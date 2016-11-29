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
	"fmt"
	"hash/crc32"
	"sync"
	"time"

	"github.com/catyguan/csf/raft"
	"github.com/catyguan/csf/raft/raftpb"

	"github.com/catyguan/csf/pkg/capnslog"
)

const (
	metadataType int64 = iota + 1
	entryType
	stateType
	crcType
	snapshotType
	confStateType

	// warnSyncDuration is the amount of time allotted to an fsync before
	// logging a warning
	warnSyncDuration = time.Second
)

func logTypeString(v uint8) string {
	switch int64(v) {
	case metadataType:
		return "metadata"
	case entryType:
		return "entry"
	case stateType:
		return "state"
	case crcType:
		return "crc"
	case snapshotType:
		return "snapshot"
	case confStateType:
		return "confState"
	default:
		return fmt.Sprintf("unknow(%d)", v)
	}
}

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

	mu sync.Mutex

	state            raftpb.HardState
	confState        raftpb.ConfState
	lastSnapshotName string
	ents             []raftpb.Entry

	snapshotter *Snapshotter
	blocks      []*logBlock
}

func InitWAL(dirpath string, clusterId uint64) (*WAL, bool, error) {
	w := &WAL{
		dir:       dirpath,
		clusterId: clusterId,
		blocks:    make([]*logBlock, 0),
		ents:      make([]raftpb.Entry, 1),
	}
	w.snapshotter = NewSnapshotter(w.dir)
	r, err := w.doInit()

	if err != nil {
		w.Close()
		return nil, false, err
	}

	return w, r, nil
}

func (this *WAL) tailBlock() *logBlock {
	return this.blocks[len(this.blocks)-1]
}

// InitialState implements the Storage interface.
func (this *WAL) InitialState() (raftpb.HardState, raftpb.ConfState, error) {
	return this.state, this.confState, nil
}

// Entries implements the Storage interface.
func (this *WAL) Entries(lo, hi, maxSize uint64) ([]raftpb.Entry, error) {
	this.mu.Lock()
	defer this.mu.Unlock()
	offset := this.ents[0].Index
	if lo <= offset {
		return nil, raft.ErrCompacted
	}
	if hi > this.lastIndex()+1 {
		plog.Panicf("entries' hi(%d) is out of bound lastindex(%d)", hi, this.lastIndex())
	}
	// only contains dummy entries.
	if len(this.ents) == 1 {
		return nil, raft.ErrUnavailable
	}

	ents := this.ents[lo-offset : hi-offset]
	return raft.LimitSize(ents, maxSize), nil
}

// Term implements the Storage interface.
func (this *WAL) Term(i uint64) (uint64, error) {
	this.mu.Lock()
	defer this.mu.Unlock()
	offset := this.ents[0].Index
	if i < offset {
		return 0, raft.ErrCompacted
	}
	if int(i-offset) >= len(this.ents) {
		return 0, raft.ErrUnavailable
	}
	return this.ents[i-offset].Term, nil
}

// LastIndex implements the Storage interface.
func (this *WAL) LastIndex() (uint64, error) {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.lastIndex(), nil
}

func (this *WAL) lastIndex() uint64 {
	return this.ents[0].Index + uint64(len(this.ents)) - 1
}

// FirstIndex implements the Storage interface.
func (this *WAL) FirstIndex() (uint64, error) {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.firstIndex(), nil
}

func (this *WAL) firstIndex() uint64 {
	return this.ents[0].Index + 1
}

// Snapshot implements the Storage interface.
func (this *WAL) Snapshot() (raftpb.Snapshot, error) {
	// ms.Lock()
	// defer ms.Unlock()
	return raftpb.Snapshot{}, nil
}

// Compact discards all log entries prior to compactIndex.
// It is the application's responsibility to not attempt to compact an index
// greater than raftLog.applied.
func (this *WAL) Compact(compactIndex uint64) error {
	this.mu.Lock()
	defer this.mu.Unlock()
	offset := this.ents[0].Index
	if compactIndex <= offset {
		return raft.ErrCompacted
	}
	if compactIndex > this.lastIndex() {
		plog.Panicf("compact %d is out of bound lastindex(%d)", compactIndex, this.lastIndex())
	}

	i := compactIndex - offset
	ents := make([]raftpb.Entry, 1, 1+uint64(len(this.ents))-i)
	ents[0].Index = this.ents[i].Index
	ents[0].Term = this.ents[i].Term
	ents = append(ents, this.ents[i+1:]...)
	this.ents = ents
	return nil
}

// Append the new entries to storage.
// TODO (xiangli): ensure the entries are continuous and
// entries[0].Index > ms.entries[0].Index
func (this *WAL) doAppend(entries []raftpb.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	first := this.firstIndex()
	last := entries[0].Index + uint64(len(entries)) - 1

	// shortcut if there is no new entry.
	if last < first {
		return nil
	}
	// truncate compacted entries
	if first > entries[0].Index {
		entries = entries[first-entries[0].Index:]
	}

	offset := entries[0].Index - this.ents[0].Index
	switch {
	case uint64(len(this.ents)) > offset:
		this.ents = append([]raftpb.Entry{}, this.ents[:offset]...)
		this.ents = append(this.ents, entries...)
	case uint64(len(this.ents)) == offset:
		this.ents = append(this.ents, entries...)
	default:
		plog.Panicf("missing log entry [last: %d, append at: %d]",
			this.lastIndex(), entries[0].Index)
	}
	return nil
}
