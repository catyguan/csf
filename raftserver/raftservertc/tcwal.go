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
	"log"
	"path/filepath"
	"time"

	"github.com/catyguan/csf/raft"
)

func create1wal() *network {
	nw := newNetwork()
	peers := []raft.Peer{
		{ID: 1},
	}

	nw.Add(newVM_WAL(1, filepath.Join(testDir, "wal1"), peers))

	return nw
}

func create1wal2mem() *network {
	nw := newNetwork()
	peers := []raft.Peer{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}

	nw.Add(newVM_WAL(1, filepath.Join(testDir, "wal3"), peers))
	nw.Add(newVM(2, peers))
	nw.Add(newVM(3, peers))

	return nw
}

func do_Test_WAL() {
	nw := create1wal()
	nw.VM(1).slog = true
	nw.VM(1).elog = true
	err := nw.StartAll()
	if err != nil {
		log.Fatal(err)
		return
	}
	defer nw.StopAll()

	time.Sleep(10 * time.Second)

	// nw.VM(1).Add(context.Background(), 100)
	time.Sleep(5 * time.Second)

	nw.DumpStatus()
	time.Sleep(1 * time.Second)
}

func do_Test_WAL_Simple() {
	nw := create1wal2mem()
	exec_Test_Simple(nw)
}
