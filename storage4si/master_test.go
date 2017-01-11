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

package storage4si

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/masterslave"
	"github.com/stretchr/testify/assert"
)

func doTestCall1(s Storage) {
	assert.NotNil(nil, nil)
}

func TestBase(t *testing.T) {
	ms := NewMemoryStorage(1024).(*MemoryStorage)
	cfg := masterslave.NewMasterConfig()
	cfg.Master = ms
	service := masterslave.NewMasterService(cfg)
	si := core.NewSimpleServiceContainer(service)
	api := masterslave.NewMasterAPI("test", si)

	ctx := context.Background()
	sid, err := api.Begin(ctx)
	assert.NoError(t, err)
	assert.True(t, sid > 0)
	if sid > 0 {
		defer api.End(ctx, sid)
	}
	lidx, snapshot, err2 := api.LastSnapshot(ctx, sid)
	assert.NoError(t, err2)
	assert.Equal(t, uint64(0), lidx)
	assert.Nil(t, snapshot)

	rlist, err3 := api.Process(ctx, sid)
	assert.NoError(t, err3)
	assert.Nil(t, rlist)
}

func TestSnapshot(t *testing.T) {
	ms := NewMemoryStorage(1024).(*MemoryStorage)
	cfg := masterslave.NewMasterConfig()
	cfg.Master = ms
	service := masterslave.NewMasterService(cfg)
	si := core.NewSimpleServiceContainer(service)
	api := masterslave.NewMasterAPI("test", si)

	ctx := context.Background()
	sid, err := api.Begin(ctx)
	assert.NoError(t, err)
	assert.True(t, sid > 0)
	if sid > 0 {
		defer api.End(ctx, sid)
	}

	buf := bytes.NewBuffer(make([]byte, 0))
	buf.WriteString("hello world")
	ms.SaveSnapshot(10, buf)

	lidx, snapshot, err2 := api.LastSnapshot(ctx, sid)
	assert.NoError(t, err2)
	assert.Equal(t, uint64(10), lidx)
	assert.Equal(t, "hello world", string(snapshot))

	rlist, err3 := api.Process(ctx, sid)
	assert.NoError(t, err3)
	assert.Nil(t, rlist)
}

func TestProcess(t *testing.T) {
	ms := NewMemoryStorage(1024).(*MemoryStorage)
	cfg := masterslave.NewMasterConfig()
	cfg.Master = ms
	service := masterslave.NewMasterService(cfg)
	si := core.NewSimpleServiceContainer(service)
	api := masterslave.NewMasterAPI("test", si)

	ctx := context.Background()
	sid, err := api.Begin(ctx)
	if !assert.NoError(t, err) {
		return
	}
	if !assert.True(t, sid > 0) {
		return
	}
	defer api.End(ctx, sid)

	lidx, snapshot, err2 := api.LastSnapshot(ctx, sid)
	assert.NoError(t, err2)
	assert.Equal(t, uint64(0), lidx)
	assert.Nil(t, snapshot)

	for i := 1; i <= 10; i++ {
		req := &corepb.Request{}
		req.ID = uint64(i)
		_, errX := ms.SaveRequest(0, req)
		assert.NoError(t, errX)
	}

	for i := 0; i < 2; i++ {
		rlist, err3 := api.Process(ctx, sid)
		assert.NoError(t, err3)
		if i == 0 {
			assert.Equal(t, 10, len(rlist))
		} else {
			assert.Equal(t, 0, len(rlist))
		}
		//plog.Infof("%v", rlist)
	}

	if api != nil {
		go func() {
			for i := 1; i <= 10; i++ {
				req := &corepb.Request{}
				req.ID = uint64(10 + i)
				_, errX := ms.SaveRequest(0, req)
				assert.NoError(t, errX)

				if i == 9 {
					time.Sleep(time.Millisecond * 100)
				}
			}
		}()
		c := 0
		for i := 0; i < 10; i++ {
			rlist, err4 := api.Process(ctx, sid)
			assert.NoError(t, err4)
			plog.Infof("%d: %v", i, rlist)
			c += len(rlist)
			if c >= 10 {
				break
			}
		}
		assert.Equal(t, 10, c)
	}
}

func TestSessionExpire(t *testing.T) {
	ms := NewMemoryStorage(1024).(*MemoryStorage)
	cfg := masterslave.NewMasterConfig()
	cfg.Master = ms
	cfg.SessionExpire = 1 * time.Second
	service := masterslave.NewMasterService(cfg)
	si := core.NewSimpleServiceContainer(service)
	api := masterslave.NewMasterAPI("test", si)

	ctx := context.Background()
	sid, err := api.Begin(ctx)
	assert.NoError(t, err)
	assert.True(t, sid > 0)
	if sid > 0 {
		defer api.End(ctx, sid)
	}
	time.Sleep(1100 * time.Millisecond)
	_, _, err2 := api.LastSnapshot(ctx, sid)
	assert.Error(t, err2)
	assert.Equal(t, masterslave.ErrSessionNotExists, err2)
}
