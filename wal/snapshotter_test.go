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
	"hash/crc32"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/catyguan/csf/raft/raftpb"
)

var testSnap = &raftpb.Snapshot{
	Data: []byte("some snapshot"),
	Metadata: raftpb.SnapshotMetadata{
		ConfState: raftpb.ConfState{
			Nodes: []uint64{1, 2, 3},
		},
		Index: 1,
		Term:  1,
	},
}

func TestSaveAndLoad(t *testing.T) {
	dir := path.Join(testDir, "snapshot")
	os.MkdirAll(dir, 0700)
	defer os.RemoveAll(dir)

	ss := NewSnapshotter(dir)
	err := ss.SaveSnap(*testSnap)
	if err != nil {
		t.Fatal(err)
	}

	name, meta, err := ss.LoadLastMetadata()
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	plog.Infof("meta = %v", meta.String())
	g, err2 := ss.LoadSnap(name)
	if err2 != nil {
		t.Fatalf("err = %v, want nil", err2)
	}
	if !reflect.DeepEqual(g, testSnap) {
		t.Errorf("snap = %#v, want %#v", g, testSnap)
	}
}

func TestBadCRC(t *testing.T) {
	dir := path.Join(testDir, "snapshot")
	os.MkdirAll(dir, 0700)
	defer os.RemoveAll(dir)

	ss := NewSnapshotter(dir)
	err := ss.SaveSnap(*testSnap)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { crcTable = crc32.MakeTable(crc32.Castagnoli) }()
	// switch to use another crc table
	// fake a crc mismatch
	crcTable = crc32.MakeTable(crc32.Koopman)

	_, err = ss.LoadSnap(fmt.Sprintf("%016x.snap", 1))
	if err == nil || err != ErrCRCMismatch {
		t.Errorf("err = %v, want %v", err, ErrCRCMismatch)
	}
}

func TestFailback(t *testing.T) {
	dir := path.Join(testDir, "snapshot")
	os.MkdirAll(dir, 0700)
	defer os.RemoveAll(dir)

	large := fmt.Sprintf("%016x-%016x-%016x.snap", 0xFFFF, 0xFFFF, 0xFFFF)
	err := ioutil.WriteFile(path.Join(dir, large), []byte("bad data"), 0666)
	if err != nil {
		t.Fatal(err)
	}

	ss := NewSnapshotter(dir)
	err = ss.SaveSnap(*testSnap)
	if err != nil {
		t.Fatal(err)
	}

	n, _, err := ss.LoadLastMetadata()
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	g, err2 := ss.LoadSnap(n)
	if err2 != nil {
		t.Errorf("err = %v, want nil", err2)
	}
	if !reflect.DeepEqual(g, testSnap) {
		t.Errorf("snap = %#v, want %#v", g, testSnap)
	}
	if f, err := os.Open(path.Join(dir, large) + ".broken"); err != nil {
		t.Fatal("broken snapshot does not exist")
	} else {
		f.Close()
	}
}

func TestSnapNames(t *testing.T) {
	dir := path.Join(testDir, "snapshot")
	os.MkdirAll(dir, 0700)
	defer os.RemoveAll(dir)
	var err error

	for i := 1; i <= 5; i++ {
		var f *os.File
		if f, err = os.Create(path.Join(dir, fmt.Sprintf("%d.snap", i))); err != nil {
			t.Fatal(err)
		} else {
			f.Close()
		}
	}
	ss := NewSnapshotter(dir)
	names, err := ss.snapNames()
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if len(names) != 5 {
		t.Errorf("len = %d, want 10", len(names))
	}
	w := []string{"5.snap", "4.snap", "3.snap", "2.snap", "1.snap"}
	if !reflect.DeepEqual(names, w) {
		t.Errorf("names = %v, want %v", names, w)
	}
}

func TestLoadNewestSnap(t *testing.T) {
	dir := path.Join(testDir, "snapshot")
	os.MkdirAll(dir, 0700)
	defer os.RemoveAll(dir)
	ss := NewSnapshotter(dir)
	err := ss.SaveSnap(*testSnap)
	if err != nil {
		t.Fatal(err)
	}

	newSnap := *testSnap
	newSnap.Metadata.Index = 5
	err = ss.SaveSnap(newSnap)
	if err != nil {
		t.Fatal(err)
	}

	n, _, err := ss.LoadLastMetadata()
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	g, err2 := ss.LoadSnap(n)
	if err2 != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if !reflect.DeepEqual(g, &newSnap) {
		t.Errorf("snap = %#v, want %#v", g, &newSnap)
	}
}

func TestNoSnapshot(t *testing.T) {
	dir := path.Join(testDir, "snapshot")
	os.MkdirAll(dir, 0700)
	defer os.RemoveAll(dir)

	ss := NewSnapshotter(dir)
	_, _, err := ss.LoadLastMetadata()
	if err != nil {
		t.Errorf("err = %v, want %v", err, ErrNoSnapshot)
	}
}

// TestAllSnapshotBroken ensures snapshotter returns
// ErrNoSnapshot if all the snapshots are broken.
func TestAllSnapshotBroken(t *testing.T) {
	dir := path.Join(testDir, "snapshot")
	os.Mkdir(dir, 0700)
	defer os.RemoveAll(dir)

	err := ioutil.WriteFile(path.Join(dir, "1.snap"), []byte("bad"), 0x700)
	if err != nil {
		t.Fatal(err)
	}

	ss := NewSnapshotter(dir)
	_, _, err = ss.LoadLastMetadata()
	if err != ErrNoSnapshot {
		t.Errorf("err = %v, want %v", err, ErrNoSnapshot)
	}
}
