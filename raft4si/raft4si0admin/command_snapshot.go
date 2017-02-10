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

func createSNAPSHOTCommand() *csfctl.Command {
	return &csfctl.Command{
		Name:        "snapshot",
		Usage:       "snapshot",
		Description: `call RaftServiceContainer.MakeSnapshot`,
		Aliases:     []string{"ss"},
		Args:        csfctl.Flags{
		// csfctl.Flag{Name: "h", Type: "bool", Usage: "show help"},
		},
		Vars: csfctl.Flags{
			csfctl.Flag{Name: "SERVICE_ADMIN_LOC", Type: "string", Usage: "service admin location"},
		},
		Action: handleSNAPSHOTCommand,
	}
}

func handleSNAPSHOTCommand(ctx context.Context, env *csfctl.Env, pwd *csfctl.CommandDir, cmdobj *csfctl.Command, args []string) error {
	loc := env.GetVarString("SERVICE_ADMIN_LOC", "")
	if loc == "" {
		return env.PrintErrorf("SERVICE_ADMIN_LOC nil")
	}

	sl, err2 := core.ParseLocation(loc)
	if err2 != nil {
		return env.PrintError(err2)
	}
	api := NewAdminAPI(DefaultAdminServiceName(sl.ServiceName), sl.Invoker)
	rv, err3 := api.MakeSnapshot(ctx)
	if err3 != nil {
		return env.PrintError(err3)
	}
	env.Printf("%v\n", rv)
	return nil
}
