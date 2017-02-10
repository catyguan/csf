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
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

func (this *Env) PopEnv() {
	this.isStop = true
}

func (this *Env) PushEnv(env *Env) {
	this.nextEnv = env
}

func (this *Env) IsStop() bool {
	if this.isStop {
		return true
	}
	if this.backend != nil {
		return this.backend.IsStop()
	}
	return false
}

func (this *Env) Prompt() string {
	p := ""
	pwd, _ := this.GetPwd()
	if pwd != nil {
		p = pwd.GetPrompt()
	}
	return p + ">> "
}

func (this *Env) iPrint(s string) bool {
	if this.out != nil {
		io.WriteString(this.out, s)
		return true
	}
	if this.backend != nil {
		if this.backend.iPrint(s) {
			return true
		}
	}
	return false
}

func (this *Env) iReadLine() (string, bool) {
	if this.in != nil {
		s, _ := this.in.ReadString('\n')
		return strings.TrimSpace(s), true
	}
	if this.backend != nil {
		s, ok := this.backend.iReadLine()
		if ok {
			return s, ok
		}
	}
	return "", false
}

func (this *Env) Println(args ...interface{}) {
	this.iPrint(fmt.Sprintln(args...))
}

func (this *Env) Printf(format string, args ...interface{}) {
	this.iPrint(fmt.Sprintf(format, args...))
}

func (this *Env) Print(args ...interface{}) {
	this.iPrint(fmt.Sprint(args...))
}

func (this *Env) PrintError(err error) error {
	this.Println("ERR: " + err.Error())
	return err
}

func (this *Env) PrintErrorf(format string, args ...interface{}) error {
	s := fmt.Sprintf(format, args...)
	this.Println("ERR: " + s)
	return errors.New(s)
}

func (this *Env) SetPwd(d *CommandDir) {
	this.pwd = d
}

func (this *Env) GetPwd() (*CommandDir, bool) {
	if this.pwd != nil {
		return this.pwd, true
	}
	if this.backend != nil {
		r, ok := this.backend.GetPwd()
		if ok {
			return r, ok
		}
	}
	return nil, false
}

func (this *Env) SetVar(n string, v string) {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.vars == nil {
		this.vars = make(map[string]string)
	}
	if v == "" {
		delete(this.vars, n)
	} else {
		this.vars[n] = v
	}
}

func (this *Env) GetVar(n string) (string, bool) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	var v string
	var ok bool
	if this.vars != nil {
		v, ok = this.vars[n]
	}
	if !ok && this.backend != nil {
		return this.backend.GetVar(n)
	}
	return v, ok
}

func (this *Env) CopyVars(r map[string]string, local bool) {
	if !local && this.backend != nil {
		this.backend.CopyVars(r, local)
	}
	this.mu.RLock()
	defer this.mu.RUnlock()
	for k, v := range this.vars {
		r[k] = v
	}
}

func (this *Env) GetVarString(n string, dv string) string {
	v, ok := this.GetVar(n)
	if !ok {
		return dv
	}
	return v
}

func (this *Env) GetVarBool(n string, dv bool) bool {
	v, ok := this.GetVar(n)
	if !ok {
		return dv
	}
	vv, err := strconv.ParseBool(v)
	if err != nil {
		return dv
	}
	return vv
}

func (this *Env) GetVarInt(n string, dv int) int {
	v, ok := this.GetVar(n)
	if !ok {
		return dv
	}
	vv, err := strconv.ParseInt(v, 0, 0)
	if err != nil {
		return dv
	}
	return int(vv)
}

func (this *Env) GetVarInt64(n string, dv int64) int64 {
	v, ok := this.GetVar(n)
	if !ok {
		return dv
	}
	vv, err := strconv.ParseInt(v, 0, 0)
	if err != nil {
		return dv
	}
	return vv
}

func (this *Env) GetVarFloat(n string, dv float64) float64 {
	v, ok := this.GetVar(n)
	if !ok {
		return dv
	}
	vv, err := strconv.ParseFloat(v, 0)
	if err != nil {
		return dv
	}
	return vv
}

func (this *Env) GetVarDuration(n string, dv time.Duration) time.Duration {
	v, ok := this.GetVar(n)
	if !ok {
		return dv
	}
	vv, err := time.ParseDuration(v)
	if err != nil {
		return dv
	}
	return vv
}

func (this *Env) FormatVars(s string) string {
	if this.GetVarBool("STRICT", false) {
		return s
	}
	return this.DoFormatVars(s)
}

func (this *Env) DoFormatVars(s string) string {
	buf := bytes.NewBuffer(make([]byte, 0))
	q1 := false
	q2 := false
	qch := rune(0)
	wbuf := bytes.NewBuffer(make([]byte, 0))
	for _, ch := range []rune(s) {
		if q1 {
			if q2 {
				q2 = false
				if ch == '{' {
					qch = '}'
					continue
				} else {
					qch = ' '
				}
			}
			if ch == qch {
				w := wbuf.String()
				nw := this.GetVarString(w, "")
				buf.WriteString(nw)
				q1 = false
			} else {
				wbuf.WriteRune(ch)
			}
		} else {
			if ch == '$' {
				q1 = true
				q2 = true
				wbuf.Reset()
			} else {
				buf.WriteRune(ch)
			}
		}
	}

	if q1 {
		w := wbuf.String()
		nw := this.GetVarString(w, "")
		buf.WriteString(nw)
	}

	return buf.String()
}

func parseArgs(s string) []string {
	r := make([]string, 0)
	buf := make([]rune, len(s))
	wbuf := buf[:0]
	var qch rune
	for _, ch := range []rune(s) {
		if qch != 0 {
			if qch == ch {
				s := string(wbuf)
				r = append(r, s)
				wbuf = buf[:0]
				qch = 0
			} else {
				wbuf = append(wbuf, ch)
			}
		} else {
			switch ch {
			case ' ':
				if len(wbuf) > 0 {
					s := string(wbuf)
					r = append(r, s)
					wbuf = buf[:0]
				}
			case '"', '\'':
				qch = ch
			default:
				wbuf = append(wbuf, ch)
			}
		}
	}
	if len(wbuf) > 0 {
		s := string(wbuf)
		r = append(r, s)
		wbuf = buf[:0]
	}
	return r
}

func (this *Env) CreateStandardCommands() {
	dir := this.RootDir()
	if dir == nil {
		panic("env RootDir is nil")
	}
	dir.AddCommand(CreateEXITCommand())
	dir.AddCommand(CreateLSCommand())
	dir.AddCommand(CreateCDCommand())
	dir.AddCommand(CreateFCDCommand())
	dir.AddCommand(CreateHELPCommand())
	dir.AddCommand(CreateENVCommand())
	dir.AddCommand(CreateEXECCommand())
	dir.AddCommand(CreateECHOCommand())
	dir.AddCommand(CreateWAITCommand())
}
