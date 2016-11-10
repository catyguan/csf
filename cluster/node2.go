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

package cluster

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"path"
	"time"

	"github.com/catyguan/csf/interfaces"
	"github.com/catyguan/csf/pkg/cors"
	runtimeutil "github.com/catyguan/csf/pkg/runtime"
	"github.com/catyguan/csf/pkg/transport"
	"github.com/catyguan/csf/rafthttp"
)

func StartNode(cfg *Config, shub []interfaces.Service) (srv *CSFNode, err error) {
	if err = cfg.Validate(); err != nil {
		return nil, err
	}

	var peerls []net.Listener
	var clientls []net.Listener
	var sctxs map[string]*serveCtx
	if peerls, err = startPeerListeners(cfg); err != nil {
		return nil, err
	}
	if sctxs, err = startClientListeners(cfg); err != nil {
		closeListeners(peerls)
		return
	}
	for _, sctx := range sctxs {
		clientls = append(clientls, sctx.listener)
	}
	cfg.shub = shub

	node, errn := doSetupNode(cfg)
	if errn != nil {
		closeListeners(peerls)
		closeListeners(clientls)
		return nil, errn
	}
	node.Peers = peerls
	node.Clients = clientls
	node.sctxs = sctxs

	// buffer channel so goroutines on closed connections won't wait forever
	node.errc = make(chan error, len(node.Peers)+len(node.Clients)+2*len(node.sctxs))

	node.doStart()
	if err = node.serve(); err != nil {
		node.doStop()
		return nil, err
	}
	// SERVICE: start
	for _, sv := range cfg.shub {
		plog.Infof("start Service(%s)", sv.ServiceID())
		sv.StartService(node)
	}

	// OnReady
	close(node.readych)
	return node, nil
}

func closeListeners(plns []net.Listener) {
	if plns == nil {
		return
	}
	for i := range plns {
		if plns[i] == nil {
			continue
		}
		s := plns[i].Addr().String()
		plns[i].Close()
		plog.Info("stopping listener on ", s)
	}
}

func startPeerListeners(cfg *Config) (plns []net.Listener, err error) {
	if cfg.PeerAutoTLS && cfg.PeerTLSInfo.Empty() {
		phosts := make([]string, len(cfg.LPUrls))
		for i, u := range cfg.LPUrls {
			phosts[i] = u.Host
		}
		cfg.PeerTLSInfo, err = transport.SelfCert(path.Join(cfg.Dir, "fixtures/peer"), phosts)
		if err != nil {
			plog.Fatalf("could not get certs - %v", err)
		}
	} else if cfg.PeerAutoTLS {
		plog.Warningf("ignoring peer auto TLS since certs given")
	}

	if !cfg.PeerTLSInfo.Empty() {
		plog.Infof("peerTLS: %s", cfg.PeerTLSInfo)
	}

	plns = make([]net.Listener, len(cfg.LPUrls))
	defer func() {
		if err == nil {
			return
		}
		closeListeners(plns)
	}()

	for i, u := range cfg.LPUrls {
		var tlscfg *tls.Config
		if u.Scheme == "http" {
			if !cfg.PeerTLSInfo.Empty() {
				plog.Warningf("The scheme of peer url %s is HTTP while peer key/cert files are presented. Ignored peer key/cert files.", u.String())
			}
			if cfg.PeerTLSInfo.ClientCertAuth {
				plog.Warningf("The scheme of peer url %s is HTTP while client cert auth (--peer-client-cert-auth) is enabled. Ignored client cert auth for this url.", u.String())
			}
		}
		if !cfg.PeerTLSInfo.Empty() {
			if tlscfg, err = cfg.PeerTLSInfo.ServerConfig(); err != nil {
				return nil, err
			}
		}
		if plns[i], err = rafthttp.NewListener(u, tlscfg); err != nil {
			return nil, err
		}
		plog.Info("listening for peers on ", u.String())
	}
	return plns, nil
}

