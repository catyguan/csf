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
)

const (
// SP_BEGIN         = "begin"
)

type SlaveService struct {
	ep  *slaveEP
	cfg *SlaveConfig
}

func NewSlaveService(cfg *SlaveConfig) *SlaveService {
	r := &SlaveService{}
	r.cfg = cfg
	r.ep = newSlaveEP(cfg)
	return r
}

func (this *SlaveService) impl() {
	_ = core.CoreService(this)
}

func (this *SlaveService) Run() error {
	return this.ep.Run()
}

func (this *SlaveService) Close() {
	this.ep.Close()
}

// BEGIN: 实现core.CoreService
func (this *SlaveService) VerifyRequest(ctx context.Context, req *corepb.Request) (bool, error) {
	return false, nil
}

func (this *SlaveService) ApplyRequest(ctx context.Context, req *corepb.Request) (*corepb.Response, error) {
	action := req.ServicePath
	// data := req.Data
	// plog.Infof("%v ApplyRequest - %s(%v)", req.ServiceName, action, len(data))
	switch action {
	default:
		return nil, fmt.Errorf("unknow action %s", action)
	}
}

func (this *SlaveService) CreateSnapshot(ctx context.Context, w io.Writer) error {
	return nil
}

func (this *SlaveService) ApplySnapshot(ctx context.Context, r io.Reader) error {
	_, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return nil
}

// END: 实现core.CoreService
