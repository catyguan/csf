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
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/catyguan/csf/raft/raftpb"
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

func TestWALBase(t *testing.T) {
	SegmentSizeBytes = 16 * 1024

	p := filepath.Join(testDir, "waltest")
	// os.Remove(p)

	w, _, err2 := InitWAL(p, 100)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	w.Close()

	time.Sleep(time.Second)
}

func TestWALSave(t *testing.T) {
	SegmentSizeBytes = 4 * 1024

	p := filepath.Join(testDir, "waltest")
	// os.RemoveAll(p)

	w, _, err2 := InitWAL(p, 100)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()

	st := raftpb.HardState{Term: 1}
	ents := make([]raftpb.Entry, 0)
	si := 6
	sz := 10
	for i := 0; i < sz; i++ {
		v := uint64(si + i)
		ents = append(ents, raftpb.Entry{Index: v, Term: v, Data: []byte("hello world")})
	}

	err3 := w.doSave(&st, ents)
	if err3 != nil {
		t.Fatalf("err3 = %v", err3)
	}

	time.Sleep(time.Second)
}

func TestWALDump(t *testing.T) {
	SegmentSizeBytes = 16 * 1024

	p := filepath.Join(testDir, "waltest")
	cid := uint64(100)

	w, _, err2 := InitWAL(p, cid)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()

	time.Sleep(time.Second)

	err3 := w.processRecord(0, 0xFFFFFFFF, func(li *logIndex, data []byte) (bool, error) {
		bs, _ := json.Marshal(li)
		plog.Infof("result3 = %v, %v", string(bs), data)
		return false, nil
	})
	if err3 != nil {
		t.Fatalf("err3 = %v", err3)
	}
}

func TestWALTerm(t *testing.T) {
	SegmentSizeBytes = 16 * 1024

	p := filepath.Join(testDir, "waltest")
	cid := uint64(100)

	w, _, err2 := InitWAL(p, cid)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()

	time.Sleep(time.Second)

	v, err3 := w.Term(14)
	if err3 != nil {
		t.Fatalf("err3 = %v", err3)
	}
	plog.Infof("result3 = %v", v)

}

func TestWALEntries(t *testing.T) {
	SegmentSizeBytes = 16 * 1024

	p := filepath.Join(testDir, "waltest")
	cid := uint64(100)

	w, _, err2 := InitWAL(p, cid)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer w.Close()

	time.Sleep(time.Second)

	v, err3 := w.Entries(3, 7, 1000*1000)
	if err3 != nil {
		t.Fatalf("err3 = %v", err3)
	}
	plog.Infof("result3 = %v", v)

}
