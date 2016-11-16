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
	"github.com/catyguan/csf/pkg/ioutil"
	"github.com/catyguan/csf/pkg/pbutil"
	"github.com/catyguan/csf/wal/walpb"
)

type logBlock struct {
	Index   uint64
	dir     string
	meta    *walpb.Metadata
	Tail    *logIndex
	wFile   *os.File
	rFile   *os.File
	bw      *ioutil.PageWriter
	rFiles  []*os.File
	prevCrc uint32
}

func newLogBlock(idx uint64, dir string, meta *walpb.Metadata) *logBlock {
	this := &logBlock{
		Index: idx,
		dir:   dir,
		meta:  meta,
	}
	return this
}

func (this *logBlock) WalFilePath() string {
	return filepath.Join(this.dir, walName(this.Index))
}

func (this *logBlock) Metadata() *walpb.Metadata {
	return this.meta
}

func (this *logBlock) Open() error {
	wfn := this.WalFilePath()

	f, err := os.OpenFile(wfn, os.O_RDONLY, fileutil.PrivateFileMode)
	if err != nil {
		return err
	}
	defer f.Close()

	lih := createLogCoder(0)
	li, b, err2 := lih.ReadRecord(f)
	if err2 != nil {
		if err2 == io.EOF {
			return ErrLogIndexError
		}
		return err2
	}
	if li.Empty() {
		return ErrLogIndexError
	}
	if li.Type != uint8(metadataType) {
		return ErrMetadataError
	}
	meta := &walpb.Metadata{}
	if err := meta.Unmarshal(b); err != nil {
		return err
	}
	if meta.ClusterID != this.meta.ClusterID {
		return fmt.Errorf("WAL metadata clusterId fail, got(%v) want(%v)", meta.ClusterID, this.meta.ClusterID)
	}
	if meta.NodeID != this.meta.NodeID {
		plog.Warningf("WAL metadata nodeId got(%v) want(%v)", meta.NodeID, this.meta.NodeID)
	}
	this.meta = meta

	return nil
}

func (this *logBlock) Create() error {
	wfn := this.WalFilePath()

	if ExistFile(wfn) {
		plog.Warningf("create WAL file fail - file(%v) exist", wfn)
		return os.ErrExist
	}

	err := allocFileSize(this.dir, wfn, SegmentSizeBytes)
	if err != nil {
		plog.Warningf("alloc WAL file fail - %v", err)
		return err
	}

	f, err := os.OpenFile(wfn, os.O_WRONLY, fileutil.PrivateFileMode)
	if err != nil {
		return err
	}
	defer f.Close()

	b := pbutil.MustMarshal(this.meta)

	lih := createLogCoder(0)
	li := createLogIndex(this.Index, 0, uint8(metadataType))
	err2 := lih.WriteRecord(f, li, b)
	if err2 != nil {
		return err2
	}

	return nil
}

func (this *logBlock) readFile() (*os.File, error) {
	if this.rFile != nil {
		return this.rFile, nil
	}
	rf, err := os.OpenFile(this.WalFilePath(), os.O_RDONLY, fileutil.PrivateFileMode)
	if err != nil {
		return nil, err
	}
	this.rFile = rf
	this.rFiles = append(this.rFiles, rf)
	return rf, nil
}

func (this *logBlock) SeekToEnd() (*logIndex, uint64, error) {
	return this.Seek(0xFFFFFFFF)
}

// 定位具体index的位置，返回对应logIndex的文件位置
func (this *logBlock) Seek(idx uint64) (*logIndex, uint64, error) {

	rf, err := this.readFile()
	if err != nil {
		return nil, 0, err
	}
	_, err = rf.Seek(0, os.SEEK_SET)
	if err != nil {
		return nil, 0, err
	}

	var pos uint64
	var lli, li *logIndex
	var buf []byte

	lih := createLogCoder(0)
	r := bufio.NewReader(rf)
	for {
		if li != nil {
			pos = pos + sizeofLogIndex + uint64(li.Size)
		}
		plog.Infof("here")
		li, buf, err = lih.ReadRecordToBuf(r, buf)
		if err != nil {
			return lli, pos, err
		}
		lli = li

		if li.Index >= idx {
			break
		}
	}
	return lli, pos, nil
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
