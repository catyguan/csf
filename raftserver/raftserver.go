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

// Package interfaces defines raftserver.
package raftserver

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/catyguan/csf/raft"
	"github.com/catyguan/csf/raft/raftpb"
	"github.com/catyguan/csf/wal"
)

type debugStorage struct {
	debuger RaftServerDebuger
	backend raft.Storage
}

func (this *debugStorage) InitialState() (raftpb.HardState, raftpb.ConfState, error) {
	r1, r2, r3 := this.backend.InitialState()
	if this.debuger != nil {
		this.debuger.StorageInitialState(r1, r2, r3)
	}
	return r1, r2, r3
}

func (this *debugStorage) Entries(lo, hi, maxSize uint64) ([]raftpb.Entry, error) {
	r1, r2 := this.backend.Entries(lo, hi, maxSize)
	if this.debuger != nil {
		this.debuger.StorageEntries(lo, hi, maxSize, r1, r2)
	}
	return r1, r2
}

func (this *debugStorage) Term(i uint64) (uint64, error) {
	r1, r2 := this.backend.Term(i)
	if this.debuger != nil {
		this.debuger.StorageTerm(i, r1, r2)
	}
	return r1, r2
}

func (this *debugStorage) LastIndex() (uint64, error) {
	r1, r2 := this.backend.LastIndex()
	if this.debuger != nil {
		this.debuger.StorageLastIndex(r1, r2)
	}
	return r1, r2
}

func (this *debugStorage) FirstIndex() (uint64, error) {
	r1, r2 := this.backend.FirstIndex()
	if this.debuger != nil {
		this.debuger.StorageFirstIndex(r1, r2)
	}
	return r1, r2
}

func (this *debugStorage) Snapshot() (raftpb.Snapshot, error) {
	r1, r2 := this.backend.Snapshot()
	if this.debuger != nil {
		this.debuger.StorageSnapshot(r1, r2)
	}
	return r1, r2
}

type RaftServer struct {
	id         uint64
	Node       raft.Node
	w          *wal.WAL
	Debuger    RaftServerDebuger
	Handler    RaftServerHandler
	Transport  RaftServerTransport
	memStorage *raft.MemoryStorage

	closec chan interface{}
	done   chan interface{}
	// a chan to send out readState
	readStateC chan raft.ReadState

	mu           sync.Mutex
	leaderId     uint64
	lastLeadTime time.Time
}

func NewRaftServer(tr RaftServerTransport, h RaftServerHandler, d RaftServerDebuger) (*RaftServer, error) {
	r := new(RaftServer)
	r.Transport = tr
	r.Handler = h
	r.Debuger = d
	r.done = make(chan interface{})
	r.readStateC = make(chan raft.ReadState, 1)
	return r, nil
}

func (this *RaftServer) NodeId() uint64 {
	if this.Node == nil {
		return 0
	}
	return this.id
}

func (this *RaftServer) LeaderId() uint64 {
	return atomic.LoadUint64(&this.leaderId)
}

func (this *RaftServer) StartRaftServer(cfg *Config) error {
	this.id = cfg.ID
	if cfg.WALDir == "" {
		ncfg := cfg.Config
		this.memStorage = raft.NewMemoryStorage()
		ncfg.Storage = this.memStorage
		if this.Debuger != nil {
			ncfg.Storage = &debugStorage{
				backend: ncfg.Storage,
				debuger: this.Debuger,
			}
		}
		this.Node = raft.StartNode(&ncfg, cfg.InitPeers)
	} else {
		w, newone, err := wal.InitWAL(cfg.WALDir, cfg.ClusterID)
		if err != nil {
			return err
		}
		err = w.Begin()
		if err != nil {
			return err
		}
		this.w = w

		ncfg := cfg.Config
		ncfg.Storage = w
		if this.Debuger != nil {
			ncfg.Storage = &debugStorage{
				backend: ncfg.Storage,
				debuger: this.Debuger,
			}
		}
		if newone {
			this.Node = raft.StartNode(&ncfg, cfg.InitPeers)
		} else {
			this.Node = raft.RestartNode(&ncfg)
		}
	}

	ticker := time.Tick(cfg.TickerDuration)
	closec := make(chan interface{})

	go this.run(closec, ticker)
	this.closec = closec

	return nil
}

