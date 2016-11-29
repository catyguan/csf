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
	"time"

	"github.com/catyguan/csf/pkg/fileutil"
	"github.com/catyguan/csf/pkg/pbutil"
	"github.com/catyguan/csf/raft/raftpb"
)

type recoderHandler func(li *logIndex, data []byte) (stop bool, err error)

type logBlock struct {
	logHeader
	id    int
	dir   string
	Tail  *logIndex
	wFile *os.File
}

func (this *logBlock) String() string {
	s := ""
	s += fmt.Sprintf("id: %d,", this.id)
	s += fmt.Sprintf("header: %s,", &this.logHeader)
	s += fmt.Sprintf("tail: %s", this.Tail)
	return s
}

func newFirstBlock(dir string, clusterId uint64) *logBlock {
	this := &logBlock{
		id:  0,
		dir: dir,
	}
	this.ClusterId = clusterId
	return this
}

func (this *logBlock) newNextBlock() *logBlock {
	t := this.Tail
	r := &logBlock{
		id:  this.id + 1,
		dir: this.dir,
	}
	r.ClusterId = this.ClusterId
	if t != nil {
		r.Index = t.Index
		r.Term = t.Term
		r.Crc = t.Crc
	} else {
		r.Index = this.Index
		r.Term = this.Term
		r.Crc = this.Crc
	}
	return r
}

func debugNewLogBlock(dir string, clusterId uint64, id int, idx, term uint64, crc uint32) *logBlock {
	r := &logBlock{
		id:  id,
		dir: dir,
	}
	r.ClusterId = clusterId
	r.Index = idx
	r.Term = term
	r.Crc = crc
	return r
}

func (this *logBlock) LastIndex() uint64 {
	if this.Tail == nil {
		return this.Index
	}
	return this.Tail.Index
}

func (this *logBlock) BlockFilePath() string {
	return filepath.Join(this.dir, blockName(uint64(this.id)))
}

func (this *logBlock) IsExists() bool {
	return ExistFile(this.BlockFilePath())
}

func (this *logBlock) Create() error {
	wfn := this.BlockFilePath()

	if ExistFile(wfn) {
		plog.Warningf("create WAL block fail - file(%v) exist", wfn)
		return os.ErrExist
	}

	err := allocFileSize(this.dir, wfn, SegmentSizeBytes)
	if err != nil {
		plog.Warningf("alloc WAL block fail - %v", err)
		return err
	}

	f, err := os.OpenFile(wfn, os.O_WRONLY, fileutil.PrivateFileMode)
	if err != nil {
		return err
	}
	lih := createLogCoder(0)
	err2 := lih.WriteHeader(f, &this.logHeader)
	if err2 != nil {
		f.Close()
		return err2
	}
	this.wFile = f
	return nil
}

func (this *logBlock) InitWrite(st *raftpb.HardState, cst *raftpb.ConfState) error {
	if this.Tail != nil {
		plog.Panicf("block[%v] already active, can't InitWrite", this.id)
	}
	bs := pbutil.MustMarshal(st)
	_, err := this.doAppend(this.Index, this.Term, uint8(stateType), bs)
	if err != nil {
		return err
	}
	bs = pbutil.MustMarshal(cst)
	_, err = this.doAppend(this.Index, this.Term, uint8(confStateType), bs)
	if err != nil {
		return err
	}
	return nil
}

func (this *logBlock) CreateAndInit(st *raftpb.HardState, cst *raftpb.ConfState) error {
	err := this.Create()
	if err != nil {
		return err
	}
	return this.InitWrite(st, cst)
}

func (this *logBlock) CreateEmpty() error {
	return this.CreateAndInit(&raftpb.HardState{}, &raftpb.ConfState{})
}

