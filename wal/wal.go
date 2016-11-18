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
	"sync"
	"sync/atomic"
	"time"

	"github.com/catyguan/csf/raft/raftpb"

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
	MaxRecordSize    int64 = 0x00FFFFFF

	plog = capnslog.NewPackageLogger("github.com/catyguan/csf", "wal")

	ErrLogIndexError    = errors.New("wal: logindex format error")
	ErrLogHeaderError   = errors.New("wal: logheader format error")
	ErrFileNotFound     = errors.New("wal: file not found")
	ErrCRCMismatch      = errors.New("wal: crc mismatch")
	ErrSnapshotMismatch = errors.New("wal: snapshot mismatch")
	ErrSnapshotNotFound = errors.New("wal: snapshot not found")
	crcTable            = crc32.MakeTable(crc32.Castagnoli)
)

type actionOfWAL struct {
	action string
	p1     interface{}
	p2     interface{}
	p3     interface{}
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

	clusterId uint64

	mu                   sync.RWMutex
	firstIndex           uint64           // atom
	lastIndex            uint64           // atom
	state                raftpb.HardState // hardstate recorded at the head of WAL
	initConfState        raftpb.ConfState
	lastSnapshotName     string
	lastSnapshotMetadata *raftpb.SnapshotMetadata

	snapshotter *Snapshotter
	blocks      []*logBlock
	actionC     chan *actionOfWAL
}

func InitWAL(dirpath string, clusterId uint64) (*WAL, bool, error) {
	w := &WAL{
		dir:        dirpath,
		clusterId:  clusterId,
		blocks:     make([]*logBlock, 0),
		actionC:    make(chan *actionOfWAL, 512),
		firstIndex: 1,
		lastIndex:  0,
	}
	w.run()

	act := new(actionOfWAL)
	act.action = "init"
	r, err := w.callAction(act)

	if err != nil {
		w.Close()
		return nil, false, err
	}
	w.snapshotter = NewSnapshotter(w.dir)
	return w, r.(bool), nil
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

func (this *WAL) Begin() error {
	act := new(actionOfWAL)
	act.action = "begin"
	_, err := this.callAction(act)

	if err != nil {
		return err
	}
	return nil
}

func (this *WAL) Save(st raftpb.HardState, entries []raftpb.Entry) error {
	act := new(actionOfWAL)
	act.action = "save"
	act.p1 = &st
	act.p2 = entries
	_, err := this.callAction(act)

	if err != nil {
		return err
	}
	return nil
}

func (this *WAL) tailBlock() *logBlock {
	return this.blocks[len(this.blocks)-1]
}

// InitialState returns the saved HardState and ConfState information.
func (this *WAL) InitialState() (raftpb.HardState, raftpb.ConfState, error) {
	return this.state, this.initConfState, nil
}

// Entries returns a slice of log entries in the range [lo,hi).
// MaxSize limits the total size of the log entries returned, but
// Entries returns at least one entry if any.
func (this *WAL) Entries(lo, hi, maxSize uint64) ([]raftpb.Entry, error) {
	act := new(actionOfWAL)
	act.action = "entries"
	act.p1 = lo
	act.p2 = hi
	act.p3 = maxSize
	r, err := this.callAction(act)

	if err != nil {
		return nil, err
	}
	return r.([]raftpb.Entry), nil
}

// Term returns the term of entry i, which must be in the range
// [FirstIndex()-1, LastIndex()]. The term of the entry before
// FirstIndex is retained for matching purposes even though the
// rest of that entry may not be available.
func (this *WAL) Term(i uint64) (uint64, error) {
	act := new(actionOfWAL)
	act.action = "term"
	act.p1 = i
	r, err := this.callAction(act)

	if err != nil {
		return 0, err
	}
	return r.(uint64), nil
}

// LastIndex returns the index of the last entry in the log.
func (this *WAL) LastIndex() (uint64, error) {
	r := atomic.LoadUint64(&this.lastIndex)
	return r, nil
}

// FirstIndex returns the index of the first log entry that is
// possibly available via Entries (older entries have been incorporated
// into the latest Snapshot; if storage only contains the dummy entry the
// first log entry is not available).
func (this *WAL) FirstIndex() (uint64, error) {
	r := atomic.LoadUint64(&this.firstIndex)
	return r, nil
}

// Snapshot returns the most recent snapshot.
// If snapshot is temporarily unavailable, it should return ErrSnapshotTemporarilyUnavailable,
// so raft state machine could know that Storage needs some time to prepare
// snapshot and call Snapshot later.
func (this *WAL) Snapshot() (raftpb.Snapshot, error) {
	act := new(actionOfWAL)
	act.action = "snapshot"
	act.p1 = uint64(0)
	act.p2 = this.lastSnapshotName
	r, err := this.callAction(act)

	if err != nil {
		return raftpb.Snapshot{}, err
	}
	snap := r.(*raftpb.Snapshot)
	return *snap, nil
}
