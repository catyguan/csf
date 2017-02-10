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

package raft4si0admin

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/pbutil"
	"github.com/catyguan/csf/raft4si"
)

const (
	SP_MAKESNAPSHOT     = "makeSnapshot"
	SP_QUERY_NODES_INFO = "queryNodesInfo"
	SP_ADD_NODE         = "addNode"
	SP_UPDATE_NODE      = "updateNode"
	SP_REMOVE_NODE      = "removeNode"
)

type AdminService struct {
	ep *raft4si.RaftServiceContainer
}

func NewAdminService(ep *raft4si.RaftServiceContainer) *AdminService {
	r := &AdminService{}
	r.ep = ep
	return r
}

func (this *AdminService) impl() {
	_ = core.CoreService(this)
}

// BEGIN: 实现core.CoreService
func (this *AdminService) VerifyRequest(ctx context.Context, req *corepb.Request) (bool, error) {
	return req.IsExecuteType(), nil
}

func (this *AdminService) ApplyRequest(ctx context.Context, req *corepb.Request) (*corepb.Response, error) {
	action := req.ServicePath
	data := req.Data
	switch action {
	case SP_MAKESNAPSHOT:
		i, err := this.ep.MakeSnapshot(ctx)
		if err != nil {
			plog.Warningf("%s MakeSnapshot fail - %s", req.ServiceName, err)
			return nil, err
		}
		plog.Infof("%s MakeSnapshot done - %v", req.ServiceName, i)

		rinfo := &PBMakeSnapshotResponse{}
		rinfo.Index = i
		r := pbutil.MustMarshal(rinfo)
		return req.CreateResponse(r, nil), nil
	case SP_QUERY_NODES_INFO:
		lid, nodes, err := this.ep.QueryNodes(ctx)
		if err != nil {
			return nil, err
		}
		nodeId := this.ep.LocalNodeId()
		nilist := make([]*PBNodeInfo, 0)
		for _, n := range nodes {
			ni := &PBNodeInfo{}
			ni.Peer = &raft4si.PBPeer{NodeId: n.NodeID, PeerLocation: n.Location}
			ni.Fail = n.IsFail()
			ni.Leader = lid == n.NodeID
			ni.Live = true
			if n.NodeID != nodeId {
				ni.Live = n.IsLive(5 * time.Second)
			}
			nilist = append(nilist, ni)
		}

		rinfo := &PBQueryNodesInfoResponse{}
		rinfo.NodeInfo = nilist
		rinfo.LocalNodeId = nodeId
		r := pbutil.MustMarshal(rinfo)
		return req.CreateResponse(r, nil), nil
	case SP_ADD_NODE:
		reqo := PBAddNodeActionRequest{}
		err := reqo.Unmarshal(data)
		if err != nil {
			return nil, err
		}
		pp := raft4si.RaftPeer{}
		pp.From(reqo.Peer)
		rid, err1 := this.ep.AddNode(ctx, &pp)
		if err1 != nil {
			return nil, err1
		}
		o := PBAddNodeActionResponse{NodeId: rid}
		rd, err2 := o.Marshal()
		if err2 != nil {
			return nil, err2
		}
		return req.CreateResponse(rd, nil), nil
	case SP_UPDATE_NODE:
		reqo := PBUpdateNodeActionRequest{}
		err := reqo.Unmarshal(data)
		if err != nil {
			return nil, err
		}
		pp := raft4si.RaftPeer{}
		pp.From(reqo.Peer)
		err1 := this.ep.UpdateNode(ctx, &pp)
		if err1 != nil {
			return nil, err1
		}
		return req.CreateResponse(nil, nil), nil
	case SP_REMOVE_NODE:
		reqo := PBRemoveNodeActionRequest{}
		err := reqo.Unmarshal(data)
		if err != nil {
			return nil, err
		}
		err1 := this.ep.RemoveNode(ctx, reqo.NodeId)
		if err1 != nil {
			return nil, err1
		}
		return req.CreateResponse(nil, nil), nil
	default:
		return nil, fmt.Errorf("unknow action %s", action)
	}
}

func (this *AdminService) CreateSnapshot(ctx context.Context, w io.Writer) error {
	return nil
}

func (this *AdminService) ApplySnapshot(ctx context.Context, r io.Reader) error {
	_, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return nil
}

// END: 实现core.CoreService
