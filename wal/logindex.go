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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash"
	"io"

	"github.com/catyguan/csf/pkg/crc"
)

const sizeofLogIndex = 8

type logIndex struct {
	Index uint64 // do not save
	Term  uint64 // do not save
	Type  uint8  // 8
	Size  uint32 // 24
	Crc   uint32 // 32
	Pos   uint64 // do not save
}

func (this *logIndex) Copy() *logIndex {
	r := &logIndex{}
	*r = *this
	return r
}

func (this *logIndex) String() string {
	bs, _ := json.Marshal(this)
	return fmt.Sprintf("%s%s", logTypeString(this.Type), string(bs))
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
	binary.LittleEndian.PutUint32(buf[0:], li.Size)
	buf[3] = li.Type
	binary.LittleEndian.PutUint32(buf[4:], li.Crc)

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
	li := &logIndex{}
	err := this.ReadIndex(r, li)
	if err != nil {
		return nil, err
	}
	return li, nil
}

func (this *logCoder) ReadIndex(r io.Reader, li *logIndex) error {
	li.Type = 0
	if this.buf == nil {
		this.buf = make([]byte, sizeofLogIndex)
	}
	buf := this.buf
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return err
	}
	if n != sizeofLogIndex {
		// if n == 0 {
		// 	return nil, io.EOF
		// }
		return io.ErrUnexpectedEOF
	}

	li.Size = binary.LittleEndian.Uint32(buf[0:]) & 0x00FFFFFF
	li.Type = buf[3]
	li.Crc = binary.LittleEndian.Uint32(buf[4:])

	return nil
}

func (this *logCoder) ReadRecord(r io.Reader) (*logIndex, []byte, error) {
	li := &logIndex{}
	b, err := this.ReadRecordToBuf(r, nil, li)
	return li, b, err
}

func (this *logCoder) ReadRecordToBuf(r io.Reader, b []byte, li *logIndex) ([]byte, error) {
	err := this.ReadIndex(r, li)
	if err != nil {
		return nil, err
	}
	if li.Empty() {
		return nil, io.EOF
	}
	if li.Size == 0 {
		return nil, nil
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
		return nil, err2
	}
	if uint32(n) != li.Size {
		return nil, io.ErrUnexpectedEOF
	}
	// crc
	if this.crc != nil {
		this.crc.Write(b2)
		if li.Crc != this.crc.Sum32() {
			return nil, ErrCRCMismatch
		}
	}
	return b, nil
}
