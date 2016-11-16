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
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/catyguan/csf/wal/walpb"
)

var (
	testDir string = "c:\\tmp"
)

func TestAlloc(t *testing.T) {
	fn := path.Join(testDir, "alloc.dat")
	os.Remove(fn)
	err := allocFileSize(testDir, fn, 1024*1024)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
}

func TestBase(t *testing.T) {
	SegmentSizeBytes = 16 * 1024

	p := filepath.Join(testDir, "waltest")
	os.Remove(p)

	w, err2 := InitWAL(p, &walpb.Metadata{})
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	w.Close()

	time.Sleep(time.Second)

	// w, err := Open(p)
	// if err != nil {
	// 	t.Fatalf("err = %v, want nil", err)
	// }
	// defer w.Close()

	// plog.Infof("names = %v", w.names)

	// a, b, c, err3 := w.InitRead(&walpb.Snapshot{})
	// plog.Infof("InitRead - %v, %v, %v", a, b, c)
	// if err3 != nil {
	// 	t.Fatalf("err3 = %v", err3)
	// }
}
