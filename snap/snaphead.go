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
	"errors"
	"fmt"
	"io"
)

var (
	ErrSnapshotMismatch = errors.New("snapshot mismatch")
	ErrSnapshotNotFound = errors.New("snapshot not found")
	sizeOfSnapHeader    = 8 /* Index */ + 4 /* MetaLen */ + 8 /* DataSize */
)

/* [Index:8, MetaLen:4, DatSize:8], [Meta:N, Data:M] */
type SnapHeader struct {
	Index    uint64
	Meta     []byte
	DataSize uint64
}

func (this *SnapHeader) String() string {
	s := ""
	s += fmt.Sprintf("Index: %v, ", this.Index)
	s += fmt.Sprintf("MetaLen: %v, ", len(this.Meta))
	s += fmt.Sprintf("DataSize: %v, ", this.DataSize)

	return s
}

func (this *SnapHeader) Write(w io.Writer) error {
	bm := make([]byte, sizeOfSnapHeader)
	binary.LittleEndian.PutUint64(bm, this.Index)
	binary.LittleEndian.PutUint32(bm[8:], uint32(len(this.Meta)))
	binary.LittleEndian.PutUint64(bm[12:], this.DataSize)

	_, err := w.Write(bm)
	if err != nil {
		return err
	}
	if this.Meta != nil {
		_, err = w.Write(this.Meta)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *SnapHeader) Read(r io.Reader) error {
	buf := make([]byte, sizeOfSnapHeader)
	n, err := io.ReadFull(r, buf)
	if n != sizeOfSnapHeader {
		return io.ErrUnexpectedEOF
	}
	this.Index = binary.LittleEndian.Uint64(buf[0:])
	ms := binary.LittleEndian.Uint32(buf[8:])
	this.DataSize = binary.LittleEndian.Uint64(buf[12:])
	if ms > 0 {
		b := make([]byte, ms)
		_, err = io.ReadFull(r, b)
		if err != nil {
			return err
		}
		this.Meta = b
	} else {
		this.Meta = nil
	}
	return nil
}
