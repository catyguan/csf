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
	"sync"
	"sync/atomic"

	"github.com/catyguan/csf/pkg/fileutil"
)

const (
	actionAppend         int = 1
	actionClose          int = 2
	actionSync           int = 3
	actionReset          int = 5
	actionTruncate       int = 6
	actionCursor         int = 7
	actionAddListener    int = 8
	actionRemoveListener int = 9
)

type walReq struct {
	action int
	p1     interface{}
	p2     interface{}
	p3     interface{}
	resp   chan Result
}

type walcore struct {
	cfg       *Config
	lastIndex uint64 // atomic access
	blocks    []*logBlock
	cursors   *list.List
	cmu       sync.Mutex

	reqCh    chan *walReq
	closed   uint64
	needSync bool
	llist    []WALListener
	newone   bool
}

func errClosedResult() <-chan Result {
	r := make(chan Result, 1)
	r <- Result{Err: ErrClosed}
	return r
}

func initWALCore(cfg *Config) (w *walcore, metadata []byte, err error) {
	w = &walcore{
		cfg:       cfg,
		lastIndex: 0,
		blocks:    make([]*logBlock, 0),
		reqCh:     make(chan *walReq, cfg.WALQueueSize),
		closed:    0,
	}
	metadata, err = w.doInit()

	if err != nil {
		w.doClose()
		return nil, nil, err
	}

	go w.run()

	return w, metadata, nil
}

func (this *walcore) doInit() ([]byte, error) {
	this.newone = false
	if !ExistDir(this.cfg.Dir) {
		plog.Infof("init walcore at %v", this.cfg.Dir)
		err := fileutil.CreateDirAll(this.cfg.Dir)
		if err != nil {
			plog.Warningf("init walcore fail - %v", err)
			return nil, err
		}
	}
	var rmeta []byte
	flb := newFirstBlock(this.cfg.Dir)
	if !flb.IsExists() {
		// init first wal block
		plog.Infof("init WAL block [0]")
		err := flb.Create(this.cfg.InitMetadata, this.cfg.BlockRollSize)
		if err != nil {
			plog.Warningf("init WAL block [0] fail - %v", err)
		}
		this.doAppendBlock(flb)
		this.lastIndex = 0
		rmeta = this.cfg.InitMetadata
		this.newone = true
	} else {
		lb := flb
		for {
			llb := !lb.IsExistsNextBlock()
			err := lb.Open(llb)
			if err != nil {
				plog.Warningf("open WAL block[%v] fail - %v", lb.id, err)
				return nil, err
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

	return rmeta, nil
}

func (this *walcore) isClosed() bool {
	return atomic.LoadUint64(&this.closed) == 1
}

// WAL implements
func NewWAL(cfg *Config) (WAL, []byte, error) {
	return initWALCore(cfg)
}

func (this *walcore) IsNew() bool {
	return this.newone
}

// Append
func (this *walcore) Append(ents []Entry, sync bool) <-chan Result {
	if this.isClosed() {
		return errClosedResult()
	}
	r := make(chan Result, 1)
	req := &walReq{
		action: actionAppend,
		p1:     ents,
		p2:     sync,
		resp:   r,
	}
	this.reqCh <- req
	return r
}

// Sync blocks
func (this *walcore) Sync() <-chan Result {
	if this.isClosed() {
		return errClosedResult()
	}
	r := make(chan Result, 1)
	req := &walReq{
		action: actionSync,
		resp:   r,
	}
	this.reqCh <- req
	return r
}

func (this *walcore) Close() {
	if this.isClosed() {
		return
	}
	atomic.StoreUint64(&this.closed, 1)
	r := make(chan Result, 1)
	req := &walReq{
		action: actionClose,
		resp:   r,
	}
	this.reqCh <- req
}

// Reset destructively clears out any pending data in the log
func (this *walcore) Reset() <-chan Result {
	if this.isClosed() {
		return errClosedResult()
	}
	r := make(chan Result, 1)
	req := &walReq{
		action: actionReset,
		resp:   r,
	}
	this.reqCh <- req
	return r
}

// Truncate to index
func (this *walcore) Truncate(idx uint64) <-chan Result {
	if this.isClosed() {
		return errClosedResult()
	}
	r := make(chan Result, 1)
	req := &walReq{
		action: actionTruncate,
		p1:     idx,
		resp:   r,
	}
	this.reqCh <- req
	return r
}

func (this *walcore) createCursor(idx uint64) (*cursor, error) {
	if this.isClosed() {
		return nil, ErrClosed
	}
	r := make(chan Result, 1)
	req := &walReq{
		action: actionCursor,
		p1:     idx,
		resp:   r,
	}
	this.reqCh <- req
	rs := <-r
	return rs.data.(*cursor), rs.Err
}

// Index returns the last index
func (this *walcore) LastIndex() uint64 {
	return atomic.LoadUint64(&this.lastIndex)
}

// GetCursor returns a Cursor at the specified index
func (this *walcore) GetCursor(idx uint64) (Cursor, error) {
	c, err := this.createCursor(idx)
	if err != nil {
		return nil, err
	}
	r := &cursorImpl{
		w: this,
		c: c,
	}
	return r, nil
}

func (this *walcore) AddListener(lis WALListener) (uint64, error) {
	if this.isClosed() {
		return 0, ErrClosed
	}
	r := make(chan Result, 1)
	req := &walReq{
		action: actionAddListener,
		p1:     lis,
		resp:   r,
	}
	this.reqCh <- req
	rs := <-r
	return rs.data.(uint64), rs.Err
}

func (this *walcore) RemoveListener(lis WALListener) {
	if this.isClosed() {
		return
	}
	r := make(chan Result, 1)
	req := &walReq{
		action: actionRemoveListener,
		p1:     lis,
		resp:   r,
	}
	this.reqCh <- req
	<-r
}
