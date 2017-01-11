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
package storage4si

import (
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/masterslave"
)

type MasterFollowerListener struct {
	Follower masterslave.MasterFollower
}

func (this *MasterFollowerListener) GetFollower() masterslave.MasterFollower {
	return this.Follower
}

func (this *MasterFollowerListener) OnReset() {
	this.Follower.OnMasterError("reset")
}

func (this *MasterFollowerListener) OnTruncate(idx uint64) {
	this.Follower.OnMasterError("truncate")
}

func (this *MasterFollowerListener) OnSaveRequest(idx uint64, req *corepb.Request) {
	this.Follower.OnMasterSaveRequest(idx, req)
}

func (this *MasterFollowerListener) OnSaveSnapshot(idx uint64) {
}

func (this *MasterFollowerListener) OnClose() {
	this.Follower.OnMasterError("close")
}
