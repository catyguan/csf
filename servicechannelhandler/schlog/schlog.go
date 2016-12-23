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
package schlog

import (
	"context"
	"sync/atomic"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/capnslog"
)

type Logger struct {
	plog *capnslog.PackageLogger
	iid  uint64
}

func NewLogger(n string) *Logger {
	r := new(Logger)
	r.plog = capnslog.NewPackageLogger("github.com/catyguan/csf", n)
	return r
}

func (this *Logger) impl() {
	_ = core.ServiceChannelHandler(this)
}

func (this *Logger) HandleRequest(ctx context.Context, si core.ServiceInvoker, creq *corepb.ChannelRequest) (*corepb.ChannelResponse, error) {
	iid := atomic.AddUint64(&this.iid, 1)
	this.plog.Infof("[%d] IN  - %s", iid, creq)
	r, err := si.InvokeRequest(ctx, creq)
	if err != nil {
		this.plog.Infof("[%d] ERR - %v", iid, err)
	} else {
		this.plog.Infof("[%d] OUT - %s", iid, r)
	}
	return r, err
}
