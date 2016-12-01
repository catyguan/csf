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

	indexTryRead uint64
	li           logIndex
	f            *os.File
	lc           *logCoder
}

func newCursor(wc *walcore, lb *logBlock, idx uint64) (*cursor, error) {
	f, err := lb.readFile()
	if err != nil {
		return nil, err
	}
	li, err2 := lb.doSeek(f, idx-1)
	if err2 != nil {
		f.Close()
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
	if err != nil {
		f.Close()
		return nil, err
	}
	c.indexTryRead = idx
	return c, nil
}

func (this *cursor) doClose() {
	// TODO: multi close not safed
	f := this.f
	if f != nil {
		this.f = nil
		f.Close()
	}
}

func (this *cursor) doRead() (bool, *Entry, error) {
	f := this.f
	if f == nil {
		return false, nil, io.ErrUnexpectedEOF
	}
	if this.indexTryRead > this.wc.LastIndex() {
		return false, nil, nil
	}
	if this.lc == nil {
		crc := this.li.Crc
		if this.li.Empty() {
			crc = this.lb.Header.Crc
		}
		this.lc = createLogCoder(crc)
	}
	data, err := this.lc.ReadRecordToBuf(f, nil, &this.li)

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
	this.indexTryRead = e.Index + 1
	return false, e, nil
}

func (this *cursor) Close() {
	this.wc.doRemoveCursor(this)
	this.doClose()
}

func (this *cursor) walBlock(lb *logBlock) bool {
	return this.lb == lb || this.lb.id == lb.id
}

func (this *cursor) walClose() {
	this.doClose()
}

type cursorImpl struct {
	w *walcore
	c *cursor
}

func (this *cursorImpl) Close() {
	if this.w != nil {
		if this.c != nil {
			this.c.Close()
			this.c = nil
		}
		this.w = nil
	}
}

func (this *cursorImpl) Read() (*Entry, error) {
	if this.w == nil {
		return nil, ErrClosed
	}
	if this.c == nil {
		return nil, nil
	}
	for {
		end, e, err := this.c.doRead()
		if !end {
			return e, err
		}
		idx := this.c.indexTryRead
		this.c.Close()
		this.c = nil

		c, err := this.w.createCursor(idx)
		if err != nil {
			return nil, err
		}
		if c == nil {
			return nil, nil
		}
		this.c = c

	}
}
