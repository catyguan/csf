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
package raft4si

import (
	"github.com/catyguan/csf/raft"
	"github.com/catyguan/csf/raft/raftpb"
)

type debugStorage struct {
	backend raft.Storage
}

func (this *debugStorage) InitialState() (raftpb.HardState, raftpb.ConfState, error) {
	r1, r2, r3 := this.backend.InitialState()
	plog.Infof("InitialState() (HardState:%v, ConfState:%v, error:%v)", r1, r2, r3)
	return r1, r2, r3
}

func (this *debugStorage) Entries(lo, hi, maxSize uint64) ([]raftpb.Entry, error) {
	r1, r2 := this.backend.Entries(lo, hi, maxSize)
	plog.Infof("Entries(lo:%d, hi:%d, maxSize:%d) ([]Entry:%v, error:%v)", lo, hi, maxSize, r1, r2)
	return r1, r2
}

func (this *debugStorage) Term(i uint64) (uint64, error) {
	r1, r2 := this.backend.Term(i)
	plog.Infof("Term(i:%d) (%v, error:%v)", i, r1, r2)
	return r1, r2
}

func (this *debugStorage) LastIndex() (uint64, error) {
	r1, r2 := this.backend.LastIndex()
	// plog.Infof("LastIndex() (%v, error:%v)", r1, r2)
	return r1, r2
}

func (this *debugStorage) FirstIndex() (uint64, error) {
	r1, r2 := this.backend.FirstIndex()
	// plog.Infof("FirstIndex() (%v, error:%v)", r1, r2)
	return r1, r2
}

func (this *debugStorage) Snapshot() (raftpb.Snapshot, error) {
	r1, r2 := this.backend.Snapshot()
	plog.Infof("Snapshot() (Snapshot:%v, error:%v)", r1, r2)
	return r1, r2
}
