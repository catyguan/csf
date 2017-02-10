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
	"bytes"
	"context"
	"strings"
)

func CreateENVCommand() *Command {
	return &Command{
		Name:  "env",
		Usage: "env, env new, env set {varname} <varvalue>, env replace {varname} <searchString> <replaceString>",
		Description: `env, list all env vars
 env new, create new env
 env set {varname} <varvalue>, set env var or remove env var
 env replace {varname} <searchString> <replaceString>, replace env var string`,
		Aliases: []string{},
		Args: Flags{
			Flag{Name: "h", Type: "bool", Usage: "show help"},
			Flag{Name: "o", Type: "bool", Usage: "show local vars only"},
		},
		Action: HandleENVCommand,
	}
}

func HandleENVCommand(ctx context.Context, env *Env, pwd *CommandDir, cmdobj *Command, args []string) error {
	vars, nargs, err := cmdobj.Args.Parse(args)
	if err != nil {
		return env.PrintErrorf(err.Error())
	}
	args = nargs

	varH := vars["h"].(bool)
	varO := vars["o"].(bool)
	if varH {
		DoHelp(ctx, env, cmdobj)
		return nil
	}

	if len(args) == 0 {
		vars := make(map[string]string)
		env.CopyVars(vars, varO)
		buf := bytes.NewBufferString("ENV VARS:\n")
		for k, v := range vars {
			buf.WriteString("  ")
			buf.WriteString(k)
			buf.WriteString("=")
			buf.WriteString(v)
			buf.WriteString("\n")
		}
		env.Print(buf.String())
		return nil
	}
	act := args[0]
	switch act {
	case "new":
		senv := env.SubEnv()
		env.PushEnv(senv)
		env.Printf("enter ENV[%p]\n", senv)
	case "set":
		args = args[1:]
		if len(args) == 0 {
			return env.PrintErrorf("env set {varname} <varvalue>")
		}
		n := args[0]
		v := ""
		if len(args) > 1 {
			v = args[1]
		}
		env.SetVar(n, v)
		env.Println("OK")
	case "replace":
		args = args[1:]
		if len(args) < 3 {
			return env.PrintErrorf("env replace {varname} <searchString> <replaceString>")
		}
		n := args[0]
		ss := args[1]
		rs := args[2]

		v, b := env.GetVar(n)
		if !b {
			return env.PrintErrorf("miss ENV '%s'", n)
		}
		nv := strings.Replace(v, ss, rs, -1)
		env.SetVar(n, nv)
		env.Printf("%s -> %s\n", v, nv)
	default:
		return env.PrintErrorf("unknow action '%s'", act)
	}
	return nil
}
