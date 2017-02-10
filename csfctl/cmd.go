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

type Commands []*Command

func (c Commands) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}

func (c Commands) Len() int {
	return len(c)
}

func (c Commands) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

type CommandAction func(ctx context.Context, env *Env, pwd *CommandDir, cmdobj *Command, args []string) error

type Command struct {
	// The name of the command
	Name string
	// A list of aliases for the command
	Aliases []string
	// A short description of the usage of this command
	Usage string

	Description string
	// Args
	Args Flags
	// EnvVar
	Vars Flags

	// The function to call when this command is invoked
	Action CommandAction

	SkipLogFormat bool
}

func (this *Command) Is(n string) bool {
	if this.Name == n {
		return true
	}
	for _, a := range this.Aliases {
		if a == n {
			return true
		}
	}
	return false
}
