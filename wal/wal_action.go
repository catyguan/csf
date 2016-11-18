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
	"fmt"
	"io"
	"sync/atomic"

	"github.com/catyguan/csf/pkg/fileutil"
	"github.com/catyguan/csf/pkg/pbutil"
	"github.com/catyguan/csf/raft"
	"github.com/catyguan/csf/raft/raftpb"
)

func (this *WAL) doClear() {
	for _, lb := range this.blocks {
		lb.Close()
	}
	this.blocks = make([]*logBlock, 0)
}

func (this *WAL) doRun(ac chan *actionOfWAL) {
	defer this.doClear()
	for {
		// pick
		select {
		case act := <-ac:
			if act == nil {
				plog.Infof("WAL worker stop")
				return
			}
			switch act.action {
			default:
				this.handleSomeAction(act)
			}
		}
	}
}

func (this *WAL) handleSomeAction(act *actionOfWAL) {
	switch act.action {
	case "init":
		r, err := this.doInit()
		if act.respC != nil {
			act.respC <- &respOfWAL{answer: r, err: err}
		}
	case "begin":
		err := this.doBegin()
		if act.respC != nil {
			act.respC <- &respOfWAL{err: err}
		}
	case "save":
		err := this.doSave(act.p1.(*raftpb.HardState), act.p2.([]raftpb.Entry))
		if act.respC != nil {
			act.respC <- &respOfWAL{err: err}
		}
	case "term":
		r, err := this.doTerm(act.p1.(uint64))
		if act.respC != nil {
			act.respC <- &respOfWAL{answer: r, err: err}
		}
	case "entries":
		r, err := this.doEntries(act.p1.(uint64), act.p2.(uint64), act.p3.(uint64))
		if act.respC != nil {
			act.respC <- &respOfWAL{answer: r, err: err}
		}
	default:
		plog.Warningf("unknow WAL action - %v", act.action)
	}
}

func (this *WAL) callAction(act *actionOfWAL) (interface{}, error) {
	act.respC = make(chan *respOfWAL, 1)
	if this.actionC != nil {
		this.actionC <- act
		r := <-act.respC
		return r.answer, r.err
	} else {
		return nil, fmt.Errorf("WAL closed")
	}
}

func (this *WAL) doInit() (bool, error) {
	newone := false
	if !ExistDir(this.dir) {
		plog.Infof("init WAL at %v", this.dir)
		err := fileutil.CreateDirAll(this.dir)
		if err != nil {
			plog.Warningf("init WAL fail - %v", err)
			return false, err
		}
	}
	flb := newFirstBlock(this.dir, this.clusterId)
	if !flb.IsExists() {
		// init first wal block
		plog.Infof("init WAL block [0]")
		err := flb.Create()
		if err != nil {
			plog.Warningf("init WAL block [0] fail - %v", err)
		}
		this.doAppendBlock(flb)
		newone = true
	} else {
		lb := flb
		for {
			if !lb.IsExists() {
				break
			}
			err := lb.Open()
			if err != nil {
				plog.Warningf("open WAL block[%v] fail - %v", lb.id, err)
				return false, err
			} else {
				plog.Infof("open WAL block[%v]", lb.id)
			}
			this.doAppendBlock(lb)
			lb = lb.newNextBlock()
		}
	}

	return newone, nil
}

func (this *WAL) doBegin() error {
	n, meta, err := this.snapshotter.LoadLastMetadata()
	if err != nil {
		return err
	}
	if meta == nil {
		meta = &raftpb.SnapshotMetadata{}
	}

	this.initConfState = meta.ConfState
	this.lastSnapshotName = n
	this.lastSnapshotMetadata = meta
	this.firstIndex = meta.Index + 1

	lb := this.tailBlock()
	err = lb.Active()
	if err != nil {
		return err
	}
	this.state = lb.state
	this.lastIndex = lb.LastIndex()

	return nil
}

func (this *WAL) processRecord(start, end uint64, h recoderHandler) error {
	lbi := this.seekBlock(start)
	plog.Infof("seek block %v - %v", start, lbi.id)
	for i := lbi.id; i < len(this.blocks); i++ {
		lb := this.blocks[i]
		plog.Infof("process block %v", lb.id)
		stop, err := lb.Process(start, end, h)
		if err != nil && err != io.EOF {
			return err
		}
		if stop {
			break
		}
	}
	return nil
}

func (this *WAL) seekBlock(idx uint64) *logBlock {
	c := len(this.blocks)
	for i := c - 1; i >= 0; i-- {
		lb := this.blocks[i]
		if idx > lb.Index {
			return lb
		}
	}
	return this.blocks[0]
}

func (this *WAL) seekStats(lb *logBlock, idx uint64) (*raftpb.HardState, error) {
	// lb := this.seekBlock(idx)
	// f, err := lb.readFile()
	// if err != nil {
	// 	return nil, err
	// }
	// _, err = f.Seek(0, os.SEEK_END)
	// if err != nil {
	// 	return nil, err
	// }
	// lih := createLogCoder(0)
	// for {
	// 	lih.ReadRecordToBuf(r, b)
	// }
	return nil, nil
}

