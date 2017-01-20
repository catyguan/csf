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
	"strconv"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/csfctl"
)

func createADDCommand() *csfctl.Command {
	return &csfctl.Command{
		Name:        "add",
		Usage:       "add {varname} {number}",
		Description: `add counter service var value`,
		Aliases:     []string{},
		Args: csfctl.Flags{
			csfctl.Flag{Name: "h", Type: "bool", Usage: "show help"},
		},
		Vars: csfctl.Flags{
			csfctl.Flag{Name: "COUNTER_LOC", Type: "string", Usage: "counter service location, default use SERVICE_LOC"},
		},
		Action: handleADDCommand,
	}
}

func handleADDCommand(ctx context.Context, env *csfctl.Env, pwd *csfctl.CommandDir, cmdobj *csfctl.Command, args []string) error {
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

	if len(args) != 2 {
		return env.PrintErrorf("%s", cmdobj.Usage)
	}
	loc := env.GetVarString("COUNTER_LOC", "")
	if loc == "" {
		loc = env.GetVarString("SERVICE_LOC", "")
	}
	if loc == "" {
		return env.PrintErrorf("COUNTER_LOC nil")
	}

	n := args[0]
	val := args[1]
	v, err4 := strconv.ParseInt(val, 0, 0)
	if err4 != nil {
		return env.PrintErrorf("parse value fail - %s", err4)
	}
	sl, err2 := core.ParseLocation(loc)
	if err2 != nil {
		return env.PrintError(err2)
	}
	api := NewCounter(sl.Invoker)
	rv, err3 := api.AddValue(ctx, n, uint64(v))
	if err3 != nil {
		return env.PrintError(err3)
	}
	env.Printf("%v\n", rv)
	return nil
}
