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
	"fmt"
)

var commandHelpTemplate = `NAME:
 %s - %s

DESCRIPTION:
 %s

ARGS:
%s

ENV VARS:
%s
`

func CreateHELPCommand() *Command {
	return &Command{
		Name:        "help",
		Usage:       "help <command name>",
		Description: "show command help",
		Aliases:     []string{},
		Args:        Flags{},
		Action:      HandleHelpCommand,
	}
}

func DoHelp(ctx context.Context, env *Env, cmd *Command) {
	var s1, s2, s3, s4, s5 string
	b1 := bytes.NewBufferString(cmd.Name)
	for _, n := range cmd.Aliases {
		b1.WriteString(",")
		b1.WriteString(n)
	}
	s1 = b1.String()

	s2 = cmd.Usage
	if s2 == "" {
		s2 = cmd.Name
	}

	s3 = cmd.Description

	b4 := bytes.NewBufferString("")
	for _, f := range cmd.Args {
		if b4.Len() != 0 {
			b4.WriteString("\n")
		}
		b4.WriteString(" -")
		b4.WriteString(f.Name)
		if f.Type != "" {
			b4.WriteString(", ")
			b4.WriteString(f.Type)
			if f.Default != nil {
				b4.WriteString("(")
				b4.WriteString(fmt.Sprintf("%v", f.Default))
				b4.WriteString(")")
			}
		}
		if f.Usage != "" {
			b4.WriteString(", ")
			b4.WriteString(f.Usage)
		}
	}
	s4 = b4.String()

	b5 := bytes.NewBufferString("")
	for _, f := range cmd.Vars {
		if b5.Len() != 0 {
			b5.WriteString("\n")
		}
		b5.WriteString(" ")
		b5.WriteString(f.Name)
		if f.Type != "" {
			b5.WriteString(", ")
			b5.WriteString(f.Type)
			if f.Default != nil {
				b5.WriteString("(")
				b5.WriteString(fmt.Sprintf("%v", f.Default))
				b5.WriteString(")")
			}
		}
		if f.Usage != "" {
			b5.WriteString(", ")
			b5.WriteString(f.Usage)
		}
	}
	s5 = b5.String()

	env.Printf(commandHelpTemplate, s1, s2, s3, s4, s5)
}

func HandleHelpCommand(ctx context.Context, env *Env, pwd *CommandDir, _ *Command, args []string) error {
	if len(args) == 0 {
		env.Println("help <command name>")
		return nil
	}
	cmds := args[0]
	cmd := LocateCommand(pwd, cmds)
	DoHelp(ctx, env, cmd)
	return nil
}
