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
	"io"
)

const sizeofLogIndex = 8 /* Index */ + 4 /* Size */ + 4 /* Crc */
const sizeOfQIndex = 8 /* Index */ + 8                  /* Pos */

type logIndex struct {
	Index uint64 // 64, log Index
	Size  uint32 // 32, data size
	Crc   uint32 // 32
	Pos   uint64 // do not save
}

func (this *logIndex) Empty() bool {
	return this.Index == 0
}

func (this *logIndex) Copy() *logIndex {
	r := &logIndex{}
	*r = *this
	return r
}

func (this *logIndex) String() string {
	bs, _ := json.Marshal(this)
	return fmt.Sprintf("%s", string(bs))
}

func (this *logIndex) Validate(crc uint32) error {
	if this.Crc == crc {
		return nil
	}
	return ErrCRCMismatch
}

func (this *logIndex) EndPos() uint64 {
	return this.Pos + uint64(sizeofLogIndex) + uint64(this.Size)
}

func (this *logIndex) DataSize() uint32 {
	return this.Size
}

func (this *logCoder) WriteIndex(w io.Writer, li *logIndex) error {
	if this.buf == nil {
		this.buf = make([]byte, sizeofLogIndex)
	}
	buf := this.buf
	binary.LittleEndian.PutUint64(buf[0:], li.Index)
	binary.LittleEndian.PutUint32(buf[8:], li.Size)
	binary.LittleEndian.PutUint32(buf[12:], li.Crc)

	_, err := w.Write(buf)
	return err
}

func (this *logCoder) WriteQIndex(w io.Writer, li *logIndex) error {
	if this.buf == nil {
		this.buf = make([]byte, sizeOfQIndex)
	}
	buf := this.buf
	binary.LittleEndian.PutUint64(buf[0:], li.Index)
	binary.LittleEndian.PutUint64(buf[8:], li.Pos)

	_, err := w.Write(buf)
	return err
}

func (this *logCoder) WriteRecord(w io.Writer, li *logIndex, b []byte) error {
	sz := len(b)
	li.Size = uint32(sz)
	if this.crc != nil {
		this.crc.Write(b)
		li.Crc = this.crc.Sum32()
	}
	err := this.WriteIndex(w, li)
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

func (this *logCoder) ReadIndex(r io.Reader) (*logIndex, error) {
	li := &logIndex{}
	err := this.ReadIndexTo(r, li)
	if err != nil {
		return nil, err
	}
	return li, nil
}

func (this *logCoder) ReadIndexTo(r io.Reader, li *logIndex) error {
	if this.buf == nil {
		this.buf = make([]byte, sizeofLogIndex)
	}
	buf := this.buf
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return err
	}
	if n != sizeofLogIndex {
		return io.ErrUnexpectedEOF
	}

	li.Index = binary.LittleEndian.Uint64(buf[0:])
	li.Size = binary.LittleEndian.Uint32(buf[8:])
	li.Crc = binary.LittleEndian.Uint32(buf[12:])

	if li.Empty() {
		return io.EOF
	}

	return nil
}

func (this *logCoder) ReadQIndexTo(r io.Reader, li *logIndex) error {
	if this.buf == nil {
		this.buf = make([]byte, sizeOfQIndex)
	}
	buf := this.buf
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return err
	}
	if n != sizeOfQIndex {
		return io.ErrUnexpectedEOF
	}

	li.Index = binary.LittleEndian.Uint64(buf[0:])
	li.Pos = binary.LittleEndian.Uint64(buf[8:])

	return nil
}

func (this *logCoder) ReadRecord(r io.Reader) (*logIndex, []byte, error) {
	li := &logIndex{}
	b, err := this.ReadRecordToBuf(r, nil, li)
	return li, b, err
}

func (this *logCoder) ReadRecordToBuf(r io.Reader, b []byte, li *logIndex) ([]byte, error) {
	err := this.ReadIndexTo(r, li)
	if err != nil {
		return nil, err
	}
	sz := li.Size
	if sz > 0 {
		var b2 []byte
		if b == nil || uint32(len(b)) < sz {
			b = make([]byte, sz)
			b2 = b
		} else {
			b2 = b[:sz]
		}
		n, err2 := io.ReadFull(r, b2)
		if err2 != nil {
			return nil, err2
		}
		if uint32(n) != sz {
			return nil, io.ErrUnexpectedEOF
		}
		// crc
		if this.crc != nil {
			this.crc.Write(b2)
			if li.Crc != this.crc.Sum32() {
				return nil, ErrCRCMismatch
			}
		}
	}
	return b, nil
}
