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
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/catyguan/csf/basepb"
	pb "github.com/catyguan/csf/cluster/clusterpb"
	"github.com/catyguan/csf/cluster/membership"
	"github.com/catyguan/csf/interfaces"
	"github.com/catyguan/csf/pkg/contention"
	"github.com/catyguan/csf/pkg/fileutil"
	"github.com/catyguan/csf/pkg/idutil"
	"github.com/catyguan/csf/pkg/pbutil"
	"github.com/catyguan/csf/pkg/runtime"
	"github.com/catyguan/csf/pkg/schedule"
	"github.com/catyguan/csf/pkg/types"
	"github.com/catyguan/csf/pkg/wait"
	"github.com/catyguan/csf/raft"
	"github.com/catyguan/csf/raft/raftpb"
	"github.com/catyguan/csf/rafthttp"
	"github.com/catyguan/csf/semver"
	"github.com/catyguan/csf/snap"
	"github.com/catyguan/csf/stats"
	"github.com/catyguan/csf/version"
	"github.com/catyguan/csf/wal"
	"golang.org/x/net/context"
)

const (
	// HealthInterval is the minimum time the cluster should be healthy
	// before accepting add member requests.
	HealthInterval = 5 * time.Second

	purgeFileInterval = 30 * time.Second

	// monitorVersionInterval should be smaller than the timeout
	// on the connection. Or we will not be able to reuse the connection
	// (since it will timeout).
	monitorVersionInterval = rafthttp.ConnWriteTimeout - time.Second

	databaseFilename = "db"
	// max number of in-flight snapshot messages etcdserver allows to have
	// This number is more than enough for most clusters with 5 machines.
	maxInFlightMsgSnap = 16

	releaseDelayAfterSnapshot = 30 * time.Second

	// maxPendingRevokes is the maximum number of outstanding expired lease revocations.
	maxPendingRevokes = 16
)

func init() {
	rand.Seed(time.Now().UnixNano())

	expvar.Publish(
		"file_descriptor_limit",
		expvar.Func(
			func() interface{} {
				n, _ := runtime.FDLimit()
				return n
			},
		),
	)
}

// CSFNode is the production implementation of the Server interface
type CSFNode struct {
	// inflightSnapshots holds count the number of snapshots currently inflight.
	inflightSnapshots int64  // must use atomic operations to access; keep 64-bit aligned.
	appliedIndex      uint64 // must use atomic operations to access; keep 64-bit aligned.
	committedIndex    uint64 // must use atomic operations to access; keep 64-bit aligned.

	// consistIndex used to hold the offset of current executing entry
	// It is initialized to 0 before executing any entry.
	consistIndex consistentIndex // must use atomic operations to access; keep 64-bit aligned.
	Cfg          *Config

	readych  chan struct{}
	raftNode raftNode

	snapCount uint64

	w  wait.Wait
	td *contention.TimeoutDetector

	readMu sync.RWMutex
	// read routine notifies etcd server that it waits for reading by sending an empty struct to
	// readwaitC
	readwaitc chan struct{}
	// readNotifier is used to notify the read routine that it can process the request
	// when there is no error
	readNotifier *notifier

	// stop signals the run goroutine should shutdown.
	stop chan struct{}
	// stopping is closed by run goroutine on shutdown.
	stopping chan struct{}
	// done is closed when all goroutines from start() complete.
	done chan struct{}

	errorc  chan error
	id      types.ID
	cluster *membership.RaftCluster

	// applyV2 ApplierV2
	// applyV3 is the applier with auth and quotas
	// applyV3 applierV3
	// applyV3Base is the core applier without auth or quotas
	// applyV3Base applierV3
	applyWait wait.WaitTime

	stats  *stats.ServerStats
	lstats *stats.LeaderStats

	SyncTicker <-chan time.Time

	// peerRt used to send requests (version, lease) to peers.
	peerRoundTripper http.RoundTripper
	reqIDGen         *idutil.Generator

	// forceVersionC is used to force the version monitor loop
	// to detect the cluster version immediately.
	forceVersionC chan struct{}

	msgSnapC chan raftpb.Message

	// wgMu blocks concurrent waitgroup mutation while server stopping
	wgMu sync.RWMutex
	// wg is used to wait for the go routines that depends on the server state
	// to exit when stopping the server.
	wg sync.WaitGroup

	// cluster fields
	Peers   []net.Listener
	Clients []net.Listener

	errc  chan error
	sctxs map[string]*serveCtx

	shub map[string]interfaces.Service
}

