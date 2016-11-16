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
)

type RaftServer struct {
	id         uint64
	Node       raft.Node
	Handler    RaftServerHandler
	memStorage *raft.MemoryStorage

	closec chan interface{}
	done   chan interface{}
	// a chan to send out readState
	readStateC chan raft.ReadState

	mu           sync.Mutex
	leaderId     uint64
	lastLeadTime time.Time
}

func NewRaftServer(h RaftServerHandler) (*RaftServer, error) {
	r := new(RaftServer)
	r.Handler = h
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
	if cfg.InMemory {
		ncfg := cfg.Config
		this.memStorage = raft.NewMemoryStorage()
		ncfg.Storage = this.memStorage
		this.Node = raft.StartNode(&ncfg, cfg.InitPeers)
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
				this.Handler.SendMessage(rd.Messages)
			}

			// TODO:  持久化日志和状态
			this.Handler.Debug(&rd.HardState, rd.Entries)
			// if err := r.storage.Save(rd.HardState, rd.Entries); err != nil {
			// 	plog.Fatalf("raft save state and entries error: %v", err)
			// }
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
				// gofail: var raftBeforeFollowerSend struct{}
				this.Handler.SendMessage(rd.Messages)
			}
			this.Handler.ApplyRaftUpdates(rd.CommittedEntries, &rd.Snapshot)
			this.Node.Advance()
		}
	}
}

func (this *RaftServer) OnRecvRaftRPC(m raftpb.Message) {
	this.onRecvRaftRPC(context.Background(), m)
}

func (this *RaftServer) onRecvRaftRPC(ctx context.Context, m raftpb.Message) {
	this.Node.Step(ctx, m)
}

func (this *RaftServer) Propose(ctx context.Context, data []byte) {
	this.Node.Propose(ctx, data)
}
