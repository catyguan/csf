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

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/pbutil"
)

type CounterInvoker struct {
	si core.ServiceInvoker
}

func NewCounter(si core.ServiceInvoker) Counter {
	return &CounterInvoker{si: si}
}

func (this *CounterInvoker) impl() {
	_ = Counter(this)
}

// BEGIN: 业务
func (this *CounterInvoker) GetValue(ctx context.Context, key string) (uint64, error) {
	obj := &CounterInfo{Name: key}
	data := pbutil.MustMarshal(obj)
	req := corepb.NewQueryRequest(SERVICE_NAME, SP_GET, data)
	resp, err := this.si.InvokeRequest(ctx, req)
	err = corepb.HandleError(resp, err)
	if err != nil {
		return 0, err
	}
	rinfo := &CounterInfo{}
	pbutil.MustUnmarshal(rinfo, resp.Data)
	return rinfo.Value, nil
}

func (this *CounterInvoker) AddValue(ctx context.Context, key string, val uint64) (uint64, error) {
	obj := &CounterInfo{Name: key, Value: val}
	data := pbutil.MustMarshal(obj)
	req := corepb.NewExecuteRequest(SERVICE_NAME, SP_ADD, data)
	resp, err := this.si.InvokeRequest(ctx, req)
	err = corepb.HandleError(resp, err)
	if err != nil {
		return 0, err
	}
	rinfo := &CounterInfo{}
	pbutil.MustUnmarshal(rinfo, resp.Data)
	return rinfo.Value, nil
}

// END: 业务