func createMembership(cfg *Config) (*membership.RaftCluster, error) {
	ms := make([]*membership.Member, 0)
	for _, peer := range cfg.ClusterPeers {
		m := membership.NewMember(cfg.ClusterName, types.ID(peer.ID), peer.Name, peer.PeerURL, peer.ClientURL)
		ms = append(ms, m)
	}
	cl := membership.NewClusterFromMembers(cfg.ClusterName, cfg.ClusterToken, ms, version.Version)
	return cl, nil
}

// NewServer creates a new CSFNode from the supplied configuration. The
// configuration is considered static for the lifetime of the CSFNode.
func doSetupNode(cfg *Config) (srv *CSFNode, err error) {

	var (
		w  *wal.WAL
		n  raft.Node
		s  *raft.MemoryStorage
		id types.ID
		cl *membership.RaftCluster
	)

	if terr := fileutil.TouchDirAll(cfg.Dir); terr != nil {
		return nil, fmt.Errorf("cannot access data directory: %v", terr)
	}

	cl, err = createMembership(cfg)
	if err != nil {
		return nil, fmt.Errorf("create membership fail - %v", err)
	}

	if !wal.Exist(cfg.WALDir()) {
		// init WAL data
		member := cl.MemberByName(cfg.Name)
		metadata := pbutil.MustMarshal(
			&basepb.Metadata{
				NodeID:    uint64(member.ID),
				ClusterID: uint64(cl.ID()),
			},
		)
		if w, err = wal.Create(cfg.WALDir(), metadata); err != nil {
			plog.Fatalf("create wal error: %v", err)
		}
		w.Close()
		w = nil
	}

	if err = fileutil.TouchDirAll(cfg.SnapDir()); err != nil {
		plog.Fatalf("create snapshot directory error: %v", err)
	}
	ss := snap.New(cfg.SnapDir())

	prt, err := rafthttp.NewRoundTripper(cfg.PeerTLSInfo, cfg.peerDialTimeout())
	if err != nil {
		return nil, err
	}
	var (
		snapshot *raftpb.Snapshot
	)

	if err = fileutil.IsDirWriteable(cfg.MemberDir()); err != nil {
		return nil, fmt.Errorf("cannot write to member directory: %v", err)
	}

	if err = fileutil.IsDirWriteable(cfg.WALDir()); err != nil {
		return nil, fmt.Errorf("cannot write to WAL directory: %v", err)
	}

	snapshot, err = ss.Load()
	if err != nil && err != snap.ErrNoSnapshot {
		return nil, err
	}
	cfg.Print()
	id, n, s, w = startNode(cfg, cl, snapshot)

	if terr := fileutil.TouchDirAll(cfg.MemberDir()); terr != nil {
		return nil, fmt.Errorf("cannot access member directory: %v", terr)
	}

	sstats := &stats.ServerStats{
		Name: cfg.Name,
		ID:   id.String(),
	}
	sstats.Initialize()
	lstats := stats.NewLeaderStats(id.String())

	heartbeat := time.Duration(cfg.TickMs) * time.Millisecond
	srv = &CSFNode{
		readych:   make(chan struct{}),
		Cfg:       cfg,
		snapCount: cfg.SnapCount,
		// set up contention detectors for raft heartbeat message.
		// expect to send a heartbeat within 2 heartbeat intervals.
		td:     contention.NewTimeoutDetector(2 * heartbeat),
		errorc: make(chan error, 1),
		raftNode: raftNode{
			Node:        n,
			ticker:      time.Tick(heartbeat),
			raftStorage: s,
			storage:     NewStorage(w, ss),
			readStateC:  make(chan raft.ReadState, 1),
		},
		id:               id,
		cluster:          cl,
		stats:            sstats,
		lstats:           lstats,
		SyncTicker:       time.Tick(500 * time.Millisecond),
		peerRoundTripper: prt,
		reqIDGen:         idutil.NewGenerator(uint16(id), time.Now()),
		forceVersionC:    make(chan struct{}),
		msgSnapC:         make(chan raftpb.Message, maxInFlightMsgSnap),
		shub:             make(map[string]interfaces.Service),
	}
	for _, sv := range cfg.shub {
		if _, ok := srv.shub[sv.ServiceID()]; ok {
			plog.Panicf("duplicate service id(%v)", sv.ServiceID())
		}
		srv.shub[sv.ServiceID()] = sv
	}

	//srv.applyV2 = &applierV2store{store: srv.store, cluster: srv.cluster}
	// minTTL := time.Duration((3*cfg.ElectionTicks())/2) * heartbeat

	// srv.consistIndex.setConsistentIndex(srv.kv.ConsistentIndex())

	// TODO: move transport initialization near the definition of remote
	tr := &rafthttp.Transport{
		TLSInfo:     cfg.PeerTLSInfo,
		DialTimeout: cfg.peerDialTimeout(),
		ID:          id,
		URLs:        cfg.LPUrls,
		ClusterID:   cl.ID(),
		Raft:        srv,
		Snapshotter: ss,
		ServerStats: sstats,
		LeaderStats: lstats,
		ErrorC:      srv.errorc,
	}
	if err = tr.Start(); err != nil {
		return nil, err
	}
	// add all remotes into transport
	// for _, m := range cl.Members() {
	// 	if m.ID != id {
	// 		tr.AddRemote(m.ID, m.PeerURLs)
	// 	}
	// }
	for _, m := range cl.Members() {
		if m.ID != id {
			tr.AddPeer(m.ID, m.PeerURLs.StringSlice())
		}
	}
	srv.raftNode.transport = tr

	return srv, nil
}

