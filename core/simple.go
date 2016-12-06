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
)

type SimpleServiceInvoker struct {
	cs CoreService
}

func (this *SimpleServiceInvoker) impl() {
	_ = ServiceChannel(this)
}

func (this *SimpleServiceInvoker) InvokeRequest(ctx context.Context, req *corepb.Request) (*corepb.Response, error) {
	_, err := this.cs.VerifyRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	return this.cs.ApplyRequest(ctx, req)
}

func (this *SimpleServiceInvoker) SendRequest(ctx context.Context, creq *corepb.ChannelRequest) (<-chan *corepb.ChannelResponse, error) {
	resp, err := this.InvokeRequest(ctx, creq.Request)
	err = corepb.HandleError(resp, err)
	if err != nil {
		return nil, err
	}
	r := make(chan *corepb.ChannelResponse, 1)
	r <- &corepb.ChannelResponse{
		Response: resp,
	}
	return r, nil
}

func NewSimpleServiceInvoker(cs CoreService) *SimpleServiceInvoker {
	return &SimpleServiceInvoker{cs: cs}
}
