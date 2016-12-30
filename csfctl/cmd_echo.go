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

import "context"

func CreateECHOCommand() *Command {
	return &Command{
		Name:        "echo",
		Usage:       "echo <content>",
		Description: `print content to Env output`,
		Aliases:     []string{"print"},
		Args:        Flags{},
		Action:      HandleECHOCommand,
	}
}

func HandleECHOCommand(ctx context.Context, env *Env, pwd *CommandDir, cmdobj *Command, args []string) error {
	str := ""
	if len(args) > 0 {
		str = args[0]
	}
	env.Println(str)
	return nil
}
