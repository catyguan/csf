// Copyright 2015 The CSF Authors
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
package raft4si

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/idutil"
	"github.com/catyguan/csf/pkg/pbutil"
	"github.com/catyguan/csf/pkg/runobj"
	"github.com/catyguan/csf/pkg/wait"
	"github.com/catyguan/csf/raft"
	"github.com/catyguan/csf/raft/raftpb"
	"github.com/catyguan/csf/snapshot"
	"github.com/catyguan/csf/wal"
)

type RaftPeer struct {
	NodeID uint64
}

type Config struct {
	ClusterID uint64 // (必填) 集群节点的唯一编号，不一定等同于raft.ID

	ElectionTick    int                 // raft.Config.ElectionTick, 投票的Tick间隔 default: ElectionTick = 10 * HeartbeatTick
	HeartbeatTick   int                 // raft.Config.HeartbeatTick, 心跳Tick间隔 default: 1
	MaxSizePerMsg   uint64              // raft.Config.MaxSizePerMsg. Append消息数限制 default 1 * 1024 * 1024
	MaxInflightMsgs int                 // raft.Config.MaxInflightMsgs 同步发送的消息数限制 default 4*1024/8
	CheckQuorum     bool                // raft.Config.CheckQuorum Leader是否检查有效的激活节点
	ReadOnlyOption  raft.ReadOnlyOption // raft.Config.ReadOnlyOption default: ReadOnlySafe

	// Number of entries for slow follower to catch-up after compacting
	// the raft storage entries.
	// We expect the follower has a millisecond level latency with the leader.
	// The max throughput is around 10K. Keep a 5K entries is enough for helping
	// follower to catch up.
	NumberOfCatchUpEntries uint64 // raft.go numberOfCatchUpEntries, default 5000
	SnapCount              uint64 // 生成Snapshot的处理数, default 10000

	InitPeers      []RaftPeer    // (必填) 初始化节点
	TickerDuration time.Duration // Tick时间间隔 default 500*ms

	MemoryMode       bool          // 内存模式，不创建WAL
	WALDir           string        // wal.Dir WAL工作目录
	BlockRollSize    uint64        // wal.BlockRollSize default 64M
	WALQueueSize     int           // wal.WALQueueSize WAL的处理队列大小，default 1000
	IdleSyncDuration time.Duration // wal.IdleSyncDuration default 30秒
	Symbol           string        // 文件标志符，用于标志WAL文件属于哪个系统
	AutoWALSync      bool          // 是否保存消息后自动Sync文件
	NotWaitWAL       bool          // 是否不等待WAL写操作返回

	ContainerQueueSize int  // Container处理队列大小,default 128
	ForceNew           bool // 是否强制启动新Raft实例(包括重置WAL)
	NotifyLeaderChange bool // 是否通知Leader改变事件
}

func NewConfig() *Config {
	r := &Config{}

	r.ElectionTick = 10
	r.HeartbeatTick = 1
	r.MaxSizePerMsg = 1 * 1024 * 1024
	r.MaxInflightMsgs = 4096 / 8
	r.NumberOfCatchUpEntries = 5000
	r.SnapCount = 10000

	r.TickerDuration = 500 * time.Millisecond

	r.BlockRollSize = wal.DefaultMaxBlockSize
	r.WALQueueSize = 1000
	r.IdleSyncDuration = 30 * time.Second

	r.ContainerQueueSize = 128

	return r
}

type RaftServiceContainer struct {
	cs  core.CoreService
	cfg *Config

	ro *runobj.RunObj

	id       uint64
	Node     raft.Node
	ms       *raft.MemoryStorage
	w        wal.WAL
	snap     *snapshot.Snapshotter
	proIDGen *idutil.Generator
	proWait  wait.Wait

	mu            sync.Mutex
	lastState     raftpb.HardState
	lastConf      *raftpb.ConfState
	lastIndex     uint64
	lastSnapIndex uint64
	leaderId      uint64
	lastLeadTime  time.Time
}

func NewRaftServiceContainer(cs core.CoreService, cfg *Config) *RaftServiceContainer {
	r := &RaftServiceContainer{
		cs:      cs,
		cfg:     cfg,
		ro:      runobj.NewRunObj(cfg.ContainerQueueSize),
		proWait: wait.New(),
	}
	return r
}