// Start prepares and starts server in a new goroutine. It is no longer safe to
// modify a server's fields after it has been sent to Start.
// It also starts a goroutine to publish its server information.
func (s *CSFNode) doStart() {
	s.start()
	s.goAttach(s.purgeFile)
	s.goAttach(func() { monitorFileDescriptor(s.stopping) })
	// s.goAttach(s.linearizableReadLoop)
}

// start prepares and starts server in a new goroutine. It is no longer safe to
// modify a server's fields after it has been sent to Start.
// This function is just used for testing.
func (s *CSFNode) start() {
	if s.snapCount == 0 {
		plog.Infof("set snapshot count to default %d", DefaultSnapCount)
		s.snapCount = DefaultSnapCount
	}
	s.w = wait.New()
	s.applyWait = wait.NewTimeList()
	s.done = make(chan struct{})
	s.stop = make(chan struct{})
	s.stopping = make(chan struct{})
	s.readwaitc = make(chan struct{}, 1)
	s.readNotifier = newNotifier()
	if s.ClusterVersion() != nil {
		plog.Infof("starting server... [version: %v]", version.Version)
	}
	// TODO: if this is an empty log, writes all peer infos
	// into the first entry
	go s.run()
}

func (s *CSFNode) purgeFile() {
	var serrc, werrc <-chan error
	if s.Cfg.MaxSnapFiles > 0 {
		serrc = fileutil.PurgeFile(s.Cfg.SnapDir(), "snap", s.Cfg.MaxSnapFiles, purgeFileInterval, s.done)
	}
	if s.Cfg.MaxWalFiles > 0 {
		werrc = fileutil.PurgeFile(s.Cfg.WALDir(), "wal", s.Cfg.MaxWalFiles, purgeFileInterval, s.done)
	}
	select {
	case e := <-werrc:
		plog.Fatalf("failed to purge wal file %v", e)
	case e := <-serrc:
		plog.Fatalf("failed to purge snap file %v", e)
	case <-s.stopping:
		return
	}
}

func (s *CSFNode) ID() types.ID { return s.id }

func (s *CSFNode) Cluster() *membership.RaftCluster { return s.cluster }

func (s *CSFNode) RaftHandler() http.Handler { return s.raftNode.transport.Handler() }

