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
	"bufio"
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
)

func CreateEXECCommand() *Command {
	return &Command{
		Name:        "exec",
		Usage:       "exec {filename}",
		Description: `read a file and process command line by line`,
		Aliases:     []string{},
		Args: Flags{
			Flag{Name: "i", Type: "bool", Usage: "ignore error"},
			Flag{Name: "n", Type: "bool", Usage: "process command in new Env"},
			Flag{Name: "h", Type: "bool", Usage: "show help"},
		},
		Vars: Flags{
			Flag{Name: "FWD", Type: "string", Usage: "File system Work Directory"},
		},
		Action: HandleEXECCommand,
	}
}

func HandleEXECCommand(ctx context.Context, env *Env, pwd *CommandDir, cmdobj *Command, args []string) error {
	vars, nargs, err := cmdobj.Args.Parse(args)
	if err != nil {
		return env.PrintErrorf(err.Error())
	}
	args = nargs

	varH := vars["h"].(bool)
	varN := vars["n"].(bool)
	varI := vars["i"].(bool)
	if varH {
		DoHelp(ctx, env, cmdobj)
		return nil
	}

	if len(args) == 0 {
		return env.PrintErrorf("exec {filename}")
	}
	dir := env.GetVarString("FWD", "")
	if dir == "" {
		dir, _ = os.Getwd()
	}
	fn := args[0]
	p := ""
	if filepath.IsAbs(fn) {
		p = filepath.Clean(fn)
	} else {
		p = filepath.Join(dir, fn)
	}

	bs, err := ioutil.ReadFile(p)
	if err != nil {
		return env.PrintErrorf(err.Error())
	}
	r := bufio.NewReader(bytes.NewBuffer(bs))
	renv := env
	if varN {
		renv = env.SubEnv()
	}
	return renv.Exec(r, varI)
}
