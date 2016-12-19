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
package httpport

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/catyguan/csf/core"
	runtimeutil "github.com/catyguan/csf/pkg/runtime"
	"github.com/catyguan/csf/pkg/transport"
)

const (
	defaultConnReadTimeout  = 5 * time.Second
	defaultConnWriteTimeout = 5 * time.Second
	defaultExecuteTimeout   = 60 * time.Second

	reservedInternalFDNum = 150
)

type Config struct {
	Addr             string
	Host             string
	ConnReadTimeout  time.Duration
	ConnWriteTimeout time.Duration
	ExcecuteTimeout  time.Duration
	ConnectionLimit  int
	transport.TLSInfo
}

type Port struct {
	cfg *Config
	mux *http.ServeMux
	l   net.Listener
}

func NewPort(cfg *Config) *Port {
	if cfg.ConnReadTimeout == 0 {
		cfg.ConnReadTimeout = defaultConnReadTimeout
	}
	if cfg.ConnWriteTimeout == 0 {
		cfg.ConnWriteTimeout = defaultConnWriteTimeout
	}
	if cfg.ExcecuteTimeout == 0 {
		cfg.ExcecuteTimeout = defaultExecuteTimeout
	}
	r := new(Port)
	r.cfg = cfg
	return r
}

func (this *Port) Start(pattern string, mux *core.ServiceMux, c Converter) error {
	h := NewHandler(mux, c, this.cfg.ExcecuteTimeout)
	hmux := http.NewServeMux()
	hmux.Handle(pattern, h)
	return this.StartServe(hmux)
}

func (this *Port) StartServe(mux *http.ServeMux) error {
	proto := "http"
	if !this.cfg.Empty() {
		proto = "https"
	}
	var cf *tls.Config
	if !this.cfg.TLSInfo.Empty() {
		ccf, err2 := this.cfg.TLSInfo.ServerConfig()
		if err2 != nil {
			return err2
		}
		cf = ccf
	}
	l, err := transport.NewTimeoutListener(this.cfg.Addr, proto, cf, this.cfg.ConnReadTimeout, this.cfg.ConnWriteTimeout)
	if err != nil {
		return err
	}
	lm := this.cfg.ConnectionLimit
	if lm == 0 {
		if fdLimit, fderr := runtimeutil.FDLimit(); fderr == nil {
			if fdLimit <= reservedInternalFDNum {
				plog.Fatalf("file descriptor limit[%d] of CSF process is too low, and should be set higher than %d to ensure internal usage", fdLimit, reservedInternalFDNum)
			}
			lm = int(fdLimit - reservedInternalFDNum)
		}
	}
	if lm > 0 {
		l = transport.LimitListener(l, lm)
	}
	// if proto == "tcp" {
	// 	if sctx.l, err = transport.NewKeepAliveListener(sctx.l, "tcp", nil); err != nil {
	// 		return nil, err
	// 	}
	// }
	this.l = l
	this.mux = mux
	return nil
}

func (this *Port) Run() {
	go this.serv()
}

func (this *Port) serv() {
	srv := &http.Server{
		Handler: this,
	}
	srv.Serve(this.l)
}

func (this *Port) Stop() {
	if this.l != nil {
		this.l.Close()
		this.l = nil
	}
}

func (this *Port) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if this.cfg.Host != "" {
		if strings.ToLower(req.Host) != strings.ToLower(this.cfg.Host) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}
	this.mux.ServeHTTP(w, req)
}
