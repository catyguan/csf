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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/runobj"
)

func (this *StorageServiceContainer) doRun(ready chan error, ach <-chan *runobj.ActionRequest, p interface{}) {
	errRC := this.doRecover()
	if errRC != nil {
		ready <- errRC
		return
	}
	close(ready)
	for {
		select {
		case a := <-ach:
			if a == nil {
				return
			}
			r, err := this.doAction(a)
			if a.Resp != nil {
				a.Resp <- &runobj.ActionResponse{R1: r, Err: err}
			}
		}
	}
}

func (this *StorageServiceContainer) doRecover() error {
	idx, reader, err := this.storage.LoadLastSnapshot()
	if err != nil {
		plog.Infof("storage load last snapshot fail - %v", err)
		return err
	}
	var data []byte
	if reader != nil {
		data, err = ioutil.ReadAll(reader)
		if err != nil {
			plog.Infof("storage read last snapshot fail - %v", err)
			return err
		}
	}
	if data == nil {
		data = make([]byte, 0)
	}
	buf := bytes.NewBuffer(data)
	ctx := context.Background()
	err = this.cs.ApplySnapshot(ctx, buf)
	if err != nil {
		plog.Infof("service apply snapshot fail - %v", buf)
		return err
	}
	this.lastIndex = idx
	cc, errB := this.storage.BeginLoad(idx + 1)
	if errB != nil {
		plog.Warningf("storage begin load (%v) fail - %v", idx+1, errB)
		return errB
	}
	for {
		last, reqs, err2 := this.storage.LoadRequest(cc, 128, nil)
		if err2 != nil {
			plog.Warningf("storage load request(%v) fail - %v", cc, err2)
			return err2
		}
		if reqs == nil {
			break
		}
		for _, req := range reqs {
			_, err2 = this.cs.ApplyRequest(ctx, req)
			if err2 != nil {
				plog.Warningf("service recover apply request fail - %v", err2)
			}
		}
		this.lastIndex = last
	}
	return nil
}

func (this *StorageServiceContainer) doAction(a *runobj.ActionRequest) (interface{}, error) {
	switch a.Type {
	case 1:
		return this.doRequest(a)
	case 2:
		// snap
		return this.doSnapshot(context.Background())
	case 3:
		ctx := a.P1.(context.Context)
		rlist := a.P2.([]*corepb.Request)
		for _, req := range rlist {
			s, err := this.cs.VerifyRequest(ctx, req)
			if err != nil {
				return nil, err
			}
			resp, err1 := this.doApplyRequest(ctx, req, s)
			err1 = corepb.HandleError(resp, err1)
			if err1 != nil {
				return nil, err1
			}
		}
		return nil, nil
	case 4:
		ctx := a.P1.(context.Context)
		idx := a.P2.(uint64)
		data := a.P3.([]byte)
		return nil, this.doApplySnapshot(ctx, idx, data)
	default:
		return nil, fmt.Errorf("unknow action type(%v)", a.Type)
	}
}

func (this *StorageServiceContainer) doRequest(a *runobj.ActionRequest) (*corepb.Response, error) {
	ctx := a.P1.(context.Context)
	req := a.P2.(*corepb.Request)
	s := a.P3.(bool)
	return this.doApplyRequest(ctx, req, s)

}

func (this *StorageServiceContainer) doApplyRequest(ctx context.Context, req *corepb.Request, save bool) (*corepb.Response, error) {
	li := this.lastIndex

	if save {
		sidx, err := this.storage.SaveRequest(0, req)
		if err != nil {
			return nil, err
		}
		this.lastIndex = sidx
	}

	resp, err := this.cs.ApplyRequest(ctx, req)
	if err != nil {
		this.lastIndex = li
		return nil, err
	}
	if resp != nil {
		resp.ActionIndex = this.lastIndex
	}

	if save && this.snapCount > 0 {
		if this.lastIndex%uint64(this.snapCount) == 0 {
			_, err = this.doSnapshot(ctx)
			if err != nil {
				return nil, err
			}
		}
	}

	return resp, nil
}

func (this *StorageServiceContainer) doSnapshot(ctx context.Context) (uint64, error) {
	buf := bytes.NewBuffer(make([]byte, 0))
	err := this.cs.CreateSnapshot(ctx, buf)
	if err != nil {
		plog.Warningf("service create snapshot fail - %v", err)
		return 0, err
	}
	err = this.storage.SaveSnapshot(this.lastIndex, buf)
	if err != nil {
		plog.Warningf("storage save snapshot fail - %v", err)
		return 0, err
	}
	return this.lastIndex, nil
}

func (this *StorageServiceContainer) doApplySnapshot(ctx context.Context, idx uint64, data []byte) error {
	rs, ok := this.storage.(Rebuildable)
	if !ok {
		return fmt.Errorf("can not ApplySnapshot")
	}
	err := rs.ApplySnapshot(idx, data)
	if err != nil {
		return err
	}
	this.lastIndex = idx
	r := bytes.NewBuffer(data)
	return this.cs.ApplySnapshot(ctx, r)
}
