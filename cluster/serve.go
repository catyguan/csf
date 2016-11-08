// Copyright 2015 The etcd Authors
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

package cluster

import (
	"crypto/tls"
	"io/ioutil"
	defaultLog "log"
	"net"
	"net/http"
	"time"

	"github.com/cockroachdb/cmux"
	"golang.org/x/net/context"
)

type serveCtx struct {
	listener net.Listener
	secure   bool
	insecure bool

	ctx    context.Context
	cancel context.CancelFunc

	userHandlers map[string]http.Handler
}

func newServeCtx() *serveCtx {
	ctx, cancel := context.WithCancel(context.Background())
	return &serveCtx{ctx: ctx, cancel: cancel}
}

// serve accepts incoming connections on the listener l,
// creating a new service goroutine for each. The service goroutines
// read requests and then call handler to reply to them.
func (sctx *serveCtx) serve(s *CSFNode, tlscfg *tls.Config, handler http.Handler, errc chan<- error) error {
	logger := defaultLog.New(ioutil.Discard, "csfhttp", 0)

	<-s.ReadyNotify()
	addr := sctx.listener.Addr().String()
	plog.Infof("ready to serve client requests on %s", addr)

	m := cmux.New(sctx.listener)

	if sctx.insecure {
		httpmux := sctx.createMux(handler)

		srvhttp := &http.Server{
			Handler:  httpmux,
			ErrorLog: logger, // do not log user error
		}
		httpl := m.Match(cmux.HTTP1())
		go func() { errc <- srvhttp.Serve(httpl) }()
		plog.Noticef("serving insecure client requests on %s, this is strongly discouraged!", addr)
	}

	if sctx.secure {
		tlsl := tls.NewListener(m.Match(cmux.Any()), tlscfg)
		// TODO: add debug flag; enable logging when debug flag is set
		httpmux := sctx.createMux(handler)

		srv := &http.Server{
			Handler:   httpmux,
			TLSConfig: tlscfg,
			ErrorLog:  logger, // do not log user error
		}
		go func() { errc <- srv.Serve(tlsl) }()

		plog.Infof("serving client requests on %s", addr)
	}

	return m.Serve()
}

func servePeerHTTP(l net.Listener, handler http.Handler) error {
	logger := defaultLog.New(ioutil.Discard, "csfhttp", 0)
	// TODO: add debug flag; enable logging when debug flag is set
	srv := &http.Server{
		Handler:     handler,
		ReadTimeout: 5 * time.Minute,
		ErrorLog:    logger, // do not log user error
	}
	return srv.Serve(l)
}

func (sctx *serveCtx) createMux(handler http.Handler) *http.ServeMux {
	httpmux := http.NewServeMux()
	for path, h := range sctx.userHandlers {
		httpmux.Handle(path, h)
	}
	httpmux.Handle("/", handler)
	return httpmux
}