func (s *CSFNode) Process(ctx context.Context, m raftpb.Message) error {
	if s.cluster.IsIDRemoved(types.ID(m.From)) {
		plog.Warningf("reject message from removed member %s", types.ID(m.From).String())
		return interfaces.NewHTTPError(http.StatusForbidden, "cannot process message from removed member")
	}
	if m.Type == raftpb.MsgApp {
		s.stats.RecvAppendReq(types.ID(m.From).String(), m.Size())
	}
	return s.raftNode.Step(ctx, m)
}

func (s *CSFNode) IsIDRemoved(id uint64) bool { return s.cluster.IsIDRemoved(types.ID(id)) }

func (s *CSFNode) ReportUnreachable(id uint64) { s.raftNode.ReportUnreachable(id) }

// ReportSnapshot reports snapshot sent status to the raft state machine,
// and clears the used snapshot from the snapshot store.
func (s *CSFNode) ReportSnapshot(id uint64, status raft.SnapshotStatus) {
	s.raftNode.ReportSnapshot(id, status)
}

type csfProgress struct {
	confState raftpb.ConfState
	snapi     uint64
	appliedi  uint64
}

func (s *CSFNode) run() {
	snap, err := s.raftNode.raftStorage.Snapshot()
	if err != nil {
		plog.Panicf("get snapshot from raft storage error: %v", err)
	}

	rh := &raftReadyHandler{
		leadershipUpdate: func() {
			// SERVICE: NOTICE Service
			ll := s.isLeader()
			for _, ser := range s.shub {
				ser.OnLeadershipUpdate(s, ll)
			}

			// TODO: remove the nil checking
			// current test utility does not provide the stats
			if s.stats != nil {
				s.stats.BecomeLeader()
			}
			if s.td != nil {
				s.td.Reset()
			}
		},
		sendMessage: func(msgs []raftpb.Message) { s.send(msgs) },
	}
	s.raftNode.start(rh)

	// asynchronously accept apply packets, dispatch progress in-order
	sched := schedule.NewFIFOScheduler()
	ep := csfProgress{
		confState: snap.Metadata.ConfState,
		snapi:     snap.Metadata.Index,
		appliedi:  snap.Metadata.Index,
	}

	defer func() {
		s.wgMu.Lock() // block concurrent waitgroup adds in goAttach while stopping
		close(s.stopping)
		s.wgMu.Unlock()

		sched.Stop()

		// wait for gouroutines before closing raft so wal stays open
		s.wg.Wait()

		// must stop raft after scheduler-- etcdserver can leak rafthttp pipelines
		// by adding a peer after raft stops the transport
		s.raftNode.stop()

		// kv, lessor and backend can be nil if running without v3 enabled
		// or running unit tests.
		// SERVICE: Close All Service
		for _, ser := range s.shub {
			ser.OnClose(s)
		}

		close(s.done)
	}()

	for {
		select {
		case ap := <-s.raftNode.apply():
			var ci uint64
			if len(ap.entries) != 0 {
				ci = ap.entries[len(ap.entries)-1].Index
			}
			if ap.snapshot.Metadata.Index > ci {
				ci = ap.snapshot.Metadata.Index
			}
			if ci != 0 {
				s.setCommittedIndex(ci)
			}
			f := func(context.Context) { s.applyAll(&ep, &ap) }
			sched.Schedule(f)
		case err := <-s.errorc:
			plog.Errorf("%s", err)
			plog.Infof("the data-dir used by this member must be removed.")
			return
		case <-s.stop:
			return
		}
	}
}

func (s *CSFNode) applyAll(ep *csfProgress, apply *apply) {
	s.applySnapshot(ep, apply)
	st := time.Now()
	s.applyEntries(ep, apply)
	d := time.Since(st)
	entriesNum := len(apply.entries)
	if entriesNum != 0 && d > time.Duration(entriesNum)*warnApplyDuration {
		plog.Warningf("apply entries took too long [%v for %d entries]", d, len(apply.entries))
		plog.Warningf("avoid queries with large range/delete range!")
	}
	proposalsApplied.Set(float64(ep.appliedi))
	s.applyWait.Trigger(ep.appliedi)
	// wait for the raft routine to finish the disk writes before triggering a
	// snapshot. or applied index might be greater than the last index in raft
	// storage, since the raft routine might be slower than apply routine.
	<-apply.raftDone

	s.triggerSnapshot(ep)
	select {
	// snapshot requested via send()
	case m := <-s.msgSnapC:
		merged := s.createMergedSnapshotMessage(m, ep.appliedi)
		s.sendMergedSnap(merged)
	default:
	}
}

