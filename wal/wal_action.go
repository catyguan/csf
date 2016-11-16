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
	"fmt"

	"github.com/catyguan/csf/pkg/fileutil"
)

func (this *WAL) doClear() {
	for _, lb := range this.blocks {
		lb.Close()
	}
	this.blocks = make([]*logBlock, 0)
}

func (this *WAL) doRun(ac chan *actionOfWAL) {
	defer this.doClear()
	for {
		// pick
		select {
		case act := <-ac:
			if act == nil {
				plog.Infof("WAL worker stop")
				return
			}
			switch act.action {
			default:
				this.handleSomeAction(act)
			}
		}
	}
}

func (this *WAL) handleSomeAction(act *actionOfWAL) {
	switch act.action {
	case "init":
		err := this.doInit()
		if act.respC != nil {
			act.respC <- &respOfWAL{err: err}
		}
	default:
		plog.Warningf("unknow WAL action - %v", act.action)
	}
}

func (this *WAL) callAction(act *actionOfWAL) (interface{}, error) {
	act.respC = make(chan *respOfWAL, 1)
	if this.actionC != nil {
		this.actionC <- act
		r := <-act.respC
		return r.answer, r.err
	} else {
		return nil, fmt.Errorf("WAL closed")
	}
}

func (this *WAL) doInit() error {
	var names []string
	if !Exist(this.dir) {
		plog.Infof("init WAL at %v", this.dir)
		err := fileutil.CreateDirAll(this.dir)
		if err != nil {
			plog.Warningf("init WAL fail - %v", err)
			return err
		}
	} else {
		names2, err := readWalNames(this.dir)
		if err != nil {
			return err
		}
		names = names2
	}
	if len(names) == 0 {
		// init first wal block
		plog.Infof("init WAL block [0]")
		lb := newLogBlock(0, this.dir, this.meta)
		err := lb.Create()
		if err != nil {
			plog.Warningf("init WAL block [0] fail - %v", err)
		}
		this.doAppendBlock(lb)
	} else {
		for _, n := range names {
			idx, err2 := parseWalName(n)
			if err2 != nil {
				plog.Warningf("WAL skip file '%v'", n)
				continue
			}
			lb := newLogBlock(idx, this.dir, this.meta)
			err := lb.Open()
			if err != nil {
				plog.Warningf("open WAL block[%v] fail - %v", idx, err)
				return err
			}
			this.doAppendBlock(lb)
		}
	}

	return nil
}

func (this *WAL) doAppendBlock(lb *logBlock) {
	this.blocks = append(this.blocks, lb)
}

func (this *WAL) doAppendRecord() {

}
