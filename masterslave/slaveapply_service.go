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

package masterslave

import (
	"bytes"
	"context"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
)

type SlaveServiceApply struct {
	sc core.ServiceHolder
}

func NewSlaveServiceApply(sc core.ServiceHolder) *SlaveServiceApply {
	return &SlaveServiceApply{sc: sc}
}

func (this *SlaveServiceApply) ApplySnapshot(idx uint64, data []byte) error {
	r := bytes.NewBuffer(data)
	ctx := context.Background()
	return this.sc.ExecuteServiceFunc(ctx, func(ctx context.Context, cs core.CoreService) error {
		return cs.ApplySnapshot(ctx, r)
	})
}

func (this *SlaveServiceApply) ApplyRequests(reqlist []*corepb.Request) error {
	ctx := context.Background()
	return this.sc.ExecuteServiceFunc(ctx, func(ctx context.Context, cs core.CoreService) error {
		for _, req := range reqlist {
			_, err := cs.VerifyRequest(ctx, req)
			if err != nil {
				return err
			}
			_, err = cs.ApplyRequest(ctx, req)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
