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
package walstorage

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/masterslave"
	"github.com/catyguan/csf/snapshot"
	"github.com/catyguan/csf/storage4si"
	"github.com/catyguan/csf/wal"
)

type Config struct {
	wal.Config
	Symbol   string
	AutoSync bool
}

func NewConfig() *Config {
	r := new(Config)
	wal.InitConfig(&r.Config)
	return r
}

type WALStorage struct {
	cfg  *Config
	snap *snapshot.Snapshotter
	w    wal.WAL

	llist storage4si.Listeners
	mu    sync.Mutex
}

func NewWALStorage(cfg *Config) (*WALStorage, error) {
	r := &WALStorage{
		cfg: cfg,
	}
	r.snap = snapshot.NewSnapshotter(cfg.Dir)

	wcfg := wal.NewConfig()
	*wcfg = cfg.Config
	wcfg.InitMetadata = []byte(cfg.Symbol)

	w, meta, err := wal.NewWAL(wcfg)
	if err != nil {
		return nil, err
	}
	if string(meta) != cfg.Symbol {
		w.Close()
		return nil, fmt.Errorf("Symbol invalid, got '%s' want '%s'", string(meta), cfg.Symbol)
	}
	r.w = w
	w.AddListener(r)

	return r, nil
}

func (this *WALStorage) impl() {
	_ = storage4si.Storage(this)
	_ = masterslave.MasterNode(this)
}

func (this *WALStorage) Close() {
	this.mu.Lock()
	this.llist.OnClose()
	this.mu.Unlock()

	if this.w != nil {
		this.w.Close()
		this.w = nil
	}
}

func (this *WALStorage) OnReset() {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.llist.OnReset()
}

func (this *WALStorage) OnTruncate(idx uint64) {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.llist.OnTruncate(idx)
}

func (this *WALStorage) OnAppendEntry(ents []wal.Entry) {
	this.mu.Lock()
	defer this.mu.Unlock()
	for _, e := range ents {
		req := &corepb.Request{}
		req.Unmarshal(e.Data)
		this.llist.OnSaveRequest(e.Index, req)
	}
}

func (this *WALStorage) OnClose() {

}

func (this *WALStorage) SaveRequest(idx uint64, req *corepb.Request) (uint64, error) {
	data, err := req.Marshal()
	if err != nil {
		return 0, err
	}
	ents := make([]wal.Entry, 1)
	ents[0].Index = idx
	ents[0].Data = data
	c := this.w.Append(ents, this.cfg.AutoSync)
	rs := <-c
	return rs.Index, rs.Err
}

type wsCursor struct {
	c     wal.Cursor
	start uint64
}

func (this *WALStorage) BeginLoad(start uint64) (interface{}, error) {
	c, err := this.w.GetCursor(start)
	if err != nil {
		return nil, err
	}
	return &wsCursor{c: c, start: start}, err
}
func (this *WALStorage) LoadRequest(c interface{}, size int, lis storage4si.StorageListener) (uint64, []*corepb.Request, error) {
	return this.doLoadRequest(c, size, lis, nil)
}

func (this *WALStorage) doLoadRequest(c interface{}, size int, lis storage4si.StorageListener, f masterslave.MasterFollower) (uint64, []*corepb.Request, error) {
	wsc := c.(*wsCursor)

	ll := uint64(0)
	r := make([]*corepb.Request, 0, size)
	for {
		if len(r) >= size {
			break
		}
		e, err4 := wsc.c.Read()
		if err4 != nil {
			return 0, nil, err4
		}
		if e == nil {
			// plog.Infof("cursor end")
			break
		}
		if e.Index >= wsc.start {
			req := &corepb.Request{}
			err := req.Unmarshal(e.Data)
			if err != nil {
				return 0, nil, err
			}
			r = append(r, req)
			ll = e.Index
		}
	}
	if len(r) == 0 {
		r = nil
		if lis != nil {
			this.AddListener(lis)
		}
		if f != nil {
			this.AddListener(&storage4si.MasterFollowerListener{Follower: f})
		}
	}
	return ll, r, nil
}

func (this *WALStorage) EndLoad(c interface{}) error {
	cr := c.(*wsCursor)
	cr.c.Close()
	return nil
}

func (this *WALStorage) SaveSnapshot(idx uint64, r io.Reader) error {
	sh := &snapshot.SnapHeader{
		Index: idx,
		Meta:  []byte(this.cfg.Symbol),
	}
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return this.snap.SaveSnap(sh, data)
}

func (this *WALStorage) LoadLastSnapshot() (uint64, io.Reader, error) {
	n, lr, err := this.snap.LoadLastHeader()
	if err != nil {
		return 0, nil, err
	}
	if lr == nil {
		return 0, nil, nil
	}
	_, data, err2 := this.snap.LoadSnapFile(n)
	if err2 != nil {
		return 0, nil, err2
	}

	return lr.Index, bytes.NewBuffer(data), nil
}

func (this *WALStorage) AddListener(lis storage4si.StorageListener) uint64 {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.llist.Add(lis)
	return this.w.LastIndex()
}

func (this *WALStorage) RemoveListener(lis storage4si.StorageListener) {
	go func() {
		this.doRemoveListener(lis)
	}()
}

func (this *WALStorage) doRemoveListener(lis storage4si.StorageListener) {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.llist.Remove(lis)
}

func (this *WALStorage) MasterLoadLastSnapshot() (uint64, []byte, error) {
	lidx, r, err := this.LoadLastSnapshot()
	if err != nil {
		return 0, nil, err
	}
	if r == nil {
		return lidx, nil, nil
	}
	b, err2 := ioutil.ReadAll(r)
	return lidx, b, err2
}

func (this *WALStorage) MasterBeginLoad(idx uint64) (interface{}, error) {
	return this.BeginLoad(idx)
}

func (this *WALStorage) MasterLoadRequest(c interface{}, size int, f masterslave.MasterFollower) ([]*corepb.Request, error) {
	_, r1, r2 := this.doLoadRequest(c, size, nil, f)
	return r1, r2
}

func (this *WALStorage) MasterEndLoad(c interface{}) {
	this.EndLoad(c)
}

func (this *WALStorage) RemoveFollower(f masterslave.MasterFollower) {
	this.RemoveListener(&storage4si.MasterFollowerListener{Follower: f})
}
