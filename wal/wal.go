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
	"time"

	"github.com/catyguan/csf/pkg/fileutil"
)

const (
	actionAppend   int = 1
	actionClose    int = 2
	actionSync     int = 3
	actionDelete   int = 4
	actionReset    int = 5
	actionTruncate int = 6
)

type walCursor interface {
	InBlock(lb *logBlock) bool
	CloseByWAL()
}

type walReq struct {
	action int
	p1     interface{}
	p2     interface{}
	p3     interface{}
	resp   chan walResp
}

type walResp struct {
	r1  interface{}
	r2  interface{}
	err error
}

type walcore struct {
	cfg       *Config
	lastIndex uint64 // atomic access
	blocks    []*logBlock
	cursors   *list.List

	reqCh  chan *walReq
	closed uint64
}

func initWALCore(cfg *Config) (w *walcore, metadata []byte, err error) {
	w = &walcore{
		cfg:       cfg,
		lastIndex: 0,
		blocks:    make([]*logBlock, 0),
		reqCh:     make(chan *walReq, cfg.WALQueueSize),
		closed:    0,
	}
	_, metadata, err = w.doInit()

	if err != nil {
		w.doClose()
		return nil, nil, err
	}

	go w.run()

	return w, metadata, nil
}

func (this *walcore) tailBlock() *logBlock {
	return this.blocks[len(this.blocks)-1]
}

func (this *walcore) doInit() (bool, []byte, error) {
	newone := false
	if !ExistDir(this.cfg.Dir) {
		plog.Infof("init walcore at %v", this.cfg.Dir)
		err := fileutil.CreateDirAll(this.cfg.Dir)
		if err != nil {
			plog.Warningf("init walcore fail - %v", err)
			return false, nil, err
		}
	}
	var rmeta []byte
	flb := newFirstBlock(this.cfg.Dir)
	if !flb.IsExists() {
		// init first wal block
		plog.Infof("init WAL block [0]")
		err := flb.Create(this.cfg.InitMetadata)
		if err != nil {
			plog.Warningf("init WAL block [0] fail - %v", err)
		}
		this.doAppendBlock(flb)
		this.lastIndex = 0
		rmeta = this.cfg.InitMetadata
		newone = true
	} else {
		lb := flb
		for {
			llb := !lb.IsExistsNextBlock()
			err := lb.Open(llb)
			if err != nil {
				plog.Warningf("open WAL block[%v] fail - %v", lb.id, err)
				return false, nil, err
			}
			if rmeta == nil {
				rmeta = lb.Header.MetaData
			}
			this.doAppendBlock(lb)
			if llb {
				this.lastIndex = lb.LastIndex()
				break
			}
			lb = lb.newNextBlock()
		}
	}

	return newone, rmeta, nil
}

func (this *walcore) run() {
	ti := time.NewTicker(this.cfg.IdleSyncDuration)
	defer func() {
		ti.Stop()
		this.doClose()
	}()

	for {
		idle := false
		select {
		case req := <-this.reqCh:
			idle = false
			switch req.action {
			case actionClose:
				this.doSync()
				close(req.resp)
				return
			case actionSync:
				err := this.doSync()
				if req.resp != nil {
					req.resp <- walResp{err: err}
				}
			case actionAppend:
				err := this.doAppend(req.p1.(uint64), req.p2.([]byte))
				if req.resp != nil {
					req.resp <- walResp{err: err}
				}
			case actionDelete:
				err := this.doDelete()
				if req.resp != nil {
					req.resp <- walResp{err: err}
				}
			case actionReset:
				err := this.doReset()
				if req.resp != nil {
					req.resp <- walResp{err: err}
				}
			case actionTruncate:
				err := this.doTruncate(req.p1.(uint64))
				if req.resp != nil {
					req.resp <- walResp{err: err}
				}
			}
		case <-ti.C:
			if idle {
				this.doSync()
			} else {
				idle = true
			}
		}
	}
}

func (this *walcore) isClosed() bool {
	return atomic.LoadUint64(&this.closed) == 1
}

func (this *walcore) doClose() {
	close(this.reqCh)
	if this.cursors != nil {
		for e := this.cursors.Front(); e != nil; e = e.Next() {
			e.Value.(walCursor).CloseByWAL()
		}
		this.cursors = nil
	}

	for _, lb := range this.blocks {
		lb.Close()
	}
	this.blocks = make([]*logBlock, 0)

	atomic.StoreUint64(&this.lastIndex, 0)
}

func (this *walcore) seekBlock(idx uint64) *logBlock {
	c := len(this.blocks)
	for i := c - 1; i >= 0; i-- {
		lb := this.blocks[i]
		if idx > lb.Header.Index {
			return lb
		}
	}
	return this.blocks[0]
}

