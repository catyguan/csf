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

type muxHandler struct {
	si ServiceInvoker
	sc ServiceChannel
}

func (this *muxHandler) NotEmpty() bool {
	return this.si != nil || this.sc != nil
}

func (this *muxHandler) Empty() bool {
	return this.si == nil && this.sc == nil
}

type muxEntry struct {
	h muxHandler
}

type muxSEntry struct {
	h muxHandler
	m map[string]*muxEntry
}

type ServiceMux struct {
	mu   sync.RWMutex
	m    map[string]*muxSEntry
	next *ServiceMux
}

func NewServiceMux() *ServiceMux {
	r := new(ServiceMux)
	r.m = make(map[string]*muxSEntry)
	return r
}

func (this *ServiceMux) impl() {
	_ = ServiceInvoker(this)
	_ = ServiceChannel(this)
}

func (this *ServiceMux) SetNext(mux *ServiceMux) {
	this.next = mux
}

var DefaultServiceMux = &defaultServiceMux

var defaultServiceMux ServiceMux

// Find a handler on a handler map given a path string
// Most-specific (longest) pattern wins
func (this *ServiceMux) match(sname string, path string) *muxHandler {
	mux, ok := this.m[sname]
	if !ok {
		return nil
	}

	if mux.m != nil {
		e, ok2 := mux.m[path]
		if ok2 {
			if e.h.NotEmpty() {
				return &e.h
			}
		}
	}
	if mux.h.NotEmpty() {
		return &mux.h
	}
	return nil
}

func (this *ServiceMux) handler(r *corepb.Request) *muxHandler {
	if r == nil || r.Info == nil {
		return nil
	}
	this.mu.RLock()
	h := this.match(r.Info.ServiceName, r.Info.ServicePath)
	this.mu.RUnlock()
	if h == nil && this.next != nil {
		return this.next.handler(r)
	}
	return nil
}

func (this *ServiceMux) InvokeRequest(ctx context.Context, req *corepb.Request) (*corepb.Response, error) {
	h := this.handler(req)
	if h == nil {
		return nil, ErrNotFound
	}
	if h.Empty() {
		return nil, ErrNotFound
	}
	if h.si != nil {
		return h.si.InvokeRequest(ctx, req)
	}
	if h.sc != nil {
		creq := &corepb.ChannelRequest{Request: req}
		r, err := DoSendRequest(h.sc, ctx, creq)
		if err != nil {
			return nil, err
		}
		if r != nil {
			return r.Response, nil
		}
	}
	return nil, nil
}

func (this *ServiceMux) SendRequest(ctx context.Context, creq *corepb.ChannelRequest) (<-chan *corepb.ChannelResponse, error) {
	h := this.handler(creq.Request)
	if h == nil {
		return nil, ErrNotFound
	}
	if h.Empty() {
		return nil, ErrNotFound
	}
	if h.sc != nil {
		return h.sc.SendRequest(ctx, creq)
	}
	if h.si != nil {
		resp, err := h.si.InvokeRequest(ctx, creq.Request)
		if err != nil {
			return nil, err
		}
		r := make(chan *corepb.ChannelResponse, 1)
		r <- &corepb.ChannelResponse{
			Response: resp,
		}
		return r, nil
	}
	return nil, nil
}

func (this *ServiceMux) AddInvoker(s string, si ServiceInvoker) bool {
	return this.execAdd(s, muxHandler{si: si})
}

func (this *ServiceMux) AddChannel(s string, sc ServiceChannel) bool {
	return this.execAdd(s, muxHandler{sc: sc})
}

func (this *ServiceMux) execAdd(s string, h muxHandler) bool {
	this.mu.Lock()
	defer this.mu.Unlock()
	_, ok := this.m[s]
	if ok {
		return false
	}
	this.m[s] = &muxSEntry{
		h: h,
	}
	return true
}

func (this *ServiceMux) HandleInvoker(s, p string, si ServiceInvoker) bool {
	return this.execHandle(s, p, muxHandler{si: si})
}

func (this *ServiceMux) HandleChannel(s, p string, sc ServiceChannel) bool {
	return this.execHandle(s, p, muxHandler{sc: sc})
}

func (this *ServiceMux) execHandle(s, p string, h muxHandler) bool {
	this.mu.Lock()
	defer this.mu.Unlock()
	m, ok := this.m[s]
	if !ok {
		m = &muxSEntry{
			h: muxHandler{},
		}
		this.m[s] = m
	}
	if m.m == nil {
		m.m = make(map[string]*muxEntry)
	}
	_, ok2 := m.m[p]
	if ok2 {
		return false
	}
	m.m[p] = &muxEntry{
		h: h,
	}
	return true
}
