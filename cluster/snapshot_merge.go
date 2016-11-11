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
	"log"

	"github.com/catyguan/csf/raft/raftpb"
	"github.com/catyguan/csf/snap"
)

func (s *CSFNode) createSnapshotMessage(m raftpb.Message, snapi uint64) snap.Message {
	fpath := s.raftNode.storage.SnapFilePath(snapi)
	ss, err2 := snap.Read(fpath)
	if err2 != nil {
		log.Panicf("read snapshot(%v) fail - %v", fpath, err2)
	}
	m.Snapshot = *ss

	return *snap.NewMessage(m)
}
