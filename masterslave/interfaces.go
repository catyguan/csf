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
package masterslave

import "github.com/catyguan/csf/core/corepb"

type MasterFollower interface {
	OnMasterError(err string)

	OnMasterSaveRequest(idx uint64, req *corepb.Request)
}

type MasterNode interface {
	MasterLoadLastSnapshot() (uint64, []byte, error)

	MasterBeginLoad(idx uint64) (interface{}, error)

	MasterLoadRequest(c interface{}, size int, f MasterFollower) ([]*corepb.Request, error)

	MasterEndLoad(c interface{})

	RemoveFollower(f MasterFollower)
}
