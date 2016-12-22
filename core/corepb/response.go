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
package corepb

import (
	"errors"
	"fmt"
)

type Response struct {
	PBResponseInfo
	Data  []byte
	Error string
}

func (this *Response) ToError() error {
	if this.Error == "" {
		return nil
	}
	return errors.New(this.Error)
}

func (m *Response) ToPB(pb *PBResponse) {
	pb.Info = &m.PBResponseInfo
	pb.Data = m.Data
	pb.Error = m.Error
}

func (m *Response) FromPB(pb *PBResponse) {
	if pb.Info != nil {
		m.PBResponseInfo = *pb.Info
	}
	m.Data = pb.Data
	m.Error = pb.Error
}

func (m *Response) Marshal() (data []byte, err error) {
	pb := &PBResponse{}
	m.ToPB(pb)
	return pb.Marshal()
}

func (m *Response) MarshalTo(data []byte) (int, error) {
	pb := &PBResponse{}
	m.ToPB(pb)
	return pb.MarshalTo(data)
}

func (m *Response) Unmarshal(data []byte) error {
	pb := &PBResponse{}
	err := pb.Unmarshal(data)
	if err != nil {
		return err
	}
	m.FromPB(pb)
	return nil
}

func (this *Response) Bind(req *Request) {
	if this.RequestID != 0 {
		return
	}
	if req == nil {
		return
	}
	this.RequestID = req.ID
}

type ChannelResponse struct {
	Response
	Header []*PBHeader
}

func (m *ChannelResponse) String() string {
	s := m.PBResponseInfo.String()
	s = fmt.Sprintf("{%s Data:%v Header:%v}", s, len(m.Data), m.Header)
	return s
}

func MakeChannelResponse(o *Response) *ChannelResponse {
	r := &ChannelResponse{}
	if o != nil {
		r.Response = *o
	}
	return r
}

func (m *ChannelResponse) ToCPB(pb *PBChannelResponse) {
	pb.Response = &PBResponse{}
	m.Response.ToPB(pb.Response)
	pb.Header = m.Header
}

func (m *ChannelResponse) FromCPB(pb *PBChannelResponse) {
	if pb.Response != nil {
		m.Response.FromPB(pb.Response)
	}
	m.Header = pb.Header
}

func (m *ChannelResponse) Marshal() (data []byte, err error) {
	pb := &PBChannelResponse{}
	m.ToCPB(pb)
	return pb.Marshal()
}

func (m *ChannelResponse) MarshalTo(data []byte) (int, error) {
	pb := &PBChannelResponse{}
	m.ToCPB(pb)
	return pb.MarshalTo(data)
}

func (m *ChannelResponse) Unmarshal(data []byte) error {
	pb := &PBChannelResponse{}
	err := pb.Unmarshal(data)
	if err != nil {
		return err
	}
	m.FromCPB(pb)
	return nil
}
