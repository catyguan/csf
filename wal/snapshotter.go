// Copyright 2015 The CSF Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package snap stores raft nodes' states with snapshots.
package wal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/catyguan/csf/pkg/fileutil"
	"github.com/catyguan/csf/pkg/pbutil"
	"github.com/catyguan/csf/raft"
	"github.com/catyguan/csf/raft/raftpb"
)

const (
	snapSuffix = ".snap"
)

var (
	ErrNoSnapshot    = errors.New("snap: no available snapshot")
	ErrEmptySnapshot = errors.New("snap: empty snapshot")
)

type Snapshotter struct {
	dir string
}

func NewSnapshotter(dir string) *Snapshotter {
	return &Snapshotter{
		dir: dir,
	}
}

func (s *Snapshotter) SnapFilePath(id uint64) string {
	fname := fmt.Sprintf("%016x%s", id, snapSuffix)
	return filepath.Join(s.dir, fname)
}

func (s *Snapshotter) SaveSnap(snapshot raftpb.Snapshot) error {
	if raft.IsEmptySnap(snapshot) {
		return nil
	}
	return s.save(&snapshot)
}

func (s *Snapshotter) save(snapshot *raftpb.Snapshot) (err error) {
	fname := s.SnapFilePath(snapshot.Metadata.Index)
	b1 := pbutil.MustMarshal(snapshot)
	crc1 := crc32.Update(0, crcTable, b1)

	b2 := pbutil.MustMarshal(&snapshot.Metadata)
	crc2 := crc32.Update(0, crcTable, b2)

	f, err2 := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileutil.PrivateFileMode)
	if err2 != nil {
		return err2
	}
	defer func() {
		f.Close()
		if err != nil {
			os.Remove(fname)
		}
	}()
	hb := make([]byte, 8)

	if hb != nil {
		binary.LittleEndian.PutUint32(hb[:4], crc2)
		if _, err = f.Write(hb[:4]); err != nil {
			return err
		}
		binary.LittleEndian.PutUint64(hb, uint64(len(b2)))
		if _, err = f.Write(hb); err != nil {
			return err
		}
		n, err3 := f.Write(b2)
		if err3 != nil {
			return err3
		}
		if n != len(b2) {
			return io.ErrShortWrite
		}
	}

	if hb != nil {
		binary.LittleEndian.PutUint32(hb[:4], crc1)
		if _, err = f.Write(hb[:4]); err != nil {
			return err
		}
		binary.LittleEndian.PutUint64(hb, uint64(len(b1)))
		if _, err = f.Write(hb); err != nil {
			return err
		}
		n, err3 := f.Write(b1)
		if err3 != nil {
			return err3
		}
		if n < len(b1) {
			return io.ErrShortWrite
		}
		err = fileutil.Fsync(f)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Snapshotter) LoadLastMetadata() (string, *raftpb.SnapshotMetadata, error) {
	names, err := s.snapNames()
	if err != nil {
		return "", nil, err
	}
	var meta raftpb.SnapshotMetadata
	var rname string
	for _, name := range names {
		f, err2 := os.OpenFile(filepath.Join(s.dir, name), os.O_RDONLY, fileutil.PrivateFileMode)
		if err2 == nil {
			defer f.Close()
			if meta, err = s.readMetadata(f); err == nil {
				rname = name
				break
			}
		} else {
			err = err2
		}
		plog.Warningf("load snapshot '%v' fail - %v", name, err)
	}
	if err != nil {
		return "", nil, ErrNoSnapshot
	}
	return rname, &meta, nil
}

func (s *Snapshotter) LoadSnap(name string) (*raftpb.Snapshot, error) {
	fpath := filepath.Join(s.dir, name)
	f, err := os.OpenFile(fpath, os.O_RDONLY, fileutil.PrivateFileMode)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err = s.readMetadata(f); err != nil {
		return nil, fmt.Errorf("read snapmetadata fail - %v", err)
	}

	snap, err := s.readSnapshot(f)
	if err != nil {
		return nil, fmt.Errorf("read snapshot fail - %v", err)
	}
	return snap, nil
}

func (s *Snapshotter) readMetadata(f *os.File) (meta raftpb.SnapshotMetadata, err error) {
	defer func() {
		if err != nil {
			f.Close()
			s.renameBroken(f.Name())
		}
	}()
	hb := make([]byte, 8)

	_, err = io.ReadFull(f, hb[:4])
	if err != nil {
		return
	}
	crc := binary.LittleEndian.Uint32(hb[:4])
	_, err = io.ReadFull(f, hb)
	if err != nil {
		return
	}
	sz := binary.LittleEndian.Uint64(hb)

	buf := make([]byte, sz)
	_, err = io.ReadFull(f, buf)
	if err != nil {
		return
	}

	bcrc := crc32.Update(0, crcTable, buf)
	if crc != bcrc {
		plog.Errorf("corrupted snapshot file %v: crc mismatch", f.Name())
		err = ErrCRCMismatch
		return
	}

	err = meta.Unmarshal(buf)
	if err != nil {
		plog.Errorf("corrupted snapshot file %v: %v", f.Name(), err)
		return
	}
	return
}

// Read reads the snapshot named by snapname and returns the snapshot.
func (s *Snapshotter) readSnapshot(f *os.File) (snap *raftpb.Snapshot, err error) {
	defer func() {
		if err != nil {
			f.Close()
			s.renameBroken(f.Name())
		}
	}()
	hb := make([]byte, 8)

	_, err = io.ReadFull(f, hb[:4])
	if err != nil {
		return
	}
	crc := binary.LittleEndian.Uint32(hb[:4])
	_, err = io.ReadFull(f, hb)
	if err != nil {
		return
	}
	sz := binary.LittleEndian.Uint64(hb)

	buf := make([]byte, sz)
	_, err = io.ReadFull(f, buf)
	if err != nil {
		return
	}

	bcrc := crc32.Update(0, crcTable, buf)
	if crc != bcrc {
		plog.Errorf("corrupted snapshot file %v: crc mismatch", f.Name())
		err = ErrCRCMismatch
		return
	}

	snap = &raftpb.Snapshot{}
	err = snap.Unmarshal(buf)
	if err != nil {
		plog.Errorf("corrupted snapshot file %v: %v", f.Name(), err)
		return
	}
	return
}

// snapNames returns the filename of the snapshots in logical time order (from newest to oldest).
// If there is no available snapshots, an ErrNoSnapshot will be returned.
func (s *Snapshotter) snapNames() ([]string, error) {
	dir, err := os.Open(s.dir)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	snaps := checkSnapNames(names)
	if len(snaps) == 0 {
		return nil, nil
	}
	sort.Sort(sort.Reverse(sort.StringSlice(snaps)))
	return snaps, nil
}

func (s *Snapshotter) renameBroken(path string) {
	brokenPath := path + ".broken"
	if err := os.Rename(path, brokenPath); err != nil {
		plog.Warningf("cannot rename broken snapshot file %v to %v: %v", path, brokenPath, err)
	}
}