func (this *walcore) doRoll(lb *logBlock) error {
	if err := lb.Sync(); err != nil {
		plog.Warningf("Cut(%v) sync fail - %v", lb.id, err)
		return err
	}
	lb.Close()

	nlb := lb.newNextBlock()
	if err := nlb.Create(lb.Header.MetaData); err != nil {
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

func (this *walcore) doAddCursor(c walCursor) {
	if this.cursors == nil {
		this.cursors = list.New()
	}
	this.cursors.PushBack(c)
}

func (this *walcore) doCloseCursor(lb *logBlock) {
	if this.cursors == nil {
		return
	}
	var n *list.Element
	for e := this.cursors.Front(); e != nil; {
		c := e.Value.(walCursor)
		if c.InBlock(lb) {
			n = e.Next()
			this.cursors.Remove(e)
			e = n
			c.CloseByWAL()
		} else {
			e = e.Next()
		}
	}
}

func (this *walcore) doRemoveCursor(p walCursor) {
	if this.cursors == nil {
		return
	}
	var n *list.Element
	for e := this.cursors.Front(); e != nil; {
		c := e.Value.(walCursor)
		if c == p {
			n = e.Next()
			this.cursors.Remove(e)
			e = n
		} else {
			e = e.Next()
		}
	}
}

func (this *walcore) newCursor(idx uint64) (*cursor, error) {
	lb := this.seekBlock(idx)
	return newCursor(this, lb, idx)
}

// WAL implements
func NewWAL(cfg *Config) (WAL, []byte, error) {
	return initWALCore(cfg)
}

// Append
func (this *walcore) Append(idx uint64, data []byte) error {
	if this.isClosed() {
		return ErrClosed
	}
	req := &walReq{
		action: actionAppend,
		p1:     idx,
		p2:     data,
	}
	if !this.cfg.PostAppend {
		req.resp = make(chan walResp, 1)
		this.reqCh <- req
		resp := <-req.resp
		return resp.err
	} else {
		this.reqCh <- req
		return nil
	}
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

// Sync blocks
func (this *walcore) doSync() error {
	lb := this.tailBlock()
	return lb.Sync()
}
func (this *walcore) Sync() error {
	if this.isClosed() {
		return ErrClosed
	}
	req := &walReq{
		action: actionSync,
		resp:   make(chan walResp, 1),
	}
	this.reqCh <- req
	resp := <-req.resp
	return resp.err
}

func (this *walcore) Close() error {
	atomic.StoreUint64(&this.closed, 1)
	req := &walReq{
		action: actionClose,
		resp:   make(chan walResp, 0),
	}
	this.reqCh <- req
	<-req.resp
	return nil
}

// Delete permanently closes the log by deleting all data
func (this *walcore) Delete() error {
	if this.isClosed() {
		return ErrClosed
	}
	req := &walReq{
		action: actionDelete,
		resp:   make(chan walResp, 1),
	}
	this.reqCh <- req
	resp := <-req.resp
	return resp.err
}

func (this *walcore) doDelete() error {
	c := len(this.blocks)
	if c > 0 {
		return nil
	}
	for i := c - 1; i >= 0; i-- {
		lb := this.blocks[i]
		// this.doCloseCursor(lb)
		err := lb.Remove()
		if err != nil {
			this.blocks = this.blocks[:i]
			return err
		}
	}
	this.doClose()
	return nil
}

// Reset destructively clears out any pending data in the log
func (this *walcore) Reset() error {
	if this.isClosed() {
		return ErrClosed
	}
	req := &walReq{
		action: actionReset,
		resp:   make(chan walResp, 1),
	}
	this.reqCh <- req
	resp := <-req.resp
	return resp.err
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

// Truncate to index
func (this *walcore) Truncate(idx uint64) error {
	if this.isClosed() {
		return ErrClosed
	}
	req := &walReq{
		action: actionTruncate,
		p1:     idx,
		resp:   make(chan walResp, 1),
	}
	this.reqCh <- req
	resp := <-req.resp
	return resp.err
}
func (this *walcore) doTruncate(idx uint64) error {

	tlb := this.seekBlock(idx)

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

// Index returns the last index
func (this *walcore) LastIndex() uint64 {
	return atomic.LoadUint64(&this.lastIndex)
}

// GetCursor returns a Cursor at the specified index
func (this *walcore) GetCursor(idx uint64) (Cursor, error) {
	c, err := this.newCursor(idx)
	if err != nil {
		return nil, err
	}
	this.doAddCursor(c)
	return c, nil
}
