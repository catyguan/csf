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

package counter

import (
	"context"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/csfctl"
)

func createGETCommand() *csfctl.Command {
	return &csfctl.Command{
		Name:        "get",
		Usage:       "get {varname}",
		Description: `get counter service var value`,
		Aliases:     []string{},
		Args: csfctl.Flags{
			csfctl.Flag{Name: "h", Type: "bool", Usage: "show help"},
		},
		Vars: csfctl.Flags{
			csfctl.Flag{Name: "COUNTER_LOC", Type: "string", Usage: "counter service location"},
		},
		Action: handleGETCommand,
	}
}

func handleGETCommand(ctx context.Context, env *csfctl.Env, pwd *csfctl.CommandDir, cmdobj *csfctl.Command, args []string) error {
	vars, nargs, err := cmdobj.Args.Parse(args)
	if err != nil {
		return env.PrintErrorf(err.Error())
	}
	args = nargs

	varH := vars["h"].(bool)
	if varH {
		csfctl.DoHelp(ctx, env, cmdobj)
		return nil
	}

	if len(args) == 0 {
		return env.PrintErrorf("%s", cmdobj.Usage)
	}
	loc := env.GetVarString("COUNTER_LOC", "")
	if loc == "" {
		return env.PrintErrorf("COUNTER_LOC nil")
	}

	n := args[0]
	sl, err2 := core.ParseLocation(loc)
	if err2 != nil {
		return env.PrintErrorf(err2.Error())
	}
	api := NewCounter(sl.Invoker)
	v, err3 := api.GetValue(ctx, n)
	if err3 != nil {
		return env.PrintErrorf(err3.Error())
	}
	env.Printf("%v\n", v)
	return nil
}
