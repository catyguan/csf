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
	"context"
	"testing"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/service/counter"
	"github.com/stretchr/testify/assert"
)

func TestMemorySeek(t *testing.T) {
	var b bool
	var pos int
	ms := NewMemoryStorage(10).(*MemoryStorage)
	b, _ = ms.seek(0)
	assert.False(t, b)
	b, _ = ms.seek(100)
	assert.False(t, b)

	for i := 1; i <= 5; i++ {
		ms.SaveRequest(uint64(i)*2, nil)
	}
	assert.Equal(t, 5, ms.Count())
	b, _ = ms.seek(0)
	assert.False(t, b)
	b, pos = ms.seek(2)
	assert.True(t, b)
	assert.Equal(t, 0, pos)

	b, pos = ms.seek(5)
	assert.True(t, b)
	assert.Equal(t, 1, pos)

	for i := 6; i <= 10; i++ {
		ms.SaveRequest(uint64(i)*2, nil)
	}
	b, pos = ms.seek(5)
	assert.True(t, b)
	assert.Equal(t, 1, pos)

	b, pos = ms.seek(16)
	assert.True(t, b)
	assert.Equal(t, 7, pos)

	b, pos = ms.seek(17)
	assert.True(t, b)
	assert.Equal(t, 7, pos)

	for i := 11; i <= 15; i++ {
		ms.SaveRequest(uint64(i)*2, nil)
	}
	b, pos = ms.seek(5)
	assert.False(t, b)

	b, pos = ms.seek(17)
	assert.True(t, b)
	// head:5/12, off:2
	assert.Equal(t, 7, pos)
}

func TestMemoryOverwrite(t *testing.T) {
	// var b bool
	// var pos int
	ms := NewMemoryStorage(10).(*MemoryStorage)
	for i := 1; i <= 5; i++ {
		ms.SaveRequest(uint64(i)*2, nil)
	}
	ms.SaveRequest(1, nil)
	assert.Equal(t, 1, ms.size)
	// plog.Infof("%v", ms.ents)

	ms.Reset()
	for i := 1; i <= 5; i++ {
		ms.SaveRequest(uint64(i)*2, nil)
	}
	ms.SaveRequest(2, nil)
	assert.Equal(t, 1, ms.size)
	// plog.Infof("%v", ms.ents)

	ms.Reset()
	for i := 1; i <= 5; i++ {
		ms.SaveRequest(uint64(i)*2, nil)
	}
	ms.SaveRequest(5, nil)
	assert.Equal(t, 3, ms.size)
	// plog.Infof("%v", ms.ents)

	ms.Reset()
	for i := 1; i <= 15; i++ {
		ms.SaveRequest(uint64(i)*2, nil)
	}
	ms.SaveRequest(15, nil)
	assert.Equal(t, 3, ms.size)
	assert.Equal(t, 0, ms.head)
	// plog.Infof("%v", ms.ents)

	ms.Reset()
	for i := 1; i <= 15; i++ {
		ms.SaveRequest(uint64(i)*2, nil)
	}
	ms.SaveRequest(22, nil)
	assert.Equal(t, 6, ms.size)
	// plog.Infof("%v", ms.ents)
}

func createCounter() (counter.Counter, *counter.CounterService) {
	s := counter.NewCounterService()
	si := core.NewSimpleServiceInvoker(s)
	return counter.NewCounter(si), s
}

func doTestCall1(t *testing.T, c counter.Counter) {
	ctx := context.Background()

	v, err := c.GetValue(ctx, "test")
	assert.NoError(t, err)
	assert.True(t, 0 == v)
	v, err = c.AddValue(ctx, "test", 1)
	assert.NoError(t, err)
	assert.True(t, 1 == v)
	v, err = c.AddValue(ctx, "test", 10)
	assert.NoError(t, err)
	assert.True(t, 11 == v)
	v, err = c.GetValue(ctx, "test")
	assert.NoError(t, err)
	assert.True(t, 11 == v)
}

func TestBase(t *testing.T) {
	ms := NewMemoryStorage(16).(*MemoryStorage)
	cfg := NewConfig()
	cfg.Storage = ms
	cfg.Service = counter.NewCounterService()
	si := NewStorageServiceInvoker(cfg)

	err := si.Run()
	assert.NoError(t, err)
	defer si.Close()

	c := counter.NewCounter(si)
	doTestCall1(t, c)

	assert.Equal(t, int(2), ms.Count())
}

func TestRecover(t *testing.T) {
	ms := NewMemoryStorage(16).(*MemoryStorage)
	cfg := NewConfig()
	cfg.Storage = ms
	cfg.Service = counter.NewCounterService()
	si := NewStorageServiceInvoker(cfg)

	si.Run()
	defer si.Close()

	c := counter.NewCounter(si)
	doTestCall1(t, c)

	cfg2 := NewConfig()
	cfg2.Storage = ms
	cfg2.Service = counter.NewCounterService()
	si2 := NewStorageServiceInvoker(cfg2)

	si2.Run()
	defer si2.Close()

	c2 := counter.NewCounter(si2)
	v2, err2 := c2.GetValue(context.Background(), "test")
	assert.NoError(t, err2)
	assert.True(t, 11 == v2)
}

func TestRecoverSnapshot(t *testing.T) {
	ms := NewMemoryStorage(16).(*MemoryStorage)
	cfg := NewConfig()
	cfg.SnapCount = 10
	cfg.Storage = ms
	cfg.Service = counter.NewCounterService()
	si := NewStorageServiceInvoker(cfg)

	si.Run()
	defer si.Close()

	ctx := context.Background()

	c := counter.NewCounter(si)
	for i := 0; i < 25; i++ {
		_, err := c.AddValue(ctx, "test", 1)
		assert.NoError(t, err)
	}

	cfg2 := NewConfig()
	cfg2.Storage = ms
	cfg2.Service = counter.NewCounterService()
	si2 := NewStorageServiceInvoker(cfg2)

	si2.Run()
	defer si2.Close()

	c2 := counter.NewCounter(si2)
	v2, err2 := c2.GetValue(context.Background(), "test")
	assert.NoError(t, err2)
	assert.Equal(t, uint64(25), v2)
}
