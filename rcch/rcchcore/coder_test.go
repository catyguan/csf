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
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestCoder(t *testing.T) {
	r := NewExecuteRequest("target", "path", []byte("hello kitty"))
	r.ID = 100

	buf := bytes.NewBuffer(make([]byte, 0))

	err1 := r.Marshal(buf)
	if !assert.NoError(t, err1) {
		return
	}
	fmt.Printf("encode -> %v\n", buf.Bytes())

	r2 := &Request{}
	err2 := r2.Unmarshal(buf)
	if !assert.NoError(t, err2) {
		return
	}
	fmt.Printf("decode -> %v\n", r2)
}

func TestChannelRequestCoder(t *testing.T) {
	r := NewExecuteRequest("target", "path", []byte("hello kitty"))
	r.ID = 100
	cr := MakeChannelRequest(r)
	cr.AddStringHeader("h", "ok")

	buf := bytes.NewBuffer(make([]byte, 0))

	e := NewChannelRequestEncoder()
	err1 := e.Marshal(buf, cr)
	if !assert.NoError(t, err1) {
		return
	}
	fmt.Printf("encode -> %v\n", buf.Bytes())

	d := NewChannelRequestDecoder()
	cr2, err2 := d.Unmarshal(buf)
	if !assert.NoError(t, err2) {
		return
	}
	fmt.Printf("decode -> %v\n", cr2)
}

func TestResponseCoder1(t *testing.T) {
	r := NewResponse()
	r.RequestID = 100
	r.Data = []byte("hello world")

	buf := bytes.NewBuffer(make([]byte, 0))

	err1 := r.Marshal(buf)
	if !assert.NoError(t, err1) {
		return
	}
	fmt.Printf("encode -> %v\n", buf.Bytes())

	r2 := &Response{}
	err2 := r2.Unmarshal(buf)
	if !assert.NoError(t, err2) {
		return
	}
	fmt.Printf("decode -> %v\n", r2)
}

func TestResponseCoder2(t *testing.T) {
	r := NewResponse()
	r.RequestID = 100
	r.BeErrorString("hello world")

	buf := bytes.NewBuffer(make([]byte, 0))

	err1 := r.Marshal(buf)
	if !assert.NoError(t, err1) {
		return
	}
	fmt.Printf("encode -> %v\n", buf.Bytes())

	r2 := &Response{}
	err2 := r2.Unmarshal(buf)
	if !assert.NoError(t, err2) {
		return
	}
	fmt.Printf("decode -> %v\n", r2)
}

func TestChannelResponseCoder1(t *testing.T) {
	r := NewPushResponse()
	r.RequestID = 100
	r.Data = []byte("hello world")
	cr := MakeChannelResponse(r)
	cr.AddStringHeader("h", "ok")

	buf := bytes.NewBuffer(make([]byte, 0))

	e := NewChannelResponseEncoder()
	err1 := e.Marshal(buf, cr)
	if !assert.NoError(t, err1) {
		return
	}
	fmt.Printf("encode -> %v\n", buf.Bytes())

	d := NewChannelResponseDecoder()
	cr2, err2 := d.Unmarshal(buf)
	if !assert.NoError(t, err2) {
		return
	}
	fmt.Printf("decode -> %v\n", cr2)
}

func TestChannelResponseCoder2(t *testing.T) {
	r := NewPushResponse()
	r.RequestID = 100
	r.BeErrorString("hello world")
	cr := MakeChannelResponse(r)
	cr.AddStringHeader("h", "ok")

	buf := bytes.NewBuffer(make([]byte, 0))

	e := NewChannelResponseEncoder()
	err1 := e.Marshal(buf, cr)
	if !assert.NoError(t, err1) {
		return
	}
	fmt.Printf("encode -> %v\n", buf.Bytes())

	d := NewChannelResponseDecoder()
	cr2, err2 := d.Unmarshal(buf)
	if !assert.NoError(t, err2) {
		return
	}
	fmt.Printf("decode -> %v\n", cr2)
}
