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
