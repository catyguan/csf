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
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/catyguan/csf/pkg/fileutil"
)

type RecoderHandler func(li *logIndex, data []byte) (stop bool, err error)

type logBlock struct {
	Header logHeader
	id     int
	dir    string
	Tail   *logIndex
	wFile  *os.File
}

func (this *logBlock) String() string {
	s := ""
	s += fmt.Sprintf("id: %d,", this.id)
	s += fmt.Sprintf("header: %s,", &this.Header)
	s += fmt.Sprintf("tail: %s", this.Tail)
	return s
}

func newFirstBlock(dir string) *logBlock {
	this := &logBlock{
		id:  0,
		dir: dir,
	}
	this.Header.Version = WALVersion
	return this
}

func (this *logBlock) newNextBlock() *logBlock {
	t := this.Tail
	r := &logBlock{
		id:  this.id + 1,
		dir: this.dir,
	}
	r.Header.Version = this.Header.Version
	r.Header.SetMetaData(this.Header.MetaData)
	if t != nil {
		r.Header.Index = t.Index
		r.Header.Crc = t.Crc
	} else {
		r.Header.Index = this.Header.Index
		r.Header.Crc = this.Header.Crc
	}
	return r
}

func (this *logBlock) LastIndex() uint64 {
	if this.Tail == nil {
		return this.Header.Index
	}
	return this.Tail.Index
}

func (this *logBlock) BlockFilePath() string {
	return filepath.Join(this.dir, blockName(uint64(this.id)))
}

func (this *logBlock) IsExistsNextBlock() bool {
	fn := filepath.Join(this.dir, blockName(uint64(this.id+1)))
	return ExistFile(fn)
}

func (this *logBlock) IsExists() bool {
	return ExistFile(this.BlockFilePath())
}

func (this *logBlock) CreateT(s string) error {
	return this.Create([]byte(s))
}

func (this *logBlock) Create(metadata []byte) error {
	wfn := this.BlockFilePath()

	if ExistFile(wfn) {
		plog.Warningf("create WAL block fail - file(%v) exist", wfn)
		return os.ErrExist
	}

	f, err := os.OpenFile(wfn, os.O_RDWR|os.O_CREATE, fileutil.PrivateFileMode)
	if err != nil {
		return err
	}
	lih := createLogCoder(0)
	this.Header.SetMetaData(metadata)
	err = lih.WriteHeader(f, &this.Header)
	if err != nil {
		f.Close()
		return err
	}
	err = lih.WriteTailSize(f, 0)
	this.wFile = f
	return nil
}

func (this *logBlock) Open(lastBlock bool) error {
	if lastBlock {
		return this.OpenWrite()
	} else {
		return this.ReadHeader()
	}
}

func (this *logBlock) ReadHeader() error {
	wfn := this.BlockFilePath()

	f, err := os.OpenFile(wfn, os.O_RDONLY, fileutil.PrivateFileMode)
	if err != nil {
		return err
	}
	defer f.Close()
	return this.doReadHeader(f)
}

func (this *logBlock) doReadHeader(f *os.File) error {
	lih := &logCoder{}
	lh, err2 := lih.ReadHeader(f)
	if err2 != nil {
		if err2 == io.EOF {
			return ErrLogHeaderError
		}
		return err2
	}
	this.Header = *lh
	return nil
}

func (this *logBlock) OpenWrite() error {
	if this.wFile != nil {
		this.wFile.Close()
		this.wFile = nil
	}
	wfn := this.BlockFilePath()
	f, err := os.OpenFile(wfn, os.O_RDWR, fileutil.PrivateFileMode)
	if err != nil {
		return err
	}
	err = this.doReadHeader(f)
	if err != nil {
		return err
	}
	_, err2 := f.Seek(-int64(sizeofLogTail), os.SEEK_END)
	if err2 != nil {
		f.Close()
		return err2
	}
	lih := &logCoder{}
	lt, err3 := lih.ReadTail(f)
	if err3 != nil {
		f.Close()
		return err3
	}
	if lt.Size != 0 {
		n, err := f.Seek(-int64(sizeofLogTail+lt.Size), os.SEEK_END)
		if err != nil {
			f.Close()
			return err
		}
		this.Tail, err = lih.ReadIndex(f)
		if err != nil {
			f.Close()
			return err
		}
		this.Tail.Pos = uint64(n)
	}
	f.Seek(0, os.SEEK_END)
	this.wFile = f
	return nil
}

func (this *logBlock) readFile() (*os.File, error) {
	rf, err := os.OpenFile(this.BlockFilePath(), os.O_RDONLY, fileutil.PrivateFileMode)
	if err != nil {
		return nil, err
	}
	return rf, nil
}

func (this *logBlock) ReadAll(h RecoderHandler) error {
	f, err2 := this.readFile()
	if err2 != nil {
		return err2
	}
	defer f.Close()

	_, err := this.doProcess(f, 0xFFFFFFFF, h)
	return err
}

