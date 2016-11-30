// Copyright 2016 The CSF Authors
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

// Package interfaces defines write ahead log.
package wal

import (
	"io"
	"os"
)

type cursor struct {
	lb *logBlock
	wc *walcore

	li logIndex
	f  *os.File
	lc *logCoder
}

func newCursor(wc *walcore, lb *logBlock, idx uint64) (*cursor, error) {
	f, err := lb.readFile()
	if err != nil {
		return nil, err
	}
	li, err2 := lb.doSeek(f, idx-1)
	if err2 != nil {
		return nil, err2
	}
	c := &cursor{
		lb: lb,
		wc: wc,
		f:  f,
	}
	off := uint64(lb.Header.Size())
	if li != nil {
		if err != nil {
			return nil, err
		}
		c.li = *li
		off = c.li.EndPos()
	}
	_, err = f.Seek(int64(off), os.SEEK_SET)

	return c, nil
}

func (this *cursor) doClose() {
	if this.f != nil {
		this.f.Close()
		this.f = nil
	}
}

func (this *cursor) Read() (*Entry, error) {
	end, e, err := this.doRead()
	if end {
		this.doClose()
	}
	return e, err
}

func (this *cursor) doRead() (bool, *Entry, error) {
	if this.f == nil {
		return false, nil, io.ErrUnexpectedEOF
	}
	if this.lc == nil {
		crc := this.li.Crc
		if this.li.Empty() {
			crc = this.lb.Header.Crc
		}
		this.lc = createLogCoder(crc)
	}
	data, err := this.lc.ReadRecordToBuf(this.f, nil, &this.li)

	if err != nil {
		if err == io.EOF {
			// read block end
			return true, nil, nil
		}
		return false, nil, err
	}
	e := &Entry{
		Index: this.li.Index,
		Data:  data,
	}
	return false, e, nil
}

func (this *cursor) Close() error {
	this.doClose()
	this.wc.doRemoveCursor(this)
	return nil
}

func (this *cursor) InBlock(lb *logBlock) bool {
	return this.lb == lb || this.lb.id == lb.id
}

func (this *cursor) CloseByWAL() {
	plog.Infof("close by wal!!!")
	this.doClose()
}
