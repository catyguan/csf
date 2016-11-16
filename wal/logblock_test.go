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
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/catyguan/csf/wal/walpb"
)

func TestLogCoder(t *testing.T) {
	li1 := &logIndex{
		Index: 1,
		Term:  2,
		Type:  7,
		Crc:   5,
		Size:  6,
	}
	if li1 != nil {
		buf := bytes.NewBuffer(make([]byte, 0))
		lih := &logCoder{}

		err := lih.Write(buf, li1)
		if err != nil {
			t.Fatalf("err1 = %v", err)
		}

		// lih = &logCoder{}
		li2, err2 := lih.Read(buf)
		if err2 != nil {
			t.Fatalf("err2 = %v", err2)
		}
		bs, _ := json.Marshal(li2)
		plog.Infof("result1 = %v", string(bs))
	}

	if li1 != nil {
		buf := bytes.NewBuffer(make([]byte, 0))
		lih := createLogCoder(0)

		err := lih.WriteRecord(buf, li1, []byte("hello world"))
		if err != nil {
			t.Fatalf("err1 = %v", err)
		}

		lih2 := createLogCoder(0)
		li2, b, err2 := lih2.ReadRecord(buf)
		if err2 != nil {
			t.Fatalf("err2 = %v", err2)
		}
		bs, _ := json.Marshal(li2)
		plog.Infof("result2 = %v, %v", string(bs), string(b))
	}

}

func TestLogBlockBase(t *testing.T) {
	SegmentSizeBytes = 16 * 1024

	p := filepath.Join(testDir, "waltest2")
	os.RemoveAll(p)
	os.MkdirAll(p, os.ModePerm)

	meta := &walpb.Metadata{}
	bid := uint64(10)

	lb := newLogBlock(bid, p, meta)
	err := lb.Create()
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	lb.Close()

	lb2 := newLogBlock(bid, p, meta)
	err2 := lb2.Open()
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer lb2.Close()

	if lb2 != nil {
		li, pos, err3 := lb2.Seek(1)
		if err3 != nil && err3 != io.EOF {
			t.Fatalf("err3 = %v", err3)
		}
		bs, _ := json.Marshal(li)
		plog.Infof("result3 = %v, %v", string(bs), pos)
	}

}
