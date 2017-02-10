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

package raft4si0admin

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/csfctl"
)

func createNODESCommand() *csfctl.Command {
	return &csfctl.Command{
		Name:        "nodes",
		Usage:       "nodes",
		Description: `call RaftServiceContainer.QueryNodesInfo`,
		Aliases:     []string{"raft.nodes"},
		Args: csfctl.Flags{
			csfctl.Flag{Name: "rec", Type: "string", Default: "", Usage: "record result to ENV Vars"},
		},
		Vars: csfctl.Flags{
			csfctl.Flag{Name: "SERVICE_ADMIN_LOC", Type: "string", Usage: "service admin location"},
		},
		Action: handleNODESCommand,
	}
}

type unt64Slice []uint64

func (p unt64Slice) Len() int           { return len(p) }
func (p unt64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p unt64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func handleNODESCommand(ctx context.Context, env *csfctl.Env, pwd *csfctl.CommandDir, cmdobj *csfctl.Command, args []string) error {
	vars, nargs, err := cmdobj.Args.Parse(args)
	if err != nil {
		return env.PrintError(err)
	}
	args = nargs

	varRec := vars["rec"].(string)

	loc := env.GetVarString("SERVICE_ADMIN_LOC", "")
	if loc == "" {
		return env.PrintErrorf("SERVICE_ADMIN_LOC nil")
	}

	sl, err2 := core.ParseLocation(loc)
	if err2 != nil {
		return env.PrintError(err2)
	}
	api := NewAdminAPI(DefaultAdminServiceName(sl.ServiceName), sl.Invoker)
	_, nodes, err3 := api.QueryNodesInfo(ctx)
	if err3 != nil {
		return env.PrintError(err3)
	}
	vmap := make(map[uint64]*PBNodeInfo)
	vilist := make([]uint64, 0)
	for _, n := range nodes {
		vmap[n.Peer.NodeId] = n
		vilist = append(vilist, n.Peer.NodeId)
	}
	sort.Sort(unt64Slice(vilist))

	buf := bytes.NewBuffer(make([]byte, 0))
	for _, nid := range vilist {
		n := vmap[nid]
		s := fmt.Sprintf("%d: %s", n.Peer.NodeId, n.Peer.PeerLocation)
		buf.WriteString(s)
		if n.Leader {
			buf.WriteString(" (*)")
		}
		if n.Fail {
			buf.WriteString(" F")
		}
		if !n.Live {
			buf.WriteString(" D")
		}
		buf.WriteString("\n")
	}
	s := buf.String()
	env.Printf("%v\n", s)
	if varRec != "" {
		env.SetVar(varRec, s)
	}
	return nil
}