func (s *CSFNode) applySnapshot(ep *csfProgress, apply *apply) {
	if raft.IsEmptySnap(apply.snapshot) {
		return
	}

	plog.Infof("applying snapshot at index %d...", ep.snapi)
	defer plog.Infof("finished applying incoming snapshot at index %d", ep.snapi)

	if apply.snapshot.Metadata.Index <= ep.appliedi {
		plog.Panicf("snapshot index [%d] should > appliedi[%d] + 1",
			apply.snapshot.Metadata.Index, ep.appliedi)
	}

	snapfn, err := s.raftNode.storage.DBFilePath(apply.snapshot.Metadata.Index)
	if err != nil {
		plog.Panicf("get snapshot file path error: %v", err)
	}

	// SERVICE: Apply Snapshot
	pbsnap, err2 := snap.Read(snapfn)
	if err2 != nil {
		plog.Panicf("read snapshot file error(%s): %v", snapfn, err2)
	}

	ssp := &basepb.ServiceSnapshotPack{}
	err3 := ssp.Unmarshal(pbsnap.Data)
	if err3 != nil {
		plog.Panicf("Unmarshal snapshot file(%s) error: %v", snapfn, err2)
	}

	for _, ss := range ssp.Snapshots {
		sid := ss.Id
		if sv, ok := s.shub[sid]; ok {
			plog.Infof("apply snapshot to service(%s)", sid)
			if err4 := sv.ApplySnapshot(s, ss.Data); err4 != nil {
				plog.Panicf("apply snapshot to service(%s) error: %v", sid, err2)
			}
		} else {
			plog.Warningf("apply snapshot miss service(%s)", sid)
		}
	}
	// SERVICE: Apply Snapshot - END

	ep.appliedi = apply.snapshot.Metadata.Index
	ep.snapi = ep.appliedi
	ep.confState = apply.snapshot.Metadata.ConfState
}

func (s *CSFNode) applyEntries(ep *csfProgress, apply *apply) {
	if len(apply.entries) == 0 {
		return
	}
	firsti := apply.entries[0].Index
	if firsti > ep.appliedi+1 {
		plog.Panicf("first index of committed entry[%d] should <= appliedi[%d] + 1", firsti, ep.appliedi)
	}
	var ents []raftpb.Entry
	if ep.appliedi+1-firsti < uint64(len(apply.entries)) {
		ents = apply.entries[ep.appliedi+1-firsti:]
	}
	if len(ents) == 0 {
		return
	}
	var shouldstop bool
	if ep.appliedi, shouldstop = s.apply(ents, &ep.confState); shouldstop {
		go s.stopWithDelay(10*100*time.Millisecond, fmt.Errorf("the member has been permanently removed from the cluster"))
	}
}

func (s *CSFNode) triggerSnapshot(ep *csfProgress) {
	if s.snapCount == 0 {
		// disable snapshot
		return
	}
	if ep.appliedi-ep.snapi <= s.snapCount {
		return
	}

	plog.Infof("start to snapshot (applied: %d, lastsnap: %d)", ep.appliedi, ep.snapi)
	s.snapshot(ep.appliedi)
	ep.snapi = ep.appliedi
}

func (s *CSFNode) isMultiNode() bool {
	return s.cluster != nil && len(s.cluster.MemberIDs()) > 1
}

func (s *CSFNode) isLeader() bool {
	return uint64(s.ID()) == s.Lead()
}

// transferLeadership transfers the leader to the given transferee.
// TODO: maybe expose to client?
func (s *CSFNode) transferLeadership(ctx context.Context, lead, transferee uint64) error {
	now := time.Now()
	interval := time.Duration(s.Cfg.TickMs) * time.Millisecond

	plog.Infof("%s starts leadership transfer from %s to %s", s.ID(), types.ID(lead), types.ID(transferee))
	s.raftNode.TransferLeadership(ctx, lead, transferee)
	for s.Lead() != transferee {
		select {
		case <-ctx.Done(): // time out
			return ErrTimeoutLeaderTransfer
		case <-time.After(interval):
		}
	}

	// TODO: drain all requests, or drop all messages to the old leader

	plog.Infof("%s finished leadership transfer from %s to %s (took %v)", s.ID(), types.ID(lead), types.ID(transferee), time.Since(now))
	return nil
}

