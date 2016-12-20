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
	cs               CoreService
	AsyncChannelSend bool
}

func (this *SimpleServiceInvoker) impl() {
	_ = ServiceInvoker(this)
}

func (this *SimpleServiceInvoker) InvokeRequest(ctx context.Context, creq *corepb.ChannelRequest) (*corepb.ChannelResponse, error) {
	req := &creq.Request
	_, err := this.cs.VerifyRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	resp, err2 := this.cs.ApplyRequest(ctx, req)
	err2 = corepb.HandleError(resp, err)
	if err2 != nil {
		return nil, err2
	}
	return corepb.MakeChannelResponse(resp), nil
}

func NewSimpleServiceInvoker(cs CoreService) *SimpleServiceInvoker {
	return &SimpleServiceInvoker{cs: cs}
}
