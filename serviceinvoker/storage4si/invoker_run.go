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
	"sync/atomic"

	"github.com/catyguan/csf/core/corepb"
)

func (this *StorageServiceInvoker) doRun(ready chan error, closef chan interface{}) {
	defer func() {
		close(closef)
	}()
	errRC := this.doRecover()
	if errRC != nil {
		atomic.StoreUint64(&this.closed, 2)
		ready <- errRC
		return
	} else {
		close(ready)
	}
	for {
		select {
		case a := <-this.actions:
			if a == nil {
				return
			}
			r, err := this.doAction(a)
			if a.resp != nil {
				a.resp <- responseOfSSI{result: r, err: err}
			}
		}
	}
}

func (this *StorageServiceInvoker) doRecover() error {
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
	for {
		idx++
		last, reqs, err2 := this.storage.LoadRequest(idx, 128)
		if err2 != nil {
			plog.Infof("storage load request(%v) fail - %v", idx, err2)
			return err2
		}
		if reqs == nil {
			break
		}
		for _, req := range reqs {
			_, err2 = this.cs.ApplyRequest(ctx, req)
			if err2 != nil {
				plog.Infof("service recover apply request fail - %v", err2)
			}
		}
		this.lastIndex = last
		idx = last
	}
	return nil
}

func (this *StorageServiceInvoker) doAction(a *actionOfSSI) (interface{}, error) {
	switch a.typ {
	case 1:
		return this.doRequest(a)
	case 2:
		// snap
		return this.doSnapshot(context.Background())
	default:
		return nil, fmt.Errorf("unknow action type(%v)", a.typ)
	}
}

func (this *StorageServiceInvoker) doRequest(a *actionOfSSI) (*corepb.Response, error) {
	ctx := a.p1.(context.Context)
	req := a.p2.(*corepb.Request)
	s := a.p3.(bool)
	li := this.lastIndex

	if s {
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

	if s && this.snapCount > 0 {
		if this.lastIndex%uint64(this.snapCount) == 0 {
			_, err = this.doSnapshot(ctx)
			if err != nil {
				return nil, err
			}
		}
	}

	return resp, nil
}

func (this *StorageServiceInvoker) doSnapshot(ctx context.Context) (uint64, error) {
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
