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

package cluster

import (
	"context"
	"time"

	pb "github.com/catyguan/csf/cluster/clusterpb"
	"github.com/catyguan/csf/interfaces"
)

func (this *CSFNode) GetACL() interfaces.ACL {
	return this.Cfg.ACL
}

func (this *CSFNode) ClientCertAuthEnabled() bool {
	return this.Cfg.ClientCertAuthEnabled
}

func (this *CSFNode) ClientRequestTime() time.Duration {
	return this.Cfg.ReqTimeout()
}

type Response struct {
	data []byte
	err  error
}

func (this *CSFNode) DoClusterAction(ctx context.Context, servceId, action string, data []byte) ([]byte, error) {
	req := &pb.Request{ServiceID: servceId, Action: action, Data: data}
	resp, err := this.processClusterAction(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.err != nil {
		return nil, resp.err
	}
	return resp.data, nil
}

func (this *CSFNode) processClusterAction(ctx context.Context, r *pb.Request) (Response, error) {
	r.ID = this.reqIDGen.Next()
	return this.processRaftRequest(ctx, r)
}

func (this *CSFNode) processRaftRequest(ctx context.Context, r *pb.Request) (Response, error) {
	data, err := r.Marshal()
	if err != nil {
		return Response{}, err
	}
	ch := this.w.Register(r.ID)

	start := time.Now()
	this.raftNode.Propose(ctx, data)
	proposalsPending.Inc()
	defer proposalsPending.Dec()

	select {
	case x := <-ch:
		resp := x.(Response)
		return resp, resp.err
	case <-ctx.Done():
		proposalsFailed.Inc()
		this.w.Trigger(r.ID, nil) // GC wait
		return Response{}, this.parseProposeCtxErr(ctx.Err(), start)
	case <-this.stopping:
	}
	return Response{}, ErrStopped
}

func (this *CSFNode) applyRequest(r *pb.Request) Response {
	// SERVICE: Apply Here
	if this.shub == nil {
		plog.Warningf("apply request(%s:%s) fail, miss ServiceHub", r.ServiceID, r.Action)
		return Response{err: ErrUnknownActions}
	}
	serv, ok := this.shub[r.ServiceID]
	if !ok {
		plog.Warningf("apply request(%s:%s) fail, miss Service", r.ServiceID, r.Action)
		return Response{err: ErrUnknownActions}
	}
	rdata, err := serv.ApplyAction(this, r.Action, r.Data)
	if err != nil {
		return Response{err: err}
	}
	return Response{data: rdata}
}