func (this *RaftServiceContainer) impl() {
	_ = core.ServiceInvoker(this)
}

func (this *RaftServiceContainer) Run() error {
	return this.ro.Run(this.doRun, nil)
}

func (this *RaftServiceContainer) safeUseWAL() bool {
	if this.cfg.MemoryMode {
		return false
	}
	if this.cfg.WALDir == "" {
		plog.Panicf("invalid WALDir when use WAL")
	}
	return true
}

func (this *RaftServiceContainer) startRaftNode() error {
	cfg := this.cfg

	if cfg.ForceNew {
		if this.safeUseWAL() {
			err := os.RemoveAll(cfg.WALDir)
			if err != nil {
				return err
			}
		}
	}

	if this.safeUseWAL() {
		this.snap = snapshot.NewSnapshotter(cfg.WALDir)

		wcfg := wal.NewConfig()
		wcfg.Dir = cfg.WALDir
		wcfg.BlockRollSize = cfg.BlockRollSize
		wcfg.IdleSyncDuration = cfg.IdleSyncDuration
		wcfg.WALQueueSize = cfg.WALQueueSize
		wcfg.InitMetadata = []byte(cfg.Symbol)

		w, meta, err := wal.NewWAL(wcfg)
		if err != nil {
			return err
		}
		if string(meta) != cfg.Symbol {
			w.Close()
			return fmt.Errorf("Symbol invalid, got '%s' want '%s'", string(meta), cfg.Symbol)
		}
		this.w = w
	}
	this.ms = raft.NewMemoryStorage()

	this.id = cfg.ClusterID
	rcfg := &raft.Config{}
	rcfg.CheckQuorum = cfg.CheckQuorum
	rcfg.ElectionTick = cfg.ElectionTick
	rcfg.HeartbeatTick = cfg.HeartbeatTick
	rcfg.MaxInflightMsgs = cfg.MaxInflightMsgs
	rcfg.MaxSizePerMsg = cfg.MaxSizePerMsg
	rcfg.ReadOnlyOption = cfg.ReadOnlyOption
	rcfg.Storage = this.ms

	// debug
	rcfg.Storage = &debugStorage{backend: this.ms}
	rcfg.Logger = plog

	this.proIDGen = idutil.NewGenerator(uint16(cfg.ClusterID), time.Now())

	if cfg.MemoryMode || this.w.IsNew() {
		rcfg.ID = cfg.ClusterID
		plist := make([]raft.Peer, 0, len(cfg.InitPeers))
		for _, rp := range cfg.InitPeers {
			plist = append(plist, raft.Peer{ID: rp.NodeID})
		}
		this.Node = raft.StartNode(rcfg, plist)
	} else {
		// recover the in-memory storage from persistent
		// snapshot, state and entries.
		rcfg.ID = cfg.ClusterID

		snapidx := uint64(0)
		if true {
			n, lr, err1 := this.snap.LoadLastHeader()
			if err1 != nil {
				return err1
			}
			if lr != nil {
				_, data, err2 := this.snap.LoadSnapFile(n)
				if err2 != nil {
					return err2
				}
				ps := PBSnapshot{}
				err3 := ps.Unmarshal(data)
				if err3 != nil {
					return err3
				}
				pSnapshot := raftpb.Snapshot{}
				pState := raftpb.HardState{}
				err4 := pSnapshot.Unmarshal(ps.Snapdata)
				if err4 != nil {
					return err4
				}
				err5 := pState.Unmarshal(ps.HardState)
				if err5 != nil {
					return err5
				}
				snapidx = lr.Index
				err := this.ms.ApplySnapshot(pSnapshot)
				if err != nil {
					return err
				}
				this.lastState = pState
			}
		}
		this.lastIndex = snapidx

		pEntries := make([]raftpb.Entry, 0)
		err := func() error {
			wsc, err1 := this.w.GetCursor(snapidx)
			if err1 != nil {
				return err1
			}
			defer wsc.Close()

			ll := uint64(0)
			for {
				e, err2 := wsc.Read()
				if err2 != nil {
					return err2
				}
				if e == nil {
					break
				}
				if e.Index >= snapidx {
					pbe := &PBEntry{}
					err3 := pbe.Unmarshal(e.Data)
					if err3 != nil {
						return err3
					}
					rte := raftpb.Entry{}
					err4 := rte.Unmarshal(pbe.Entry)
					if err4 != nil {
						return err4
					}
					if pbe.HardState != nil {
						st := raftpb.HardState{}
						err5 := st.Unmarshal(pbe.HardState)
						if err5 != nil {
							return err5
						}
						this.lastState = st
					}
					if rte.Index != e.Index {
						plog.Fatalf("Raft.Entry.Index(%d) != WAL.Entry.Index(%d)", rte.Index, e.Index)
					}
					pEntries = append(pEntries, rte)
					ll = e.Index
				}
			}
			this.lastIndex = ll
			return nil
		}()
		if err != nil {
			return err
		}
		err = this.ms.SetHardState(this.lastState)
		if err != nil {
			return err
		}
		this.ms.Append(pEntries)

		this.Node = raft.RestartNode(rcfg)
	}

	for i := 0; i < cfg.ElectionTick-1; i++ {
		this.Node.Tick()
	}

	return nil
}

