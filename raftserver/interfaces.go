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
	ClusterID      uint64
	WALDir         string
	InitPeers      []raft.Peer
	TickerDuration time.Duration
}

func NewConfig() *Config {
	r := &Config{
		TickerDuration: 500 * time.Millisecond,
	}
	r.ID = 0x01
	r.ElectionTick = 10
	r.HeartbeatTick = 1
	r.MaxSizePerMsg = 4096
	r.MaxInflightMsgs = 256
	return r
}

type RaftServerDebuger interface {
	ReceivedMessage(msg raftpb.Message)
	SaveToWAL(st *raftpb.HardState, entries []raftpb.Entry)

	// Storage Debuger
	StorageInitialState(rst raftpb.HardState, conf raftpb.ConfState, rerr error)
	StorageEntries(lo, hi, maxSize uint64, rents []raftpb.Entry, rerr error)
	StorageTerm(i uint64, rterm uint64, rerr error)
	StorageLastIndex(rindex uint64, rerr error)
	StorageFirstIndex(rindex uint64, rerr error)
	StorageSnapshot(rss raftpb.Snapshot, rerr error)
}

type RaftServerHandler interface {
	ApplyRaftSnapshot(snapshot raftpb.Snapshot) error

	ApplyRaftUpdate(entry raftpb.Entry) error

	LeadershipUpdate(isLocalLeader bool)
}

type RaftServerTransport interface {
	SendMessage(msgs []raftpb.Message)
}
