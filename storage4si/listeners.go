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
package storage4si

import (
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/masterslave"
	"github.com/catyguan/csf/pkg/capnslog"
)

type Listeners struct {
	llist []StorageListener
}

func (this *Listeners) same(l1 StorageListener, l2 StorageListener) bool {
	if l1 == l2 {
		return true
	}
	if l1.GetFollower() != nil && l1.GetFollower() == l2.GetFollower() {
		return true
	}
	return false
}

func (this *Listeners) Add(lis StorageListener) {
	for _, l := range this.llist {
		if this.same(l, lis) {
			return
		}
	}
	this.llist = append(this.llist, lis)
}

func (this *Listeners) Remove(lis StorageListener) {
	for i, l := range this.llist {
		if this.same(l, lis) {
			c := len(this.llist)
			if i < c-1 {
				this.llist[i] = this.llist[c-1]
			}
			this.llist = this.llist[:c-1]
			return
		}
	}
}

func (this *Listeners) OnReset() {
	for _, lis := range this.llist {
		lis.OnReset()
	}
}

func (this *Listeners) OnTruncate(idx uint64) {
	for _, lis := range this.llist {
		lis.OnTruncate(idx)
	}
}

func (this *Listeners) OnSaveRequest(idx uint64, req *corepb.Request) {
	for _, lis := range this.llist {
		lis.OnSaveRequest(idx, req)
	}
}

func (this *Listeners) OnSaveSnapshot(idx uint64) {
	for _, lis := range this.llist {
		lis.OnSaveSnapshot(idx)
	}
}

func (this *Listeners) OnClose() {
	for _, lis := range this.llist {
		lis.OnClose()
	}
}

type LogListener struct {
	name string
	l    *capnslog.PackageLogger
}

func NewLogListener(n string, l *capnslog.PackageLogger) StorageListener {
	if n == "" {
		n = "STORAGE"
	}
	r := &LogListener{
		name: n,
		l:    l,
	}
	return r
}

func (this *LogListener) GetFollower() masterslave.MasterFollower {
	return nil
}

func (this *LogListener) OnReset() {
	this.l.Infof("%v OnReset", this.name)
}

func (this *LogListener) OnTruncate(idx uint64) {
	this.l.Infof("%v OnTruncate(%v)", this.name, idx)
}

func (this *LogListener) OnSaveRequest(idx uint64, req *corepb.Request) {
	this.l.Infof("%v OnSaveRequest(%v, %v)", this.name, idx, req)
}

func (this *LogListener) OnSaveSnapshot(idx uint64) {
	this.l.Infof("%v OnSaveSnapshot(%v)", this.name, idx)
}

func (this *LogListener) OnClose() {
	this.l.Infof("%v OnClose()", this.name)
}
