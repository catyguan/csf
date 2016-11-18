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
	"io"
)

const sizeofLogHead = 3*8 + 4

type logHeader struct {
	ClusterId uint64 // 64
	Index     uint64 // 64, 上一个Block的最后的Index, 本Block的FirstIndex = Index + 1
	Term      uint64 // 64, 上一个Block的最后的Term
	Crc       uint32 // 32, 上一个Block的最后的Crc
}

func (this *logHeader) FirstIndex() uint64 {
	return this.Index + 1
}

func (this *logCoder) WriteHeader(w io.Writer, lh *logHeader) error {
	buf := make([]byte, sizeofLogHead)
	binary.LittleEndian.PutUint64(buf[0:], lh.ClusterId)
	binary.LittleEndian.PutUint64(buf[8:], lh.Index)
	binary.LittleEndian.PutUint64(buf[16:], lh.Term)
	binary.LittleEndian.PutUint32(buf[24:], lh.Crc)
	_, err := w.Write(buf)
	return err
}

func (this *logCoder) ReadHeader(r io.Reader) (*logHeader, error) {
	buf := make([]byte, sizeofLogHead)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}
	if n != sizeofLogHead {
		return nil, io.ErrUnexpectedEOF
	}

	lh := &logHeader{}
	lh.ClusterId = binary.LittleEndian.Uint64(buf[0:])
	lh.Index = binary.LittleEndian.Uint64(buf[8:])
	lh.Term = binary.LittleEndian.Uint64(buf[16:])
	lh.Crc = binary.LittleEndian.Uint32(buf[24:])

	return lh, nil
}
