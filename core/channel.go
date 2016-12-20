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

type scInvoker struct {
	sc   *ServiceChannel
	h    ServiceChannelHandler
	next *scInvoker
}

func (this *scInvoker) InvokeRequest(ctx context.Context, creq *corepb.ChannelRequest) (*corepb.ChannelResponse, error) {
	if this.h != nil {
		return this.h.HandleRequest(ctx, this.next, creq)
	}
	if this.sc.sink == nil {
		plog.Fatalf("nil ServiceChannel sink")
	}
	return this.sc.sink.InvokeRequest(ctx, creq)
}

type ServiceChannel struct {
	head *scInvoker
	sink ServiceInvoker
}

func NewServiceChannel() *ServiceChannel {
	r := new(ServiceChannel)
	r.head = &scInvoker{sc: r}
	return r
}

func (this *ServiceChannel) impl() {
	_ = ServiceInvoker(this)
}

func (this *ServiceChannel) InvokeRequest(ctx context.Context, creq *corepb.ChannelRequest) (*corepb.ChannelResponse, error) {
	return this.head.InvokeRequest(ctx, creq)
}

func (this *ServiceChannel) Next(h ServiceChannelHandler) *ServiceChannel {
	i := this.head
	for {
		if i.h == nil {
			i.h = h
			i.next = &scInvoker{sc: this}
			break
		}
		i = i.next
	}
	return this
}

func (this *ServiceChannel) Sink(si ServiceInvoker) {
	this.sink = si
}
