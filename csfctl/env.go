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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
)

var (
	ErrNotFound = errors.New("command not found")
)

type Env struct {
	isStop  bool
	nextEnv *Env

	root *CommandDir
	pwd  *CommandDir
	mu   sync.RWMutex
	vars map[string]string

	out     io.Writer
	in      *bufio.Reader
	backend *Env
}

func newEnv() *Env {
	r := &Env{}
	return r
}

func NewRootEnv() *Env {
	r := newEnv()
	r.root = &CommandDir{
		Name: "",
	}
	r.pwd = r.root
	return r
}

func NewEnv(backend *Env) *Env {
	r := newEnv()
	r.backend = backend
	return r
}

func (this *Env) SubEnv() *Env {
	r := newEnv()
	r.backend = this
	return r
}

func (this *Env) BackEnd() *Env {
	return this.backend
}

func (this *Env) RootDir() *CommandDir {
	return this.root
}

type envAsWriter struct {
	env *Env
}

func (this *envAsWriter) Write(b []byte) (int, error) {
	this.env.Print(string(b))
	return len(b), nil
}

func (this *Env) AsOutput() io.Writer {
	return &envAsWriter{env: this}
}

func (this *Env) RunAsConsole() {
	renv := NewEnv(this)
	renv.in = bufio.NewReader(os.Stdin)
	renv.out = os.Stdout
	renv.doRunConsole()
}

func (this *Env) doRunConsole() {
	for {
		if this.IsStop() {
			return
		}
		this.Println("")
		this.Print(this.Prompt())
		cmd, _ := this.iReadLine()
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}
		pwd, _ := this.GetPwd()
		_, ok := ProcessCommand(this, pwd, cmd)
		if !ok {
			this.Printf("sorry, don't know how to do '%v'\n", cmd)
		}
		if this.nextEnv != nil {
			ne := this.nextEnv
			this.nextEnv = nil
			ne.doRunConsole()
		}
	}
}

func (this *Env) Exec(nr *bufio.Reader, ignoreErr bool) error {
	for {
		if this.IsStop() {
			return nil
		}
		cmd, errR := nr.ReadString('\n')
		cmd = strings.TrimSpace(cmd)
		if strings.HasPrefix(cmd, "#") {
			// 注释, skip
			continue
		}
		if cmd == "" {
			if errR != nil {
				if errR == io.EOF {
					errR = nil
				}
				if errR != nil {
					this.Printf("ERR: read command fail - %v", errR)
				}
				if ignoreErr {
					errR = nil
				}
				return errR
			}
			continue
		}
		this.Printf("exec '%v'\n", cmd)
		pwd, _ := this.GetPwd()
		errE, ok := ProcessCommand(this, pwd, cmd)
		if !ok {
			this.Printf("ERR: sorry, don't know how to do '%v'\n", cmd)
		}
		if errE != nil {
			if ignoreErr {
				continue
			}
			return errE
		}
		if this.nextEnv != nil {
			ne := this.nextEnv
			this.nextEnv = nil
			ne.Exec(nr, ignoreErr)
		}
	}
}

func execCommand(cmd *Command, env *Env, pwd *CommandDir, cmdobj *Command, args []string) error {
	ctx := context.Background()
	return cmd.Action(ctx, env, pwd, cmdobj, args)
}

type subprocesWriter struct {
	nl bool
	t  string
	w  io.Writer
}

func (this *subprocesWriter) doWrite(p []byte) error {
	if this.nl {
		_, err1 := this.w.Write([]byte(this.t))
		if err1 != nil {
			return err1
		}
		this.nl = false
	}
	_, err2 := this.w.Write(p)
	if err2 != nil {
		return err2
	}
	return nil
}

func (this *subprocesWriter) Write(p []byte) (int, error) {
	s := 0
	for i, b := range p {
		if b == '\n' {
			err2 := this.doWrite(p[s : i+1])
			this.nl = true
			if err2 != nil {
				return 0, err2
			}
			s = i + 1
		}
	}
	if s < len(p) {
		err2 := this.doWrite(p[s:])
		if err2 != nil {
			return 0, err2
		}
	}
	return len(p), nil
}

func LocateDir(pwd *CommandDir, pn string) *CommandDir {
	var p string
	if path.IsAbs(pn) {
		p = path.Clean(pn)
	} else {
		p = path.Join(pwd.GetPath(), pn)
	}
	slist := strings.Split(p, "/")
	cdir := pwd.GetRootDir()
	for _, n := range slist {
		if n == "" {
			continue
		}
		o := cdir.GetDir(n)
		if o == nil {
			return nil
		}
		cdir = o
	}
	return cdir
}

func LocateCommand(pwd *CommandDir, cmd string) *Command {
	var p string
	if path.IsAbs(cmd) {
		p = path.Clean(cmd)
	} else {
		p = path.Join(pwd.GetPath(), cmd)
	}
	slist := strings.Split(p, "/")
	cdir := pwd.GetRootDir()
	l := len(slist) - 1
	for i := 0; i < l; i++ {
		n := slist[i]
		if n == "" {
			continue
		}
		o := cdir.GetDir(n)
		if o == nil {
			return nil
		}
		cdir = o
	}
	co, _ := cdir.FindCommand(slist[l])
	return co
}

// co, ok := pwd.FindCommand(cmd)
// if !ok {
// 	return ErrNotFound, false
// }
// return co, ok

func ProcessCommand(env *Env, pwd *CommandDir, cmdline string) (error, bool) {
	str := env.FormatVars(cmdline)
	if str != cmdline {
		env.Printf("-> %s\n", str)
		cmdline = str
	}

	slist := strings.SplitN(cmdline, " ", 2)
	cmd := slist[0]
	argline := ""
	if len(slist) > 1 {
		argline = strings.TrimSpace(slist[1])
	}
	co := LocateCommand(pwd, cmd)
	if co == nil {
		return ErrNotFound, false
	}
	args := parseArgs(argline)
	if len(args) > 0 {
		tails := args[len(args)-1]
		if tails == "&" {
			// fork
			nenv := env.SubEnv()
			title := fmt.Sprintf("%p - ", nenv)
			nenv.out = &subprocesWriter{nl: true, t: title, w: env.out}
			env.Printf("fork ENV[%p]...\n", nenv)
			go func(cmd *Command, env *Env, pwd *CommandDir, cmdobj *Command, args []string) {
				err := execCommand(co, env, pwd, cmdobj, args)
				env.Printf("ENV[%p] exit. error: %v\n", env, err)
			}(co, nenv, pwd, co, args[:len(args)-1])
			return nil, true
		}
	}
	err := execCommand(co, env, pwd, co, args)
	return err, true
}
