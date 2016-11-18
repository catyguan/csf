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
	"encoding/binary"
	"fmt"

	"github.com/catyguan/csf/pkg/capnslog"
	"github.com/catyguan/csf/raft"
	"github.com/catyguan/csf/raft/raftpb"
	"github.com/catyguan/csf/raftserver"
)

type vm struct {
	id     uint64
	cfg    *raftserver.Config
	nw     *network
	elog   bool
	slog   bool
	log    *capnslog.PackageLogger
	server *raftserver.RaftServer

	value int64

	debugLastIndex  uint64
	debugFirstIndex uint64
}

func newVM(id uint64, peers []raft.Peer) *vm {
	cfg := raftserver.NewConfig()
	cfg.ID = id
	cfg.ClusterID = 1
	cfg.InitPeers = peers

	r := new(vm)
	r.id = id
	r.cfg = cfg
	r.elog = false
	r.log = capnslog.NewPackageLogger("github.com/catyguan/csf", fmt.Sprintf("vm_%v", r.id))
	return r
}

func newVM_WAL(id uint64, dir string, peers []raft.Peer) *vm {
	r := newVM(id, peers)
	r.cfg.WALDir = dir
	return r
}

func (this *vm) EnableLog(e bool) {
	this.elog = e
}

func (this *vm) Start() error {
	if this.server != nil {
		this.server.StopRaftServer()
		this.server = nil
	}
	s, err := raftserver.NewRaftServer(this, this, this)
	if err != nil {
		return fmt.Errorf("new raft server fail - %v", err)
	}
	this.server = s
	err = this.server.StartRaftServer(this.cfg)
	if err != nil {
		return fmt.Errorf("start raft server fail - %v", err)
	}
	return nil
}

func (this *vm) Stop() {
	if this.server != nil {
		this.server.StopRaftServer()
		this.server = nil
	}
}

func (this *vm) ReceivedMessage(msg raftpb.Message) {
	if !this.elog {
		return
	}
	switch msg.Type {
	case raftpb.MsgHeartbeat, raftpb.MsgHeartbeatResp:
		return
	}
	this.log.Printf("Received Message --- %v", msg)
}

func (this *vm) SaveToWAL(st *raftpb.HardState, entries []raftpb.Entry) {
	if !this.elog {
		return
	}
	if !raftpb.IsEmptyHardState(*st) {
		this.log.Printf("SaveToWAL HardState - %v", st.String())
	}
	if len(entries) > 0 {
		this.log.Printf("SaveToWAL Entries: - %v", len(entries))
		for i, e := range entries {
			this.log.Printf(" --- %v: %v", i, e.String())
		}
	}
}

func (this *vm) StorageInitialState(rst raftpb.HardState, conf raftpb.ConfState, rerr error) {
	if !this.slog {
		return
	}
	this.log.Infof("Storage.InitialState() (HardState:%v, ConfState:%v, error:%v)", rst, conf, rerr)
}

func (this *vm) StorageEntries(lo, hi, maxSize uint64, rents []raftpb.Entry, rerr error) {
	if !this.slog {
		return
	}
	this.log.Infof("Storage.Entries(lo:%v, hi:%v, maxSize:%v) ([]Entry:%v, error:%v)", lo, hi, maxSize, rents, rerr)
}

func (this *vm) StorageTerm(i uint64, rterm uint64, rerr error) {
	if !this.slog {
		return
	}
	this.log.Infof("Storage.Term(index:%v) (Term:%v, error:%v)", i, rterm, rerr)
}

func (this *vm) StorageLastIndex(rindex uint64, rerr error) {
	if !this.slog {
		return
	}
	if rerr == nil {
		if this.debugLastIndex == rindex {
			return
		}
		this.debugLastIndex = rindex
	} else {
		this.debugLastIndex = 0
	}
	this.log.Infof("Storage.LastIndex() (Index:%v, error:%v)", rindex, rerr)
}

func (this *vm) StorageFirstIndex(rindex uint64, rerr error) {
	if !this.slog {
		return
	}
	if rerr == nil {
		if this.debugFirstIndex == rindex {
			return
		}
		this.debugFirstIndex = rindex
	} else {
		this.debugFirstIndex = 0
	}
	this.log.Infof("Storage.FirstIndex() (Index:%v, error:%v)", rindex, rerr)
}

func (this *vm) StorageSnapshot(rss raftpb.Snapshot, rerr error) {
	if !this.slog {
		return
	}
	this.log.Infof("Storage.Snapshot() (Snapshot:%v, error:%v)", rss, rerr)
}

func (this *vm) ApplyRaftSnapshot(snapshot raftpb.Snapshot) error {
	if this.elog {
		this.log.Printf("ApplyRaftsnapshot - %v", snapshot)
	}
	return nil
}

func (this *vm) ApplyRaftUpdate(e raftpb.Entry) error {
	if this.elog {
		this.log.Printf("ApplyRaftUpdate - %v", e.String())
	}

	if e.Type == raftpb.EntryNormal && e.Data != nil {
		b := e.Data
		v := binary.LittleEndian.Uint32(b)
		s := b[3]
		if s == 0 {
			this.value += int64(v)
		} else {
			this.value -= int64(v)
		}
	}
	return nil
}

func (this *vm) LeadershipUpdate(isLocalLeader bool) {
	if this.elog {
		this.log.Printf("LeadershipUpdate - %v", isLocalLeader)
	}
}

func (this *vm) SendMessage(msgs []raftpb.Message) {
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
		if this.elog && l {
			if c == 0 {
				this.log.Printf("SendMessage")
			}
			c++
			this.log.Printf(" --- %v: %v", i, msg)
		}
		this.nw.SendMessage(this.id, msg)
	}
}

func (this *vm) Dump() string {
	s := fmt.Sprintf("V=%v, S=", this.value)
	if this.server == nil {
		s += "nil"
	} else {
		st := this.server.Node.Status()
		s += st.String()
	}
	return s
}

func (this *vm) Add(ctx context.Context, v uint32) {
	if this.server != nil {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, v)
		b[3] = 0
		this.server.Propose(ctx, b)
	}
}

func (this *vm) Sub(ctx context.Context, v uint32) {
	if this.server != nil {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, v)
		b[3] = 1
		this.server.Propose(ctx, b)
	}
}

func (this *vm) Value() int64 {
	return this.value
}
