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
	"github.com/catyguan/csf/pkg/fileutil"
	"github.com/catyguan/csf/raft"
	"github.com/catyguan/csf/raft/raftpb"
)

func (this *WAL) Close() {
	for _, lb := range this.blocks {
		lb.Close()
	}
	this.blocks = make([]*logBlock, 0)
	this.ents = nil
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
		err := flb.CreateAndInit(&raftpb.HardState{}, &raftpb.ConfState{})
		if err != nil {
			plog.Warningf("init WAL block [0] fail - %v", err)
		}
		this.doAppendBlock(flb)
		this.state.Reset()
		this.confState.Reset()
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

	n, lr, err := this.snapshotter.LoadLastHeader()
	if err != nil {
		return false, err
	}
	if lr == nil {
		lr = &snapHeader{}
	}

	err = this.Begin(n, lr)
	if err != nil {
		return false, err
	}

	return newone, nil
}

func (this *WAL) Begin(lastSnapshotName string, lr *snapHeader) error {
	this.state = lr.state
	this.confState = lr.confState
	this.lastSnapshotName = lastSnapshotName

	flb := this.seekBlock(lr.index)
	for i := flb.id; i < len(this.blocks); i++ {
		lb := this.blocks[i]
		ents := make([]raftpb.Entry, 0)
		err := lb.Active(func(li *logIndex, data []byte) (bool, error) {
			// read all stats & sync state
			switch int64(li.Type) {
			case stateType:
				st := raftpb.HardState{}
				err := st.Unmarshal(data)
				if err == nil {
					this.state = st
				}
			case confStateType:
				cst := raftpb.ConfState{}
				err := cst.Unmarshal(data)
				if err == nil {
					this.confState = cst
				}
			case entryType:
				if li.Index >= lr.index {
					e := raftpb.Entry{}
					err := e.Unmarshal(data)
					if err == nil {
						ents = append(ents, e)
						sz := len(ents)
						if sz > 1 {
							if ents[sz-2].Index+1 != ents[sz-1].Index {
								plog.Panicf("missing log entry [last: %d, at: %d]",
									ents[sz-2].Index, ents[sz-1].Index)
							}
						}
					}
				}
			}
			return false, nil
		})
		if err != nil {
			plog.Warningf("WAL Begin() active block[%v] fail - %v", lb.id, err)
			return err
		}
		this.doAppend(ents)
		if lb != this.tailBlock() {
			lb.Close()
		}
	}

	plog.Infof("WAL begin at %v, ents=%v, state=%v, confState=%v",
		lr.index, len(this.ents), this.state.String(), this.confState.String())

	return nil
}

// func (this *WAL) processRecord(start, end uint64, h recoderHandler) error {
// 	lbi := this.seekBlock(start)
// 	for i := lbi.id; i < len(this.blocks); i++ {
// 		lb := this.blocks[i]
// 		// plog.Infof("process block %v", lb.id)
// 		stop, err := lb.Process(start, end, h)
// 		if err != nil && err != io.EOF {
// 			return err
// 		}
// 		if stop {
// 			break
// 		}
// 	}
// 	return nil
// }

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

func mustSync(st, prevst *raftpb.HardState, entsnum int) bool {
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

func (this *WAL) Save(st *raftpb.HardState, ents []raftpb.Entry) error {
	// short cut, do not call sync
	ebst := raft.IsEmptyHardState(*st)
	if ebst && len(ents) == 0 {
		return nil
	}

	this.mu.Lock()
	defer this.mu.Unlock()

	err := this.doAppend(ents)
	if err != nil {
		return err
	}
	if !sameState(st, &this.state) {
		this.state = *st
	}

	/*
		lb, err2 := this.checkTailBlock(ents)
		if err2 != nil {
			return err2
		}

		mustSync := mustSync(st, &this.state, len(ents))

		this.doAppendEntries(ents)
		for i := range ents {
			e := &ents[i]
			b := pbutil.MustMarshal(e)
			_, err := lb.Append(e.Index, e.Term, uint8(entryType), b)
			if err != nil {
				return err
			}
			atomic.StoreUint64(&this.lastIndex, e.Index)
		}

		if !ebst {
			if !sameState(st, &this.state) {
				b := pbutil.MustMarshal(st)
				_, err := lb.Append(0, 0, uint8(stateType), b)
				if err != nil {
					return err
				}
				this.state = *st
			}
		}

		curOff := lb.Tail.EndPos()
		if curOff < uint64(SegmentSizeBytes) {
			if mustSync {
				return lb.sync()
			}
		}
		return this.Cut(lb)
	*/
	return nil
}

func (this *WAL) Cut(lb *logBlock) error {
	if err := lb.sync(); err != nil {
		return err
	}
	lb.Close()

	nlb := lb.newNextBlock()
	if err := nlb.CreateAndInit(&this.state, &this.confState); err != nil {
		plog.Warningf("create new WAL block [%v] fail - %v", nlb.id, err)
		return err
	}
	this.doAppendBlock(nlb)

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
	if tlb != nil && tlb.Tail == nil {
		err := tlb.Active(nil)
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

func (this *WAL) ApplyConfState(cst *raftpb.ConfState) error {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.confState = *cst
	return nil
}
