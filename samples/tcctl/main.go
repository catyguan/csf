// Copyright 2016 The etcd Authors
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

// etcdctl is a command line application that controls etcd.
package main

import (
	"os"

	"github.com/catyguan/csf/csfctl"
	_ "github.com/catyguan/csf/httpsc/http4si"
	_ "github.com/catyguan/csf/servicechannelhandler/schsign"
)

func main() {
	// for _, s := range os.Args {
	// 	fmt.Println(s)
	// }

	root := csfctl.NewRootEnv()
	BuildRootEnv2(root)
	rdir := root.RootDir()
	rdir.CreateDir("test")
	root.RunAsConsole()
	os.Exit(1)
}
