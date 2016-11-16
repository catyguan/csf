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

// Package raftserver defines raftnode + storage layer + transportation layer
package raftserver

import (
	"time"

	"github.com/catyguan/csf/raft"
	"github.com/catyguan/csf/raft/raftpb"
)

type Config struct {
	raft.Config
	WALDir         string
	InitPeers      []raft.Peer
	InMemory       bool
	TickerDuration time.Duration
}

func NewConfig() *Config {
	r := &Config{
		InMemory:       false,
		TickerDuration: 500 * time.Millisecond,
	}
	r.ID = 0x01
	r.ElectionTick = 10
	r.HeartbeatTick = 1
	r.MaxSizePerMsg = 4096
	r.MaxInflightMsgs = 256
	return r
}

type RaftServerHandler interface {
	Debug(st *raftpb.HardState, entries []raftpb.Entry)

	ApplyRaftUpdates(entries []raftpb.Entry, snapshot *raftpb.Snapshot) error

	LeadershipUpdate(isLocalLeader bool)

	SendMessage(msgs []raftpb.Message)
}
