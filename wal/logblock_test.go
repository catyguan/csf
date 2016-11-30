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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

var (
	testDir       string = "c:\\tmp"
	testBlockSize uint64 = 16 * 1024
)

func TestLogCoder(t *testing.T) {
	if t != nil {
		lh1 := &logHeader{
			Index: 1,
			Crc:   5,
		}
		lh1.SetMetaData([]byte("hello world"))
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
		li1 := &logIndex{
			Index: 1,
			Crc:   5,
			Size:  6,
			Pos:   7,
		}
		if li1 != nil {
			buf := bytes.NewBuffer(make([]byte, 0))
			lih := &logCoder{}

			err := lih.WriteIndex(buf, li1)
			if err != nil {
				t.Fatalf("err1 = %v", err)
			}
			errq := lih.WriteQIndex(buf, li1)
			if errq != nil {
				t.Fatalf("err1q = %v", errq)
			}

			// lih = &logCoder{}
			li2, err2 := lih.ReadIndex(buf)
			if err2 != nil {
				t.Fatalf("err2 = %v", err2)
			}
			liq := &logIndex{}
			err2q := lih.ReadQIndexTo(buf, liq)
			if err2q != nil {
				t.Fatalf("err2q = %v", err2q)
			}
			plog.Infof("result1 = %v, %v", li2.String(), liq)
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
		lr1 := &logTail{
			Size: 123,
		}
		if lr1 != nil {
			buf := bytes.NewBuffer(make([]byte, 0))
			lih := &logCoder{}

			err := lih.WriteTail(buf, lr1)
			if err != nil {
				t.Fatalf("h err1 = %v", err)
			}

			// lih = &logCoder{}
			lr2, err2 := lih.ReadTail(buf)
			if err2 != nil {
				t.Fatalf("h err2 = %v", err2)
			}
			plog.Infof("RT result1 = %v", lr2.String())
		}
	}
}

func TestLogBlockBase(t *testing.T) {
	p := filepath.Join(testDir, "waltest2")
	os.RemoveAll(p)
	os.MkdirAll(p, os.ModePerm)

	lb := newFirstBlock(p)
	if lb != nil {
		err := lb.CreateT("hello world")
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		lb.Close()
	}

	lb2 := newFirstBlock(p)
	err2 := lb2.OpenWrite()
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer lb2.Close()
}

func TestLogBlockRemove(t *testing.T) {
	p := filepath.Join(testDir, "waltest2")
	os.RemoveAll(p)
	os.MkdirAll(p, os.ModePerm)

	lb := newFirstBlock(p)
	err := lb.CreateT("hello world")
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
	p := filepath.Join(testDir, "waltest2")
	os.RemoveAll(p)
	os.MkdirAll(p, os.ModePerm)

	lb := newFirstBlock(p)
	err := lb.CreateT("hello world")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	defer lb.Close()
	doTestLogBlockAppend(lb, t)
}

func TestLogBlockAppend2(t *testing.T) {
	p := filepath.Join(testDir, "waltest2")

	lb2 := newFirstBlock(p)
	err2 := lb2.OpenWrite()
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer lb2.Close()
	doTestLogBlockAppend(lb2, t)
}

func doTestLogBlockAppend(lb *logBlock, t *testing.T) {
	if lb != nil {
		sz := 100
		for i := 1; i < sz; i++ {
			li, err4 := lb.Append(0, []byte(fmt.Sprintf("hello world %v", i)))
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
	p := filepath.Join(testDir, "waltest")
	bid := 0

	fname := filepath.Join(p, blockName(uint64(bid)))
	f, err := os.OpenFile(fname, os.O_RDONLY, 0666)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	defer f.Close()

	lih := &logCoder{}

	lh, err2 := lih.ReadHeader(f)
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	plog.Infof("header = %v", lh.String())
	//f.Seek(int64(lh.Size()), os.SEEK_SET)
	lih.ReadTail(f)

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
			li.Index = lli.Index + 1
			li.Pos = lli.EndPos()
		} else {
			li.Index = lh.Index + 1
			li.Pos = uint64(lh.Size())
		}
		lli = li
		plog.Infof("index = %v", li.String())
	}
}

func TestLogBlockDump(t *testing.T) {
	p := filepath.Join(testDir, "waltest2")
	bid := 0

	lb2 := newFirstBlock(p)
	lb2.id = bid
	err2 := lb2.ReadHeader()
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer lb2.Close()

	if lb2 != nil {
		err3 := lb2.ReadAll(func(li *logIndex, data []byte) (bool, error) {
			plog.Infof("result3 = %v, %v", li.String(), data)
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

	lb2 := newFirstBlock(p)
	err2 := lb2.ReadHeader()
	if err2 != nil {
		t.Fatalf("err2 = %v", err2)
	}
	defer lb2.Close()

	_, err3 := lb2.Truncate(180)
	if err3 != nil && err3 != io.EOF {
		t.Fatalf("err3 = %v", err3)
	}
	if lb2 != nil {
		plog.Infof("Block = %v", lb2.String())
	}
}
