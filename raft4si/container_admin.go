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
package raft4si

import (
	"context"

	"github.com/catyguan/csf/pkg/runobj"
)

func (this *RaftServiceContainer) MakeSnapshot(ctx context.Context) (uint64, error) {
	a := &runobj.ActionRequest{
		Type: actionOfMakeSnapshot,
		P1:   ctx,
	}
	ar, err1 := this.ro.ContextCall(ctx, a)
	if err1 != nil {
		return 0, err1
	}
	if ar.Err != nil {
		return 0, ar.Err
	}
	return ar.R1.(uint64), nil
}

func (this *RaftServiceContainer) QueryNodes(ctx context.Context) (uint64, []*RaftPeer, error) {
	a := &runobj.ActionRequest{
		Type: actionOfQueryNodes,
		P1:   ctx,
	}
	ar, err1 := this.ro.ContextCall(ctx, a)
	if err1 != nil {
		return 0, nil, err1
	}
	if ar.Err != nil {
		return 0, nil, ar.Err
	}
	return ar.R1.(uint64), ar.R2.([]*RaftPeer), nil
}

func (this *RaftServiceContainer) AddNode(ctx context.Context, peer *RaftPeer) (uint64, error) {
	a := &runobj.ActionRequest{
		Type: actionOfAddNode,
		P1:   ctx,
		P2:   peer,
	}
	ar, err1 := this.ro.ContextCall(ctx, a)
	if err1 != nil {
		return 0, err1
	}
	if ar.Err != nil {
		return 0, ar.Err
	}
	return ar.R1.(uint64), nil
}

func (this *RaftServiceContainer) UpdateNode(ctx context.Context, peer *RaftPeer) error {
	a := &runobj.ActionRequest{
		Type: actionOfUpdateNode,
		P1:   ctx,
		P2:   peer,
	}
	ar, err1 := this.ro.ContextCall(ctx, a)
	if err1 != nil {
		return err1
	}
	if ar.Err != nil {
		return ar.Err
	}
	return nil
}

func (this *RaftServiceContainer) RemoveNode(ctx context.Context, nodeId uint64) error {
	a := &runobj.ActionRequest{
		Type: actionOfRemoveNode,
		P1:   ctx,
		P2:   nodeId,
	}
	ar, err1 := this.ro.ContextCall(ctx, a)
	if err1 != nil {
		return err1
	}
	if ar.Err != nil {
		return ar.Err
	}
	return nil
}