func (this *RaftServer) StopRaftServer() {
	if this.closec != nil {
		select {
		case this.closec <- nil:
			// Not already stopped, so trigger it
		case <-this.done:
			// Node has already been stopped - no need to do anything
			return
		}
		// Block until the stop has been acknowledged by run()
		<-this.done
	}
}

func (this *RaftServer) onClose() {
	plog.Infof("[%v] closing RaftServer", this.id)
	close(this.done)
}

func (this *RaftServer) run(closec chan interface{}, ticker <-chan time.Time) {
	islead := false
	defer this.onClose()

	for {
		select {
		case <-closec:
			// onClose
			return
		case <-ticker:
			// Leader:Send Heartbeat, Follower:Election timeout
			this.Node.Tick()
		case rd := <-this.Node.Ready():
			if rd.SoftState != nil {
				if lead := atomic.LoadUint64(&this.leaderId); rd.SoftState.Lead != raft.None && lead != rd.SoftState.Lead {
					this.mu.Lock()
					this.lastLeadTime = time.Now()
					this.mu.Unlock()
					leaderChanges.Inc()
				}

				if rd.SoftState.Lead == raft.None {
					hasLeader.Set(0)
				} else {
					hasLeader.Set(1)
				}

				atomic.StoreUint64(&this.leaderId, rd.SoftState.Lead)
				islead = rd.RaftState == raft.StateLeader
				this.Handler.LeadershipUpdate(islead)
			}
			if len(rd.ReadStates) != 0 {
				select {
				case this.readStateC <- rd.ReadStates[len(rd.ReadStates)-1]:
				case <-closec:
					return
				}
			}

			// the leader can write to its disk in parallel with replicating to the followers and them
			// writing to their disks.
			// For more details, check raft thesis 10.2.1
			if islead {
				this.Transport.SendMessage(rd.Messages)
			}

			// TODO:  持久化日志和状态
			if this.Debuger != nil {
				this.Debuger.SaveToWAL(&rd.HardState, rd.Entries)
			}
			if this.w != nil {
				if err := this.w.Save(rd.HardState, rd.Entries); err != nil {
					plog.Fatalf("raft save state and entries error: %v", err)
				}
			}
			if !raft.IsEmptyHardState(rd.HardState) {
				proposalsCommitted.Set(float64(rd.HardState.Commit))
			}

			// 持久化快照
			if !raft.IsEmptySnap(rd.Snapshot) {
				plog.Infof("raft apply incoming snapshot at index %d", rd.Snapshot.Metadata.Index)
				// if err := r.storage.SaveSnap(rd.Snapshot); err != nil {
				// 	plog.Fatalf("raft save snapshot error: %v", err)
				// }
				if this.memStorage != nil {
					this.memStorage.ApplySnapshot(rd.Snapshot)
				}
			}
			if this.memStorage != nil {
				this.memStorage.Append(rd.Entries)
			}

			if !islead {
				this.Transport.SendMessage(rd.Messages)
			}
			if !raft.IsEmptySnap(rd.Snapshot) {
				this.Handler.ApplyRaftSnapshot(rd.Snapshot)
			}
			for _, entry := range rd.CommittedEntries {
				this.Handler.ApplyRaftUpdate(entry)
				// lastIndex
				if entry.Type == raftpb.EntryConfChange {
					var cc raftpb.ConfChange
					cc.Unmarshal(entry.Data)
					plog.Infof("ApplyConfChange - %v", cc.String())
					this.Node.ApplyConfChange(cc)
				}
			}
			this.Node.Advance()
		}
	}
}

func (this *RaftServer) OnRecvRaftRPC(m raftpb.Message) {
	if this.Debuger != nil {
		this.Debuger.ReceivedMessage(m)
	}
	this.onRecvRaftRPC(context.Background(), m)
}

func (this *RaftServer) onRecvRaftRPC(ctx context.Context, m raftpb.Message) {
	this.Node.Step(ctx, m)
}

func (this *RaftServer) Propose(ctx context.Context, data []byte) {
	this.Node.Propose(ctx, data)
}
