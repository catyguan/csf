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
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testDir = "c:\\tmp"
)

func TestSaveAndLoad(t *testing.T) {
	dir := path.Join(testDir, "snapshot")
	os.RemoveAll(dir)
	os.MkdirAll(dir, 0700)

	data := "hello world"
	sh := &SnapHeader{
		Index: 1,
		Meta:  []byte("version 123"),
	}

	ss := NewSnapshotter(dir)
	err := ss.SaveSnap(sh, []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	n, lr, err := ss.LoadLastHeader()
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	assert.NotNil(t, lr)
	if lr == nil {
		return
	}
	plog.Infof("RT = %v", lr.String())
	_, g, err2 := ss.LoadSnapFile(n)
	if err2 != nil {
		t.Fatalf("err = %v, want nil", err2)
	}
	assert.Equal(t, string(g), data)
}

func TestFailback(t *testing.T) {
	dir := path.Join(testDir, "snapshot")
	os.RemoveAll(dir)
	os.MkdirAll(dir, 0700)

	large := fmt.Sprintf("%016x-%016x-%016x.snap", 0xFFFF, 0xFFFF, 0xFFFF)
	err := ioutil.WriteFile(path.Join(dir, large), []byte("bad data"), 0666)
	if err != nil {
		t.Fatal(err)
	}

	data := "hello world"
	ss := NewSnapshotter(dir)
	err = ss.SaveSnap(&SnapHeader{Index: 1}, []byte(data))
	if err != nil {
		t.Fatal(err)
	}

	n, _, err := ss.LoadLastHeader()
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	_, g, err2 := ss.LoadSnapFile(n)
	if err2 != nil {
		t.Errorf("err = %v, want nil", err2)
	}
	if string(g) != data {
		t.Errorf("snap = %#v, want %#v", g, data)
	}
	if f, err := os.Open(path.Join(dir, large) + ".broken"); err != nil {
		t.Fatal("broken snapshot does not exist")
	} else {
		f.Close()
	}
}

func TestSnapNames(t *testing.T) {
	dir := path.Join(testDir, "snapshot")
	os.RemoveAll(dir)
	os.MkdirAll(dir, 0700)

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
	os.RemoveAll(dir)
	os.MkdirAll(dir, 0700)

	ss := NewSnapshotter(dir)
	err := ss.SaveSnap(&SnapHeader{Index: 1}, []byte("111"))
	if err != nil {
		t.Fatal(err)
	}

	err = ss.SaveSnap(&SnapHeader{Index: 5}, []byte("555"))
	if err != nil {
		t.Fatal(err)
	}

	n, _, err := ss.LoadLastHeader()
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	_, g, err2 := ss.LoadSnapFile(n)
	if err2 != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if string(g) != "555" {
		t.Errorf("snap = %#v, want %#v", string(g), "555")
	}
}

func TestNoSnapshot(t *testing.T) {
	dir := path.Join(testDir, "snapshot")
	os.RemoveAll(dir)
	os.MkdirAll(dir, 0700)

	ss := NewSnapshotter(dir)
	_, _, err := ss.LoadLastHeader()
	if err != nil {
		t.Errorf("err = %v, want %v", err, ErrNoSnapshot)
	}
}

// TestAllSnapshotBroken ensures snapshotter returns
// ErrNoSnapshot if all the snapshots are broken.
func TestAllSnapshotBroken(t *testing.T) {
	dir := path.Join(testDir, "snapshot")
	os.RemoveAll(dir)
	os.Mkdir(dir, 0700)

	err := ioutil.WriteFile(path.Join(dir, "1.snap"), []byte("bad"), 0x700)
	if err != nil {
		t.Fatal(err)
	}

	ss := NewSnapshotter(dir)
	_, _, err = ss.LoadLastHeader()
	if err != ErrNoSnapshot {
		t.Errorf("err = %v, want %v", err, ErrNoSnapshot)
	}
}
