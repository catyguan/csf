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
	"bufio"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"os"

	"github.com/catyguan/csf/pkg/crc"
)

const sizeofLogIndex = 3 * 8

type logIndex struct {
	Index uint64 // 64
	Term  uint64 // 64
	Size  uint32 // 24
	Type  uint8  // 8
	Crc   uint32 // 32
	Pos   uint64 // do not save
	Id    int32  // do not save
}

func (this *logIndex) Empty() bool {
	return this.Type == 0
}

func (this *logIndex) Validate(crc uint32) error {
	if this.Crc == crc {
		return nil
	}
	return ErrCRCMismatch
}

func (this *logIndex) EndPos() uint64 {
	return this.Pos + uint64(this.Size) + sizeofLogIndex
}

type logCoder struct {
	crc hash.Hash32
	buf []byte
}

func createLogCoder(prevCrc uint32) *logCoder {
	return &logCoder{crc: crc.New(prevCrc, crcTable)}
}

func (this *logCoder) Write(w io.Writer, li *logIndex) error {
	if this.buf == nil {
		this.buf = make([]byte, sizeofLogIndex)
	}
	buf := this.buf
	binary.LittleEndian.PutUint64(buf[0:], li.Index)
	binary.LittleEndian.PutUint64(buf[8:], li.Term)
	binary.LittleEndian.PutUint32(buf[16:], li.Size)
	buf[19] = li.Type
	binary.LittleEndian.PutUint32(buf[20:], li.Crc)

	_, err := w.Write(buf)
	return err
}

func (this *logCoder) WriteRecord(w io.Writer, li *logIndex, b []byte) error {
	li.Size = uint32(len(b))
	if this.crc != nil {
		this.crc.Write(b)
		li.Crc = this.crc.Sum32()
	}
	err := this.Write(w, li)
	if err != nil {
		return err
	}
	n, err2 := w.Write(b)
	if err2 != nil {
		return err2
	}
	if n != len(b) {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func (this *logCoder) Read(r io.Reader) (*logIndex, error) {
	if this.buf == nil {
		this.buf = make([]byte, sizeofLogIndex)
	}
	buf := this.buf
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}
	if n != sizeofLogIndex {
		// if n == 0 {
		// 	return nil, io.EOF
		// }
		return nil, io.ErrUnexpectedEOF
	}

	li := &logIndex{}
	li.Index = binary.LittleEndian.Uint64(buf[0:])
	li.Term = binary.LittleEndian.Uint64(buf[8:])
	li.Size = binary.LittleEndian.Uint32(buf[16:]) & 0x00FFFFFF
	li.Type = buf[19]
	li.Crc = binary.LittleEndian.Uint32(buf[20:])

	return li, nil
}

func (this *logCoder) ReadRecord(r io.Reader) (*logIndex, []byte, error) {
	return this.ReadRecordToBuf(r, nil)
}

func (this *logCoder) ReadRecordToBuf(r io.Reader, b []byte) (*logIndex, []byte, error) {
	li, err := this.Read(r)
	if err != nil {
		return nil, nil, err
	}
	if li.Empty() {
		return nil, nil, io.EOF
	}
	if li.Size == 0 {
		return li, nil, nil
	}
	var b2 []byte
	if b == nil || uint32(len(b)) < li.Size {
		b = make([]byte, li.Size)
		b2 = b
	} else {
		b2 = b[:li.Size]
	}
	n, err2 := io.ReadFull(r, b2)
	if err2 != nil {
		return nil, nil, err2
	}
	if uint32(n) != li.Size {
		return nil, nil, io.ErrUnexpectedEOF
	}
	// crc
	if this.crc != nil {
		this.crc.Write(b2)
		if li.Crc != this.crc.Sum32() {
			return nil, nil, ErrCRCMismatch
		}
	}
	return li, b, nil
}

func (this *logCoder) PickEntry(f *os.File, index uint64) (*logIndex, error) {
	if f != nil {
		return nil, fmt.Errorf("not impl")
	}
	r := bufio.NewReader(f)
	for {
		li, err := this.Read(r)
		if err != nil {
			if err == io.EOF {
				return nil, nil
			}
			return nil, err
		}
		if li.Type == uint8(entryType) && li.Index >= index {
			return li, nil
		}
	}
}
