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
	"github.com/catyguan/csf/raft/raftpb"
)

type recoderHandler func(li *logIndex, data []byte) (stop bool, err error)

type logBlock struct {
	logHeader
	id         int
	dir        string
	Tail       *logIndex
	SeekIndexs []*logIndex
	wFile      *os.File
	rFile      *os.File
	rFiles     []*os.File
	active     bool
	state      raftpb.HardState
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

func (this *logBlock) LastIndex() uint64 {
	if this.Tail != nil {
		return this.Tail.Index
	}
	return this.Index
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
	this.active = true
	return nil
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

func (this *logBlock) readFile() (*os.File, error) {
	if this.rFile != nil {
		return this.rFile, nil
	}
	rf, err := os.OpenFile(this.BlockFilePath(), os.O_RDONLY, fileutil.PrivateFileMode)
	if err != nil {
		return nil, err
	}
	this.rFile = rf
	this.rFiles = append(this.rFiles, rf)
	return rf, nil
}

func (this *logBlock) findSeekIndex(idx uint64) *logIndex {
	if this.SeekIndexs == nil {
		return nil
	}
	// plog.Infof("Find %v at %v", idx, this.SeekIndexs)
	var lli *logIndex
	for _, si := range this.SeekIndexs {
		if idx <= si.Index {
			return lli
		}
		lli = si
	}
	return lli
}

func (this *logBlock) Seek(idx uint64) (*logIndex, error) {
	var r *logIndex
	_, err := this.Process(idx-1, idx+1, func(li *logIndex, b []byte) (bool, error) {
		r = li
		return false, nil
	})
	return r, err
}

func (this *logBlock) Process(sidx, eidx uint64, h recoderHandler) (nstop bool, err error) {
	err = this.Active()
	if err != nil {
		return false, err
	}
	rf, err2 := this.readFile()
	if err2 != nil {
		return false, err2
	}
	return this.doProcess(rf, sidx, eidx, h)
}

func (this *logBlock) doProcess(f *os.File, sidx, eidx uint64, h recoderHandler) (nstop bool, err error) {
	ski := this.findSeekIndex(sidx)

	off := uint64(sizeofLogHead)
	lid := int32(0)
	lcrc := this.Crc
	if ski != nil {
		off = ski.EndPos()
		lid = ski.Id
		lcrc = ski.Crc
	}
	_, err = f.Seek(int64(off), os.SEEK_SET)
	if err != nil {
		return false, err
	}

	var lli, li *logIndex
	var buf []byte

	lih := createLogCoder(lcrc)
	var r io.Reader
	if f != this.wFile {
		r = bufio.NewReader(f)
	} else {
		r = f
	}
	// plog.Infof("doProcess(%v, %v) at %v, %v", sidx, eidx, off, lid)
	for {
		li, buf, err = lih.ReadRecordToBuf(r, buf)
		if err != nil {
			return false, err
		}
		if li.Empty() {
			return true, nil
		}
		// if li.Index < sidx {
		// 	continue
		// }
		if li.Index >= eidx {
			return true, nil
		}
		if lli != nil {
			li.Pos = lli.EndPos()
			li.Id = lli.Id + 1
		} else {
			li.Pos = uint64(off)
			li.Id = lid
		}
		lli = li
		stop, err2 := h(li, buf[:li.Size])
		if err2 != nil {
			return false, err2
		}
		if stop {
			return true, nil
		}
	}
	return false, nil
}

func (this *logBlock) Active() error {
	if this.active {
		return nil
	}
	f, err := this.readFile()
	if err != nil {
		return err
	}
	_, err = this.doProcess(f, 0, 0xFFFFFFFF, func(li *logIndex, data []byte) (bool, error) {
		this.Tail = li
		if isSeekIndex(li) {
			this.SeekIndexs = append(this.SeekIndexs, li)
		}
		if li.Type == uint8(stateType) {
			st := raftpb.HardState{}
			err := st.Unmarshal(data)
			if err == nil {
				this.state = st
			}
		}
		return false, nil
	})
	if err != nil && err != io.EOF {
		// plog.Panicf("active fail - %v", err)
		return fmt.Errorf("block[%v] active fail - %v", this.id, err)
	}
	this.active = true
	return nil
}

func isSeekIndex(li *logIndex) bool {
	return (li.Id+1)%64 == 0
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
		r.Id = this.Tail.Id + 1
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
		r.Id = 0
	}
	return r
}

func (this *logBlock) AppendState(idx uint64, term uint64, st *raftpb.HardState) (*logIndex, error) {
	b, _ := st.Marshal()
	r, err := this.Append(idx, term, uint8(stateType), b)
	if err != nil {
		return nil, err
	}
	this.state = *st
	return r, err
}

func (this *logBlock) Append(idx uint64, term uint64, typ uint8, data []byte) (*logIndex, error) {
	err := this.Active()
	if err != nil {
		return nil, err
	}
	f, err2 := this.writesFile()
	if err2 != nil {
		return nil, err2
	}
	if typ == 0 {
		typ = 0xFF
	}
	li := this.createLogIndex(idx, term, typ)
	crc := this.Crc
	if this.Tail != nil {
		crc = this.Tail.Crc
	}
	lc := createLogCoder(crc)
	// n, _ := this.wFile.Seek(0, os.SEEK_CUR)
	// plog.Infof("cur2 = %v", n)
	err = lc.WriteRecord(f, li, data)
	if err != nil {
		return nil, err
	}
	if isSeekIndex(li) {
		this.SeekIndexs = append(this.SeekIndexs, li)
	}
	this.Tail = li
	return li, nil
}

func (this *logBlock) Close() {
	if this.wFile != nil {
		this.wFile.Close()
		this.wFile = nil
	}
	for _, rfile := range this.rFiles {
		rfile.Close()
	}
	this.rFile = nil
	this.rFiles = nil
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
	err := this.Active()
	if err != nil {
		return err
	}
	rf, err2 := this.readFile()
	if err2 != nil {
		return err2
	}
	var nseeks []*logIndex
	var lli *logIndex
	var tli *logIndex
	_, err = this.doProcess(rf, 0, 0xFFFFFFFF, func(li *logIndex, data []byte) (bool, error) {
		if li.Index >= idx {
			tli = lli
			return true, nil
		}
		lli = li
		if isSeekIndex(li) {
			nseeks = append(nseeks, li)
		}
		return false, nil
	})
	if err != nil && err != io.EOF {
		return err
	}
	pos := uint64(sizeofLogHead)
	if tli != nil {
		pos = tli.EndPos()
	}
	plog.Infof("block[%v] truncate to [%v]", this.id, pos)
	wf, err3 := this.writesFile()
	if err3 != nil {
		return err3
	}
	_, err = wf.Seek(int64(pos), os.SEEK_SET)
	if err != nil {
		return err
	}
	err = fileutil.ZeroToEnd(this.wFile)
	if err != nil {
		return err
	}
	this.SeekIndexs = nseeks
	this.Tail = tli
	this.state.Reset()
	return nil
}
