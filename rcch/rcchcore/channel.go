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
package rcchcore

import "context"

type chInvoker struct {
	ch   *Channel
	h    ChannelHandler
	next *chInvoker
}

func (this *chInvoker) Invoke(ctx context.Context, creq *ChannelRequest) (*ChannelResponse, error) {
	if this.h != nil {
		return this.h.HandleRequest(ctx, this.next, creq)
	}
	if this.ch.sink == nil {
		plog.Fatalf("nil Channel sink")
	}
	return this.ch.sink.Invoke(ctx, creq)
}

type Channel struct {
	head *chInvoker
	sink Invoker
}

func NewChannel() *Channel {
	r := new(Channel)
	r.head = &chInvoker{ch: r}
	return r
}

func (this *Channel) impl() {
	_ = Invoker(this)
}

func (this *Channel) Invoke(ctx context.Context, creq *ChannelRequest) (*ChannelResponse, error) {
	return this.head.Invoke(ctx, creq)
}

func (this *Channel) Begin(h ChannelHandler) *Channel {
	i := this.head
	this.head = &chInvoker{ch: this, h: h, next: i}
	return this
}

func (this *Channel) Next(h ChannelHandler) *Channel {
	i := this.head
	for {
		if i.h == nil {
			i.h = h
			i.next = &chInvoker{ch: this}
			break
		}
		i = i.next
	}
	return this
}

func (this *Channel) Sink(ink Invoker) {
	this.sink = ink
}