func (this *RaftServiceContainer) doRun(ready chan error, ach <-chan *runobj.ActionRequest, p interface{}) {
	defer func() {
		this.doClose()
	}()
	err := this.startRaftNode()
	if err != nil {
		ready <- err
		return
	} else {
		close(ready)
	}
	ticker := time.Tick(this.cfg.TickerDuration)
	islead := false
	for {
		select {
		case <-ticker:
			// Leader:Send Heartbeat, Follower:Election timeout
			this.Node.Tick()
		case rd := <-this.Node.Ready():
			// plog.Infof("ready -> %v", rd)
			if rd.SoftState != nil {
				if lead := atomic.LoadUint64(&this.leaderId); rd.SoftState.Lead != raft.None && lead != rd.SoftState.Lead {
					this.mu.Lock()
					this.lastLeadTime = time.Now()
					this.mu.Unlock()
				}
				atomic.StoreUint64(&this.leaderId, rd.SoftState.Lead)
				islead = rd.RaftState == raft.StateLeader
				this.doLeadershipUpdate(islead)
			}

			// the leader can write to its disk in parallel with replicating to the followers and them
			// writing to their disks.
			// For more details, check raft thesis 10.2.1
			if islead {
				this.doSendMessage(rd.Messages)
			}

			// 持久化日志和状态
			stE := raft.IsEmptyHardState(rd.HardState)
			if this.w != nil {
				l := len(rd.Entries)
				ents := make([]wal.Entry, 0, l)
				for i := 0; i < l; i++ {
					e := &rd.Entries[i]
					pbe := PBEntry{}
					pbe.Entry = pbutil.MustMarshal(e)
					if i == l-1 && !stE {
						pbe.HardState = pbutil.MustMarshal(&rd.HardState)
					}
					we := wal.Entry{}
					we.Index = e.Index
					we.Data = pbutil.MustMarshal(&pbe)
					ents = append(ents, we)
				}
				rsc := this.w.Append(ents, this.cfg.AutoWALSync)
				if !this.cfg.NotWaitWAL {
					rs := <-rsc
					if rs.Err != nil {
						plog.Fatalf("raft save state and entries error: %v", rs.Err)
					}
				}
			}
			if !stE {
				this.lastState = rd.HardState
			}

			// 持久化快照
			if !raft.IsEmptySnap(rd.Snapshot) {
				plog.Infof("raft apply incoming snapshot at index %d", rd.Snapshot.Metadata.Index)
				pbs := &PBSnapshot{}
				if stE {
					pbs.HardState = pbutil.MustMarshal(&this.lastState)
				} else {
					pbs.HardState = pbutil.MustMarshal(&rd.HardState)
				}
				pbs.Snapdata = pbutil.MustMarshal(&rd.Snapshot)
				sh := &snapshot.SnapHeader{}
				sh.Index = rd.Snapshot.Metadata.Index
				sh.Meta = []byte(this.cfg.Symbol)
				if this.snap != nil {
					err := this.snap.SaveSnap(sh, pbutil.MustMarshal(pbs))
					if err != nil {
						plog.Fatalf("raft save snapshot error: %v", err)
					}
				}
				if this.ms != nil {
					this.ms.ApplySnapshot(rd.Snapshot)
				}
			}
			if this.ms != nil {
				this.ms.Append(rd.Entries)
			}

			if !islead {
				this.doSendMessage(rd.Messages)
			}
			if !raft.IsEmptySnap(rd.Snapshot) {
				this.doApplyRaftSnapshot(rd.Snapshot)
			}
			for _, entry := range rd.CommittedEntries {
				this.doApplyRaftUpdate(entry)
				// lastIndex
				if entry.Type == raftpb.EntryConfChange {
					var cc raftpb.ConfChange
					pbutil.MustUnmarshal(&cc, entry.Data)
					plog.Infof("ApplyConfChange - %v", cc.String())
					cst := this.Node.ApplyConfChange(cc)
					if cst != nil {
						this.lastConf = cst
					}
				}
			}
			this.Node.Advance()
			this.triggerSnapshot()
		case a := <-ach:
			if a == nil {
				return
			}
			switch a.Type {
			case 1:
				ctx := a.P1.(context.Context)
				req := a.P2.(*corepb.Request)
				r, err := this.doApplyRequest(ctx, req)
				if a.Resp != nil {
					if err != nil {
						r = core.MakeErrorResponse(r, err)
					}
					a.Resp <- &runobj.ActionResponse{R1: r, Err: err}
				}
			}
		}
	}
}

