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
	"fmt"
	"io"
	"io/ioutil"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/pbutil"
)

const (
	SP_BEGIN         = "begin"
	SP_END           = "end"
	SP_LAST_SNAPSHOT = "lastShapshot"
	SP_PROCESS       = "process"
)

type MasterService struct {
	ep  *masterEP
	cfg *MasterConfig
}

func NewMasterService(cfg *MasterConfig) *MasterService {
	r := &MasterService{}
	r.cfg = cfg
	r.ep = newMasterEP(cfg)
	return r
}

func (this *MasterService) impl() {
	_ = core.CoreService(this)
}

// BEGIN: 实现core.CoreService
func (this *MasterService) VerifyRequest(ctx context.Context, req *corepb.Request) (bool, error) {
	return false, nil
}

func (this *MasterService) ApplyRequest(ctx context.Context, req *corepb.Request) (*corepb.Response, error) {
	action := req.ServicePath
	data := req.Data
	// plog.Infof("%v ApplyRequest - %s(%v)", req.ServiceName, action, len(data))
	switch action {
	case SP_BEGIN:
		info := &PBRequestFollow{}
		pbutil.MustUnmarshal(info, data)
		rsid := this.ep.Begin()
		rinfo := &PBResponseFollow{}
		rinfo.SessionId = rsid
		r := pbutil.MustMarshal(rinfo)
		return req.CreateResponse(r, nil), nil
	case SP_END:
		info := &PBRequestFollow{}
		pbutil.MustUnmarshal(info, data)
		this.ep.End(info.SessionId)
		rinfo := &PBResponseFollow{}
		r := pbutil.MustMarshal(rinfo)
		return req.CreateResponse(r, nil), nil
	case SP_LAST_SNAPSHOT:
		info := &PBRequestFollow{}
		pbutil.MustUnmarshal(info, data)
		i, d, err := this.ep.LastSnapshot(info.SessionId)
		if err != nil {
			return nil, err
		}
		rinfo := &PBResponseFollow{}
		rinfo.Snapshot = &PBLastSnapshot{Index: i, Data: d}
		r := pbutil.MustMarshal(rinfo)
		return req.CreateResponse(r, nil), nil
	case SP_PROCESS:
		info := &PBRequestFollow{}
		pbutil.MustUnmarshal(info, data)
		rlist, err := this.ep.Process(info.SessionId)
		if err != nil {
			return nil, err
		}
		rdlist := make([][]byte, 0)
		for _, rreq := range rlist {
			rd := pbutil.MustMarshal(rreq)
			rdlist = append(rdlist, rd)
		}
		rinfo := &PBResponseFollow{}
		rinfo.Requests = rdlist
		r := pbutil.MustMarshal(rinfo)
		return req.CreateResponse(r, nil), nil
	default:
		return nil, fmt.Errorf("unknow action %s", action)
	}
}

func (this *MasterService) CreateSnapshot(ctx context.Context, w io.Writer) error {
	return nil
}

func (this *MasterService) ApplySnapshot(ctx context.Context, r io.Reader) error {
	_, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return nil
}

// END: 实现core.CoreService
