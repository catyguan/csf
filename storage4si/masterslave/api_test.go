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
	"testing"
	"time"

	"github.com/catyguan/csf/httpsc/http4si"
	"github.com/stretchr/testify/assert"
)

func testClient() (*http4si.HttpServiceInvoker, error) {
	cfg := &http4si.Config{}
	cfg.URL = "http://localhost:8086/peer"
	cfg.ExcecuteTimeout = 10 * time.Second
	si, err := http4si.NewHttpServiceInvoker(cfg, nil)
	return si, err
}

func TestAPIBase(t *testing.T) {
	si, err0 := testClient()
	if !assert.NoError(t, err0) {
		return
	}

	api := NewMasterAPI("service-counter#master", si)

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
