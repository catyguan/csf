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
package http4si

import (
	"context"
	"net/http"
	"time"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/transport"
)

type Config struct {
	URL             string
	Host            string
	DialTimeout     time.Duration
	ExcecuteTimeout time.Duration
	transport.TLSInfo
}

type HttpServiceInvoker struct {
	cl        *http.Client
	cfg       *Config
	converter Converter
}

func NewHttpServiceInvoker(cfg *Config, c Converter) (*HttpServiceInvoker, error) {
	r := &HttpServiceInvoker{}
	r.cfg = cfg
	r.converter = c
	if r.converter == nil {
		r.converter = &DefaultConverter{}
	}
	if r.cfg.ExcecuteTimeout == 0 {
		r.cfg.ExcecuteTimeout = defaultExecuteTimeout
	}
	if r.cfg.DialTimeout == 0 {
		r.cfg.DialTimeout = defaultDialTimeout
	}

	tr, err := newTransport(&r.cfg.TLSInfo, r.cfg.DialTimeout)
	if err != nil {
		return nil, err
	}
	r.cl = &http.Client{Transport: tr}
	return r, nil
}

func (this *HttpServiceInvoker) impl() {
	_ = core.ServiceInvoker(this)
	_ = core.ServiceChannel(this)
}

func (this *HttpServiceInvoker) InvokeRequest(ctx context.Context, req *corepb.Request) (*corepb.Response, error) {
	creq := &corepb.ChannelRequest{}
	creq.Request = *req
	hreq, err := this.converter.BuildRequest(this.cfg.URL, creq)
	if err != nil {
		return nil, err
	}

	nctx, _ := context.WithTimeout(ctx, this.cfg.ExcecuteTimeout)
	hreq = hreq.WithContext(nctx)
	hresp, err1 := this.cl.Do(hreq)
	if err1 != nil {
		return nil, err1
	}
	r, err2 := this.converter.HandleResponse(hresp)
	if err2 != nil {
		return nil, err2
	}
	return &r.Response, nil
}

func (this *HttpServiceInvoker) SendRequest(ctx context.Context, creq *corepb.ChannelRequest) (<-chan *corepb.ChannelResponse, error) {
	hreq, err := this.converter.BuildRequest(this.cfg.URL, creq)
	if err != nil {
		return nil, err
	}

	rch := make(chan *corepb.ChannelResponse, 1)
	nctx, _ := context.WithTimeout(ctx, this.cfg.ExcecuteTimeout)
	hreq.WithContext(nctx)
	go func() {
		hresp, err1 := this.cl.Do(hreq)
		if err1 != nil {
			resp := &corepb.ChannelResponse{}
			resp.Response = *core.MakeErrorResponse(nil, err1)
			rch <- resp
			return
		}
		r, err2 := this.converter.HandleResponse(hresp)
		if err2 != nil {
			r := &corepb.ChannelResponse{}
			r.Response = *core.MakeErrorResponse(nil, err2)
		}
		rch <- r
	}()
	return rch, nil
}