// TransferLeadership transfers the leader to the chosen transferee.
func (s *CSFNode) TransferLeadership() error {
	if !s.isLeader() {
		plog.Printf("skipped leadership transfer for stopping non-leader member")
		return nil
	}

	if !s.isMultiNode() {
		plog.Printf("skipped leadership transfer for single member cluster")
		return nil
	}

	transferee, ok := longestConnected(s.raftNode.transport, s.cluster.MemberIDs())
	if !ok {
		return ErrUnhealthy
	}

	tm := s.Cfg.ReqTimeout()
	ctx, cancel := context.WithTimeout(context.TODO(), tm)
	err := s.transferLeadership(ctx, s.Lead(), uint64(transferee))
	cancel()
	return err
}

// HardStop stops the server without coordination with other members in the cluster.
func (s *CSFNode) HardStop() {
	select {
	case s.stop <- struct{}{}:
	case <-s.done:
		return
	}
	<-s.done
}

// Stop stops the server gracefully, and shuts down the running goroutine.
// Stop should be called after a Start(s), otherwise it will block forever.
// When stopping leader, Stop transfers its leadership to one of its peers
// before stopping the server.
func (s *CSFNode) doStop() {
	if err := s.TransferLeadership(); err != nil {
		plog.Warningf("%s failed to transfer leadership (%v)", s.ID(), err)
	}
	s.HardStop()
}

// ReadyNotify returns a channel that will be closed when the server
// is ready to serve client requests
func (s *CSFNode) ReadyNotify() <-chan struct{} { return s.readych }

func (s *CSFNode) stopWithDelay(d time.Duration, err error) {
	select {
	case <-time.After(d):
	case <-s.done:
	}
	select {
	case s.errorc <- err:
	default:
	}
}

// StopNotify returns a channel that receives a empty struct
// when the server is stopped.
func (s *CSFNode) StopNotify() <-chan struct{} { return s.done }

func (s *CSFNode) SelfStats() []byte { return s.stats.JSON() }

func (s *CSFNode) LeaderStats() []byte {
	lead := atomic.LoadUint64(&s.raftNode.lead)
	if lead != uint64(s.id) {
		return nil
	}
	return s.lstats.JSON()
}

// Implement the RaftTimer interface

func (s *CSFNode) Index() uint64 { return atomic.LoadUint64(&s.raftNode.index) }

func (s *CSFNode) Term() uint64 { return atomic.LoadUint64(&s.raftNode.term) }

// Lead is only for testing purposes.
// TODO: add Raft server interface to expose raft related info:
// Index, Term, Lead, Committed, Applied, LastIndex, etc.
func (s *CSFNode) Lead() uint64 { return atomic.LoadUint64(&s.raftNode.lead) }

func (s *CSFNode) Leader() types.ID { return types.ID(s.Lead()) }

func (s *CSFNode) IsPprofEnabled() bool { return s.Cfg.EnablePprof }

// TODO: move this function into raft.go
func (s *CSFNode) send(ms []raftpb.Message) {
	sentAppResp := false
	for i := len(ms) - 1; i >= 0; i-- {
		if s.cluster.IsIDRemoved(types.ID(ms[i].To)) {
			ms[i].To = 0
		}

		if ms[i].Type == raftpb.MsgAppResp {
			if sentAppResp {
				ms[i].To = 0
			} else {
				sentAppResp = true
			}
		}

		if ms[i].Type == raftpb.MsgSnap {
			// There are two separate data store: the store for v2, and the KV for v3.
			// The msgSnap only contains the most recent snapshot of store without KV.
			// So we need to redirect the msgSnap to etcd server main loop for merging in the
			// current store snapshot and KV snapshot.
			select {
			case s.msgSnapC <- ms[i]:
			default:
				// drop msgSnap if the inflight chan if full.
			}
			ms[i].To = 0
		}
		if ms[i].Type == raftpb.MsgHeartbeat {
			ok, exceed := s.td.Observe(ms[i].To)
			if !ok {
				// TODO: limit request rate.
				plog.Warningf("failed to send out heartbeat on time (exceeded the %dms timeout for %v)", s.Cfg.TickMs, exceed)
				plog.Warningf("server is likely overloaded")
			}
		}
	}

	s.raftNode.transport.Send(ms)
}

