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

type Response struct {
	PBResponseInfo
	Data  []byte
	Error string
}

func NewResponse() *Response {
	r := &Response{}
	r.Type = ResponseType_RESPONSE
	return r
}

func NewACKResponse() *Response {
	r := &Response{}
	r.Type = ResponseType_ACK
	return r
}

func NewPushResponse() *Response {
	r := &Response{}
	r.Type = ResponseType_PUSH
	return r
}

func (m *Response) String() string {
	s := m.PBResponseInfo.String()
	if m.Error != "" {
		s = fmt.Sprintf("%s Error:%v", s, m.Error)
	} else {
		s = fmt.Sprintf("%s Data:%v", s, len(m.Data))
	}
	return s
}

func (this *Response) BeError(err error) {
	this.HasError = true
	this.Error = err.Error()
}

func (this *Response) BeErrorString(err string) {
	this.HasError = true
	this.Error = err
}

func (this *Response) ToError() error {
	if this.Error == "" {
		return nil
	}
	return errors.New(this.Error)
}

func (m *Response) Marshal(w io.Writer) error {
	m.HasError = m.Error != ""
	d1, err := m.PBResponseInfo.Marshal()
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
	var data []byte
	if m.HasError {
		data = []byte(m.Error)
	} else {
		data = m.Data
	}
	n = binary.PutUvarint(b, uint64(len(data)))
	err2 = doWrite(w, b[:n])
	if err2 != nil {
		return err2
	}
	if data != nil {
		err2 = doWrite(w, data)
		if err2 != nil {
			return err2
		}
	}
	return nil
}

func (m *Response) Unmarshal(r StreamReader) error {
	n, err := binary.ReadUvarint(r)
	if err != nil {
		return err
	}
	buf := make([]byte, n)
	err2 := doRead(r, buf)
	if err2 != nil {
		return err2
	}
	err = m.PBResponseInfo.Unmarshal(buf)
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
	if m.HasError {
		m.Error = string(buf)
	} else {
		m.Data = buf
	}
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
	if m.Error != "" {
		s = fmt.Sprintf("{%s Error:%v Header:%v}", s, m.Error, m.Header)
	} else {
		s = fmt.Sprintf("{%s Data:%v Header:%v}", s, len(m.Data), m.Header)
	}
	return s
}

func MakeChannelResponse(o *Response) *ChannelResponse {
	r := &ChannelResponse{}
	if o != nil {
		r.Response = *o
	}
	return r
}

func (m *ChannelResponse) Copy() *ChannelResponse {
	r := new(ChannelResponse)
	*r = *m
	return r
}

func (m *ChannelResponse) AddHeader(n string, data []byte) {
	h := &PBHeader{}
	h.Name = n
	h.Data = data
	m.Header = append(m.Header, h)
}

func (m *ChannelResponse) AddStringHeader(n string, s string) {
	h := &PBHeader{}
	h.Name = n
	h.Value = s
	m.Header = append(m.Header, h)
}

func (m *ChannelResponse) SetHeader(n string, data []byte) {
	h := m.GetHeaderInfo(n)
	if h == nil {
		m.AddHeader(n, data)
	} else {
		h.Data = data
	}
}

func (m *ChannelResponse) SetStringHeader(n string, s string) {
	h := m.GetHeaderInfo(n)
	if h == nil {
		m.AddStringHeader(n, s)
	} else {
		h.Value = s
	}
}

func (m *ChannelResponse) GetHeaderInfo(n string) *PBHeader {
	var r *PBHeader
	for _, h := range m.Header {
		if h.Name == n {
			r = h
		}
	}
	return r
}

func (m *ChannelResponse) ListHeaderInfo(n string) []*PBHeader {
	var r []*PBHeader
	for _, h := range m.Header {
		if h.Name == n {
			r = append(r, h)
		}
	}
	return r
}

type ChannelResponseEncoder struct {
	fbin []byte
	rbin []byte
	hbin []byte
	dbin []byte
}

func NewChannelResponseEncoder() *ChannelResponseEncoder {
	r := &ChannelResponseEncoder{}
	r.fbin = make([]byte, 4) // 32bit
	r.rbin = make([]byte, binary.MaxVarintLen32)
	r.hbin = make([]byte, binary.MaxVarintLen32)
	r.dbin = make([]byte, binary.MaxVarintLen32)
	return r
}

func (this *ChannelResponseEncoder) Marshal(w io.Writer, m *ChannelResponse) error {
	//	[FrameSize:32bit, not include this]
	fsz := uint32(0)
	//[RequestSize:varuint64]
	//[PBRequestInfo]
	m.Response.HasError = m.Response.Error != ""
	d1, err1 := m.Response.PBResponseInfo.Marshal()
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
	var data []byte
	if m.HasError {
		data = []byte(m.Error)
	} else {
		data = m.Data
	}
	fsz += uint32(len(data))
	dn := binary.PutUvarint(this.dbin, uint64(len(data)))
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
	if data != nil {
		err1 = doWrite(w, data)
		if err1 != nil {
			return err1
		}
	}
	return nil
}

type ChannelResponseDecoder struct {
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

func NewChannelResponseDecoder() *ChannelResponseDecoder {
	r := &ChannelResponseDecoder{}
	r.Fbin = make([]byte, 4) // 32bit
	r.Rbin = make([]byte, binary.MaxVarintLen32)
	r.Hbin = make([]byte, binary.MaxVarintLen32)
	r.Dbin = make([]byte, binary.MaxVarintLen32)
	return r
}

func (this *ChannelResponseDecoder) Reset() {
	this.Fsize = 0
	this.Freaded = 0
	this.Rsize = 0
	this.Hsize = 0
	this.Dsize = 0
	this.Ddata = nil
}

func (this *ChannelResponseDecoder) alloc(b []byte, n uint32) []byte {
	if b == nil {
		return make([]byte, n)
	}
	if uint32(len(b)) < n {
		return make([]byte, n)
	}
	return b
}

func (this *ChannelResponseDecoder) ReadFrameSize(r StreamReader) error {
	err := doRead(r, this.Fbin)
	if err != nil {
		return err
	}
	this.Fsize = binary.LittleEndian.Uint32(this.Fbin)
	this.Freaded = 0
	return nil
}

func (this *ChannelResponseDecoder) CanRead(n uint32) error {
	if this.Freaded+n > this.Fsize {
		return errors.New("read overflow")
	}
	return nil
}

func (this *ChannelResponseDecoder) ReadResponseInfo(r StreamReader) error {
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

func (this *ChannelResponseDecoder) ReadHeaders(r StreamReader) error {
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

func (this *ChannelResponseDecoder) ReadPayload(r StreamReader) error {
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

func (this *ChannelResponseDecoder) Build() (*ChannelResponse, error) {
	cr := &ChannelResponse{}
	err := cr.Response.PBResponseInfo.Unmarshal(this.Rdata[:this.Rsize])
	if err != nil {
		return nil, err
	}
	hs := PBHeaders{}
	err = hs.Unmarshal(this.Hdata[:this.Hsize])
	if err != nil {
		return nil, err
	}
	cr.Header = hs.Header
	if cr.HasError {
		cr.Error = string(this.Ddata)
	} else {
		cr.Data = this.Ddata
	}
	return cr, nil
}

func (this *ChannelResponseDecoder) Unmarshal(r StreamReader) (*ChannelResponse, error) {
	this.Reset()
	err := this.ReadFrameSize(r)
	if err != nil {
		return nil, err
	}
	err = this.ReadResponseInfo(r)
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
