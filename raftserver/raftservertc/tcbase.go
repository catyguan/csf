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

// Package raftservertc defines a test app for raftserver.
package main

import (
	"context"
	"log"
	"time"

	"github.com/catyguan/csf/raft"
)

func create3mem() *network {
	nw := newNetwork()
	peers := []raft.Peer{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}

	nw.Add(newVM(1, peers))
	nw.Add(newVM(2, peers))
	nw.Add(newVM(3, peers))

	return nw
}

func create1vm() *network {
	nw := newNetwork()
	peers := []raft.Peer{
		{ID: 1},
	}

	nw.Add(newVM(1, peers))

	return nw
}

func do_Test_Simple() {
	nw := create1vm()
	exec_Test_Simple(nw)
}

func exec_Test_Simple(nw *network) {
	nw.VM(1).elog = true
	nw.VM(1).slog = true
	err := nw.StartAll()
	if err != nil {
		log.Fatal(err)
		return
	}
	defer nw.StopAll()

	time.Sleep(10 * time.Second)
	nw.VM(1).Add(context.Background(), 100)
	time.Sleep(10 * time.Second)
	nw.DumpStatus()
	time.Sleep(1 * time.Second)
}

func do_Test_Base() {
	nw := create3mem()
	err := nw.StartAll()
	if err != nil {
		log.Fatal(err)
		return
	}
	defer nw.StopAll()

	time.Sleep(10 * time.Second)

	nw.DumpStatus()

	nw.VM(1).Add(context.Background(), 100)
	time.Sleep(1 * time.Second)
	nw.DumpStatus()

	time.Sleep(1 * time.Second)
}

func do_Test_LeaderDisconn() {
	nw := create3mem()
	nw.DisableLog(0)
	err := nw.StartAll()
	if err != nil {
		log.Fatal(err)
		return
	}
	defer nw.StopAll()

	time.Sleep(10 * time.Second)
	nw.DumpStatus()

	// find leader and disconnect it
	ls := nw.FindLeader()
	if len(ls) == 0 {
		log.Printf("!!!!!! no leader")
		return
	}

	lvm := ls[0]
	log.Printf("!!!!!! %v disconnect", lvm.id)
	lvm.EnableLog(true)
	nw.ConnBreak(lvm.id, false)

	ovm := nw.PickOther(lvm.id)
	if ovm == nil {
		log.Printf("!!!!!! no other")
		return
	}

	time.Sleep(10 * time.Second)
	ovm.Add(context.Background(), 100)
	lvm.Add(context.Background(), 200)
	time.Sleep(2 * time.Second)
	nw.DumpStatus()

	// reconnect
	log.Printf("!!!!!! %v  reconnect", lvm.id)
	nw.ConnBreak(lvm.id, true)

	time.Sleep(10 * time.Second)
	nw.DumpStatus()

	time.Sleep(1 * time.Second)
}
