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
	"flag"
	"time"
)

type Flags []Flag

func (c Flags) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}

func (c Flags) Len() int {
	return len(c)
}

func (c Flags) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

type noneWriter struct {
}

func (c *noneWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (c Flags) Parse(args []string) (map[string]interface{}, []string, error) {
	vars := make(map[string]interface{})

	fs := flag.NewFlagSet("", flag.ContinueOnError)
	for _, f := range c {
		var p interface{}
		switch f.Type {
		case "bool":
			dv := false
			if v, ok := f.Default.(bool); ok {
				dv = v
			}
			p = fs.Bool(f.Name, dv, f.Usage)
		case "string":
			dv := ""
			if v, ok := f.Default.(string); ok {
				dv = v
			}
			p = fs.String(f.Name, dv, f.Usage)
		case "int":
			dv := 0
			if v, ok := f.Default.(int); ok {
				dv = v
			}
			p = fs.Int(f.Name, dv, f.Usage)
		case "int64":
			dv := int64(0)
			if v, ok := f.Default.(int64); ok {
				dv = v
			}
			p = fs.Int64(f.Name, dv, f.Usage)
		case "float":
			dv := float64(0)
			if v, ok := f.Default.(float64); ok {
				dv = v
			}
			p = fs.Float64(f.Name, dv, f.Usage)
		case "duration":
			dv := time.Duration(0)
			if v, ok := f.Default.(time.Duration); ok {
				dv = v
			}
			if v, ok := f.Default.(string); ok {
				v2, err2 := time.ParseDuration(v)
				if err2 != nil {
					panic("pares duration fail - " + v)
				}
				dv = v2
			}
			p = fs.Duration(f.Name, dv, f.Usage)
		default:
			// skip
			panic("unknow flag type '" + f.Type + "'")
		}
		if p != nil {
			vars[f.Name] = p
		}
	}
	fs.SetOutput(&noneWriter{})
	err := fs.Parse(args)
	if err != nil {
		return nil, nil, err
	}

	r := make(map[string]interface{})
	for k, v := range vars {
		var nv interface{}
		switch p := v.(type) {
		case *bool:
			nv = *p
		case *int:
			nv = *p
		case *int64:
			nv = *p
		case *float64:
			nv = *p
		case *time.Duration:
			nv = *p
		case *string:
			nv = *p
		}
		r[k] = nv
	}

	return r, fs.Args(), nil
}

type Flag struct {
	Name    string
	Type    string
	Default interface{}
	Usage   string
}
