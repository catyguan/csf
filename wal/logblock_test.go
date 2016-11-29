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
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/catyguan/csf/raft/raftpb"
)

func TestLogCoder(t *testing.T) {
	if t != nil {
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
			plog.Infof("result1 = %v", li2.String())
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
			plog.Infof("result2 = %v, %v", li2.String(), string(b))
		}
	}
	if t != nil {
		lh1 := &logHeader{
			ClusterId: 100,
			Index:     1,
			Term:      2,
			Crc:       5,
		}
		if lh1 != nil {
			buf := bytes.NewBuffer(make([]byte, 0))
			lih := &logCoder{}

			err := lih.WriteHeader(buf, lh1)
			if err != nil {
				t.Fatalf("h err1 = %v", err)
			}

			// lih = &logCoder{}
			lh2, err2 := lih.ReadHeader(buf)
			if err2 != nil {
				t.Fatalf("h err2 = %v", err2)
			}
			plog.Infof("h result1 = %v", lh2.String())
		}
	}

	if t != nil {
		lr1 := &snapHeader{}
		if lr1 != nil {
			buf := bytes.NewBuffer(make([]byte, 0))
			lih := &logCoder{}

			err := lih.WriteSnap(buf, lr1)
			if err != nil {
				t.Fatalf("h err1 = %v", err)
			}

			// lih = &logCoder{}
			lr2, err2 := lih.ReadSnap(buf)
			if err2 != nil {
				t.Fatalf("h err2 = %v", err2)
			}
			plog.Infof("RT result1 = %v", lr2.String())
		}
	}
}

func TestLogBlockBase(t *testing.T) {
	SegmentSizeBytes = 16 * 1024

	p := filepath.Join(testDir, "waltest2")
	os.RemoveAll(p)
	os.MkdirAll(p, os.ModePerm)

	cid := uint64(100)

	lb := newFirstBlock(p, cid)
	err := lb.CreateEmpty()
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	lb.Close()

	lb2 := newFirstBlock(p, cid)
	err2 := lb2.Open()
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer lb2.Close()

	lb2.Active(nil)
}

func TestLogBlockRemove(t *testing.T) {
	SegmentSizeBytes = 16 * 1024

	p := filepath.Join(testDir, "waltest2")
	os.RemoveAll(p)
	os.MkdirAll(p, os.ModePerm)

	cid := uint64(100)

	lb := newFirstBlock(p, cid)
	err := lb.CreateEmpty()
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	defer lb.Close()

	err2 := lb.Remove()
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
}

func TestLogBlockAppend(t *testing.T) {
	SegmentSizeBytes = 16 * 1024

	p := filepath.Join(testDir, "waltest2")
	os.RemoveAll(p)
	os.MkdirAll(p, os.ModePerm)

	cid := uint64(100)

	lb := newFirstBlock(p, cid)
	err := lb.CreateEmpty()
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	defer lb.Close()
	if lb != nil {
		sz := 100
		for i := 1; i < sz; i++ {
			v := uint64(i * 10)
			li, err4 := lb.Append(v, v, 5, []byte("hello world"))
			if err4 != nil {
				t.Fatalf("err4 = %v", err4)
			}
			if li != nil {
				plog.Infof("Append Result = %v", li.String())
			}
		}
	}
}

func TestLogBlockQDump(t *testing.T) {
	p := filepath.Join(testDir, "waltest2")
	bid := 0

	fname := filepath.Join(p, blockName(uint64(bid)))
	f, err := os.OpenFile(fname, os.O_RDONLY, 0666)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	defer f.Close()

	lih := &logCoder{}
	if f != nil {
		lh, err2 := lih.ReadHeader(f)
		if err2 != nil {
			t.Fatalf("err2 = %v", err2)
		}
		plog.Infof("header = %v", lh.String())
	}
	var lli *logIndex
	for {
		li, _, err3 := lih.ReadRecord(f)
		if err3 != nil {
			if err3 == io.EOF {
				plog.Infof("dump end")
				break
			}
			t.Fatalf("err3 = %v", err3)
		}
		if lli != nil {
			li.Pos = lli.EndPos()
		} else {
			li.Pos = sizeofLogHead
		}
		lli = li
		plog.Infof("index = %v", li.String())
	}
}

func TestLogBlockDump(t *testing.T) {
	SegmentSizeBytes = 16 * 1024

	p := filepath.Join(testDir, "wal3")
	bid := 0

	cid := uint64(1)

	lb2 := newFirstBlock(p, cid)
	lb2.id = bid
	err2 := lb2.Open()
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer lb2.Close()

	if lb2 != nil {
		err3 := lb2.Active(func(li *logIndex, data []byte) (bool, error) {
			var v interface{}
			v = data
			switch int64(li.Type) {
			case entryType:
				e := &raftpb.Entry{}
				e.Unmarshal(data)
				v = e
			case stateType:
				st := &raftpb.HardState{}
				st.Unmarshal(data)
				v = st
			}
			plog.Infof("result3 = %v, %v", li.String(), v)
			return false, nil
		})
		if err3 != nil {
			if err3 != io.EOF {
				t.Fatalf("err3 = %v", err3)
			} else {
				plog.Infof("dump end")
			}
		}
	}

}

func TestLogBlockTruncate(t *testing.T) {
	p := filepath.Join(testDir, "waltest2")
	cid := uint64(100)

	lb2 := newFirstBlock(p, cid)
	err2 := lb2.Open()
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer lb2.Close()

	err3 := lb2.Truncate(800)
	if err3 != nil && err3 != io.EOF {
		t.Fatalf("err3 = %v", err3)
	}
	if lb2 != nil {
		plog.Infof("Block = %v", lb2.String())
	}
}
