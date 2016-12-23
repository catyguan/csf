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
package snapshot

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/catyguan/csf/pkg/capnslog"
	"github.com/catyguan/csf/pkg/fileutil"
)

const (
	snapSuffix = ".snap"
)

var (
	ErrNoSnapshot    = errors.New("snap: no available snapshot")
	ErrEmptySnapshot = errors.New("snap: empty snapshot")

	plog = capnslog.NewPackageLogger("github.com/catyguan/csf", "snap")
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

func (s *Snapshotter) SaveSnap(sh *SnapHeader, data []byte) (err error) {
	sh.DataSize = uint64(len(data))

	fname := s.SnapFilePath(sh.Index)

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

	err = sh.Write(f)
	if err != nil {
		return
	}
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	err = fileutil.Fsync(f)
	if err != nil {
		return err
	}
	return nil
}

func (s *Snapshotter) LoadLastHeader() (string, *SnapHeader, error) {
	names, err := s.snapNames()
	if err != nil {
		return "", nil, err
	}
	var sh *SnapHeader
	var rname string
	for _, name := range names {
		f, err2 := os.OpenFile(filepath.Join(s.dir, name), os.O_RDONLY, fileutil.PrivateFileMode)
		if err2 == nil {
			defer f.Close()
			if sh, err = s.readHeader(f); err == nil {
				rname = name
				break
			}
		} else {
			err = err2
		}
		plog.Warningf("read snapshot '%v' header fail - %v", name, err)
	}
	if err != nil {
		return "", nil, ErrNoSnapshot
	}
	return filepath.Join(s.dir, rname), sh, nil
}

func (s *Snapshotter) onError(err error, f *os.File) {
	if err != nil {
		f.Close()
		s.renameBroken(f.Name())
	}
}

func (s *Snapshotter) LoadSnap(idx uint64) (sh *SnapHeader, data []byte, err error) {
	return s.LoadSnapFile(s.SnapFilePath(idx))
}

func (s *Snapshotter) LoadSnapFile(fpath string) (sh *SnapHeader, data []byte, err error) {
	f, err2 := os.OpenFile(fpath, os.O_RDONLY, fileutil.PrivateFileMode)
	if err2 != nil {
		return nil, nil, err2
	}
	sh, err = s.readHeader(f)
	if err != nil {
		return nil, nil, fmt.Errorf("read snapshot header fail - %v", err)
	}
	defer func() {
		s.onError(err, f)
	}()
	data = make([]byte, sh.DataSize)
	n, err3 := io.ReadFull(f, data)
	if err3 != nil {
		return nil, nil, fmt.Errorf("read snapshot fail - %v", err3)
	}
	if uint64(n) != sh.DataSize {
		return nil, nil, io.ErrUnexpectedEOF
	}
	return sh, data, nil
}

func (s *Snapshotter) readHeader(f *os.File) (lr *SnapHeader, err error) {
	defer func() {
		s.onError(err, f)
	}()
	lr = &SnapHeader{}
	err = lr.Read(f)
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
