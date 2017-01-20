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

package storage4si0admin

import (
	"context"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/pbutil"
)

type AdminInvoker struct {
	sn string
	si core.ServiceInvoker
}

func NewAdminAPI(serviceName string, si core.ServiceInvoker) AdminAPI {
	return &AdminInvoker{sn: serviceName, si: si}
}

// BEGIN: 业务
func (this *AdminInvoker) MakeSnapshot(ctx context.Context) (uint64, error) {
	req := corepb.NewExecuteRequest(this.sn, SP_MAKESNAPSHOT, nil)
	resp, err := core.Invoke(this.si, ctx, req)
	err = corepb.HandleError(resp, err)
	if err != nil {
		return 0, err
	}
	rinfo := &PBMakeSnapshotResult{}
	pbutil.MustUnmarshal(rinfo, resp.Data)
	return rinfo.Index, nil
}

// END: 业务
