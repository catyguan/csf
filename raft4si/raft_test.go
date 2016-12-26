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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/service/counter"
	"github.com/catyguan/csf/wal"
	"github.com/stretchr/testify/assert"
)

var (
	testDir = "c:\\tmp"
)

func testRaft(cs core.CoreService) *RaftServiceContainer {
	peers := make([]RaftPeer, 1)
	peers[0] = RaftPeer{NodeID: 1}

	cfg := NewConfig()
	cfg.ClusterID = 1
	cfg.InitPeers = peers
	// cfg.WALConfig.Dir = filepath.Join(testDir, "raft1")
	// cfg.WALConfig.BlockRollSize = 16 * 1024
	cfg.MemoryMode = true
	cfg.SnapCount = 16
	cfg.NumberOfCatchUpEntries = 16

	cfg.NotifyLeaderChange = true

	return NewRaftServiceContainer(cs, cfg)
}

func TestBase(t *testing.T) {
	cs := counter.NewCounterService()
	rsc := testRaft(cs)

	err := rsc.Run()
	defer rsc.Close()
	if !assert.NoError(t, err) {
		return
	}

	// c := counter.NewCounter(si)
	// doTestCall1(t, c)

	// assert.Equal(t, int(2), ms.Count())
	time.Sleep(10 * time.Second)
}

func TestInvoker(t *testing.T) {
	tm := time.AfterFunc(60*time.Second, func() {
		fmt.Printf("exec timeout")
		os.Exit(-1)
	})
	defer tm.Stop()

	cs := counter.NewCounterService()
	rsc := testRaft(cs)

	err := rsc.Run()
	defer rsc.Close()
	if !assert.NoError(t, err) {
		return
	}

	cl := counter.NewCounter(rsc)
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		v, errX := cl.AddValue(ctx, "test", 1)
		assert.NoError(t, errX)
		assert.Equal(t, i+1, int(v))
	}

	// assert.Equal(t, int(2), ms.Count())
	time.Sleep(1 * time.Second)
}

func TestDump(t *testing.T) {
	wal.DumpBlockLogIndex(filepath.Join(testDir, "raft1"), 0, func(s string) {
		plog.Info(s)
	})
}

func TestTriggerSnapshot(t *testing.T) {
	tm := time.AfterFunc(60*time.Second, func() {
		fmt.Printf("exec timeout")
		os.Exit(-1)
	})
	defer tm.Stop()

	cs := counter.NewCounterService()
	rsc := testRaft(cs)

	err := rsc.Run()
	defer rsc.Close()
	if !assert.NoError(t, err) {
		return
	}

	cl := counter.NewCounter(rsc)
	ctx := context.Background()
	for i := 0; i < 40; i++ {
		v, errX := cl.AddValue(ctx, "test", 1)
		assert.NoError(t, errX)
		assert.Equal(t, i+1, int(v))
	}

	// assert.Equal(t, int(2), ms.Count())
	time.Sleep(1 * time.Second)
}
