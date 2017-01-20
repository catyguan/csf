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
	"fmt"
	"io"
	"io/ioutil"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/pbutil"
	"github.com/catyguan/csf/storage4si"
)

const (
	SP_MAKESNAPSHOT = "makeSnapshot"
)

type AdminService struct {
	ep *storage4si.StorageServiceContainer
}

func NewAdminService(ep *storage4si.StorageServiceContainer) *AdminService {
	r := &AdminService{}
	r.ep = ep
	return r
}

func (this *AdminService) impl() {
	_ = core.CoreService(this)
}

// BEGIN: 实现core.CoreService
func (this *AdminService) VerifyRequest(ctx context.Context, req *corepb.Request) (bool, error) {
	return req.IsExecuteType(), nil
}

func (this *AdminService) ApplyRequest(ctx context.Context, req *corepb.Request) (*corepb.Response, error) {
	action := req.ServicePath
	// data := req.Data
	switch action {
	case SP_MAKESNAPSHOT:
		i, err := this.ep.MakeSnapshot()
		if err != nil {
			plog.Warningf("%s MakeSnapshot fail - %s", req.ServiceName, err)
			return nil, err
		}
		plog.Infof("%s MakeSnapshot done - %v", req.ServiceName, i)

		rinfo := &PBMakeSnapshotResult{}
		rinfo.Index = i
		r := pbutil.MustMarshal(rinfo)
		return req.CreateResponse(r, nil), nil
	default:
		return nil, fmt.Errorf("unknow action %s", action)
	}
}

func (this *AdminService) CreateSnapshot(ctx context.Context, w io.Writer) error {
	return nil
}

func (this *AdminService) ApplySnapshot(ctx context.Context, r io.Reader) error {
	_, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return nil
}

// END: 实现core.CoreService
