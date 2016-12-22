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

package masterslave

import (
	"context"
	"time"

	"github.com/catyguan/csf/pkg/runobj"
)

type slaveEP struct {
	apply  SlaveApply
	master MasterAPI
	cfg    *SlaveConfig

	sid uint64
	ro  *runobj.RunObj
}

func newSlaveEP(cfg *SlaveConfig) *slaveEP {
	r := new(slaveEP)
	r.apply = cfg.Apply
	r.master = cfg.Master
	r.cfg = cfg
	r.ro = runobj.NewRunObj(cfg.QueueSize)
	return r
}

func (this *slaveEP) Run() error {
	if this.master == nil {
		plog.Fatalf("master nil")
	}
	if this.apply == nil {
		plog.Fatalf("apply nil")
	}
	return this.ro.Run(this.doRun, nil)
}

func (this *slaveEP) IsClosed() bool {
	return this.ro.IsClosed()
}

func (this *slaveEP) Close() {
	this.ro.Close()
}

func (this *slaveEP) doRun(ready chan error, ach <-chan *runobj.ActionRequest, p interface{}) {
	close(ready)
	defer func() {
		if this.sid > 0 {
			ctx := context.Background()
			this.master.End(ctx, this.sid)
		}
	}()

	stc := make(chan int, 1)
	stc <- 0

	lerr := false
	for {
		select {
		case a := <-ach:
			if a == nil {
				return
			}
			r, err := this.doAction(a)
			if a.Resp != nil {
				a.Resp <- &runobj.ActionResponse{R1: r, Err: err}
			}
		case st := <-stc:
			nt, err := this.doFollow(st)
			if err != nil {
				if !lerr {
					plog.Warningf("slave action fail - %v", err)
					lerr = true
					// lerr = false
				}
				time.AfterFunc(this.cfg.RetryTime, func() {
					stc <- 0
				})
			} else {
				lerr = false
				stc <- nt
			}
		}

	}
}

func (this *slaveEP) doAction(a *runobj.ActionRequest) (interface{}, error) {
	return nil, nil
}

func (this *slaveEP) doReset() {
	if this.sid == 0 {
		return
	}
	plog.Infof("slave[%d] end", this.sid)
	ctx := context.Background()
	this.master.End(ctx, this.sid)
	this.sid = 0
}

func (this *slaveEP) doFollow(step int) (next int, errR error) {
	ctx := context.Background()
	defer func() {
		if errR != nil {
			this.doReset()
		}
	}()
	nctx, cancel := context.WithTimeout(ctx, this.cfg.ExecuteTimeout)
	tm := time.AfterFunc(this.cfg.ExecuteTimeout, func() {
		cancel()
	})
	defer tm.Stop()

	switch step {
	case 0:
		sid, err := this.master.Begin(nctx)
		if err != nil {
			return 0, err
		}
		this.sid = sid
		if sid > 0 {
			plog.Infof("slave[%d] begin success", sid)
			return 1, nil
		}
		return 0, nil
	case 1:
		lidx, snapshot, err2 := this.master.LastSnapshot(nctx, this.sid)
		if err2 != nil {
			return 0, err2
		}
		err3 := this.apply.ApplySnapshot(lidx, snapshot)
		if err3 != nil {
			plog.Infof("slave[%d] apply snapshot(%d) fail - %v", this.sid, lidx, err3)
			return 0, err3
		}
		plog.Infof("slave[%d] apply snapshot(%d) success ", this.sid, lidx)
		return 2, nil
	case 2:
		rlist, err2 := this.master.Process(nctx, this.sid)
		if err2 != nil {
			return 0, err2
		}
		err3 := this.apply.ApplyRequests(rlist)
		if err3 != nil {
			plog.Infof("slave[%d] apply requests(%d) fail - %v", this.sid, len(rlist), err3)
			return 0, err3
		}
		if len(rlist) > 0 {
			plog.Infof("slave[%d] apply requests(%d) success ", this.sid, len(rlist))
		}
		return 2, nil
	}

	plog.Fatalf("unknow step %v", step)
	return 0, nil
}
