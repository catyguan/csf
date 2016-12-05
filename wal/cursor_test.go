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
	"testing"
	"time"
)

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

	// rsc := w.Reset()
	// rs := <-rsc
	// if rs.Err != nil {
	// 	t.Fatalf("err = %v", rs.Err)
	// }

	time.Sleep(time.Second)
}

func TestFollowRead(t *testing.T) {
	// TestWALSave(t)
	cfg := tc_config1()

	w, _, err2 := initWALCore(cfg)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()

	c, err3 := w.GetFollow(80)
	if err3 != nil {
		t.Fatalf("err3 = %v", err3)
	}
	ti := time.After(5 * time.Second)
	ch := c.EntryCh()
	func() {
		for {
			select {
			case e := <-ch:
				if e == nil {
					t.Fatalf("err4 = %v", c.Error())
				}
				plog.Infof("result = %v, %v", e.Index, string(e.Data))
			case <-ti:
				plog.Infof("timeout,exit")
				return
			}
		}
	}()
	c.Close()

	time.Sleep(time.Second)
}

func TestFollowPush(t *testing.T) {
	// TestWALSave(t)
	cfg := tc_config1()

	w, _, err2 := initWALCore(cfg)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()

	c, err3 := w.GetFollow(80)
	if err3 != nil {
		t.Fatalf("err3 = %v", err3)
	}
	defer c.Close()
	ch := c.EntryCh()

	go func() {
		for {
			select {
			case e := <-ch:
				if e == nil {
					plog.Infof("follow channel fail = %v", c.Error())
					return
				}
				plog.Infof("result = %v, %v", e.Index, string(e.Data))
			}
		}
	}()

	time.Sleep(time.Second)

	sz := 10
	ents := make([]Entry, 0)
	for i := 0; i < sz; i++ {
		s := fmt.Sprintf("hello world2 %v", i)
		ents = append(ents, Entry{Index: 0, Data: []byte(s)})
	}
	rsc := w.Append(ents, true)
	rs := <-rsc

	if rs.Err != nil {
		t.Fatalf("err4 = %v", rs.Err)
	}

	time.Sleep(2 * time.Second)

	// <-w.Reset()

	c.Close()

	time.Sleep(time.Second)
}
