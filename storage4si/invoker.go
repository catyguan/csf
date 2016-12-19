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
package storage4si

import (
	"context"
	"sync/atomic"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
)

type Config struct {
	SnapCount int // 多少个Request保存一次快照，0表示不进行
	QueueSize int
	Service   core.CoreService
	Storage   Storage
}

func NewConfig() *Config {
	cfg := &Config{
		QueueSize: 1024,
		SnapCount: 0,
	}
	return cfg
}

type StorageServiceInvoker struct {
	cs        core.CoreService
	storage   Storage
	snapCount int
	lastIndex uint64

	closed  uint64
	closech chan interface{}
	actions chan *actionOfSSI
}

func NewStorageServiceInvoker(cfg *Config) *StorageServiceInvoker {
	r := &StorageServiceInvoker{
		cs:        cfg.Service,
		storage:   cfg.Storage,
		snapCount: cfg.SnapCount,
		actions:   make(chan *actionOfSSI, cfg.QueueSize),
	}
	return r
}

func (this *StorageServiceInvoker) Run() error {
	rd := make(chan error, 1)
	this.closech = make(chan interface{}, 0)
	go this.doRun(rd, this.closech)
	err := <-rd
	return err
}

type actionOfSSI struct {
	typ  int
	p1   interface{}
	p2   interface{}
	p3   interface{}
	resp chan responseOfSSI
}

type responseOfSSI struct {
	result interface{}
	err    error
}

func (this *StorageServiceInvoker) InvokeRequest(ctx context.Context, req *corepb.Request) (*corepb.Response, error) {
	s, err := this.cs.VerifyRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	c := make(chan responseOfSSI, 1)
	a := &actionOfSSI{
		typ:  1,
		p1:   ctx,
		p2:   req,
		p3:   s,
		resp: c,
	}
	if this.IsClosed() {
		return nil, ErrClosed
	}
	this.actions <- a
	re := <-c
	var resp *corepb.Response
	if re.result != nil {
		resp = re.result.(*corepb.Response)
	}
	return resp, re.err
}

func (this *StorageServiceInvoker) MakeSnapshot() (uint64, error) {
	c := make(chan responseOfSSI, 1)
	a := &actionOfSSI{
		typ:  2,
		resp: c,
	}
	if this.IsClosed() {
		return 0, ErrClosed
	}
	this.actions <- a
	re := <-c
	var resp uint64
	if re.result == nil {
		resp = re.result.(uint64)
	}
	return resp, re.err
}

func (this *StorageServiceInvoker) IsClosed() bool {
	return atomic.LoadUint64(&this.closed) > 0
}

func (this *StorageServiceInvoker) Close() {
	if !atomic.CompareAndSwapUint64(&this.closed, 0, 1) {
		return
	}
	this.actions <- nil
	<-this.closech
}