func (this *RaftServiceContainer) InvokeRequest(ctx context.Context, creq *corepb.ChannelRequest) (*corepb.ChannelResponse, error) {
	req := &creq.Request

	switch creq.ServicePath {
	case RaftRPCPath:
		m := raftpb.Message{}
		err := m.Unmarshal(creq.Data)
		if err != nil {
			plog.Warningf("unmarshal RPC message fail - %v", err)
			return nil, err
		}
		this.onRecvRaftRPC(ctx, m)
		resp := creq.CreateResponse(nil, nil)
		return corepb.MakeChannelResponse(resp), nil
	}

	saved, err := this.cs.VerifyRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	var resp *corepb.Response
	var resperr error
	if !saved {
		a := &runobj.ActionRequest{
			Type: 1,
			P1:   ctx,
			P2:   req,
		}
		ar, err1 := this.ro.ContextCall(ctx, a)
		if err1 != nil {
			return nil, err1
		}
		if ar.R1 != nil {
			resp = ar.R1.(*corepb.Response)
		}
		resperr = ar.Err
	} else {
		data, err2 := req.Marshal()
		if err2 != nil {
			return nil, err2
		}
		prop := PBPropose{}
		prop.ProposeID = this.proIDGen.Next()
		prop.Request = data
		pd := pbutil.MustMarshal(&prop)

		if this.IsClosed() {
			return nil, core.ErrClosed
		}

		ch := this.proWait.Register(prop.ProposeID)
		if err := this.Node.Propose(ctx, pd); err != nil {
			this.proWait.Trigger(prop.ProposeID, nil)
			return nil, err
		}
		select {
		case x := <-ch:
			if x == nil {
				return nil, core.ErrClosed
			}
			if err2, ok := x.(error); ok {
				return nil, err2
			}
			resp = x.(*corepb.Response)
			break
		case <-ctx.Done():
			this.proWait.Trigger(prop.ProposeID, nil) // GC wait
			return nil, ctx.Err()
		}
	}
	err2 := corepb.HandleError(resp, resperr)
	if err2 != nil {
		return nil, err2
	}
	return corepb.MakeChannelResponse(resp), nil
}

func (this *RaftServiceContainer) IsClosed() bool {
	return this.ro.IsClosed()
}

func (this *RaftServiceContainer) Close() {
	this.ro.Close()
}

func (this *RaftServiceContainer) doClose() {
	if this.Node != nil {
		this.Node.Stop()
	}
	if this.w != nil {
		this.w.Close()
	}
}

func (this *RaftServiceContainer) doApplyRaftSnapshot(snapshot raftpb.Snapshot) error {
	psnap := PBSnapshot{}
	err := psnap.Unmarshal(snapshot.Data)
	if err != nil {
		return err
	}
	st := raftpb.HardState{}
	pbutil.MustUnmarshal(&st, psnap.HardState)
	if !raftpb.IsEmptyHardState(st) {
		this.lastState = st
	}
	ctx := context.Background()
	return this.cs.ApplySnapshot(ctx, bytes.NewBuffer(psnap.Snapdata))
}

