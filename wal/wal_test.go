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
	ents := make([]Entry, 0)
	for i := 0; i < sz; i++ {
		s := fmt.Sprintf("hello world %v", i)
		ents = append(ents, Entry{Index: uint64(i), Data: []byte(s)})
	}
	rsc := w.Append(ents, true)
	rs := <-rsc

	plog.Infof("save: %v, lastIndex: %v", rs.Index, w.LastIndex())

	time.Sleep(time.Second)
}

func TestWALSync(t *testing.T) {
	cfg := tc_config1()

	w, _, err2 := initWALCore(cfg)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()

	rsc := w.Sync()
	rs := <-rsc
	if rs.Err != nil {
		t.Fatalf("err = %v", rs.Err)
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

	c, _ := w.GetCursor(10)
	if c != nil {
		e, err2 := c.Read()
		plog.Infof("cursor read1 %v, %v", e, err2)
	}

	rsc := w.Reset()
	rs := <-rsc
	if rs.Err != nil {
		t.Fatalf("err = %v", rs.Err)
	}

	if c != nil {
		e, err2 := c.Read()
		plog.Infof("cursor read2 %v, %v", e, err2)
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

	ents := make([]Entry, 0)
	sz := 200
	for i := 0; i < sz; i++ {
		s := fmt.Sprintf("hello world %v", i)
		ents = append(ents, Entry{Index: 0, Data: []byte(s)})
	}
	rsc := w.Append(ents, true)
	rs := <-rsc

	plog.Infof("result:%v, lastIndex: %v", rs.Index, w.LastIndex())

	time.Sleep(time.Second)
}

func TestWALTruncate(t *testing.T) {
	cfg := tc_config1()

	w, _, err2 := initWALCore(cfg)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()

	rsc := w.Truncate(100)
	rs := <-rsc
	if rs.Err != nil {
		t.Fatalf("err = %v", rs.Err)
	}

	time.Sleep(time.Second)
}

func TestWALListener(t *testing.T) {
	cfg := tc_config1()

	w, _, err2 := initWALCore(cfg)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()

	w.AddListener(NewLogListener("TEST", plog))

	<-w.Reset()
	ents := make([]Entry, 1)
	ents[0].Index = 1
	ents[0].Data = []byte("hello")
	<-w.Append(ents, false)

	<-w.Truncate(100)

	time.Sleep(time.Second)
}
