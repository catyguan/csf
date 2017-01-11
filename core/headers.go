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

import "github.com/catyguan/csf/core/corepb"

var (
	HEADER_REMOTE_ADDR = "REMOTE_ADDR"
	HEADER_ERROR_TRACE = "ERROR_TRACE"
)

var (
	CommonHeaders CommonHeaderHelper
)

type CommonHeaderHelper struct {
}

func (this *CommonHeaderHelper) GetRemoteAddr(cr *corepb.ChannelRequest) string {
	h := cr.GetHeaderInfo(HEADER_REMOTE_ADDR)
	if h == nil {
		return ""
	}
	return h.Value
}

func (this *CommonHeaderHelper) SetRemoteAddr(cr *corepb.ChannelRequest, ip string) {
	cr.AddStringHeader(HEADER_REMOTE_ADDR, ip)
}

func (this *CommonHeaderHelper) GetErrorTrace(cr *corepb.ChannelRequest) []*corepb.PBErrorTrace {
	hl := cr.ListHeaderInfo(HEADER_ERROR_TRACE)
	r := make([]*corepb.PBErrorTrace, 0, len(hl))
	for _, h := range hl {
		et := &corepb.PBErrorTrace{}
		et.Unmarshal(h.Data)
		r = append(r, et)
	}
	return r
}

func (this *CommonHeaderHelper) AddErrorTrace(cr *corepb.ChannelRequest, port, err string) {
	et := &corepb.PBErrorTrace{Port: port, Err: err}
	data, _ := et.Marshal()
	cr.AddHeader(HEADER_ERROR_TRACE, data)
}