func (s *CSFNode) sendMergedSnap(merged snap.Message) {
	atomic.AddInt64(&s.inflightSnapshots, 1)

	s.raftNode.transport.SendSnapshot(merged)
	s.goAttach(func() {
		select {
		case ok := <-merged.CloseNotify():
			// delay releasing inflight snapshot for another 30 seconds to
			// block log compaction.
			// If the follower still fails to catch up, it is probably just too slow
			// to catch up. We cannot avoid the snapshot cycle anyway.
			if ok {
				select {
				case <-time.After(releaseDelayAfterSnapshot):
				case <-s.stopping:
				}
			}
			atomic.AddInt64(&s.inflightSnapshots, -1)
		case <-s.stopping:
			return
		}
	})
}

// apply takes entries received from Raft (after it has been committed) and
// applies them to the current state of the CSFNode.
// The given entries should not be empty.
func (s *CSFNode) apply(es []raftpb.Entry, confState *raftpb.ConfState) (uint64, bool) {
	var applied uint64
	var shouldstop bool
	for i := range es {
		e := es[i]
		switch e.Type {
		case raftpb.EntryNormal:
			s.applyEntryNormal(&e)
		case raftpb.EntryConfChange:
			plog.Infof("skip EntryConfChange")
		default:
			plog.Panicf("entry type(%v) should be either EntryNormal or EntryConfChange", e.Type)
		}
		atomic.StoreUint64(&s.raftNode.index, e.Index)
		atomic.StoreUint64(&s.raftNode.term, e.Term)
		applied = e.Index
	}
	return applied, shouldstop
}

// applyEntryNormal apples an EntryNormal type raftpb request to the CSFNode
func (s *CSFNode) applyEntryNormal(e *raftpb.Entry) {
	if e.Index > s.consistIndex.ConsistentIndex() {
		// set the consistent index of current executing entry
		s.consistIndex.setConsistentIndex(e.Index)
	}
	defer s.setAppliedIndex(e.Index)

	// raft state machine may generate noop entry when leader confirmation.
	// skip it in advance to avoid some potential bug in the future
	if len(e.Data) == 0 {
		select {
		case s.forceVersionC <- struct{}{}:
		default:
		}
		// promote lessor when the local member is leader and finished
		// applying all entries from the last term.
		if s.isLeader() {
			// BUG?????
			// s.lessor.Promote(s.Cfg.electionTimeout())
		}
		return
	}

	req := &pb.Request{}
	pbutil.MustUnmarshal(req, e.Data)
	s.w.Trigger(req.ID, s.applyRequest(req))
}

// TODO: non-blocking snapshot
func (s *CSFNode) snapshot(snapi uint64) {
	s.goAttach(func() {
		// SERVICE: make a snapshot
		ssp := &basepb.ServiceSnapshotPack{}
		ssp.Snapshots = make([]*basepb.ServiceSnapshot, 0)
		for _, ser := range s.shub {
			ss := &basepb.ServiceSnapshot{}
			ss.Id = ser.ServiceID()
			data, err2 := ser.CreateSnapshot(s)
			if err2 != nil {
				plog.Panicf("service(%s) create snapshot error %v", ser.ServiceID(), err2)
			}
			ss.Data = data
			ssp.Snapshots = append(ssp.Snapshots, ss)
		}

		d, err3 := ssp.Marshal()
		if err3 != nil {
			plog.Panicf("create snapshot error %v", err3)
		}
		// SERVICE: make a snapshot end

		snap, err := s.raftNode.raftStorage.CreateSnapshot(snapi, nil, d)
		if err != nil {
			// the snapshot was done asynchronously with the progress of raft.
			// raft might have already got a newer snapshot.
			if err == raft.ErrSnapOutOfDate {
				return
			}
			plog.Panicf("unexpected create snapshot error %v", err)
		}
		// SaveSnap saves the snapshot and releases the locked wal files
		// to the snapshot index.
		if err = s.raftNode.storage.SaveSnap(snap); err != nil {
			plog.Fatalf("save snapshot error: %v", err)
		}
		plog.Infof("saved snapshot at index %d", snap.Metadata.Index)

		// When sending a snapshot, etcd will pause compaction.
		// After receives a snapshot, the slow follower needs to get all the entries right after
		// the snapshot sent to catch up. If we do not pause compaction, the log entries right after
		// the snapshot sent might already be compacted. It happens when the snapshot takes long time
		// to send and save. Pausing compaction avoids triggering a snapshot sending cycle.
		if atomic.LoadInt64(&s.inflightSnapshots) != 0 {
			plog.Infof("skip compaction since there is an inflight snapshot")
			return
		}

		// keep some in memory log entries for slow followers.
		compacti := uint64(1)
		if snapi > numberOfCatchUpEntries {
			compacti = snapi - numberOfCatchUpEntries
		}
		err = s.raftNode.raftStorage.Compact(compacti)
		if err != nil {
			// the compaction was done asynchronously with the progress of raft.
			// raft log might already been compact.
			if err == raft.ErrCompacted {
				return
			}
			plog.Panicf("unexpected compaction error %v", err)
		}
		plog.Infof("compacted raft log at %d", compacti)
	})
}

