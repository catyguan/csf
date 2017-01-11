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
package schblocker

import (
	"context"
	"fmt"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
)

type rule struct {
	typ  int /* 1-Accept, 2-Refuse, 3-ReadOnly */
	path string
}

type Blocker struct {
	rules []*rule
}

func NewBlocker() *Blocker {
	r := new(Blocker)
	r.rules = make([]*rule, 0)
	return r
}

func (this *Blocker) impl() {
	_ = core.ServiceChannelHandler(this)
}

func (this *Blocker) HandleRequest(ctx context.Context, si core.ServiceInvoker, creq *corepb.ChannelRequest) (*corepb.ChannelResponse, error) {
	ok := false
	for _, ru := range this.rules {
		switch ru.typ {
		case 1:
			if ru.path == "*" {
				ok = true
				break
			} else if ru.path == creq.ServicePath {
				ok = true
				break
			}
		case 2:
			if ru.path == "*" {
				ok = false
				break
			} else if ru.path == creq.ServicePath {
				ok = false
				break
			}
		case 3:
			if creq.IsQueryType() {
				ok = true
			} else {
				ok = false
			}
			break
		}
	}

	if ok {
		return si.InvokeRequest(ctx, creq)
	}
	return nil, fmt.Errorf("can not access '%s:%s'", creq.ServiceName, creq.ServicePath)
}

func (this *Blocker) Accept(p string) *Blocker {
	this.rules = append(this.rules, &rule{typ: 1, path: p})
	return this
}

func (this *Blocker) Refuse(p string) *Blocker {
	this.rules = append(this.rules, &rule{typ: 2, path: p})
	return this
}

func (this *Blocker) ReadOnly() *Blocker {
	this.rules = append(this.rules, &rule{typ: 3})
	return this
}