func startClientListeners(cfg *Config) (sctxs map[string]*serveCtx, err error) {
	if cfg.ClientAutoTLS && cfg.ClientTLSInfo.Empty() {
		chosts := make([]string, len(cfg.LCUrls))
		for i, u := range cfg.LCUrls {
			chosts[i] = u.Host
		}
		cfg.ClientTLSInfo, err = transport.SelfCert(path.Join(cfg.Dir, "fixtures/client"), chosts)
		if err != nil {
			plog.Fatalf("could not get certs - %v", err)
		}
	} else if cfg.ClientAutoTLS {
		plog.Warningf("ignoring client auto TLS since certs given")
	}

	sctxs = make(map[string]*serveCtx)
	for _, u := range cfg.LCUrls {
		sctx := newServeCtx()

		if u.Scheme == "http" || u.Scheme == "unix" {
			if !cfg.ClientTLSInfo.Empty() {
				plog.Warningf("The scheme of client url %s is HTTP while peer key/cert files are presented. Ignored key/cert files.", u.String())
			}
			if cfg.ClientTLSInfo.ClientCertAuth {
				plog.Warningf("The scheme of client url %s is HTTP while client cert auth (--client-cert-auth) is enabled. Ignored client cert auth for this url.", u.String())
			}
		}
		if (u.Scheme == "https" || u.Scheme == "unixs") && cfg.ClientTLSInfo.Empty() {
			return nil, fmt.Errorf("TLS key/cert (--cert-file, --key-file) must be provided for client url %s with HTTPs scheme", u.String())
		}

		proto := "tcp"
		if u.Scheme == "unix" || u.Scheme == "unixs" {
			proto = "unix"
		}

		sctx.secure = u.Scheme == "https" || u.Scheme == "unixs"
		sctx.insecure = !sctx.secure
		if oldctx := sctxs[u.Host]; oldctx != nil {
			oldctx.secure = oldctx.secure || sctx.secure
			oldctx.insecure = oldctx.insecure || sctx.insecure
			continue
		}

		if sctx.listener, err = net.Listen(proto, u.Host); err != nil {
			return nil, err
		}

		if fdLimit, fderr := runtimeutil.FDLimit(); fderr == nil {
			if fdLimit <= reservedInternalFDNum {
				plog.Fatalf("file descriptor limit[%d] of CSF process is too low, and should be set higher than %d to ensure internal usage", fdLimit, reservedInternalFDNum)
			}
			sctx.listener = transport.LimitListener(sctx.listener, int(fdLimit-reservedInternalFDNum))
		}

		if proto == "tcp" {
			if sctx.listener, err = transport.NewKeepAliveListener(sctx.listener, "tcp", nil); err != nil {
				return nil, err
			}
		}

		plog.Info("listening for client requests on ", u.Host)
		defer func() {
			if err != nil {
				sctx.listener.Close()
				plog.Info("stopping listening for client requests on ", u.Host)
			}
		}()
		sctx.userHandlers = cfg.UserHandlers
		sctxs[u.Host] = sctx
	}
	return sctxs, nil
}

func (e *CSFNode) Close() {
	for _, sctx := range e.sctxs {
		sctx.cancel()
	}
	for i := range e.Peers {
		if e.Peers[i] != nil {
			e.Peers[i].Close()
		}
	}
	for i := range e.Clients {
		if e.Clients[i] != nil {
			e.Clients[i].Close()
		}
	}
	// SERVICE: stop
	for sid, sv := range e.shub {
		plog.Infof("stop Service(%s)", sid)
		sv.StopService(e)
	}
	e.doStop()
}

func (e *CSFNode) OnReady(waitDuration time.Duration) bool {
	select {
	case <-e.ReadyNotify():
		return true
	case <-time.After(waitDuration):
		return false
	}
}

func (e *CSFNode) serve() (err error) {
	var ctlscfg *tls.Config
	if !e.Cfg.ClientTLSInfo.Empty() {
		plog.Infof("ClientTLS: %s", e.Cfg.ClientTLSInfo)
		if ctlscfg, err = e.Cfg.ClientTLSInfo.ServerConfig(); err != nil {
			return err
		}
	}

	if e.Cfg.CorsInfo.String() != "" {
		plog.Infof("cors = %s", e.Cfg.CorsInfo)
	}

	// Start the peer server in a goroutine
	ph := e.newPeerHandler()
	for _, l := range e.Peers {
		go func(l net.Listener) {
			e.errc <- servePeerHTTP(l, ph)
		}(l)
	}

	// Start a client server goroutine for each listen address
	ch := http.Handler(&cors.CORSHandler{
		Handler: e.newClientHandler(),
		Info:    e.Cfg.CorsInfo,
	})
	for _, sctx := range e.sctxs {
		// read timeout does not work with http close notify
		// TODO: https://github.com/golang/go/issues/9524
		go func(s *serveCtx) {
			e.errc <- s.serve(e, ctlscfg, ch, e.errc)
		}(sctx)
	}
	return nil
}

func (e *CSFNode) newClientHandler() http.Handler {
	mux := http.NewServeMux()
	// SERVICE: build client handler
	for _, sv := range e.Cfg.shub {
		plog.Infof("start Service(%s)", sv.ServiceID())
		sv.BuildClientHandler(e, mux)
	}

	if e.Cfg.ClientHandlerFactory != nil {
		return e.Cfg.ClientHandlerFactory(e, mux)
	}
	return mux
}

func (e *CSFNode) newPeerHandler() http.Handler {
	raftHandler := e.RaftHandler()
	mux := http.NewServeMux()
	mux.HandleFunc("/", http.NotFound)
	mux.Handle(rafthttp.RaftPrefix, raftHandler)
	mux.Handle(rafthttp.RaftPrefix+"/", raftHandler)
	return mux
}

func (e *CSFNode) Err() <-chan error { return e.errc }
