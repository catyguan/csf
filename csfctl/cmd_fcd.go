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

package csfctl

import (
	"context"
	"os"
	"path"
	"path/filepath"
)

func CreateFCDCommand() *Command {
	return &Command{
		Name:        "fcd",
		Usage:       "fcd | fcd <dir>",
		Description: "display or change current FileSystem dir",
		Aliases:     []string{"fdir"},
		Args: Flags{
			Flag{Name: "rec", Type: "string", Default: "", Usage: "record dir path to ENV Vars"},
		},
		Action: HandleFCDCommand,
	}
}

func HandleFCDCommand(ctx context.Context, env *Env, pwd *CommandDir, cmdobj *Command, args []string) error {
	vars, nargs, err2 := cmdobj.Args.Parse(args)
	if err2 != nil {
		return env.PrintError(err2)
	}
	args = nargs

	varRec := vars["rec"].(string)

	p, err0 := os.Getwd()
	if err0 != nil {
		return env.PrintError(err0)
	}
	if len(args) == 0 {
		env.Printf("<%s>\n", p)
		if varRec != "" {
			env.SetVar(varRec, p)
		}
		return nil
	}
	dirn := args[0]
	var ndirn string
	if path.IsAbs(dirn) {
		ndirn = filepath.Clean(dirn)
	} else {
		ndirn = filepath.Join(p, dirn)
	}

	err := os.Chdir(ndirn)
	if err != nil {
		return env.PrintError(err)
	}
	env.Printf("<%s>\n", ndirn)
	if varRec != "" {
		env.SetVar(varRec, ndirn)
	}
	return nil
}