func (this *WAL) doTerm(i uint64) (uint64, error) {
	// check TailBlock
	lb := this.tailBlock()
	if lb.Tail != nil {
		if lb.Tail.Index == i {
			plog.Infof("term at TailBlock.Tail")
			return lb.Tail.Term, nil
		}
	} else {
		if lb.Index == i {
			plog.Infof("term at TailBlock.Header")
			return lb.Term, nil
		}
	}

	// search
	plog.Infof("search term records")
	var r *logIndex
	err := this.processRecord(i-1, i+1, func(li *logIndex, b []byte) (bool, error) {
		if li.Index == i {
			r = li
			return true, nil
		}
		return false, nil
	})
	if err != nil && err != io.EOF {
		return 0, err
	}
	if r != nil {
		return r.Term, nil
	}
	return 0, nil
}

func (this *WAL) doEntries(lo, hi, maxSize uint64) ([]raftpb.Entry, error) {
	var r []raftpb.Entry
	var totalSize = uint64(0)
	err := this.processRecord(lo, hi, func(li *logIndex, b []byte) (bool, error) {
		if li.Type != uint8(entryType) {
			return false, nil
		}
		if li.Index >= hi {
			return true, nil
		}
		if li.Index < lo {
			return false, nil
		}
		totalSize += uint64(len(b))
		if len(r) > 0 && totalSize > maxSize {
			return true, nil
		}
		e := raftpb.Entry{}
		if err := e.Unmarshal(b); err != nil {
			return false, err
		}
		r = append(r, e)
		return false, nil
	})
	if err != nil && err != io.EOF {
		return nil, err
	}
	if r != nil {
		return r, nil
	}
	return make([]raftpb.Entry, 0), nil
}

func mustSync(st, prevst raftpb.HardState, entsnum int) bool {
	// Persistent state on all servers:
	// (Updated on stable storage before responding to RPCs)
	// currentTerm
	// votedFor
	// log entries[]
	return entsnum != 0 || st.Vote != prevst.Vote || st.Term != prevst.Term
}

func sameState(st1, st2 *raftpb.HardState) bool {
	return st1.Vote == st2.Vote && st1.Term == st2.Term && st1.Commit == st2.Commit
}

func (this *WAL) doSave(st *raftpb.HardState, ents []raftpb.Entry) error {
	// short cut, do not call sync
	if raft.IsEmptyHardState(*st) && len(ents) == 0 {
		return nil
	}

	mustSync := mustSync(*st, this.state, len(ents))

	lb, err2 := this.checkTailBlock(ents)
	if err2 != nil {
		return err2
	}
	for i := range ents {
		e := &ents[i]
		b := pbutil.MustMarshal(e)
		_, err := lb.Append(e.Index, e.Term, uint8(entryType), b)
		if err != nil {
			return err
		}
		atomic.StoreUint64(&this.lastIndex, e.Index)
	}

	if !raft.IsEmptyHardState(*st) {
		this.state = *st
		if !sameState(st, &lb.state) {
			_, err := lb.AppendState(0, 0, st)
			if err != nil {
				return err
			}
		}
	}

	curOff := lb.Tail.EndPos()

	if curOff < uint64(SegmentSizeBytes) {
		if mustSync {
			return lb.sync()
		}
		return nil
	}
	return this.cut(lb)
}

func (this *WAL) cut(lb *logBlock) error {
	if err := lb.sync(); err != nil {
		return err
	}
	lb.Close()

	nlb := lb.newNextBlock()
	if err := nlb.Create(); err != nil {
		plog.Warningf("create new WAL block [%v] fail - %v", nlb.id, err)
		return err
	}
	this.doAppendBlock(nlb)

	if !raft.IsEmptyHardState(this.state) {
		_, err := nlb.AppendState(0, 0, &this.state)
		if err != nil {
			return err
		}
	}
	if err := nlb.sync(); err != nil {
		return err
	}
	plog.Infof("segmented WAL block[%v] is created", nlb.id)
	return nil
}

func (this *WAL) doAppendBlock(lb *logBlock) {
	this.blocks = append(this.blocks, lb)
}

func (this *WAL) checkTailBlock(ents []raftpb.Entry) (*logBlock, error) {
	tlb := this.tailBlock()
	if len(ents) == 0 {
		return tlb, nil
	}
	if tlb != nil {
		err := tlb.Active()
		if err != nil {
			return nil, err
		}
	}

	e := &ents[0]
	if e.Index > tlb.LastIndex() {
		return tlb, nil
	}
	// 覆盖原有的数据
	plog.Infof("entry[%v, %v] overwrite WAL block[%v, %v]", e.Index, e.Term, tlb.id, tlb.Index)
	// find lb
	c := len(this.blocks)
	for i := c - 1; i >= 0; i-- {
		tlb = this.blocks[i]
		if e.Index > tlb.Index {
			break
		}
	}
	for i := c - 1; i > tlb.id; i-- {
		lb := this.blocks[i]
		err := lb.Remove()
		if err != nil {
			this.blocks = this.blocks[:i]
			return nil, err
		}
		plog.Infof("drop WAL block[%v, %v]", lb.id, lb.Index)
	}
	this.blocks = this.blocks[:tlb.id+1]
	// Truncate
	err := tlb.Truncate(e.Index)
	return tlb, err
}
