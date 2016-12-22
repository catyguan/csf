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
	"testing"
	"time"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/storage4si"
	"github.com/stretchr/testify/assert"
)

func testMasterAPI() (MasterAPI, storage4si.Storage) {
	ms := storage4si.NewMemoryStorage(1024)
	cfg := NewMasterConfig()
	cfg.Storage = ms
	service := NewMasterService(cfg)
	si := core.NewSimpleServiceInvoker(service)
	api := NewMasterAPI("test", si)
	return api, ms
}

func TestSlaveBase(t *testing.T) {
	master, _ := testMasterAPI()

	cfg := NewSlaveConfig()
	cfg.Master = master
	cfg.Apply = &fakeSlaveApply{}
	service := NewSlaveService(cfg)

	err := service.Run()
	if !assert.NoError(t, err) {
		return
	}
	defer service.Close()

	time.Sleep(1 * time.Second)
}

func TestSlaveFollow(t *testing.T) {
	master, ms := testMasterAPI()

	cfg := NewSlaveConfig()
	cfg.Master = master
	cfg.Apply = &fakeSlaveApply{}
	service := NewSlaveService(cfg)

	err := service.Run()
	if !assert.NoError(t, err) {
		return
	}
	defer service.Close()

	time.Sleep(100 * time.Millisecond)

	go func() {
		for i := 1; i <= 10; i++ {
			req := &corepb.Request{}
			req.ID = uint64(i)
			_, errX := ms.SaveRequest(0, req)
			assert.NoError(t, errX)

			if i == 5 {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	time.Sleep(1 * time.Second)
}