func (this *logBlock) doSeek(f *os.File, eidx uint64) (*logIndex, error) {
	var lli logIndex
	_, err := this.doProcess(f, eidx+1, func(li *logIndex, data []byte) (bool, error) {
		lli = *li
		return false, nil
	})
	if err != nil && err != io.EOF {
		return nil, err
	}
	if lli.Empty() {
		return nil, nil
	}
	return &lli, nil
}

func (this *logBlock) doProcess(f *os.File, eidx uint64, h RecoderHandler) (nstop bool, err error) {
	off := this.Header.Size()
	lcrc := this.Header.Crc

	_, err = f.Seek(int64(off), os.SEEK_SET)
	if err != nil {
		return false, err
	}

	var lli, li logIndex
	var buf []byte

	lih := createLogCoder(lcrc)
	r := bufio.NewReader(f)
	for {
		buf, err = lih.ReadRecordToBuf(r, buf, &li)
		if err != nil {
			return false, err
		}
		if lli.Empty() {
			li.Pos = uint64(off)
		} else {
			li.Pos = lli.EndPos()
		}
		if li.Index >= eidx {
			return true, nil
		}
		lli = li
		relbuf := buf[:li.Size]
		nstop, err = h(&li, relbuf)
		if err != nil {
			return false, err
		}
		if nstop {
			return true, nil
		}
	}
	return false, nil
}

func (this *logBlock) createLogIndex(idx uint64) *logIndex {
	r := &logIndex{}
	if this.Tail != nil {
		if idx != 0 {
			r.Index = idx
		} else {
			r.Index = this.Tail.Index + 1
		}
		r.Pos = this.Tail.EndPos()
	} else {
		if idx != 0 {
			r.Index = idx
		} else {
			r.Index = this.Header.Index + 1
		}
		r.Pos = uint64(this.Header.Size())
	}
	return r
}

func (this *logBlock) Append(idx uint64, data []byte) (*logIndex, error) {
	if this.wFile == nil {
		plog.Panicf("block[%v] not OpenWrite, can't Append", this.id)
	}
	return this.doAppend(idx, data)
}

func (this *logBlock) doAppend(idx uint64, data []byte) (*logIndex, error) {
	f := this.wFile
	li := this.createLogIndex(idx)
	crc := this.Header.Crc
	if this.Tail != nil {
		crc = this.Tail.Crc
	}
	lc := createLogCoder(crc)
	err := lc.WriteRecord(f, li, data)
	if err != nil {
		plog.Warningf("block[%v].doAppend() write record fail - %v", this.id, err)
		return nil, err
	}
	this.Tail = li
	return li, nil
}

func (this *logBlock) Close() {
	if this.wFile != nil {
		this.wFile.Close()
		this.wFile = nil
	}
}

func (this *logBlock) Sync() error {
	if this.wFile == nil {
		return nil
	}
	err := fileutil.Fdatasync(this.wFile)
	return err
}

func (this *logBlock) Remove() error {
	this.Close()
	return os.Remove(this.BlockFilePath())
}

func (this *logBlock) Truncate(idx uint64) (uint64, error) {
	wfn := this.BlockFilePath()

	f, err := os.OpenFile(wfn, os.O_RDONLY, fileutil.PrivateFileMode)
	if err != nil {
		plog.Warningf("block[%v].Truncate() open file fail - %v", this.id, err)
		return 0, err
	}
	tli := logIndex{}
	_, err = this.doProcess(f, 0xFFFFFFFF, func(li *logIndex, data []byte) (bool, error) {
		if li.Index > idx {
			return true, nil
		}
		tli = *li
		return false, nil
	})
	f.Close()

	if err != nil && err != io.EOF {
		plog.Warningf("block[%v].Truncate() seek %v fail - %v", this.id, idx, err)
		return 0, err
	}
	pos := uint64(this.Header.Size())
	if !tli.Empty() {
		pos = tli.EndPos()
	}

	this.Close()
	plog.Infof("block[%v] Truncate(%v) at [%v]", this.id, idx, pos)
	err = os.Truncate(this.BlockFilePath(), int64(pos))
	if err != nil {
		plog.Warningf("block[%v].Truncate() truncate file fail - %v", this.id, err)
		return 0, err
	}
	if tli.Empty() {
		this.Tail = nil
	} else {
		this.Tail = &tli
	}

	f, err = os.OpenFile(wfn, os.O_RDWR, fileutil.PrivateFileMode)
	if err != nil {
		plog.Warningf("block[%v].Truncate() reopen file fail - %v", this.id, err)
		return 0, err
	}
	_, err = f.Seek(0, os.SEEK_END)
	if err != nil {
		return 0, err
	}
	this.wFile = f

	return this.LastIndex(), nil
}
