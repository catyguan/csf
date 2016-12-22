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
	"context"
	"net/http"
	"time"

	"github.com/catyguan/csf/core"
)

type Handler struct {
	mux         *core.ServiceMux
	converter   Converter
	execTimeout time.Duration
}

func (this *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	cr, err := this.converter.BuildRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx := context.Background()
	nctx, cancel := context.WithTimeout(ctx, this.execTimeout)
	dl, _ := nctx.Deadline()
	tm := time.AfterFunc(dl.Sub(time.Now()), func() {
		cancel()
		plog.Warningf("execute timeout")
	})
	cresp, err2 := this.mux.InvokeRequest(nctx, cr)
	tm.Stop()
	if err2 != nil {
		http.Error(w, err2.Error(), http.StatusInternalServerError)
		return
	}

	if cresp != nil {
		cresp.Bind(&cr.Request)
	}
	err3 := this.converter.WriteResponse(w, cresp)
	if err3 != nil {
		http.Error(w, err3.Error(), http.StatusInternalServerError)
		return
	}
}

func NewHandler(mux *core.ServiceMux, c Converter, execTimeout time.Duration) *Handler {
	if execTimeout == 0 {
		execTimeout = defaultExecuteTimeout
	}
	h := &Handler{
		mux:         mux,
		converter:   c,
		execTimeout: execTimeout,
	}
	if h.converter == nil {
		h.converter = &DefaultConverter{}
	}
	return h
}
