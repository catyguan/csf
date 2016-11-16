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
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/catyguan/csf/pkg/capnslog"
	"github.com/catyguan/csf/raft"
	"github.com/catyguan/csf/raft/raftpb"
	"github.com/catyguan/csf/raftserver"
)

type handler struct {
	log     *capnslog.PackageLogger
	servers []*raftserver.RaftServer
}

func (this *handler) Debug(st *raftpb.HardState, entries []raftpb.Entry) {
	if !raftpb.IsEmptyHardState(*st) {
		this.log.Printf("DEBUG HardState - %v", st.String())
	}
	if len(entries) > 0 {
		this.log.Printf("DEBUG Entries: - %v", len(entries))
		for i, e := range entries {
			this.log.Printf(" --- %v: %v", i, e.String())
		}
	}
}

func (this *handler) ApplyRaftUpdates(entries []raftpb.Entry, snapshot *raftpb.Snapshot) error {
	if len(entries) == 0 && raft.IsEmptySnap(*snapshot) {
		return nil
	}

	this.log.Printf("ApplyRaftUpdates - %v", len(entries))
	for i, e := range entries {
		this.log.Printf(" --- %v: %v", i, e.String())
	}
	if !raft.IsEmptySnap(*snapshot) {
		this.log.Printf(" --- snapshot: %v", snapshot)
	}
	return nil
}

func (this *handler) LeadershipUpdate(isLocalLeader bool) {
	this.log.Printf("LeadershipUpdate - %v", isLocalLeader)
}

func (this *handler) SendMessage(msgs []raftpb.Message) {
	if len(msgs) == 0 {
		return
	}
	c := 0
	for i, msg := range msgs {
		l := true
		switch msg.Type {
		case raftpb.MsgHeartbeat, raftpb.MsgHeartbeatResp:
			l = false
		}
		if l {
			c++
			this.log.Printf(" --- %v: %v", i, msg)
		}
		for _, s := range this.servers {
			if s.NodeId() == msg.To {
				s.OnRecvRaftRPC(msg)
			}
		}
	}
	if c > 0 {
		this.log.Printf("SendMessage - len(%v)", len(msgs))
	}
}

func main() {
	// wal.SegmentSizeBytes = 16 * 1024

	var configFile string
	flag.StringVar(&configFile, "C", "", "Path to the server configuration file")
	flag.Parse()

	// if configFile == "" {
	// 	fmt.Printf("config file -C invalid")
	// 	os.Exit(-1)
	// }
	nodes := []uint64{1, 2, 3}
	peers := make([]raft.Peer, len(nodes))
	sers := make([]*raftserver.RaftServer, len(nodes))
	for i, id := range nodes {
		peers[i] = raft.Peer{ID: id}
		h := &handler{
			log:     capnslog.NewPackageLogger("github.com/catyguan/csf", fmt.Sprintf("node_%v", id)),
			servers: sers,
		}
		sers[i], _ = raftserver.NewRaftServer(h)
	}

	for i, id := range nodes {
		cfg := raftserver.NewConfig()
		cfg.ID = id
		cfg.InMemory = true
		cfg.InitPeers = peers

		ser := sers[i]
		err := ser.StartRaftServer(cfg)
		if err != nil {
			log.Fatalf("start raft server fail - %v", err)
			return
		}
		defer ser.StopRaftServer()
	}

	time.Sleep(10 * time.Second)

	dumpStatus(sers)

	sers[0].Propose(context.Background(), []byte("hello world"))
	time.Sleep(1 * time.Second)
	dumpStatus(sers)

	time.Sleep(50 * time.Second)

	log.Printf("BYE~~~~~\n")
}

func dumpStatus(sers []*raftserver.RaftServer) {
	for _, s := range sers {
		st := s.Node.Status()
		log.Printf("!!!!!! Server(%v) - %v", s.NodeId(), st.String())
	}
}
