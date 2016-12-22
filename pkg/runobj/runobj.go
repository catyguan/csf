// Copyright 2016 The CSF Authors
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

package runobj

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

var (
	ErrClosed  = errors.New("closed")
	ErrTimeout = errors.New("timeout")
)

type ActionResponse struct {
	R1  interface{}
	R2  interface{}
	R3  interface{}
	Err error
}

type ActionRequest struct {
	Type int
	P1   interface{}
	P2   interface{}
	P3   interface{}
	P4   interface{}
	P5   interface{}
	Resp chan *ActionResponse
}

type RunFunc func(ready chan error, ach <-chan *ActionRequest, p interface{})

type RunObj struct {
	closed  uint64
	closef  chan interface{}
	actions chan *ActionRequest
}

func NewRunObj(actionQueueSize int) *RunObj {
	r := new(RunObj)
	r.closef = make(chan interface{}, 0)
	r.actions = make(chan *ActionRequest, actionQueueSize)
	return r
}

func (this *RunObj) Run(rf RunFunc, p interface{}) error {
	rd := make(chan error, 1)
	go this.doRun(rd, rf, p)
	err := <-rd
	return err
}

func (this *RunObj) IsClosed() bool {
	return atomic.LoadUint64(&this.closed) > 0
}

func (this *RunObj) Close() {
	if !atomic.CompareAndSwapUint64(&this.closed, 0, 1) {
		return
	}
	this.actions <- nil
	<-this.closef
}

func (this *RunObj) Call(a *ActionRequest) (*ActionResponse, error) {
	if this.IsClosed() {
		return nil, ErrClosed
	}
	if a.Resp == nil {
		a.Resp = make(chan *ActionResponse, 1)
	}
	this.actions <- a
	ar := <-a.Resp
	if ar == nil {
		return nil, nil
	}
	if ar.Err != nil {
		return nil, ar.Err
	}
	return ar, nil
}

func (this *RunObj) CallWithTimeout(a *ActionRequest, du time.Duration) (*ActionResponse, error) {
	if this.IsClosed() {
		return nil, ErrClosed
	}
	if a.Resp == nil {
		a.Resp = make(chan *ActionResponse, 1)
	}
	this.actions <- a
	tm := time.NewTimer(du)
	defer tm.Stop()
	select {
	case ar := <-a.Resp:
		if ar == nil {
			return nil, nil
		}
		if ar.Err != nil {
			return nil, ar.Err
		}
		return ar, nil
	case <-tm.C:
		return nil, ErrTimeout
	}
}

func (this *RunObj) AsyncCall(a *ActionRequest) (<-chan *ActionResponse, error) {
	if this.IsClosed() {
		return nil, ErrClosed
	}
	if a.Resp == nil {
		a.Resp = make(chan *ActionResponse, 1)
	}
	this.actions <- a
	return a.Resp, nil
}

func (this *RunObj) Post(a *ActionRequest) error {
	if this.IsClosed() {
		return ErrClosed
	}
	this.actions <- a
	return nil
}

func (this *RunObj) ContextCall(ctx context.Context, a *ActionRequest) (*ActionResponse, error) {
	c, err1 := this.AsyncCall(a)
	if err1 != nil {
		return nil, err1
	}
	done := ctx.Done()
	select {
	case re := <-c:
		return re, nil
	case <-done:
		return nil, ctx.Err()
	}
}
