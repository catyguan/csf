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

// Package setupnode defines a csf sample app.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/catyguan/csf/core"
	_ "github.com/catyguan/csf/httpsc/http4si"
	"github.com/catyguan/csf/httpsc/httpport"
	"github.com/catyguan/csf/pkg/osutil"
	"github.com/catyguan/csf/raft4si"
	"github.com/catyguan/csf/service/counter"
	"github.com/catyguan/csf/servicechannelhandler/schlog"
)

func main3() {

	smux := core.NewServiceMux()
	pmux := core.NewServiceMux()
	amux := core.NewServiceMux()

	var dir string
	cdir, errDir := os.Getwd()
	if errDir != nil {
		fmt.Printf("error - %s", cdir)
		return
	}
	var acccessCode string
	var np int
	flag.IntVar(&np, "n", 0, "node pos")
	flag.StringVar(&dir, "d", filepath.Join(cdir, "wal"), "WAL dir")
	flag.StringVar(&acccessCode, "ac", "testkey", "access code")
	flag.Parse()

	cpeers := make([]raft4si.RaftPeer, 3)
	cpeers[0] = raft4si.RaftPeer{
		NodeID:   8086,
		Location: "http://127.0.0.1:8086/peer?SERVICE=raft4si",
	}
	cpeers[1] = raft4si.RaftPeer{
		NodeID:   8087,
		Location: "http://127.0.0.1:8087/peer?SERVICE=raft4si",
	}
	cpeers[2] = raft4si.RaftPeer{
		NodeID:   8088,
		Location: "http://127.0.0.1:8088/peer?SERVICE=raft4si",
	}
	peers := []raft4si.RaftPeer{
		cpeers[0],
		cpeers[1],
		cpeers[2],
	}

	lport := cpeers[np].NodeID

	var rsc *raft4si.RaftServiceContainer
	if true {
		s := counter.NewCounterService()

		cfg := raft4si.NewConfig()
		// cfg.MemoryMode = true
		cfg.WALDir = fmt.Sprintf("%s_%d", dir, lport)
		cfg.BlockRollSize = 16 * 1024
		cfg.Symbol = "tcserver3"
		cfg.NodeID = lport
		cfg.InitPeers = peers
		cfg.SnapCount = 16
		cfg.NumberOfCatchUpEntries = 16
		cfg.MemberSeq = 10000
		cfg.AccessCode = acccessCode

		si := raft4si.NewRaftServiceContainer(s, cfg)
		rsc = si

		errS := si.Run()
		if errS != nil {
			fmt.Printf("run RaftServiceContainer fail - %v", errS)
			return
		}
		defer si.Close()

		sc := core.NewServiceChannel()
		sc.Next(schlog.NewLogger("SERVICE"))
		sc.Sink(si)

		smux.AddInvoker(counter.SERVICE_NAME, sc)
	}

	if rsc != nil {
		sc := core.NewServiceChannel()
		// sc.Next(schlog.NewLogger("RAFT"))
		sc.Next(rsc.CreateSign().ErrorShowDetail(true))
		sc.Sink(rsc.PeerInvoker())

		pmux.AddInvoker("raft4si", sc)
	}

	hmux := http.NewServeMux()

	pcfg := &httpport.Config{}
	pcfg.Addr = fmt.Sprintf(":%d", lport)
	pcfg.Host = ""

	port := httpport.NewPort(pcfg)
	port.BuildHttpMux(hmux, "/service", smux, nil)
	port.BuildHttpMux(hmux, "/peer", pmux, nil)
	port.BuildHttpMux(hmux, "/admin", amux, nil)

	err0 := port.StartServe(hmux)
	if err0 != nil {
		fmt.Printf("start fail - %v", err0)
		return
	}
	port.Run()
	defer port.Stop()

	log.Printf("Server is ready!")

	osutil.HandleInterrupts()

	tt := time.NewTicker(60 * time.Minute)

	select {
	case <-tt.C:
		log.Printf("BYE~~~~~\n")
	}

	osutil.Exit(0)
}
