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
	"context"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/pbutil"
)

type MasterInvoker struct {
	sn string
	si core.ServiceInvoker
}

func NewMasterAPI(serviceName string, si core.ServiceInvoker) MasterAPI {
	return &MasterInvoker{sn: serviceName, si: si}
}

// BEGIN: 业务
func (this *MasterInvoker) Begin(ctx context.Context) (uint64, error) {
	obj := &PBRequestFollow{}
	data := pbutil.MustMarshal(obj)
	req := corepb.NewQueryRequest(this.sn, SP_BEGIN, data)
	resp, err := core.Invoke(this.si, ctx, req)
	err = corepb.HandleError(resp, err)
	if err != nil {
		return 0, err
	}
	rinfo := &PBResponseFollow{}
	pbutil.MustUnmarshal(rinfo, resp.Data)
	return rinfo.SessionId, nil
}

func (this *MasterInvoker) End(ctx context.Context, sid uint64) error {
	obj := &PBRequestFollow{SessionId: sid}
	data := pbutil.MustMarshal(obj)
	req := corepb.NewQueryRequest(this.sn, SP_END, data)
	resp, err := core.Invoke(this.si, ctx, req)
	err = corepb.HandleError(resp, err)
	if err != nil {
		return err
	}
	// rinfo := &PBResponseFollow{}
	// pbutil.MustUnmarshal(rinfo, resp.Data)
	return nil
}

func (this *MasterInvoker) LastSnapshot(ctx context.Context, sid uint64) (uint64, []byte, error) {
	obj := &PBRequestFollow{SessionId: sid}
	data := pbutil.MustMarshal(obj)
	req := corepb.NewQueryRequest(this.sn, SP_LAST_SNAPSHOT, data)
	resp, err := core.Invoke(this.si, ctx, req)
	err = corepb.HandleError(resp, err)
	if err != nil {
		return 0, nil, err
	}
	rinfo := &PBResponseFollow{}
	pbutil.MustUnmarshal(rinfo, resp.Data)
	if rinfo.Snapshot == nil {
		return 0, nil, nil
	}
	return rinfo.Snapshot.Index, rinfo.Snapshot.Data, nil
}

func (this *MasterInvoker) Process(ctx context.Context, sid uint64) ([]*corepb.Request, error) {
	obj := &PBRequestFollow{SessionId: sid}
	data := pbutil.MustMarshal(obj)
	req := corepb.NewQueryRequest(this.sn, SP_PROCESS, data)
	resp, err := core.Invoke(this.si, ctx, req)
	err = corepb.HandleError(resp, err)
	if err != nil {
		return nil, err
	}
	rinfo := &PBResponseFollow{}
	pbutil.MustUnmarshal(rinfo, resp.Data)
	if rinfo.Requests == nil {
		return nil, nil
	}
	rlist := make([]*corepb.Request, 0, len(rinfo.Requests))
	for _, rd := range rinfo.Requests {
		rr := &corepb.Request{}
		pbutil.MustUnmarshal(rr, rd)
		rlist = append(rlist, rr)
	}
	return rlist, nil
}

// END: 业务
