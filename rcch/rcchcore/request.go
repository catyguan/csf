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
package rcchcore

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type Request struct {
	PBRequestInfo
	Data []byte
}

func NewRequest(t, p string, typ RequestType, data []byte) *Request {
	r := &Request{}
	r.ID = 0
	r.Type = typ
	r.Target = t
	r.Path = p
	r.Data = data
	return r
}

func NewQueryRequest(t, p string, data []byte) *Request {
	return NewRequest(t, p, RequestType_QUERY, data)
}

func NewExecuteRequest(t, p string, data []byte) *Request {
	return NewRequest(t, p, RequestType_EXECUTE, data)
}

func NewMessageRequest(t, p string, data []byte) *Request {
	return NewRequest(t, p, RequestType_MESSAGE, data)
}

func (m *Request) String() string {
	s := m.PBRequestInfo.String()
	s = fmt.Sprintf("%s Data:%v", s, len(m.Data))
	return s
}

func (m *Request) Marshal(w io.Writer) error {
	d1, err := m.PBRequestInfo.Marshal()
	if err != nil {
		return err
	}
	b := make([]byte, binary.MaxVarintLen32)
	n := binary.PutUvarint(b, uint64(len(d1)))
	err2 := doWrite(w, b[:n])
	if err2 != nil {
		return err2
	}
	err2 = doWrite(w, d1)
	if err2 != nil {
		return err2
	}
	n = binary.PutUvarint(b, uint64(len(m.Data)))
	err2 = doWrite(w, b[:n])
	if err2 != nil {
		return err2
	}
	if m.Data != nil {
		err2 = doWrite(w, m.Data)
		if err2 != nil {
			return err2
		}
	}
	return nil
}

func (m *Request) Unmarshal(r StreamReader) error {
	n, err := binary.ReadUvarint(r)
	if err != nil {
		return err
	}
	buf := make([]byte, n)
	err2 := doRead(r, buf)
	if err2 != nil {
		return err2
	}
	err = m.PBRequestInfo.Unmarshal(buf)
	if err != nil {
		return err
	}

	n, err = binary.ReadUvarint(r)
	if err != nil {
		return err
	}
	buf = make([]byte, n)
	err2 = doRead(r, buf)
	if err2 != nil {
		return err2
	}
	m.Data = buf
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
	s = fmt.Sprintf("%s Data:%v Header:%v", s, len(m.Data), m.Header)
	return s
}

func MakeChannelRequest(o *Request) *ChannelRequest {
	r := &ChannelRequest{}
	r.Request = *o
	return r
}

func (m *ChannelRequest) Copy() *ChannelRequest {
	r := new(ChannelRequest)
	*r = *m
	return r
}

func (m *ChannelRequest) AddHeader(n string, data []byte) {
	h := &PBHeader{}
	h.Name = n
	h.Data = data
	m.Header = append(m.Header, h)
}

func (m *ChannelRequest) AddStringHeader(n string, s string) {
	h := &PBHeader{}
	h.Name = n
	h.Value = s
	m.Header = append(m.Header, h)
}

func (m *ChannelRequest) SetHeader(n string, data []byte) {
	h := m.GetHeaderInfo(n)
	if h == nil {
		m.AddHeader(n, data)
	} else {
		h.Data = data
	}
}

func (m *ChannelRequest) SetStringHeader(n string, s string) {
	h := m.GetHeaderInfo(n)
	if h == nil {
		m.AddStringHeader(n, s)
	} else {
		h.Value = s
	}
}

func (m *ChannelRequest) GetHeaderInfo(n string) *PBHeader {
	var r *PBHeader
	for _, h := range m.Header {
		if h.Name == n {
			r = h
		}
	}
	return r
}

func (m *ChannelRequest) ListHeaderInfo(n string) []*PBHeader {
	var r []*PBHeader
	for _, h := range m.Header {
		if h.Name == n {
			r = append(r, h)
		}
	}
	return r
}

type ChannelRequestEncoder struct {
	fbin []byte
	rbin []byte
	hbin []byte
	dbin []byte
}

func NewChannelRequestEncoder() *ChannelRequestEncoder {
	r := &ChannelRequestEncoder{}
	r.fbin = make([]byte, 4) // 32bit
	r.rbin = make([]byte, binary.MaxVarintLen32)
	r.hbin = make([]byte, binary.MaxVarintLen32)
	r.dbin = make([]byte, binary.MaxVarintLen32)
	return r
}

func (this *ChannelRequestEncoder) Marshal(w io.Writer, m *ChannelRequest) error {
	//	[FrameSize:32bit, not include this]
	fsz := uint32(0)
	//[RequestSize:varuint64]
	//[PBRequestInfo]
	d1, err1 := m.Request.PBRequestInfo.Marshal()
	if err1 != nil {
		return err1
	}
	fsz += uint32(len(d1))
	rn := binary.PutUvarint(this.rbin, uint64(len(d1)))
	fsz += uint32(rn)
	//[HeadersSize:varuint64]
	//[PBHeaders]
	hs := &PBHeaders{}
	hs.Header = m.Header
	d2, err2 := hs.Marshal()
	if err2 != nil {
		return err2
	}
	fsz += uint32(len(d2))
	hn := binary.PutUvarint(this.hbin, uint64(len(d2)))
	fsz += uint32(hn)
	//[PayloadSize:varuint64]
	//[Payload:...]
	fsz += uint32(len(m.Request.Data))
	dn := binary.PutUvarint(this.dbin, uint64(len(m.Request.Data)))
	fsz += uint32(dn)

	binary.LittleEndian.PutUint32(this.fbin, fsz)

	// begin write
	err1 = doWrite(w, this.fbin)
	if err1 != nil {
		return err1
	}
	err1 = doWrite(w, this.rbin[:rn])
	if err1 != nil {
		return err1
	}
	err1 = doWrite(w, d1)
	if err1 != nil {
		return err1
	}
	err1 = doWrite(w, this.hbin[:hn])
	if err1 != nil {
		return err1
	}
	err1 = doWrite(w, d2)
	if err1 != nil {
		return err1
	}
	err1 = doWrite(w, this.dbin[:dn])
	if err1 != nil {
		return err1
	}
	if m.Request.Data != nil {
		err1 = doWrite(w, m.Request.Data)
		if err1 != nil {
			return err1
		}
	}
	return nil
}

