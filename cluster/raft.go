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

package cluster

import (
	"expvar"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/catyguan/csf/cluster/membership"
	"github.com/catyguan/csf/pkg/capnslog"
	"github.com/catyguan/csf/pkg/pbutil"
	"github.com/catyguan/csf/pkg/types"
	"github.com/catyguan/csf/raft"
	"github.com/catyguan/csf/raft/raftpb"
	"github.com/catyguan/csf/rafthttp"
	"github.com/catyguan/csf/wal"
	"github.com/catyguan/csf/wal/walpb"
)

const (
	// Number of entries for slow follower to catch-up after compacting
	// the raft storage entries.
	// We expect the follower has a millisecond level latency with the leader.
	// The max throughput is around 10K. Keep a 5K entries is enough for helping
	// follower to catch up.
	numberOfCatchUpEntries = 5000

	// The max throughput of etcd will not exceed 100MB/s (100K * 1KB value).
	// Assuming the RTT is around 10ms, 1MB max size is large enough.
	maxSizePerMsg = 1 * 1024 * 1024
	// Never overflow the rafthttp buffer, which is 4096.
	// TODO: a better const?
	maxInflightMsgs = 4096 / 8
)

var (
	// protects raftStatus
	raftStatusMu sync.Mutex
	// indirection for expvar func interface
	// expvar panics when publishing duplicate name
	// expvar does not support remove a registered name
	// so only register a func that calls raftStatus
	// and change raftStatus as we need.
	raftStatus func() raft.Status
)

func init() {
	raft.SetLogger(capnslog.NewPackageLogger("github.com/catyguan/csf", "raft"))
	expvar.Publish("raft.status", expvar.Func(func() interface{} {
		raftStatusMu.Lock()
		defer raftStatusMu.Unlock()
		return raftStatus()
	}))
}

type RaftTimer interface {
	Index() uint64
	Term() uint64
}

// apply contains entries, snapshot to be applied. Once
// an apply is consumed, the entries will be persisted to
// to raft storage concurrently; the application must read
// raftDone before assuming the raft messages are stable.
type apply struct {
	entries  []raftpb.Entry
	snapshot raftpb.Snapshot
	raftDone <-chan struct{} // rx {} after raft has persisted messages
}

type raftNode struct {
	// Cache of the latest raft index and raft term the server has seen.
	// These three unit64 fields must be the first elements to keep 64-bit
	// alignment for atomic access to the fields.
	index uint64
	term  uint64
	lead  uint64

	mu sync.Mutex
	// last lead elected time
	lt time.Time

	raft.Node

	// a chan to send out apply
	applyc chan apply

	// a chan to send out readState
	readStateC chan raft.ReadState

	// utility
	ticker      <-chan time.Time
	raftStorage *raft.MemoryStorage
	storage     Storage

	// transport specifies the transport to send and receive msgs to members.
	// Sending messages MUST NOT block. It is okay to drop messages, since
	// clients should timeout and reissue their messages.
	// If transport is nil, server will panic.
	transport rafthttp.Transporter

	stopped chan struct{}
	done    chan struct{}
}

// start prepares and starts raftNode in a new goroutine. It is no longer safe
// to modify the fields after it has been started.
func (r *raftNode) start(rh *raftReadyHandler) {
	r.applyc = make(chan apply)
	r.stopped = make(chan struct{})
	r.done = make(chan struct{})

	go func() {
		defer r.onStop()
		islead := false

		for {
			select {
			case <-r.ticker:
				r.Tick()
			case rd := <-r.Ready():
				if rd.SoftState != nil {
					if lead := atomic.LoadUint64(&r.lead); rd.SoftState.Lead != raft.None && lead != rd.SoftState.Lead {
						r.mu.Lock()
						r.lt = time.Now()
						r.mu.Unlock()
						leaderChanges.Inc()
					}

					if rd.SoftState.Lead == raft.None {
						hasLeader.Set(0)
					} else {
						hasLeader.Set(1)
					}

					atomic.StoreUint64(&r.lead, rd.SoftState.Lead)
					islead = rd.RaftState == raft.StateLeader
					rh.leadershipUpdate()
				}

				if len(rd.ReadStates) != 0 {
					select {
					case r.readStateC <- rd.ReadStates[len(rd.ReadStates)-1]:
					case <-r.stopped:
						return
					}
				}

				raftDone := make(chan struct{}, 1)
				ap := apply{
					entries:  rd.CommittedEntries,
					snapshot: rd.Snapshot,
					raftDone: raftDone,
				}

				select {
				case r.applyc <- ap:
				case <-r.stopped:
					return
				}

				// the leader can write to its disk in parallel with replicating to the followers and them
				// writing to their disks.
				// For more details, check raft thesis 10.2.1
				if islead {
					// gofail: var raftBeforeLeaderSend struct{}
					rh.sendMessage(rd.Messages)
				}

				// gofail: var raftBeforeSave struct{}
				if err := r.storage.Save(rd.HardState, rd.Entries); err != nil {
					plog.Fatalf("raft save state and entries error: %v", err)
				}
				if !raft.IsEmptyHardState(rd.HardState) {
					proposalsCommitted.Set(float64(rd.HardState.Commit))
				}
				// gofail: var raftAfterSave struct{}

				if !raft.IsEmptySnap(rd.Snapshot) {
					// gofail: var raftBeforeSaveSnap struct{}
					if err := r.storage.SaveSnap(rd.Snapshot); err != nil {
						plog.Fatalf("raft save snapshot error: %v", err)
					}
					// gofail: var raftAfterSaveSnap struct{}
					r.raftStorage.ApplySnapshot(rd.Snapshot)
					plog.Infof("raft applied incoming snapshot at index %d", rd.Snapshot.Metadata.Index)
					// gofail: var raftAfterApplySnap struct{}
				}

				r.raftStorage.Append(rd.Entries)

				if !islead {
					// gofail: var raftBeforeFollowerSend struct{}
					rh.sendMessage(rd.Messages)
				}
				raftDone <- struct{}{}
				r.Advance()
			case <-r.stopped:
				return
			}
		}
	}()
}

