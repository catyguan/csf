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

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/runobj"
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

	ro *runobj.RunObj
}

func NewStorageServiceInvoker(cfg *Config) *StorageServiceInvoker {
	r := &StorageServiceInvoker{
		cs:        cfg.Service,
		storage:   cfg.Storage,
		snapCount: cfg.SnapCount,
		ro:        runobj.NewRunObj(cfg.QueueSize),
	}
	return r
}

func (this *StorageServiceInvoker) impl() {
	_ = core.ServiceInvoker(this)
}

func (this *StorageServiceInvoker) Run() error {
	return this.ro.Run(this.doRun, nil)
}

func (this *StorageServiceInvoker) InvokeRequest(ctx context.Context, creq *corepb.ChannelRequest) (*corepb.ChannelResponse, error) {
	req := &creq.Request
	s, err := this.cs.VerifyRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	a := &runobj.ActionRequest{
		Type: 1,
		P1:   ctx,
		P2:   req,
		P3:   s,
	}
	ar, err1 := this.ro.ContextCall(ctx, a)
	if err1 != nil {
		return nil, err1
	}
	var resp *corepb.Response
	if ar.R1 != nil {
		resp = ar.R1.(*corepb.Response)
	}
	err = corepb.HandleError(resp, ar.Err)
	if err != nil {
		return nil, err
	}
	return corepb.MakeChannelResponse(resp), nil
}

func (this *StorageServiceInvoker) MakeSnapshot() (uint64, error) {
	a := &runobj.ActionRequest{
		Type: 2,
	}
	re, err := this.ro.Call(a)
	if err != nil {
		return 0, err
	}
	var resp uint64
	if re.R1 != nil {
		resp = re.R1.(uint64)
	}
	return resp, nil
}

func (this *StorageServiceInvoker) ApplyRequests(ctx context.Context, rlist []*corepb.Request) error {
	a := &runobj.ActionRequest{
		Type: 3,
		P1:   ctx,
		P2:   rlist,
	}
	ar, err := this.ro.ContextCall(ctx, a)
	if err != nil {
		return err
	}
	return ar.Err
}

func (this *StorageServiceInvoker) ApplySnapshot(ctx context.Context, idx uint64, data []byte) error {
	a := &runobj.ActionRequest{
		Type: 4,
		P1:   ctx,
		P2:   idx,
		P3:   data,
	}
	ar, err := this.ro.ContextCall(ctx, a)
	if err != nil {
		return err
	}
	return ar.Err
}

func (this *StorageServiceInvoker) IsClosed() bool {
	return this.ro.IsClosed()
}

func (this *StorageServiceInvoker) Close() {
	this.ro.Close()
}
