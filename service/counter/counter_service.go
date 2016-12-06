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

package counter

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
	SERVICE_NAME = "service-counter"
	SP_GET       = "get"
	SP_ADD       = "add"
)

type CounterService struct {
	ep counterEP
}

func NewCounterService() *CounterService {
	return &CounterService{}
}

func (this *CounterService) impl() {
	_ = core.CoreService(this)
}

// BEGIN: 实现core.CoreService
func (this *CounterService) VerifyRequest(ctx context.Context, req *corepb.Request) (bool, error) {
	return req.IsExecuteType(), nil
}

func (this *CounterService) ApplyRequest(ctx context.Context, req *corepb.Request) (*corepb.Response, error) {
	action := req.Info.ServicePath
	data := req.Data
	plog.Infof("%v ApplyRequest - %s(%v)", req.Info.ServiceName, action, len(data))
	switch action {
	case SP_GET:
		info := &CounterInfo{}
		pbutil.MustUnmarshal(info, req.Data)
		info.Value = this.ep.GetValue(info.Name)
		r := pbutil.MustMarshal(info)
		return req.CreateResponse(r), nil
	case SP_ADD:
		info := &CounterInfo{}
		pbutil.MustUnmarshal(info, req.Data)
		info.Value = this.ep.AddValue(info.Name, info.Value)
		r := pbutil.MustMarshal(info)
		return req.CreateResponse(r), nil
	default:
		return nil, fmt.Errorf("unknow action %s", action)
	}
}

func (this *CounterService) CreateSnapshot(ctx context.Context, w io.Writer) error {
	plog.Infof("%v CreateSnapshot", SERVICE_NAME)
	cs := &CounterSnapshot{}
	cs.Info = make([]*CounterInfo, 0)
	this.ep.mu.Lock()
	for k, v := range this.ep.data {
		info := &CounterInfo{}
		info.Name = k
		info.Value = v
		cs.Info = append(cs.Info, info)
	}
	this.ep.mu.Unlock()
	r := pbutil.MustMarshal(cs)
	_, err := w.Write(r)
	return err
}

func (this *CounterService) ApplySnapshot(ctx context.Context, r io.Reader) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	plog.Infof("%v ApplySnapshot - %v", SERVICE_NAME, len(data))

	cs := &CounterSnapshot{}
	pbutil.MustUnmarshal(cs, data)
	this.ep.mu.Lock()
	defer this.ep.mu.Unlock()
	this.ep.data = make(map[string]uint64)
	for _, info := range cs.Info {
		this.ep.data[info.Name] = info.Value
	}
	return nil
}

// END: 实现core.CoreService