// CutPeer drops messages to the specified peer.
func (s *CSFNode) CutPeer(id types.ID) {
	tr, ok := s.raftNode.transport.(*rafthttp.Transport)
	if ok {
		tr.CutPeer(id)
	}
}

// MendPeer recovers the message dropping behavior of the given peer.
func (s *CSFNode) MendPeer(id types.ID) {
	tr, ok := s.raftNode.transport.(*rafthttp.Transport)
	if ok {
		tr.MendPeer(id)
	}
}

func (s *CSFNode) PauseSending() { s.raftNode.pauseSending() }

func (s *CSFNode) ResumeSending() { s.raftNode.resumeSending() }

func (s *CSFNode) ClusterVersion() *semver.Version {
	if s.cluster == nil {
		return nil
	}
	return s.cluster.Version()
}

func (s *CSFNode) parseProposeCtxErr(err error, start time.Time) error {
	switch err {
	case context.Canceled:
		return ErrCanceled
	case context.DeadlineExceeded:
		curLeadElected := s.raftNode.leadElectedTime()
		prevLeadLost := curLeadElected.Add(-2 * time.Duration(s.Cfg.ElectionTicks()) * time.Duration(s.Cfg.TickMs) * time.Millisecond)
		if start.After(prevLeadLost) && start.Before(curLeadElected) {
			return ErrTimeoutDueToLeaderFail
		}

		lead := types.ID(atomic.LoadUint64(&s.raftNode.lead))
		switch lead {
		case types.ID(raft.None):
			// TODO: return error to specify it happens because the cluster does not have leader now
		case s.ID():
			if !isConnectedToQuorumSince(s.raftNode.transport, start, s.ID(), s.cluster.Members()) {
				return ErrTimeoutDueToConnectionLost
			}
		default:
			if !isConnectedSince(s.raftNode.transport, start, lead) {
				return ErrTimeoutDueToConnectionLost
			}
		}

		return ErrTimeout
	default:
		return err
	}
}

func (s *CSFNode) getAppliedIndex() uint64 {
	return atomic.LoadUint64(&s.appliedIndex)
}

func (s *CSFNode) setAppliedIndex(v uint64) {
	atomic.StoreUint64(&s.appliedIndex, v)
}

func (s *CSFNode) getCommittedIndex() uint64 {
	return atomic.LoadUint64(&s.committedIndex)
}

func (s *CSFNode) setCommittedIndex(v uint64) {
	atomic.StoreUint64(&s.committedIndex, v)
}

// goAttach creates a goroutine on a given function and tracks it using
// the etcdserver waitgroup.
func (s *CSFNode) goAttach(f func()) {
	s.wgMu.RLock() // this blocks with ongoing close(s.stopping)
	defer s.wgMu.RUnlock()
	select {
	case <-s.stopping:
		plog.Warning("server has stopped (skipping goAttach)")
		return
	default:
	}

	// now safe to add since waitgroup wait has not started yet
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		f()
	}()
}
