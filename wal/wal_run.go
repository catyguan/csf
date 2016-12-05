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
	"container/list"
	"sync/atomic"

	"github.com/catyguan/csf/pkg/idleticker"
)

func (this *walcore) result(err error) Result {
	return Result{Index: this.LastIndex(), Err: err}
}

func (this *walcore) tailBlock() *logBlock {
	return this.blocks[len(this.blocks)-1]
}

func (this *walcore) doSeekBlock(idx uint64) *logBlock {
	c := len(this.blocks)
	for i := c - 1; i >= 0; i-- {
		lb := this.blocks[i]
		if idx > lb.Header.Index {
			return lb
		}
	}
	return this.blocks[0]
}

func (this *walcore) run() {
	ti := idleticker.NewTicker(this.cfg.IdleSyncDuration)
	defer func() {
		ti.Stop()
		this.doClose()
		this.emu.Lock()
		this.econd.Broadcast()
		this.emu.Unlock()
		close(this.reqCh)
	}()

	for {
		select {
		case req := <-this.reqCh:
			ti.Reset()
			switch req.action {
			case actionClose:
				this.doSync()
				close(req.resp)
				return
			case actionSync:
				err := this.doSync()
				if req.resp != nil {
					req.resp <- this.result(err)
				}
			case actionAppend:
				err := this.doAppendEnts(req.p1.([]Entry), req.p2.(bool))
				if req.resp != nil {
					req.resp <- this.result(err)
				}
			case actionReset:
				err := this.doReset()
				if req.resp != nil {
					req.resp <- this.result(err)
				}
			case actionTruncate:
				err := this.doTruncate(req.p1.(uint64))
				if req.resp != nil {
					req.resp <- this.result(err)
				}
			case actionCursor:
				c, err := this.doNewCursor(req.p1.(uint64))
				if req.resp != nil {
					req.resp <- Result{Err: err, data: c}
				}
			}
		case <-ti.C:
			if this.needSync {
				plog.Infof("execute idle sync")
				this.doSync()
			}
		}
	}
}

func (this *walcore) doClose() {
	// close(this.reqCh)
	this.cmu.Lock()
	if this.cursors != nil {
		for e := this.cursors.Front(); e != nil; e = e.Next() {
			e.Value.(*cursor).walClose()
		}
		this.cursors = nil
	}
	this.cmu.Unlock()

	for _, lb := range this.blocks {
		lb.Close()
	}
	this.blocks = make([]*logBlock, 0)

	atomic.StoreUint64(&this.lastIndex, 0)
}

func (this *walcore) doRoll(lb *logBlock) error {
	if err := lb.Sync(); err != nil {
		plog.Warningf("Cut(%v) sync fail - %v", lb.id, err)
		return err
	}
	lb.Close()

	nlb := lb.newNextBlock()
	if err := nlb.Create(lb.Header.MetaData, this.cfg.BlockRollSize); err != nil {
		plog.Warningf("create new WAL block [%v] fail - %v", nlb.id, err)
		return err
	}
	this.doAppendBlock(nlb)

	if err := nlb.Sync(); err != nil {
		return err
	}
	plog.Infof("segmented WAL block[%v] is created", nlb.id)
	return nil
}

func (this *walcore) doAppendBlock(lb *logBlock) {
	this.blocks = append(this.blocks, lb)
}

func (this *walcore) doAddCursor(c *cursor) {
	this.cmu.Lock()
	defer this.cmu.Unlock()
	if this.cursors == nil {
		this.cursors = list.New()
	}
	this.cursors.PushBack(c)
}

func (this *walcore) doCloseCursor(lb *logBlock) {
	this.execCloseCursor(lb)
	this.emu.Lock()
	this.econd.Broadcast()
	this.emu.Unlock()
}

func (this *walcore) execCloseCursor(lb *logBlock) {
	this.cmu.Lock()
	defer this.cmu.Unlock()
	if this.cursors == nil {
		return
	}
	var n *list.Element
	for e := this.cursors.Front(); e != nil; {
		c := e.Value.(*cursor)
		if c.walBlock(lb) {
			n = e.Next()
			this.cursors.Remove(e)
			e = n
			c.walClose()
		} else {
			e = e.Next()
		}
	}
}

func (this *walcore) doRemoveCursor(p *cursor) {
	this.cmu.Lock()
	defer this.cmu.Unlock()
	if this.cursors == nil {
		return
	}
	var n *list.Element
	for e := this.cursors.Front(); e != nil; {
		c := e.Value.(*cursor)
		if c == p {
			n = e.Next()
			this.cursors.Remove(e)
			e = n
		} else {
			e = e.Next()
		}
	}
}

func (this *walcore) doAppendEnts(ents []Entry, sync bool) error {
	defer func() {
		this.emu.Lock()
		this.econd.Broadcast()
		this.emu.Unlock()
	}()
	for _, e := range ents {
		err := this.doAppend(e.Index, e.Data)
		if err != nil {
			return err
		}
		this.needSync = true
	}
	if sync {
		err := this.doSync()
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *walcore) doAppend(idx uint64, data []byte) error {
	lb := this.tailBlock()

	li, err := lb.Append(idx, data)
	if err != nil {
		return err
	}
	atomic.StoreUint64(&this.lastIndex, li.Index)

	curOff := li.EndPos()
	if curOff < this.cfg.BlockRollSize {
		return nil
	}
	return this.doRoll(lb)
}

func (this *walcore) doSync() error {
	lb := this.tailBlock()
	err := lb.Sync()
	if err == nil {
		this.needSync = false
	}
	return err
}

func (this *walcore) doDelete() error {
	c := len(this.blocks)
	for i := c - 1; i >= 0; i-- {
		lb := this.blocks[i]
		this.doCloseCursor(lb)
		err := lb.Remove()
		if err != nil {
			this.blocks = this.blocks[:i]
			return err
		}
	}
	this.doClose()
	return nil
}

func (this *walcore) doReset() error {
	if err := this.doDelete(); err != nil {
		return err
	}
	if _, _, err := this.doInit(); err != nil {
		return err
	}
	return nil
}

func (this *walcore) doTruncate(idx uint64) error {

	tlb := this.doSeekBlock(idx)

	c := len(this.blocks)
	for i := c - 1; i > tlb.id; i-- {
		lb := this.blocks[i]
		this.doCloseCursor(lb)
		err := lb.Remove()
		if err != nil {
			this.blocks = this.blocks[:i]
			return err
		}
		plog.Infof("drop WAL block[%v, %v]", lb.id, lb.Header.Index)
	}
	this.blocks = this.blocks[:tlb.id+1]
	// Truncate
	this.doCloseCursor(tlb)
	lidx, err := tlb.Truncate(idx)
	atomic.StoreUint64(&this.lastIndex, lidx)
	return err
}

func (this *walcore) doNewCursor(idx uint64) (*cursor, error) {
	if idx > this.lastIndex {
		return nil, nil
	}
	lb := this.doSeekBlock(idx)
	c, err := newCursor(this, lb, idx)
	if err != nil {
		return nil, err
	}
	this.doAddCursor(c)
	return c, nil
}
