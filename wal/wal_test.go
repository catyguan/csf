// Copyright 2015 The etcd Authors
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

package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tc_config1() *Config {
	r := NewConfig()
	r.Dir = filepath.Join(testDir, "waltest")
	r.BlockRollSize = 16 * 1024
	r.InitMetadata = []byte("test")
	return r
}

func TestWALBase(t *testing.T) {
	cfg := tc_config1()
	os.RemoveAll(cfg.Dir)

	w, _, err2 := initWALCore(cfg)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	w.Close()

	time.Sleep(time.Second)
}

func TestWALSave(t *testing.T) {
	cfg := tc_config1()
	// cfg.PostAppend = true

	w, bs, err2 := initWALCore(cfg)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()
	plog.Infof("TestWALSave meta %v", string(bs))

	sz := 100
	for i := 0; i < sz; i++ {
		s := fmt.Sprintf("hello world %v", i)
		w.Append(0, []byte(s))
	}

	plog.Infof("lastIndex: %v", w.LastIndex())

	time.Sleep(time.Second)
}

func TestWALSync(t *testing.T) {
	cfg := tc_config1()

	w, _, err2 := initWALCore(cfg)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()

	err := w.Sync()
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	time.Sleep(time.Second)
}

func TestWALDelete(t *testing.T) {
	cfg := tc_config1()

	w, _, err2 := initWALCore(cfg)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()

	// w.GetCursor(10)

	err := w.Delete()
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	time.Sleep(time.Second)
}

func TestWALReset(t *testing.T) {
	cfg := tc_config1()

	w, _, err2 := initWALCore(cfg)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()

	// w.GetCursor(10)

	err := w.Reset()
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	time.Sleep(time.Second)
}

func TestWALRoll(t *testing.T) {
	cfg := tc_config1()
	cfg.BlockRollSize = 4 * 1024

	w, bs, err2 := initWALCore(cfg)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()
	plog.Infof("TestWALSave meta %v", string(bs))

	sz := 200
	for i := 0; i < sz; i++ {
		s := fmt.Sprintf("hello world %v", i)
		w.Append(0, []byte(s))
	}

	plog.Infof("lastIndex: %v", w.LastIndex())

	time.Sleep(time.Second)
}

func TestWALTruncate(t *testing.T) {
	cfg := tc_config1()

	w, _, err2 := initWALCore(cfg)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()

	err := w.Truncate(100)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	time.Sleep(time.Second)
}

func TestCursor(t *testing.T) {
	// TestWALSave(t)
	cfg := tc_config1()

	w, _, err2 := initWALCore(cfg)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()

	c, err3 := w.GetCursor(80)
	if err3 != nil {
		t.Fatalf("err3 = %v", err3)
	}
	for {
		e, err4 := c.Read()
		if err4 != nil {
			t.Fatalf("err4 = %v", err4)
		}
		if e == nil {
			plog.Infof("cursor end")
			break
		}
		plog.Infof("result = %v, %v", e.Index, string(e.Data))
	}
	c.Close()

	err := w.Delete()
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	time.Sleep(time.Second)
}
