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

package storage4si

import (
	"context"

	"github.com/catyguan/csf/core/corepb"
)

type SlaveStorageContainerApply struct {
	ssc *StorageServiceContainer
}

func NewSlaveStorageContainerApply(ssc *StorageServiceContainer) *SlaveStorageContainerApply {
	r := &SlaveStorageContainerApply{ssc: ssc}
	return r
}

func (this *SlaveStorageContainerApply) ApplySnapshot(idx uint64, data []byte) error {
	ctx := context.Background()
	return this.ssc.ApplySnapshot(ctx, idx, data)
}

func (this *SlaveStorageContainerApply) ApplyRequests(reqlist []*corepb.Request) error {
	ctx := context.Background()
	return this.ssc.ApplyRequests(ctx, reqlist)
}
