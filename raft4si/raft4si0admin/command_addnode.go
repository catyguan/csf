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
	"context"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/csfctl"
)

func createADDNODECommand() *csfctl.Command {
	return &csfctl.Command{
		Name:        "addnode",
		Usage:       "addnode <peerLocation>",
		Description: `call RaftServiceContainer.AddNode, return nodeId`,
		Aliases:     []string{"raft.addnode"},
		Args:        csfctl.Flags{
		// csfctl.Flag{Name: "h", Type: "bool", Usage: "show help"},
		},
		Vars: csfctl.Flags{
			csfctl.Flag{Name: "SERVICE_ADMIN_LOC", Type: "string", Usage: "service admin location"},
		},
		Action: handleADDNODECommand,
	}
}

func handleADDNODECommand(ctx context.Context, env *csfctl.Env, pwd *csfctl.CommandDir, cmdobj *csfctl.Command, args []string) error {
	if len(args) != 1 {
		csfctl.DoHelp(ctx, env, cmdobj)
		return nil
	}
	loc := env.GetVarString("SERVICE_ADMIN_LOC", "")
	if loc == "" {
		return env.PrintErrorf("SERVICE_ADMIN_LOC nil")
	}

	sl, err2 := core.ParseLocation(loc)
	if err2 != nil {
		return env.PrintError(err2)
	}
	api := NewAdminAPI(DefaultAdminServiceName(sl.ServiceName), sl.Invoker)
	rv, err3 := api.AddNode(ctx, 0, args[0])
	if err3 != nil {
		return env.PrintError(err3)
	}
	env.Printf("%v\n", rv)
	return nil
}
