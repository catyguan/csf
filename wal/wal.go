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

package wal

import (
	"errors"
	"hash/crc32"
	"time"

	"github.com/catyguan/csf/raft/raftpb"
	"github.com/catyguan/csf/wal/walpb"

	"github.com/catyguan/csf/pkg/capnslog"
)

const (
	metadataType int64 = iota + 1
	entryType
	stateType
	crcType
	snapshotType

	// warnSyncDuration is the amount of time allotted to an fsync before
	// logging a warning
	warnSyncDuration = time.Second
)

var (
	// SegmentSizeBytes is the preallocated size of each wal segment file.
	// The actual size might be larger than this. In general, the default
	// value should be used, but this is defined as an exported variable
	// so that tests can set a different segment size.
	SegmentSizeBytes int64 = 32 * 1024 * 1024
	IndexsegSize     int64 = 256 * 1024

	plog = capnslog.NewPackageLogger("github.com/catyguan/csf", "wal")

	ErrLogIndexError    = errors.New("wal: logindex format error")
	ErrMetadataError    = errors.New("wal: metadata format error")
	ErrFileNotFound     = errors.New("wal: file not found")
	ErrCRCMismatch      = errors.New("wal: crc mismatch")
	ErrSnapshotMismatch = errors.New("wal: snapshot mismatch")
	ErrSnapshotNotFound = errors.New("wal: snapshot not found")
	crcTable            = crc32.MakeTable(crc32.Castagnoli)
)

type actionOfWAL struct {
	action string
	respC  chan *respOfWAL
}

type respOfWAL struct {
	answer interface{}
	err    error
}

// WAL is a logical representation of the stable storage.
// WAL is either in read mode or append mode but not both.
// A newly created WAL is in append mode, and ready for appending records.
// A just opened WAL is in read mode, and ready for reading records.
// The WAL will be ready for appending after reading out all the previous records.
type WAL struct {
	dir string // the living directory of the underlay files

	meta *walpb.Metadata // metadata recorded at the head of each WAL

	state raftpb.HardState // hardstate recorded at the head of WAL
	enti  uint64           // index of the last entry saved to the wal

	blocks  []*logBlock
	actionC chan *actionOfWAL
}

func InitWAL(dirpath string, meta *walpb.Metadata) (*WAL, error) {
	w := &WAL{
		dir:     dirpath,
		meta:    meta,
		blocks:  make([]*logBlock, 0),
		actionC: make(chan *actionOfWAL, 512),
	}
	w.run()

	act := new(actionOfWAL)
	act.action = "init"
	_, err := w.callAction(act)

	if err != nil {
		w.Close()
		return nil, err
	}
	return w, nil
}

func (this *WAL) Close() {
	if this.actionC != nil {
		close(this.actionC)
		this.actionC = nil
	}
}

func (this *WAL) run() {
	ac := this.actionC
	go this.doRun(ac)
}
