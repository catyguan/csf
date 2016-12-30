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

// CommandDirs is a slice of *CommandDir.
type CommandDirs []*CommandDir

func (c CommandDirs) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}

func (c CommandDirs) Len() int {
	return len(c)
}

func (c CommandDirs) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// CommandDir is a category containing commands.
type CommandDir struct {
	Name      string
	Prompt    string
	ParentDir *CommandDir
	SubDirs   CommandDirs
	Commands  Commands
}

func (this *CommandDir) CreateDir(n string) *CommandDir {
	for _, dir := range this.SubDirs {
		if dir.Name == n {
			return dir
		}
	}
	dir := &CommandDir{Name: n}
	this.AddDir(dir)
	return dir
}

func (this *CommandDir) AddDir(dir *CommandDir) *CommandDir {
	this.SubDirs = append(this.SubDirs, dir)
	dir.ParentDir = this
	return this
}

func (this *CommandDir) AddCommand(cmd *Command) *CommandDir {
	this.Commands = append(this.Commands, cmd)
	return this
}

func (this *CommandDir) GetCommand(s string) *Command {
	for _, cmd := range this.Commands {
		if cmd.Is(s) {
			return cmd
		}
	}
	return nil
}

func (this *CommandDir) FindCommand(s string) (*Command, bool) {
	cmd := this.GetCommand(s)
	if cmd != nil {
		return cmd, true
	}
	if this.ParentDir != nil {
		return this.ParentDir.FindCommand(s)
	}
	return nil, false
}

func (this *CommandDir) GetDir(s string) *CommandDir {
	for _, dir := range this.SubDirs {
		if dir.Name == s {
			return dir
		}
	}
	return nil
}

func (this *CommandDir) GetPath() string {
	p := this.path()
	if p == "" {
		return "/"
	}
	return p
}
func (this *CommandDir) path() string {
	if this.ParentDir != nil {
		return this.ParentDir.path() + "/" + this.Name
	}
	return this.Name
}

func (this *CommandDir) GetRootDir() *CommandDir {
	if this.ParentDir == nil {
		return this
	}
	return this.ParentDir.GetRootDir()
}

func (this *CommandDir) GetPrompt() string {
	if this.Prompt != "" {
		return this.Prompt
	}
	if this.ParentDir == nil {
		return ""
	}
	return this.GetPath()
}
