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

// Package etcdlike defines a csf app just like etcd.
package spcommons

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/catyguan/csf/interfaces"
	"github.com/catyguan/csf/pkg/capnslog"
	"github.com/catyguan/csf/pkg/pbutil"
)

var (
	plog = capnslog.NewPackageLogger("github.com/catyguan/csf", "spcommons")
)

type Counter struct {
	data  map[string]uint64
	mutex sync.RWMutex
}

func (this *Counter) GetValue(sm interfaces.ServiceManager, key string) uint64 {
	this.mutex.RLock()
	defer func() {
		this.mutex.RUnlock()
	}()
	if v, ok := this.data[key]; ok {
		return v
	}
	return 0
}

func (this *Counter) AddValue(ctx context.Context, sm interfaces.ServiceManager, key string, val uint64) (uint64, error) {
	req := &CounterInfo{Name: key, Value: val}
	data := pbutil.MustMarshal(req)
	r, err := sm.DoClusterAction(ctx, this.ServiceID(), "add", data)
	if err != nil {
		return 0, err
	}
	resp := &CounterInfo{}
	pbutil.MustUnmarshal(resp, r)
	// plog.Infof("add value return - %v, %v", resp, r)
	return resp.Value, nil
}

func (this *Counter) doAddValue(sm interfaces.ServiceManager, key string, val uint64) uint64 {
	if val == 0 {
		val = 1
	}
	this.mutex.Lock()
	defer func() {
		this.mutex.Unlock()
	}()
	if this.data == nil {
		this.data = make(map[string]uint64)
	}
	if v, ok := this.data[key]; ok {
		this.data[key] = v + val
		return v + val
	} else {
		this.data[key] = val
		return val
	}
}

func (this *Counter) ServiceID() string {
	return "test-counter"
}

func (this *Counter) StartService(sm interfaces.ServiceManager) {
	plog.Infof("%v start done", this.ServiceID())
}

func (this *Counter) StopService(sm interfaces.ServiceManager) {
	plog.Infof("%v stop done", this.ServiceID())
}

func (this *Counter) OnLeadershipUpdate(sm interfaces.ServiceManager, localIsLeader bool) {
	plog.Infof("%v Leadership Update - localIsLeader=%v", this.ServiceID(), localIsLeader)
}

func (this *Counter) OnClose(sm interfaces.ServiceManager) {
	plog.Infof("%v close")
}

func (this *Counter) ApplySnapshot(sm interfaces.ServiceManager, data []byte) error {
	plog.Infof("%v ApplySnapshot - %v", this.ServiceID(), len(data))
	return nil
}

func (this *Counter) CreateSnapshot(sm interfaces.ServiceManager) ([]byte, error) {
	plog.Infof("%v CreateSnapshot", this.ServiceID())
	return nil, nil
}

func (this *Counter) BuildClientHandler(sm interfaces.ServiceManager, mux *http.ServeMux) {
	plog.Infof("%v BuildClientHandler", this.ServiceID())
	this.doBuildClientHandler(sm, mux)
}

func (this *Counter) ApplyAction(sm interfaces.ServiceManager, action string, data []byte) ([]byte, error) {
	plog.Infof("%v ApplyAction - %s(%v)", this.ServiceID(), action, len(data))
	switch action {
	case "add":
		info := &CounterInfo{}
		pbutil.MustUnmarshal(info, data)
		info.Value = this.doAddValue(sm, info.Name, info.Value)
		r := pbutil.MustMarshal(info)
		return r, nil
	default:
		return nil, fmt.Errorf("unknow action %s", action)
	}
}
