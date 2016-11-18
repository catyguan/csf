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
			bs, _ := json.Marshal(lh2)
			plog.Infof("h result1 = %v", string(bs))
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
	err := lb.Create()
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

	lb2.Active()
}

func TestLogBlockRemove(t *testing.T) {
	SegmentSizeBytes = 16 * 1024

	p := filepath.Join(testDir, "waltest2")
	os.RemoveAll(p)
	os.MkdirAll(p, os.ModePerm)

	cid := uint64(100)

	lb := newFirstBlock(p, cid)
	err := lb.Create()
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
	err := lb.Create()
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	defer lb.Close()
	if lb != nil {
		for i := 1; i < 100; i++ {
			v := uint64(i * 10)
			li, err4 := lb.Append(v, v, 5, []byte("hello world"))
			if err4 != nil {
				t.Fatalf("err4 = %v", err4)
			}
			if li != nil {
				// bs, _ := json.Marshal(li)
				// plog.Infof("Append Result = %v", string(bs))
			}
		}
	}
}

func TestLogBlockSeek(t *testing.T) {
	SegmentSizeBytes = 16 * 1024

	p := filepath.Join(testDir, "waltest2")

	cid := uint64(100)

	lb2 := newFirstBlock(p, cid)
	err2 := lb2.Open()
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer lb2.Close()

	if lb2 != nil {
		li, err3 := lb2.Seek(780 - 1)
		if err3 != nil && err3 != io.EOF {
			t.Fatalf("err3 = %v", err3)
		}
		bs, _ := json.Marshal(li)
		plog.Infof("result3 = %v", string(bs))
	}

}

func TestLogBlockQDump(t *testing.T) {
	p := filepath.Join(testDir, "wal3")
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
		bs, _ := json.Marshal(lh)
		plog.Infof("header = %v", string(bs))
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
		bs, _ := json.Marshal(li)
		plog.Infof("index = %v", string(bs))
	}
}

func TestLogBlockDump(t *testing.T) {
	SegmentSizeBytes = 16 * 1024

	p := filepath.Join(testDir, "waltest")

	cid := uint64(100)

	lb2 := newFirstBlock(p, cid)
	lb2.id = 1
	err2 := lb2.Open()
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer lb2.Close()

	if lb2 != nil {
		err4 := lb2.Active()
		if err4 != nil {
			t.Fatalf("err4 = %v", err4)
		}

		_, err3 := lb2.Process(0, 0xFFFFFFFF, func(li *logIndex, data []byte) (bool, error) {
			bs, _ := json.Marshal(li)
			plog.Infof("result3 = %v, %v", string(bs), data)
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
		bs, _ := json.Marshal(lb2.Tail)
		plog.Infof("Block Tail = %v", string(bs))
		plog.Infof("Block SeekIndexs = %v", lb2.SeekIndexs)
	}
}
