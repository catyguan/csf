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
	"fmt"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/raft/raftpb"
	"github.com/catyguan/csf/servicechannelhandler/schsign"
)

type peerInvoker struct {
	rsc *RaftServiceContainer
}

func (this *RaftServiceContainer) PeerInvoker() core.ServiceInvoker {
	return &peerInvoker{rsc: this}
}

func (this *RaftServiceContainer) CreateSign() *schsign.Sign {
	return schsign.NewSign(this.cfg.AccessCode, schsign.SIGN_REQUEST_VERIFY, false)
}

func (this *peerInvoker) InvokeRequest(ctx context.Context, creq *corepb.ChannelRequest) (*corepb.ChannelResponse, error) {
	switch creq.ServicePath {
	case SP_MESSAGE:
		m := raftpb.Message{}
		err := m.Unmarshal(creq.Data)
		if err != nil {
			plog.Warningf("unmarshal RPC message fail - %v", err)
			return nil, err
		}

		if m.To != this.rsc.cfg.NodeID {
			return nil, fmt.Errorf("invalid message.TO(%d), me(%d)", m.To, this.rsc.cfg.NodeID)
		}

		this.rsc.onRecvRaftMessage(ctx, m)
		resp := creq.CreateResponse(nil, nil)
		return corepb.MakeChannelResponse(resp), nil
	default:
		return nil, core.ErrNotFound
	}
}
