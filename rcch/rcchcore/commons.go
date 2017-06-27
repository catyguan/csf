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
	"context"
	"errors"
	"io"

	"github.com/catyguan/csf/pkg/capnslog"
)

var (
	ErrClosed   = errors.New("closed")
	ErrNotFound = errors.New("Not Found")
	ErrNil      = errors.New("Nil Pointer")

	plog = capnslog.NewPackageLogger("github.com/catyguan/csf", "core")
)

func Invoke(inv Invoker, ctx context.Context, req *Request) (*Response, error) {
	creq := &ChannelRequest{}
	creq.Request = *req
	cresp, err := inv.Invoke(ctx, creq)
	err = HandleChannelError(cresp, err)
	if err != nil {
		return nil, err
	}
	if cresp != nil {
		return &cresp.Response, nil
	}
	resp := &Response{}
	resp.Bind(req)
	return resp, nil
}

func MakeErrorResponse(r *Response, err error) *Response {
	if r == nil {
		r = new(Response)
	}
	r.Error = err.Error()
	return r
}

func doWrite(w io.Writer, b []byte) error {
	n, err := w.Write(b)
	if err != nil {
		return err
	}
	if n != len(b) {
		return io.ErrShortWrite
	}
	return nil
}

func doRead(r io.Reader, b []byte) error {
	n, err := io.ReadFull(r, b)
	if err != nil {
		return err
	}
	if n != len(b) {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func doReadUvarint(r io.ByteReader) (uint64, error, int) {
	var x uint64
	var s uint
	for i := 0; ; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return x, err, i + 1
		}
		if b < 0x80 {
			if i > 9 || i == 9 && b > 1 {
				return x, errors.New("overflow"), i + 1
			}
			return x | uint64(b)<<s, nil, i + 1
		}
		x |= uint64(b&0x7f) << s
		s += 7
	}
}
