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
package core

import "strings"

var (
	locBuilders = make(map[string]ServiceLocationBuilder)
)

func RegisterServiceLocationBuilder(typ string, lb ServiceLocationBuilder) {
	locBuilders[typ] = lb
}

type ServiceLocation struct {
	Invoker     ServiceInvoker
	ServiceName string
	Params      map[string]string
}

func ParseLocation(s string) (*ServiceLocation, error) {
	ps := strings.SplitN(s, ":", 2)
	typ := ps[0]
	content := ""
	if len(ps) > 1 {
		content = ps[1]
	}
	lb, ok := locBuilders[typ]
	if !ok {
		plog.Panicf("ServiceLocationBuilder[%v] not found", typ)
	}

	r := &ServiceLocation{}
	r.Params = make(map[string]string)
	err := lb(r, typ, content)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (this *ServiceLocation) Copy(sl *ServiceLocation) {
	this.Invoker = sl.Invoker
	this.ServiceName = sl.ServiceName
	this.Params = make(map[string]string)
	for k, v := range sl.Params {
		this.Params[k] = v
	}
}

func (this *ServiceLocation) GetParam(s string) string {
	r, ok := this.Params[s]
	if !ok {
		return ""
	}
	return r
}
