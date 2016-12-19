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
package core

import (
	"context"
	"sync/atomic"

	"github.com/catyguan/csf/core/corepb"
)

type actionOfSTI struct {
	ctx  context.Context
	req  *corepb.Request
	rch1 chan *corepb.Response
	rch2 chan *corepb.ChannelResponse
}

type SingeThreadServiceInvoker struct {
	cs CoreService

	closed  uint64
	closech chan interface{}
	actions chan *actionOfSTI
}

func NewSingeThreadServiceInvoker(cs CoreService, queueSize int) *SingeThreadServiceInvoker {
	r := &SingeThreadServiceInvoker{
		cs:      cs,
		closech: make(chan interface{}, 0),
		actions: make(chan *actionOfSTI, queueSize),
	}
	rd := make(chan interface{}, 0)
	go r.doRun(rd, r.closech)
	<-rd
	return r
}

func (this *SingeThreadServiceInvoker) impl() {
	_ = ServiceInvoker(this)
	_ = ServiceChannel(this)
}

func (this *SingeThreadServiceInvoker) doRun(ready chan interface{}, closef chan interface{}) {
	defer func() {
		close(closef)
	}()
	close(ready)
	for {
		select {
		case a := <-this.actions:
			if a == nil {
				return
			}
			r, err := this.cs.ApplyRequest(a.ctx, a.req)
			if err != nil {
				r = MakeErrorResponse(r, err)
			}
			if a.rch1 != nil {
				a.rch1 <- r
			}
			if a.rch2 != nil {
				re := &corepb.ChannelResponse{}
				re.Response = *r
				a.rch2 <- re
			}
		}
	}
}

func (this *SingeThreadServiceInvoker) InvokeRequest(ctx context.Context, req *corepb.Request) (*corepb.Response, error) {
	_, err := this.cs.VerifyRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	c := make(chan *corepb.Response, 1)
	a := &actionOfSTI{
		ctx:  ctx,
		req:  req,
		rch1: c,
	}
	if this.IsClosed() {
		return nil, ErrClosed
	}
	this.actions <- a
	re := <-c
	return re, nil
}

func (this *SingeThreadServiceInvoker) SendRequest(ctx context.Context, creq *corepb.ChannelRequest) (<-chan *corepb.ChannelResponse, error) {
	req := &creq.Request
	_, err := this.cs.VerifyRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	c := make(chan *corepb.ChannelResponse, 1)
	a := &actionOfSTI{
		ctx:  ctx,
		req:  req,
		rch2: c,
	}
	if this.IsClosed() {
		return nil, ErrClosed
	}
	this.actions <- a
	return c, nil
}

func (this *SingeThreadServiceInvoker) IsClosed() bool {
	return atomic.LoadUint64(&this.closed) > 0
}

func (this *SingeThreadServiceInvoker) Close() {
	if !atomic.CompareAndSwapUint64(&this.closed, 0, 1) {
		return
	}
	this.actions <- nil
	<-this.closech
}
