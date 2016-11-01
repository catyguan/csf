// Copyright 2016 The CSF Authors
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

// Package etcdlike defines a csf app just like etcd.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/catyguan/csf/cluster"
	"github.com/catyguan/csf/pkg/osutil"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "C", "", "Path to the server configuration file")
	flag.Parse()

	if configFile == "" {
		fmt.Printf("config file -C invalid")
		os.Exit(-1)
	}

	cfg, err := cluster.ConfigFromFile(configFile)
	if err != nil {
		log.Fatal(err)
		os.Exit(3)
	}

	node, err := cluster.SetupNode(cfg)
	if err != nil {
		log.Fatal(err)
		os.Exit(4)
	}
	node.Start()
	defer node.Stop()

	if !node.OnReady(60 * time.Second) {
		node.Stop()
		log.Printf("Server took too long to start!")
		os.Exit(5)
	}
	log.Printf("Server is ready!")

	osutil.HandleInterrupts()

	time.Sleep(60 * time.Second)

	// if !cl.OnReady(60 * time.Second) {
	// 	cl.Server.Stop() // trigger a shutdown
	// 	log.Printf("Server took too long to start!")
	// 	os.Exit(5)
	// }
	// log.Printf("Server is ready!")

	// stopped := cl.Server.StopNotify()
	// errc := cl.Err()

	// osutil.HandleInterrupts()

	// select {
	// case lerr := <-errc:
	// 	// fatal out on listener errors
	// 	log.Fatal(lerr)
	// case <-stopped:
	// }

	osutil.Exit(0)

}