func (this *logBlock) Open() error {
	wfn := this.BlockFilePath()

	f, err := os.OpenFile(wfn, os.O_RDONLY, fileutil.PrivateFileMode)
	if err != nil {
		return err
	}
	defer f.Close()

	lih := createLogCoder(0)
	lh, err2 := lih.ReadHeader(f)
	if err2 != nil {
		if err2 == io.EOF {
			return ErrLogHeaderError
		}
		return err2
	}
	if lh.ClusterId != this.ClusterId {
		return fmt.Errorf("WAL block clusterId fail, got(%v) want(%v)", lh.ClusterId, this.ClusterId)
	}
	this.logHeader = *lh

	return nil
}

func (this *logBlock) writesFile() (*os.File, error) {
	if this.wFile != nil {
		return this.wFile, nil
	}
	f, err := os.OpenFile(this.BlockFilePath(), os.O_WRONLY, fileutil.PrivateFileMode)
	if err != nil {
		return nil, err
	}
	this.wFile = f
	off := int64(sizeofLogHead)
	if this.Tail != nil {
		off = int64(this.Tail.EndPos())
	}
	_, err = f.Seek(off, os.SEEK_SET)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (this *logBlock) closeWriteFile(f *os.File, err error) {
	if err != nil {
		f.Close()
		this.wFile = nil
	}
}

func (this *logBlock) readFile() (*os.File, error) {
	rf, err := os.OpenFile(this.BlockFilePath(), os.O_RDONLY, fileutil.PrivateFileMode)
	if err != nil {
		return nil, err
	}
	return rf, nil
}

func (this *logBlock) closeReadFile(f *os.File, err error) {
	f.Close()
}

func (this *logBlock) Active(h recoderHandler) (err error) {
	if h == nil && this.Tail != nil {
		return nil
	}
	f, err2 := this.readFile()
	if err2 != nil {
		return err2
	}
	defer func() {
		this.closeReadFile(f, err)
	}()

	this.Tail = &logIndex{}
	_, err = this.doProcess(f, 0xFFFFFFFF, func(li *logIndex, data []byte) (bool, error) {
		*this.Tail = *li
		if h != nil {
			_, err3 := h(li, data)
			if err3 != nil {
				return false, err3
			}
		}
		return false, nil
	})
	if err != nil && err != io.EOF {
		return fmt.Errorf("block[%v] active fail - %v", this.id, err)
	}
	if this.Tail.Empty() {
		plog.Panicf("block[%v] no record, invalid format", this.id)
	}
	return nil
}

func (this *logBlock) Process(eidx uint64, h recoderHandler) (nstop bool, err error) {
	rf, err2 := this.readFile()
	if err2 != nil {
		return false, err2
	}
	defer func() {
		this.closeReadFile(rf, err)
	}()
	return this.doProcess(rf, eidx, h)
}

func (this *logBlock) doProcess(f *os.File, eidx uint64, h recoderHandler) (nstop bool, err error) {
	off := uint64(sizeofLogHead)
	lcrc := this.Crc

	_, err = f.Seek(int64(off), os.SEEK_SET)
	if err != nil {
		return false, err
	}

	var lli, li logIndex
	var buf []byte

	lih := createLogCoder(lcrc)
	r := bufio.NewReader(f)
	e := raftpb.Entry{}
	// plog.Infof("doProcess(%v, %v) at %v, %v", sidx, eidx, off, lid)
	for {
		buf, err = lih.ReadRecordToBuf(r, buf, &li)
		if err != nil {
			return false, err
		}
		if li.Empty() {
			return true, nil
		}
		relbuf := buf[:li.Size]
		if li.Type == uint8(entryType) {
			e.Reset()
			pbutil.MustUnmarshal(&e, relbuf)
			li.Index = e.Index
			li.Term = e.Term
		} else {
			if lli.Empty() {
				li.Index = this.Index
				li.Term = this.Term
			} else {
				li.Index = lli.Index
				li.Term = lli.Term
			}
		}
		if li.Index >= eidx {
			return true, nil
		}
		if lli.Empty() {
			li.Pos = uint64(off)
		} else {
			li.Pos = lli.EndPos()
		}
		lli = li
		stop, err2 := h(&li, relbuf)
		if err2 != nil {
			return false, err2
		}
		if stop {
			return true, nil
		}
	}
	return false, nil
}

func (this *logBlock) createLogIndex(idx uint64, term uint64, typ uint8) *logIndex {
	r := new(logIndex)
	r.Type = typ
	if this.Tail != nil {
		if idx == 0 {
			r.Index = this.Tail.Index
		} else {
			r.Index = idx
		}
		if term == 0 {
			r.Term = this.Tail.Term
		} else {
			r.Term = term
		}
		r.Pos = this.Tail.EndPos()
	} else {
		if idx == 0 {
			r.Index = this.Index
		} else {
			r.Index = idx
		}
		if term == 0 {
			r.Term = this.Term
		} else {
			r.Term = term
		}
		r.Pos = sizeofLogHead
	}
	return r
}

func (this *logBlock) Append(idx uint64, term uint64, typ uint8, data []byte) (*logIndex, error) {
	if this.Tail == nil {
		plog.Panicf("block[%v] not active, can't Append", this.id)
	}
	return this.doAppend(idx, term, typ, data)
}

func (this *logBlock) doAppend(idx uint64, term uint64, typ uint8, data []byte) (li *logIndex, err error) {
	f, err2 := this.writesFile()
	if err2 != nil {
		plog.Warningf("block[%v].doAppend() open write file fail - %v", this.id, err)
		return nil, err2
	}
	defer func() {
		this.closeWriteFile(f, err)
	}()
	if typ == 0 {
		typ = 0xFF
	}
	li = this.createLogIndex(idx, term, typ)
	crc := this.Crc
	if this.Tail != nil {
		crc = this.Tail.Crc
	}
	lc := createLogCoder(crc)
	err = lc.WriteRecord(f, li, data)
	if err != nil {
		plog.Warningf("block[%v].doAppend() write record file fail - %v", this.id, err)
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

func (this *logBlock) sync() error {
	if this.wFile == nil {
		return nil
	}

	start := time.Now()
	err := fileutil.Fdatasync(this.wFile)

	duration := time.Since(start)
	if duration > warnSyncDuration {
		plog.Warningf("sync duration of %v, expected less than %v", duration, warnSyncDuration)
	}
	syncDurations.Observe(duration.Seconds())

	return err
}

func (this *logBlock) Remove() error {
	this.Close()
	return os.Remove(this.BlockFilePath())
}

func (this *logBlock) Truncate(idx uint64) error {
	rf, err2 := this.readFile()
	if err2 != nil {
		plog.Warningf("block[%v].Truncate() open read file fail - %v", this.id, err2)
		return err2
	}
	defer this.closeReadFile(rf, nil)

	lli := logIndex{}
	tli := logIndex{}
	_, err := this.doProcess(rf, 0xFFFFFFFF, func(li *logIndex, data []byte) (bool, error) {
		if li.Index >= idx {
			tli = lli
			return true, nil
		}
		lli = *li
		return false, nil
	})
	if err != nil && err != io.EOF {
		return err
	}
	pos := uint64(sizeofLogHead)
	if !tli.Empty() {
		pos = tli.EndPos()
	}
	plog.Infof("block[%v] truncate to [%v]", this.id, pos)
	wf, err3 := this.writesFile()
	if err3 != nil {
		plog.Warningf("block[%v].Truncate() open write file fail - %v", this.id, err3)
		return err3
	}
	defer this.closeWriteFile(wf, nil)

	_, err = wf.Seek(int64(pos), os.SEEK_SET)
	if err != nil {
		return err
	}
	err = fileutil.ZeroToEnd(wf)
	if err != nil {
		return err
	}
	this.Tail = &tli
	return nil
}
