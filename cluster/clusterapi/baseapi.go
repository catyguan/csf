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

package clusterapi

import (
	"encoding/json"
	"expvar"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/catyguan/csf/cluster"
	"github.com/catyguan/csf/cluster/membership"
	csfErr "github.com/catyguan/csf/error"
	"github.com/catyguan/csf/interfaces"
	"github.com/catyguan/csf/pkg/capnslog"
	"github.com/catyguan/csf/pkg/types"
	"github.com/catyguan/csf/raft"
	"github.com/catyguan/csf/stats"
	"github.com/catyguan/csf/version"
	"github.com/jonboulle/clockwork"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
)

const (
	shutdownPrefix = "/csf/shutdown"
	snapshotPrefix = "/csf/snapshot"
	membersPrefix  = "/csf/members"
	statsPrefix    = "/csf/stats"
	metricsPath    = "/csf/metrics"
	healthPath     = "/csf/health"
	versionPath    = "/csf/version"
	varsPath       = "/csf/debug/vars"
	configPath     = "/csf/config"
	pprofPrefix    = "/csf/debug/pprof"
)

// NewClientHandler generates a muxed http.Handler with the given parameters to serve etcd client requests.
func NewClientHandler(server *cluster.CSFNode, mux *http.ServeMux) http.Handler {
	sec := server.Cfg.ACL

	sh := &statsHandler{
		stats: server,
	}

	mh := &membersHandler{
		sec:     sec,
		server:  server,
		cluster: server.Cluster(),
		timeout: server.ClientRequestTime(),
		clock:   clockwork.NewRealClock(),
		clientCertAuthEnabled: server.Cfg.ClientCertAuthEnabled,
	}

	mux.HandleFunc("/", http.NotFound)
	mux.Handle(healthPath, healthHandler(server))
	mux.HandleFunc(versionPath, versionHandler(server.Cluster(), serveVersion))
	mux.HandleFunc(shutdownPrefix, shutdowServerHandler(server))
	mux.HandleFunc(snapshotPrefix, snapshotHandler(server))
	// mux.HandleFunc(statsPrefix+"/csf/store", sh.serveStore)
	mux.HandleFunc(statsPrefix+"/self", sh.serveSelf)
	mux.HandleFunc(statsPrefix+"/leader", sh.serveLeader)
	mux.HandleFunc(varsPath, serveVars)
	mux.HandleFunc(configPath+"/local/log", logHandleFunc)
	mux.Handle(metricsPath, prometheus.Handler())
	mux.Handle(membersPrefix, mh)
	mux.Handle(membersPrefix+"/", mh)

	if server.IsPprofEnabled() {
		plog.Infof("pprof is enabled under %s", pprofPrefix)

		mux.HandleFunc(pprofPrefix+"/", pprof.Index)
		mux.HandleFunc(pprofPrefix+"/profile", pprof.Profile)
		mux.HandleFunc(pprofPrefix+"/symbol", pprof.Symbol)
		mux.HandleFunc(pprofPrefix+"/cmdline", pprof.Cmdline)

		// TODO: currently, we don't create an entry for pprof.Trace,
		// because go 1.4 doesn't provide it. After support of go 1.4 is dropped,
		// we should add the entry.

		mux.Handle(pprofPrefix+"/heap", pprof.Handler("heap"))
		mux.Handle(pprofPrefix+"/goroutine", pprof.Handler("goroutine"))
		mux.Handle(pprofPrefix+"/threadcreate", pprof.Handler("threadcreate"))
		mux.Handle(pprofPrefix+"/block", pprof.Handler("block"))
	}

	return requestLogger(mux)
}

type membersHandler struct {
	sec                   interfaces.ACL
	server                *cluster.CSFNode
	cluster               Cluster
	timeout               time.Duration
	clock                 clockwork.Clock
	clientCertAuthEnabled bool
}

func (h *membersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !interfaces.AllowMethod(w, r.Method, "GET") {
		return
	}
	if !interfaces.CheckCSFAccess(h.server, r) {
		interfaces.WriteNoAuth(w, r)
		return
	}
	// w.Header().Set("X-CSF-Cluster-ID", h.cluster.ID().String())

	_, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	switch r.Method {
	case "GET":
		switch interfaces.TrimPrefix(r.URL.Path, membersPrefix) {
		case "":
			mc := newMemberCollection(h.cluster.Members())
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(mc); err != nil {
				plog.Warningf("failed to encode members response (%v)", err)
			}
		case "leader":
			id := h.server.Leader()
			if id == 0 {
				interfaces.WriteError(w, r, interfaces.NewHTTPError(http.StatusServiceUnavailable, "During election"))
				return
			}
			m := newMember(h.cluster.Member(id))
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(m); err != nil {
				plog.Warningf("failed to encode members response (%v)", err)
			}
		default:
			interfaces.WriteError(w, r, interfaces.NewHTTPError(http.StatusNotFound, "Not found"))
		}
	}
}

type statsHandler struct {
	stats stats.Stats
}

