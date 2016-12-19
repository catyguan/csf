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

import "fmt"

type Request struct {
	PBRequestInfo
	Data []byte
}

func NewRequest(sn, sp string, typ RequestType, data []byte) *Request {
	r := &Request{}
	r.ID = 0
	r.Type = typ
	r.ServiceName = sn
	r.ServicePath = sp
	r.Data = data
	return r
}

func NewQueryRequest(serviceName, servicePath string, data []byte) *Request {
	return NewRequest(serviceName, servicePath, RequestType_QUERY, data)
}

func NewExecuteRequest(serviceName, servicePath string, data []byte) *Request {
	return NewRequest(serviceName, servicePath, RequestType_EXECUTE, data)
}

func NewMessageRequest(serviceName, servicePath string, data []byte) *Request {
	return NewRequest(serviceName, servicePath, RequestType_MESSAGE, data)
}

func (m *Request) String() string {
	s := m.PBRequestInfo.String()
	s = fmt.Sprintf("%s, Data = %v", s, len(m.Data))
	return s
}

func (m *Request) ToPB(pb *PBRequest) {
	pb.Info = &m.PBRequestInfo
	pb.Data = m.Data
}

func (m *Request) FromPB(pb *PBRequest) {
	if pb.Info != nil {
		m.PBRequestInfo = *pb.Info
	}
	m.Data = pb.Data
}

func (m *Request) Marshal() (data []byte, err error) {
	pb := &PBRequest{}
	m.ToPB(pb)
	return pb.Marshal()
}

func (m *Request) MarshalTo(data []byte) (int, error) {
	pb := &PBRequest{}
	m.ToPB(pb)
	return pb.MarshalTo(data)
}

func (m *Request) Unmarshal(data []byte) error {
	pb := &PBRequest{}
	err := pb.Unmarshal(data)
	if err != nil {
		return err
	}
	m.FromPB(pb)
	return nil
}

func (this *Request) CreateResponse(data []byte, err error) *Response {
	r := new(Response)
	r.RequestID = this.ID
	r.Data = data
	if err != nil {
		r.Error = err.Error()
	}
	return r
}

func (this *Request) IsQueryType() bool {
	return this.Type == RequestType_QUERY
}

func (this *Request) IsExecuteType() bool {
	return this.Type == RequestType_EXECUTE
}

func (this *Request) IsMessageType() bool {
	return this.Type == RequestType_MESSAGE
}

type ChannelRequest struct {
	Request
	Header []*PBHeader
}

func (m *ChannelRequest) String() string {
	s := m.PBRequestInfo.String()
	s = fmt.Sprintf("%s, Data=%v, Header=%v", s, len(m.Data), m.Header)
	return s
}

func (m *ChannelRequest) ToCPB(pb *PBChannelRequest) {
	pb.Request = &PBRequest{}
	m.Request.ToPB(pb.Request)
	pb.Header = m.Header
}

func (m *ChannelRequest) FromCPB(pb *PBChannelRequest) {
	if pb.Request != nil {
		m.Request.FromPB(pb.Request)
	}
	m.Header = pb.Header
}

func (m *ChannelRequest) Marshal() (data []byte, err error) {
	pb := &PBChannelRequest{}
	m.ToCPB(pb)
	return pb.Marshal()
}

func (m *ChannelRequest) MarshalTo(data []byte) (int, error) {
	pb := &PBChannelRequest{}
	m.ToCPB(pb)
	return pb.MarshalTo(data)
}

func (m *ChannelRequest) Unmarshal(data []byte) error {
	pb := &PBChannelRequest{}
	err := pb.Unmarshal(data)
	if err != nil {
		return err
	}
	m.FromCPB(pb)
	return nil
}
