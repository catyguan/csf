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
	"io"
)

const sizeofLogHeader = 8 /* Index */ + 4 /* CRC */ + 1 /* Version */ + 2 /* MetaSize */ + 1 /* Padding */

type logHeader struct {
	Index    uint64 // 64, 上一个Block的最后的Index, 本Block的FirstIndex = Index + 1
	Crc      uint32 // 32, 上一个Block的最后的Crc
	Version  uint8  // 8,  编码版本
	MetaSize uint16 // 16, MetaData的大小
	Padding  uint8  // 8,  填充大小
	MetaData []byte // N
}

func (this *logHeader) String() string {
	bs, _ := json.Marshal(this)
	return string(bs)
}

func (this *logHeader) Size() uint16 {
	return sizeofLogHeader + this.MetaSize + uint16(this.Padding) + sizeofLogTail
}

func (this *logHeader) SetMetaData(m []byte) {
	this.MetaData = m
	this.MetaSize = uint16(len(m))
	if this.MetaSize%8 == 0 {
		this.Padding = 0
	} else {
		this.Padding = uint8(8 - this.MetaSize%8)
	}
}

func (this *logCoder) WriteHeader(w io.Writer, lh *logHeader) error {
	buf := make([]byte, sizeofLogHeader)
	binary.LittleEndian.PutUint64(buf[0:], lh.Index)
	binary.LittleEndian.PutUint32(buf[8:], lh.Crc)
	buf[12] = lh.Version
	buf[13] = lh.Padding
	binary.LittleEndian.PutUint16(buf[14:], lh.MetaSize)
	_, err := w.Write(buf)
	if err != nil {
		return err
	}
	w.Write(lh.MetaData)
	if lh.Padding > 0 {
		_, err = w.Write(buf[:lh.Padding])
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *logCoder) ReadHeader(r io.Reader) (*logHeader, error) {
	buf := make([]byte, sizeofLogHeader)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}
	if n != sizeofLogHeader {
		return nil, io.ErrUnexpectedEOF
	}

	lh := &logHeader{}
	lh.Index = binary.LittleEndian.Uint64(buf[0:])
	lh.Crc = binary.LittleEndian.Uint32(buf[8:])
	lh.Version = buf[12]
	lh.Padding = buf[13]
	lh.MetaSize = binary.LittleEndian.Uint16(buf[14:])
	rsz := lh.MetaSize + uint16(lh.Padding)
	data := make([]byte, rsz)
	n, err = io.ReadFull(r, data)
	if err != nil {
		return nil, err
	}
	if uint16(n) != rsz {
		return nil, io.ErrUnexpectedEOF
	}
	lh.MetaData = data[:lh.MetaSize]
	return lh, nil
}