func (this *RaftServiceContainer) doApplyRaftUpdate(entry raftpb.Entry) {
	if entry.Type != raftpb.EntryNormal {
		return
	}
	if entry.Data == nil {
		return
	}
	pp := PBPropose{}
	err0 := pp.Unmarshal(entry.Data)
	if err0 != nil {
		plog.Warningf("Entry[%d] unmarshal PBPropose fail - %v", entry.Index, err0)
		return
	}
	req := &corepb.Request{}
	pbutil.MustUnmarshal(req, pp.Request)
	ctx := context.Background()
	resp, err := this.doApplyRequest(ctx, req)
	if err != nil {
		this.proWait.Trigger(pp.ProposeID, err)
	} else {
		this.proWait.Trigger(pp.ProposeID, resp)
	}
}

func (this *RaftServiceContainer) doApplyRequest(ctx context.Context, req *corepb.Request) (*corepb.Response, error) {
	return this.cs.ApplyRequest(ctx, req)
}

func (this *RaftServiceContainer) doLeadershipUpdate(isLocalLeader bool) {
	if this.cfg.NotifyLeaderChange {
		msg := PBLeadershipUpdateMessage{LocalLeader: isLocalLeader}
		ctx := context.Background()
		req := corepb.NewMessageRequest("", "onLeadershipUpdate", pbutil.MustMarshal(&msg))
		this.doApplyRequest(ctx, req)
	}
}

func (this *RaftServiceContainer) doSendMessage(msgs []raftpb.Message) {
	for _, msg := range msgs {
		// msg.To
	}
}

func (this *RaftServiceContainer) doMakeSnapshot() (uint64, error) {
	snapi := this.lastState.Commit
	confState := this.lastConf

	ctx := context.Background()
	buf := bytes.NewBuffer(make([]byte, 0))
	err := this.cs.CreateSnapshot(ctx, buf)
	if err != nil {
		plog.Warningf("service create snapshot fail - %v", err)
		return snapi, err
	}

	snap, err2 := this.ms.CreateSnapshot(snapi, confState, buf.Bytes())
	if err2 != nil {
		// the snapshot was done asynchronously with the progress of raft.
		// raft might have already got a newer snapshot.
		if err2 == raft.ErrSnapOutOfDate {
			return snapi, nil
		}
		plog.Panicf("unexpected create snapshot error %v", err2)
	}
	if this.snap != nil {
		sh := &snapshot.SnapHeader{
			Index: snapi,
			Meta:  []byte(this.cfg.Symbol),
		}
		psnap := PBSnapshot{
			HardState: pbutil.MustMarshal(&this.lastState),
			Snapdata:  pbutil.MustMarshal(&snap),
		}
		if err = this.snap.SaveSnap(sh, pbutil.MustMarshal(&psnap)); err != nil {
			plog.Fatalf("save snapshot error: %v", err)
		}
		plog.Infof("saved snapshot at index %d", snapi)
	}
	this.lastSnapIndex = snapi

	// keep some in memory log entries for slow followers.
	compacti := uint64(1)
	if snapi > this.cfg.NumberOfCatchUpEntries {
		compacti = snapi - this.cfg.NumberOfCatchUpEntries
	}
	err = this.ms.Compact(compacti)
	if err != nil {
		// the compaction was done asynchronously with the progress of raft.
		// raft log might already been compact.
		if err == raft.ErrCompacted {
			return snapi, nil
		}
		plog.Panicf("unexpected compaction error %v", err)
	}
	plog.Infof("compacted MemoryStorage log at %d", compacti)
	return snapi, nil
}

func (this *RaftServiceContainer) triggerSnapshot() {
	if this.lastState.Commit-this.lastSnapIndex <= this.cfg.SnapCount {
		return
	}
	plog.Infof("start to snapshot (applied: %d, lastsnap: %d)", this.lastState.Commit, this.lastSnapIndex)
	this.doMakeSnapshot()
}

func (this *RaftServiceContainer) onRecvRaftRPC(ctx context.Context, m raftpb.Message) {
	if this.Node != nil {
		this.Node.Step(ctx, m)
	}
}
