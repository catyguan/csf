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

// Package raftservertc defines a test app for raftserver.
package main

import (
	"fmt"
	"log"

	"github.com/catyguan/csf/raft/raftpb"
)

type trSetting struct {
	disconn bool
}

func (this *trSetting) Valid() bool {
	return true
}

func trkey(from, to uint64) string {
	return fmt.Sprintf("%v-%v", from, to)
}

type network struct {
	trs map[string]*trSetting
	vms map[uint64]*vm
}

func newNetwork() *network {
	r := new(network)
	r.trs = make(map[string]*trSetting)
	r.vms = make(map[uint64]*vm)
	return r
}

func (this *network) VM(id uint64) *vm {
	vm, ok := this.vms[id]
	if ok {
		return vm
	}
	return nil
}

func (this *network) Add(n *vm) {
	if _, ok := this.vms[n.id]; ok {
		return
	}
	n.nw = this
	this.vms[n.id] = n
}

func (this *network) SendMessage(from uint64, msg raftpb.Message) {
	ts, ok := this.trs[trkey(from, msg.To)]
	if ok {
		if ts.disconn {
			return
		}
	}
	go func() {
		for _, s := range this.vms {
			if s.server != nil && s.id == msg.To {
				s.server.OnRecvRaftRPC(msg)
			}
		}
	}()
}

func (this *network) StartAll() error {
	for _, vm := range this.vms {
		err := vm.Start()
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *network) StopAll() {
	for _, vm := range this.vms {
		vm.Stop()
	}
}

func (this *network) DumpStatus() {
	log.Printf("-------------------------------------")
	for _, n := range this.vms {
		log.Printf("DUMP[%v] - %v", n.id, n.Dump())
	}
}

func (this *network) sureTRS(from, to uint64) *trSetting {
	k1 := trkey(from, to)
	ts, ok := this.trs[k1]
	if !ok {
		ts = new(trSetting)
		this.trs[k1] = ts
	}
	return ts
}

func (this *network) DisableLog(idexp uint64) {
	for _, vm := range this.vms {
		if vm.id != idexp {
			vm.EnableLog(false)
		}
	}
}

func (this *network) FindLeader() []*vm {
	r := make([]*vm, 0)
	for _, vm := range this.vms {
		if vm.server != nil {
			if vm.server.LeaderId() == vm.id {
				r = append(r, vm)
			}
		}
	}
	return r
}

func (this *network) PickOther(id uint64) *vm {
	for _, vm := range this.vms {
		if vm.server != nil && vm.id != id {
			return vm
		}
	}
	return nil
}

func (this *network) TRConnect(from, to uint64, conn bool) {
	ts := this.sureTRS(from, to)
	ts.disconn = !conn
}

func (this *network) ConnBreak(id uint64, conn bool) {
	for _, vm := range this.vms {
		if vm.id == id {
			continue
		}
		this.TRConnect(vm.id, id, conn)
		this.TRConnect(id, vm.id, conn)
	}
}
