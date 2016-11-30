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

const sizeofLogTail = 4 /* Size */

type logTail struct {
	Size uint32 // 32bit, 上一个logdata的大小，包含logIndex
}

func (this *logTail) String() string {
	bs, _ := json.Marshal(this)
	return string(bs)
}

func (this *logCoder) WriteTail(w io.Writer, lt *logTail) error {
	buf := make([]byte, sizeofLogTail)
	binary.LittleEndian.PutUint32(buf[0:], lt.Size)
	_, err := w.Write(buf)
	return err
}

func (this *logCoder) WriteTailSize(w io.Writer, sz uint32) error {
	buf := make([]byte, sizeofLogTail)
	binary.LittleEndian.PutUint32(buf[0:], sz)
	_, err := w.Write(buf)
	return err
}

func (this *logCoder) ReadTailTo(r io.Reader, lt *logTail) error {
	buf := make([]byte, sizeofLogTail)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return err
	}
	if n != sizeofLogTail {
		return io.ErrUnexpectedEOF
	}

	lt.Size = binary.LittleEndian.Uint32(buf[0:])

	return nil
}

func (this *logCoder) ReadTail(r io.Reader) (*logTail, error) {
	lt := &logTail{}
	err := this.ReadTailTo(r, lt)
	if err != nil {
		return nil, err
	}
	return lt, nil
}
