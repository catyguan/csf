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

	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/runobj"
)

type SingleThreadServiceContainer struct {
	cs CoreService

	ro *runobj.RunObj
}

func NewSingleThreadServiceContainer(cs CoreService, queueSize int) *SingleThreadServiceContainer {
	r := &SingleThreadServiceContainer{
		cs: cs,
		ro: runobj.NewRunObj(queueSize),
	}
	r.ro.Run(r.doRun, nil)
	return r
}

func (this *SingleThreadServiceContainer) impl() {
	_ = ServiceInvoker(this)
	_ = ServiceHolder(this)
}

func (this *SingleThreadServiceContainer) doRun(ready chan error, ach <-chan *runobj.ActionRequest, p interface{}) {
	close(ready)
	for {
		select {
		case a := <-ach:
			if a == nil {
				return
			}

			switch a.Type {
			case 1:
				ctx := a.P1.(context.Context)
				req := a.P2.(*corepb.Request)
				r, err := this.cs.ApplyRequest(ctx, req)
				if a.Resp != nil {
					if err != nil {
						r = MakeErrorResponse(r, err)
					}
					a.Resp <- &runobj.ActionResponse{R1: r, Err: err}
				}
			case 2:
				ctx := a.P1.(context.Context)
				sf := a.P2.(ServiceFunc)
				err := sf(ctx, this.cs)
				if a.Resp != nil {
					a.Resp <- &runobj.ActionResponse{Err: err}
				}
			}
		}
	}
}

func (this *SingleThreadServiceContainer) InvokeRequest(ctx context.Context, creq *corepb.ChannelRequest) (*corepb.ChannelResponse, error) {
	req := &creq.Request
	_, err := this.cs.VerifyRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	a := &runobj.ActionRequest{
		Type: 1,
		P1:   ctx,
		P2:   req,
	}
	ar, err1 := this.ro.ContextCall(ctx, a)
	if err1 != nil {
		return nil, err1
	}
	var resp *corepb.Response
	if ar.R1 != nil {
		resp = ar.R1.(*corepb.Response)
	}
	err2 := corepb.HandleError(resp, ar.Err)
	if err2 != nil {
		return nil, err2
	}
	return corepb.MakeChannelResponse(resp), nil
}

func (this *SingleThreadServiceContainer) IsClosed() bool {
	return this.ro.IsClosed()
}

func (this *SingleThreadServiceContainer) Close() {
	this.ro.Close()
}

func (this *SingleThreadServiceContainer) ExecuteServiceFunc(ctx context.Context, sfunc ServiceFunc) error {
	a := &runobj.ActionRequest{
		Type: 2,
		P1:   ctx,
		P2:   sfunc,
	}
	ar, err1 := this.ro.ContextCall(ctx, a)
	if err1 != nil {
		return err1
	}
	return ar.Err
}