type ChannelRequestDecoder struct {
	Fsize   uint32
	Fbin    []byte
	Freaded uint32

	Rsize uint32
	Rbin  []byte
	Rdata []byte

	Hsize uint32
	Hbin  []byte
	Hdata []byte

	Dsize uint32
	Dbin  []byte
	Ddata []byte
}

func NewChannelRequestDecoder() *ChannelRequestDecoder {
	r := &ChannelRequestDecoder{}
	r.Fbin = make([]byte, 4) // 32bit
	r.Rbin = make([]byte, binary.MaxVarintLen32)
	r.Hbin = make([]byte, binary.MaxVarintLen32)
	r.Dbin = make([]byte, binary.MaxVarintLen32)
	return r
}

func (this *ChannelRequestDecoder) Reset() {
	this.Fsize = 0
	this.Freaded = 0
	this.Rsize = 0
	this.Hsize = 0
	this.Dsize = 0
	this.Ddata = nil
}

func (this *ChannelRequestDecoder) alloc(b []byte, n uint32) []byte {
	if b == nil {
		return make([]byte, n)
	}
	if uint32(len(b)) < n {
		return make([]byte, n)
	}
	return b
}

func (this *ChannelRequestDecoder) ReadFrameSize(r StreamReader) error {
	err := doRead(r, this.Fbin)
	if err != nil {
		return err
	}
	this.Fsize = binary.LittleEndian.Uint32(this.Fbin)
	this.Freaded = 0
	return nil
}

func (this *ChannelRequestDecoder) CanRead(n uint32) error {
	if this.Freaded+n > this.Fsize {
		return errors.New("read overflow")
	}
	return nil
}

func (this *ChannelRequestDecoder) ReadRequestInfo(r StreamReader) error {
	err := this.CanRead(1)
	if err != nil {
		return err
	}
	v, err2, n := doReadUvarint(r)
	if err2 != nil {
		return err2
	}
	this.Rsize = uint32(v)
	this.Freaded += uint32(n)
	err = this.CanRead(this.Rsize)
	if err != nil {
		return err
	}
	this.Rdata = this.alloc(this.Rdata, this.Rsize)
	err = doRead(r, this.Rdata[:this.Rsize])
	if err != nil {
		return err
	}
	this.Freaded += this.Rsize
	return nil
}

func (this *ChannelRequestDecoder) ReadHeaders(r StreamReader) error {
	err := this.CanRead(1)
	if err != nil {
		return err
	}
	v, err2, n := doReadUvarint(r)
	if err2 != nil {
		return err2
	}
	this.Hsize = uint32(v)
	this.Freaded += uint32(n)
	err = this.CanRead(this.Hsize)
	if err != nil {
		return err
	}
	this.Hdata = this.alloc(this.Hdata, this.Hsize)
	err = doRead(r, this.Hdata[:this.Hsize])
	if err != nil {
		return err
	}
	this.Freaded += this.Hsize
	return nil
}

func (this *ChannelRequestDecoder) ReadPayload(r StreamReader) error {
	err := this.CanRead(1)
	if err != nil {
		return err
	}
	v, err2, n := doReadUvarint(r)
	if err2 != nil {
		return err2
	}
	this.Dsize = uint32(v)
	this.Freaded += uint32(n)
	err = this.CanRead(this.Dsize)
	if err != nil {
		return err
	}
	this.Ddata = this.alloc(this.Ddata, this.Dsize)
	err = doRead(r, this.Ddata[:this.Dsize])
	if err != nil {
		return err
	}
	this.Freaded += this.Dsize
	return nil
}

func (this *ChannelRequestDecoder) Build() (*ChannelRequest, error) {
	cr := &ChannelRequest{}
	err := cr.Request.PBRequestInfo.Unmarshal(this.Rdata[:this.Rsize])
	if err != nil {
		return nil, err
	}
	hs := PBHeaders{}
	err = hs.Unmarshal(this.Hdata[:this.Hsize])
	if err != nil {
		return nil, err
	}
	cr.Header = hs.Header
	cr.Data = this.Ddata
	return cr, nil
}

func (this *ChannelRequestDecoder) Unmarshal(r StreamReader) (*ChannelRequest, error) {
	this.Reset()
	err := this.ReadFrameSize(r)
	if err != nil {
		return nil, err
	}
	err = this.ReadRequestInfo(r)
	if err != nil {
		return nil, err
	}
	err = this.ReadHeaders(r)
	if err != nil {
		return nil, err
	}
	err = this.ReadPayload(r)
	if err != nil {
		return nil, err
	}
	return this.Build()
}
