// Copyright 2016 The CSF Authors
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

package masterslave

import (
	"errors"
	"io/ioutil"
	"sync"
	"time"

	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/idleticker"
	"github.com/catyguan/csf/storage4si"
)

var (
	ErrSessionNotExists = errors.New("session not exists")
)

type slaveAgent struct {
	id        uint64
	lastIndex uint64
	lastError error
	ling      bool
	cursor    interface{}
	requests  []*corepb.Request
	reqc      chan bool
	mu        sync.Mutex
	ti        *idleticker.Ticker
}

func (this *slaveAgent) impl() {
	_ = storage4si.StorageListener(this)
}

func (this *slaveAgent) beError(err string) {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.lastError = errors.New(err)
}

func (this *slaveAgent) checkError() (bool, error) {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.ling, this.lastError
}

func (this *slaveAgent) CopyR(sz int) []*corepb.Request {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.doCopyR(sz)
}

func (this *slaveAgent) doCopyR(sz int) []*corepb.Request {
	// must lock before invoke
	if len(this.requests) == 0 {
		return nil
	}
	s := sz
	if len(this.requests) < sz {
		s = len(this.requests)
	}
	r := make([]*corepb.Request, s)
	copy(r, this.requests[:s])
	if s < len(this.requests) {
		l := len(this.requests) - s
		copy(this.requests[:l], this.requests[s:])
		this.requests = this.requests[:l]
	} else {
		this.requests = this.requests[:0]
	}
	return r
}

func (this *slaveAgent) OnReset() {
	this.beError("reseted")
}

func (this *slaveAgent) OnTruncate(idx uint64) {
	this.beError("truncated")
}

func (this *slaveAgent) OnSaveRequest(idx uint64, req *corepb.Request) {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.requests = append(this.requests, req)
	if this.reqc == nil {
		this.reqc = make(chan bool, 1)
	}
	select {
	case this.reqc <- true:
	default:
	}
}

func (this *slaveAgent) OnSaveSanepshot(idx uint64) {
}

func (this *slaveAgent) OnClose() {
	this.beError("closed")
}

type masterEP struct {
	storage storage4si.Storage
	cfg     *MasterConfig

	slaveId uint64
	slaves  map[uint64]*slaveAgent
	mu      sync.RWMutex
}

func newMasterEP(cfg *MasterConfig) *masterEP {
	r := new(masterEP)
	r.storage = cfg.Storage
	r.cfg = cfg
	r.slaveId = uint64(time.Now().UnixNano())
	r.slaves = make(map[uint64]*slaveAgent)
	return r
}

func (this *masterEP) lookup(sid uint64) (*slaveAgent, error) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	if sa, ok := this.slaves[sid]; ok {
		return sa, nil
	}
	return nil, ErrSessionNotExists
}

func (this *masterEP) Begin() uint64 {
	this.mu.Lock()
	defer this.mu.Unlock()

	this.slaveId++
	sid := this.slaveId
	sa := &slaveAgent{}
	sa.id = sid
	sa.reqc = make(chan bool, 1)
	sa.ti = idleticker.NewTicker(this.cfg.SessionExpire)
	sa.ti.OnIdle(func() {
		this.End(sa.id)
	})
	this.slaves[sid] = sa

	return sid
}

func (this *masterEP) closeAgent(sa *slaveAgent) {
	if sa.ti != nil {
		sa.ti.Stop()
	}
	this.storage.RemoveListener(sa)
	if sa.cursor != nil {
		this.storage.EndLoad(sa.cursor)
		sa.cursor = nil
	}
}

func (this *masterEP) End(sid uint64) {
	this.mu.Lock()
	sa, ok := this.slaves[sid]
	if ok {
		delete(this.slaves, sid)
	}
	this.mu.Unlock()
	if sa != nil {
		this.closeAgent(sa)
	}
}

func (this *masterEP) LastSnapshot(sid uint64) (uint64, []byte, error) {
	sa, err := this.lookup(sid)
	if err != nil {
		return 0, nil, err
	}
	sa.ti.Reset()

	lidx, r, err1 := this.storage.LoadLastSnapshot()
	if err1 != nil {
		return 0, nil, err1
	}
	var b []byte
	if r == nil {
		b = make([]byte, 0, 0)
	} else {
		b, err = ioutil.ReadAll(r)
		if err != nil {
			return 0, nil, err
		}
	}
	sa.lastIndex = lidx
	return lidx, b, nil
}

func (this *masterEP) Process(sid uint64) ([]*corepb.Request, error) {
	sa, err := this.lookup(sid)
	if err != nil {
		return nil, err
	}
	sa.ti.Reset()

	ling, err1 := sa.checkError()
	if err1 != nil {
		return nil, err1
	}
	if !ling {
		// query
		if sa.cursor == nil {
			sa.cursor, err = this.storage.BeginLoad(sa.lastIndex)
			if err != nil {
				return nil, err
			}
		}
		_, rlist, err1 := this.storage.LoadRequest(sa.cursor, this.cfg.QuerySize, sa)
		if err1 != nil {
			return nil, err1
		}
		if len(rlist) != 0 {
			return rlist, nil
		}
		this.storage.EndLoad(sa.cursor)
		sa.cursor = nil
		sa.ling = true
	}

	rlist := sa.CopyR(this.cfg.QuerySize)
	if len(rlist) > 0 {
		// plog.Infof("atonce - remain:%v", sa.requests)
		return rlist, nil
	}

	// wait
	wt := time.NewTimer(this.cfg.PullWaitTime)
	defer wt.Stop()
	for {
		select {
		case <-sa.reqc:
			rlist = sa.CopyR(this.cfg.QuerySize)
			if len(rlist) > 0 {
				// plog.Infof("pull - remain:%v", sa.requests)
				return rlist, nil
			}
		case <-wt.C:
			// plog.Infof("pull timeout")
			return nil, nil
		}
	}
}