func (h *statsHandler) serveSelf(w http.ResponseWriter, r *http.Request) {
	if !interfaces.AllowMethod(w, r.Method, "GET") {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(h.stats.SelfStats())
}

func (h *statsHandler) serveLeader(w http.ResponseWriter, r *http.Request) {
	if !interfaces.AllowMethod(w, r.Method, "GET") {
		return
	}
	stats := h.stats.LeaderStats()
	if stats == nil {
		interfaces.WriteError(w, r, interfaces.NewHTTPError(http.StatusForbidden, "not current leader"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(stats)
}

func serveVars(w http.ResponseWriter, r *http.Request) {
	if !interfaces.AllowMethod(w, r.Method, "GET") {
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{\n")
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}

// TODO: change etcdserver to raft interface when we have it.
//       add test for healthHandler when we have the interface ready.
func healthHandler(server *cluster.CSFNode) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !interfaces.AllowMethod(w, r.Method, "GET") {
			return
		}

		if uint64(server.Leader()) == raft.None {
			http.Error(w, `{"health": "false", "leader" : "none"}`, http.StatusServiceUnavailable)
			return
		}

		// wait for raft's progress
		index := server.Index()
		for i := 0; i < 3; i++ {
			time.Sleep(250 * time.Millisecond)
			if server.Index() > index {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"health": "true"}`))
				return
			}
		}

		http.Error(w, `{"health": "false", "progress" : "false"}`, http.StatusServiceUnavailable)
		return
	}
}

func versionHandler(c Cluster, fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v := c.Version()
		if v != nil {
			fn(w, r, v.String())
		} else {
			fn(w, r, "not_decided")
		}
	}
}

func shutdowServerHandler(server *cluster.CSFNode) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !interfaces.AllowMethod(w, r.Method, "GET") {
			return
		}
		if !interfaces.CheckCSFAccess(server, r) {
			interfaces.WriteNoAuth(w, r)
			return
		}
		vs := make(map[string]bool)
		vs["result"] = true

		w.Header().Set("Content-Type", "application/json")
		b, err := json.Marshal(&vs)
		if err != nil {
			plog.Panicf("cannot marshal versions to json (%v)", err)
		}
		w.Write(b)
		server.HardStop(false)
	}
}

func snapshotHandler(server *cluster.CSFNode) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !interfaces.AllowMethod(w, r.Method, "GET") {
			return
		}
		if !interfaces.CheckCSFAccess(server, r) {
			interfaces.WriteNoAuth(w, r)
			return
		}
		vs := make(map[string]bool)
		vs["result"] = server.AskSnapshot()

		w.Header().Set("Content-Type", "application/json")
		b, err := json.Marshal(&vs)
		if err != nil {
			plog.Panicf("cannot marshal versions to json (%v)", err)
		}
		w.Write(b)
	}
}

func serveVersion(w http.ResponseWriter, r *http.Request, clusterV string) {
	if !interfaces.AllowMethod(w, r.Method, "GET") {
		return
	}
	vs := version.Versions{
		Server:  version.Version,
		Cluster: clusterV,
	}

	w.Header().Set("Content-Type", "application/json")
	b, err := json.Marshal(&vs)
	if err != nil {
		plog.Panicf("cannot marshal versions to json (%v)", err)
	}
	w.Write(b)
}

func logHandleFunc(w http.ResponseWriter, r *http.Request) {
	if !interfaces.AllowMethod(w, r.Method, "PUT") {
		return
	}

	in := struct{ Level string }{}

	d := json.NewDecoder(r.Body)
	if err := d.Decode(&in); err != nil {
		interfaces.WriteError(w, r, interfaces.NewHTTPError(http.StatusBadRequest, "Invalid json body"))
		return
	}

	logl, err := capnslog.ParseLevel(strings.ToUpper(in.Level))
	if err != nil {
		interfaces.WriteError(w, r, interfaces.NewHTTPError(http.StatusBadRequest, "Invalid log level "+in.Level))
		return
	}

	plog.Noticef("globalLogLevel set to %q", logl.String())
	capnslog.SetGlobalLogLevel(logl)
	w.WriteHeader(http.StatusNoContent)
}

func trimErrorPrefix(err error, prefix string) error {
	if e, ok := err.(*csfErr.Error); ok {
		e.Cause = strings.TrimPrefix(e.Cause, prefix)
	}
	return err
}

func unmarshalRequest(r *http.Request, req json.Unmarshaler, w http.ResponseWriter) bool {
	ctype := r.Header.Get("Content-Type")
	semicolonPosition := strings.Index(ctype, ";")
	if semicolonPosition != -1 {
		ctype = strings.TrimSpace(strings.ToLower(ctype[0:semicolonPosition]))
	}
	if ctype != "application/json" {
		interfaces.WriteError(w, r, interfaces.NewHTTPError(http.StatusUnsupportedMediaType, fmt.Sprintf("Bad Content-Type %s, accept application/json", ctype)))
		return false
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		interfaces.WriteError(w, r, interfaces.NewHTTPError(http.StatusBadRequest, err.Error()))
		return false
	}
	if err := req.UnmarshalJSON(b); err != nil {
		interfaces.WriteError(w, r, interfaces.NewHTTPError(http.StatusBadRequest, err.Error()))
		return false
	}
	return true
}

func getID(p string, w http.ResponseWriter) (types.ID, bool) {
	idStr := interfaces.TrimPrefix(p, membersPrefix)
	if idStr == "" {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return 0, false
	}
	id, err := types.IDFromString(idStr)
	if err != nil {
		interfaces.WriteError(w, nil, interfaces.NewHTTPError(http.StatusNotFound, fmt.Sprintf("No such member: %s", idStr)))
		return 0, false
	}
	return id, true
}

func newMemberCollection(ms []*membership.Member) *apiMemberCollection {
	c := apiMemberCollection(make([]apiMember, len(ms)))

	for i, m := range ms {
		c[i] = newMember(m)
	}

	return &c
}

func newMember(m *membership.Member) apiMember {
	tm := apiMember{
		ID:         m.ID.String(),
		Name:       m.Name,
		PeerURLs:   make([]string, len(m.PeerURLs)),
		ClientURLs: make([]string, len(m.ClientURLs)),
	}

	copy(tm.PeerURLs, m.PeerURLs.StringSlice())
	copy(tm.ClientURLs, m.ClientURLs.StringSlice())

	return tm
}
