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

import (
	"context"
	"errors"

	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/capnslog"
)

var (
	ErrClosed   = errors.New("closed")
	ErrNotFound = errors.New("Not Found")
	ErrNil      = errors.New("Nil Pointer")

	plog = capnslog.NewPackageLogger("github.com/catyguan/csf", "core")
)

func Invoke(si ServiceInvoker, ctx context.Context, req *corepb.Request) (*corepb.Response, error) {
	creq := &corepb.ChannelRequest{}
	creq.Request = *req
	cresp, err := si.InvokeRequest(ctx, creq)
	err = corepb.HandleChannelError(cresp, err)
	if err != nil {
		return nil, err
	}
	if cresp != nil {
		return &cresp.Response, nil
	}
	resp := &corepb.Response{}
	resp.Bind(req)
	return resp, nil
}

func MakeErrorResponse(r *corepb.Response, err error) *corepb.Response {
	if r == nil {
		r = new(corepb.Response)
	}
	r.Error = err.Error()
	return r
}