func (r *raftNode) apply() chan apply {
	return r.applyc
}

func (r *raftNode) leadElectedTime() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lt
}

func (r *raftNode) stop() {
	r.stopped <- struct{}{}
	<-r.done
}

func (r *raftNode) onStop() {
	r.Stop()
	r.transport.Stop()
	if err := r.storage.Close(); err != nil {
		plog.Panicf("raft close storage error: %v", err)
	}
	close(r.done)
}

// for testing
func (r *raftNode) pauseSending() {
	p := r.transport.(rafthttp.Pausable)
	p.Pause()
}

func (r *raftNode) resumeSending() {
	p := r.transport.(rafthttp.Pausable)
	p.Resume()
}

// advanceTicksForElection advances ticks to the node for fast election.
// This reduces the time to wait for first leader election if bootstrapping the whole
// cluster, while leaving at least 1 heartbeat for possible existing leader
// to contact it.
func advanceTicksForElection(n raft.Node, electionTicks int) {
	for i := 0; i < electionTicks-1; i++ {
		n.Tick()
	}
}

func startNode(cfg *Config, cl *membership.RaftCluster, snapshot *raftpb.Snapshot) (types.ID, raft.Node, *raft.MemoryStorage, *wal.WAL) {
	var walsnap walpb.Snapshot
	if snapshot != nil {
		walsnap.Index, walsnap.Term = snapshot.Metadata.Index, snapshot.Metadata.Term
	}
	w, st, ents := readWAL(cfg.WALDir(), walsnap)

	self := cl.MemberByName(cfg.Name)
	id := self.ID
	cid := cl.ID()

	plog.Infof("starting member %s in cluster %s at commit index %d", id, cid, st.Commit)

	s := raft.NewMemoryStorage()
	if snapshot != nil {
		s.ApplySnapshot(*snapshot)
	}
	s.SetHardState(st)
	s.Append(ents)
	c := &raft.Config{
		ID:              uint64(id),
		ElectionTick:    cfg.ElectionTicks(),
		HeartbeatTick:   1,
		Storage:         s,
		MaxSizePerMsg:   maxSizePerMsg,
		MaxInflightMsgs: maxInflightMsgs,
		CheckQuorum:     true,
	}

	ms := cl.Members()
	ps := make([]raft.Peer, 0, len(ms))
	for _, m := range ms {
		ps = append(ps, raft.Peer{ID: uint64(m.ID), Context: []byte(m.Name)})
	}

	n := raft.BuildNode(c, ps)
	raftStatusMu.Lock()
	raftStatus = n.Status
	raftStatusMu.Unlock()
	advanceTicksForElection(n, c.ElectionTick)
	return id, n, s, w
}

// getIDs returns an ordered set of IDs included in the given snapshot and
// the entries. The given snapshot/entries can contain two kinds of
// ID-related entry:
// - ConfChangeAddNode, in which case the contained ID will be added into the set.
// - ConfChangeRemoveNode, in which case the contained ID will be removed from the set.
func getIDs(snap *raftpb.Snapshot, ents []raftpb.Entry) []uint64 {
	ids := make(map[uint64]bool)
	if snap != nil {
		for _, id := range snap.Metadata.ConfState.Nodes {
			ids[id] = true
		}
	}
	for _, e := range ents {
		if e.Type != raftpb.EntryConfChange {
			continue
		}
		var cc raftpb.ConfChange
		pbutil.MustUnmarshal(&cc, e.Data)
		switch cc.Type {
		case raftpb.ConfChangeAddNode:
			ids[cc.NodeID] = true
		case raftpb.ConfChangeRemoveNode:
			delete(ids, cc.NodeID)
		case raftpb.ConfChangeUpdateNode:
			// do nothing
		default:
			plog.Panicf("ConfChange Type should be either ConfChangeAddNode or ConfChangeRemoveNode!")
		}
	}
	sids := make(types.Uint64Slice, 0, len(ids))
	for id := range ids {
		sids = append(sids, id)
	}
	sort.Sort(sids)
	return []uint64(sids)
}
