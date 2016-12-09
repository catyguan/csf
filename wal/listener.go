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

import "github.com/catyguan/csf/pkg/capnslog"

type LogListener struct {
	name string
	l    *capnslog.PackageLogger
}

func NewLogListener(n string, l *capnslog.PackageLogger) WALListener {
	if n == "" {
		n = "WAL"
	}
	r := &LogListener{
		name: n,
		l:    l,
	}
	return r
}

func (this *LogListener) OnReset() {
	this.l.Infof("%v OnReset", this.name)
}

func (this *LogListener) OnTruncate(idx uint64) {
	this.l.Infof("%v OnTruncate(%v)", this.name, idx)
}

func (this *LogListener) OnAppendEntry(ents []Entry) {
	this.l.Infof("%v OnAppendEntry(%v)", this.name, ents)
}

func (this *LogListener) OnClose() {
	this.l.Infof("%v OnClose()", this.name)
}
