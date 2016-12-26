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

func NewHttpServiceInvokerWithURL(url string) (*HttpServiceInvoker, error) {
	cfg := &Config{}
	cfg.URL = url
	return NewHttpServiceInvoker(cfg, nil)
}

func NewHttpServiceInvoker(cfg *Config, c Converter) (*HttpServiceInvoker, error) {
	tr, err := NewTransport(cfg)
	if err != nil {
		return nil, err
	}
	return CreateInvoker(cfg, tr, nil, c), nil
}

func CreateInvoker(cfg *Config, tr *http.Transport, cl *http.Client, c Converter) *HttpServiceInvoker {
	r := &HttpServiceInvoker{}
	r.cfg = cfg
	r.converter = c
	if r.converter == nil {
		r.converter = &DefaultConverter{}
	}
	if r.cfg.ExcecuteTimeout == 0 {
		r.cfg.ExcecuteTimeout = defaultExecuteTimeout
	}
	if cl == nil {
		cl = CreateClient(cfg, tr)
	}
	r.cl = cl
	return r
}

func (this *HttpServiceInvoker) impl() {
	_ = core.ServiceInvoker(this)
}

func (this *HttpServiceInvoker) InvokeRequest(ctx context.Context, creq *corepb.ChannelRequest) (*corepb.ChannelResponse, error) {
	hreq, err := this.converter.BuildRequest(this.cfg.URL, creq)
	if err != nil {
		return nil, err
	}
	if this.cfg.Host != "" {
		hreq.Host = this.cfg.Host
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
	return r, nil
}
