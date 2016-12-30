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
	"path"
	"sort"
)

func CreateLSCommand() *Command {
	return &Command{
		Name:        "ls",
		Description: "list commands and dirs",
		Aliases:     []string{"ll"},
		Args: Flags{
			Flag{Name: "d", Type: "bool", Usage: "show dirs only"},
			Flag{Name: "c", Type: "bool", Usage: "show command only"},
			Flag{Name: "o", Type: "bool", Usage: "show command under PWD only"},
			Flag{Name: "h", Type: "bool", Usage: "show help"},
		},
		Action: HandleLSCommand,
	}
}

func listCommandNames(dir *CommandDir, np string, local bool) []string {
	var r []string
	for _, co := range dir.Commands {
		n := co.Name
		if np != "" {
			n = path.Join(np, n)
		}
		r = append(r, n)
	}
	if !local && dir.ParentDir != nil {
		np = path.Join(dir.ParentDir.GetPath(), np)
		// if np != "" {
		// 	np = dir.ParentDir.GetPath() + "/" + np
		// } else {
		// 	np = dir.ParentDir.GetPath()
		// }
		r = append(r, listCommandNames(dir.ParentDir, np, local)...)
	}
	return r
}

func HandleLSCommand(ctx context.Context, env *Env, pwd *CommandDir, cmdobj *Command, args []string) error {
	vars, nargs, err := cmdobj.Args.Parse(args)
	if err != nil {
		return env.PrintErrorf(err.Error())
	}
	args = nargs

	varH := vars["h"].(bool)
	varD := vars["d"].(bool)
	varC := vars["c"].(bool)
	varO := vars["o"].(bool)

	if varH {
		DoHelp(ctx, env, cmdobj)
		return nil
	}

	head := true
	if !varC {
		var dirs CommandDirs
		dirs = append(dirs, pwd.SubDirs...)
		sort.Sort(dirs)
		for _, dir := range dirs {
			if !head {
				env.Print(" ")
			}
			head = false
			env.Printf("<%s>", dir.Name)
		}
	}
	if !varD {
		cmds := listCommandNames(pwd, "", varO)
		sort.Strings(cmds)
		for _, co := range cmds {
			if !head {
				env.Print(" ")
			}
			head = false
			env.Printf("%s", co)
		}
	}
	env.Println("")
	return nil
}
