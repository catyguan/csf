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
	"strings"
)

func CreateCDCommand() *Command {
	return &Command{
		Name:        "cd",
		Usage:       "cd | cd <dir>",
		Description: "display or change current command dir",
		Aliases:     []string{"dir"},
		Args:        Flags{},
		Action:      HandleCDCommand,
	}
}

func HandleCDCommand(ctx context.Context, env *Env, pwd *CommandDir, _ *Command, args []string) error {
	p := pwd.GetPath()
	if len(args) == 0 {
		env.Printf("<%s>\n", p)
		return nil
	}
	dirn := args[0]
	var ndirn string
	if path.IsAbs(dirn) {
		ndirn = path.Clean(dirn)
	} else {
		ndirn = path.Join(pwd.GetPath(), dirn)
	}

	plist := strings.Split(ndirn, "/")
	cdir := pwd.GetRootDir()
	for _, n := range plist {
		if n == "" {
			continue
		}
		o := cdir.GetDir(n)
		if o == nil {
			return env.PrintErrorf("can't found dir '%s' at '%s'", n, cdir.GetPath())
		}
		cdir = o
	}
	env.SetPwd(cdir)
	env.Printf("<%s>\n", cdir.GetPath())
	return nil
}
