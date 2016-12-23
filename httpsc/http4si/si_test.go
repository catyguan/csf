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
package http4si

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/catyguan/csf/service/counter"
	"github.com/stretchr/testify/assert"
)

func testClient() (*HttpServiceInvoker, error) {
	port := 8086
	cfg := &Config{}
	cfg.URL = fmt.Sprintf("http://localhost:%d/service", port)
	cfg.ExcecuteTimeout = 10 * time.Second
	si, err := NewHttpServiceInvoker(cfg, nil)
	return si, err
}

func TestBase(t *testing.T) {
	time.AfterFunc(11*time.Second, func() {
		fmt.Printf("exec timeout")
		os.Exit(-1)
	})

	si, err := testClient()
	assert.NoError(t, err)

	octx := context.Background()
	ctx, _ := context.WithTimeout(octx, 10*time.Second)
	c := counter.NewCounter(si)
	v, err2 := c.AddValue(ctx, "test", 1)
	assert.NoError(t, err2)
	assert.Equal(t, uint64(1), v)
}

func TestGet(t *testing.T) {
	time.AfterFunc(11*time.Second, func() {
		fmt.Printf("exec timeout")
		os.Exit(-1)
	})

	si, err := testClient()
	assert.NoError(t, err)

	octx := context.Background()
	ctx, _ := context.WithTimeout(octx, 10*time.Second)
	c := counter.NewCounter(si)
	v, err2 := c.GetValue(ctx, "test")
	assert.NoError(t, err2)
	assert.Equal(t, uint64(1), v)
}

func TestRaiseError(t *testing.T) {
	si, err := testClient()
	assert.NoError(t, err)

	octx := context.Background()
	ctx, _ := context.WithTimeout(octx, 10*time.Second)
	c := counter.NewCounter(si)
	err2 := c.RaiseError(ctx, "test")
	assert.Error(t, err2)
	assert.Equal(t, "error:(500), error:test\n", err2.Error())
}

func TestClientTimeout(t *testing.T) {
	c, err := net.Dial("tcp", "localhost:8086")
	assert.NoError(t, err)
	defer c.Close()
	c.SetDeadline(time.Now().Add(31 * time.Second))
	b := make([]byte, 10)
	_, err2 := c.Read(b)
	assert.Error(t, err2)
	assert.Equal(t, "EOF", err2.Error())
}

func TestServerTimeout(t *testing.T) {
	si, err := testClient()
	assert.NoError(t, err)

	octx := context.Background()
	ctx, _ := context.WithTimeout(octx, 10*time.Second)
	c := counter.NewCounter(si)
	err2 := c.Sleep(ctx, 1000*30)
	assert.Error(t, err2)
	if err2 != nil {
		t.Logf("invoke error - %v", err2)
		assert.NotEqual(t, -1, strings.Index(err2.Error(), "canceled"))
	}
}
