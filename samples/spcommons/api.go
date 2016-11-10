// Copyright 2016 The CSF Authors
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

// Package etcdlike defines a csf app just like etcd.
package spcommons

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/catyguan/csf/interfaces"
)

var (
	apiPrefix string = "/counter"
)

func (this *Counter) doBuildClientHandler(sm interfaces.ServiceManager, mux *http.ServeMux) {

	ah := &apiHandler{
		sm:   sm,
		this: this,
	}
	// mux.HandleFunc(statsPrefix+"/leader", sh.serveLeader)
	mux.Handle(apiPrefix, ah)
	mux.Handle(apiPrefix+"/", ah)
}

type apiHandler struct {
	sm   interfaces.ServiceManager
	this *Counter
}

func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !interfaces.AllowMethod(w, r.Method, "GET") {
		return
	}
	if !interfaces.CheckAccess(h.sm, r, apiPrefix, "rw", "") {
		interfaces.WriteNoAuth(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), h.sm.ClientRequestTime())
	defer cancel()

	switch r.Method {
	case "GET":
		path := interfaces.TrimPrefix(r.URL.Path, apiPrefix)
		switch path {
		case "", "get":
			err := r.ParseForm()
			if err != nil {
				interfaces.WriteError(w, r, err)
				return
			}
			key := r.FormValue("k")
			if key == "" {
				key = "default"
			}
			val := h.this.GetValue(h.sm, key)
			result := make(map[string]uint64)
			result[key] = val
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(result); err != nil {
				plog.Warningf("failed to encode response (%v)", err)
			}
		case "add":
			err3 := r.ParseForm()
			if err3 != nil {
				interfaces.WriteError(w, r, err3)
				return
			}
			key := r.FormValue("k")
			val, _ := interfaces.GetUint64(r.Form, "v")
			if key == "" {
				key = "default"
			}
			val, err3 = h.this.AddValue(ctx, h.sm, key, val)
			if err3 != nil {
				interfaces.WriteError(w, r, err3)
				return
			}
			result := make(map[string]uint64)
			result[key] = val
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(result); err != nil {
				plog.Warningf("failed to encode response (%v)", err)
			}
		default:
			interfaces.WriteError(w, r, interfaces.NewHTTPError(http.StatusNotFound, "Not found"))
		}
	}
}
