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
	"sync"

	"github.com/catyguan/csf/core/corepb"
)

type LockerServiceInvoker struct {
	l                *sync.RWMutex
	cs               CoreService
	AsyncChannelSend bool
}

func (this *LockerServiceInvoker) impl() {
	_ = ServiceInvoker(this)
	_ = ServiceChannel(this)
}

func (this *LockerServiceInvoker) InvokeRequest(ctx context.Context, req *corepb.Request) (*corepb.Response, error) {
	_, err := this.cs.VerifyRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	var l sync.Locker
	if req.IsQueryType() {
		l = this.l.RLocker()
	} else {
		l = this.l
	}
	l.Lock()
	defer l.Unlock()
	return this.cs.ApplyRequest(ctx, req)
}

func (this *LockerServiceInvoker) SendRequest(ctx context.Context, creq *corepb.ChannelRequest) (<-chan *corepb.ChannelResponse, error) {
	r := make(chan *corepb.ChannelResponse, 1)
	if this.AsyncChannelSend {
		go func() {
			err := this.doSendRequest(ctx, creq, r)
			if err != nil {
				resp := MakeErrorResponse(nil, err)
				re := &corepb.ChannelResponse{}
				re.Response = *resp
				r <- re
			}
		}()
		return r, nil
	} else {
		err := this.doSendRequest(ctx, creq, r)
		if err != nil {
			return nil, err
		}
		return r, nil
	}
}

func (this *LockerServiceInvoker) doSendRequest(ctx context.Context, creq *corepb.ChannelRequest, r chan *corepb.ChannelResponse) error {
	resp, err := this.InvokeRequest(ctx, &creq.Request)
	err = corepb.HandleError(resp, err)
	if err != nil {
		return err
	}
	re := &corepb.ChannelResponse{}
	re.Response = *resp
	r <- re
	return nil
}

func NewLockerServiceInvoker(cs CoreService, l *sync.RWMutex) *LockerServiceInvoker {
	if l == nil {
		l = new(sync.RWMutex)
	}
	return &LockerServiceInvoker{cs: cs, l: l}
}
