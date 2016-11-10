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
	"github.com/catyguan/csf/cluster/clusterapi"
	"github.com/catyguan/csf/interfaces"
	"github.com/catyguan/csf/pkg/osutil"
	"github.com/catyguan/csf/samples/spcommons"
	"github.com/catyguan/csf/wal"
)

func main() {
	wal.SegmentSizeBytes = 16 * 1024

	var configFile string
	flag.StringVar(&configFile, "C", "", "Path to the server configuration file")
	flag.Parse()

	if configFile == "" {
		fmt.Printf("config file -C invalid")
		os.Exit(-1)
	}

	pcfg := cluster.NewConfig()
	pcfg.EnablePprof = true

	cfg, err := cluster.BuildConfigFromFile(configFile, pcfg)
	if err != nil {
		log.Fatal(err)
		os.Exit(3)
	}

	shub := make([]interfaces.Service, 1)
	shub[0] = &spcommons.Counter{}
	cfg.ClientHandlerFactory = clusterapi.NewClientHandler

	node, err := cluster.StartNode(cfg, shub)
	if err != nil {
		log.Fatal(err)
		os.Exit(4)
	}
	defer node.Close()

	if !node.OnReady(60 * time.Second) {
		node.Close()
		log.Printf("Server took too long to start!")
		os.Exit(5)
	}
	log.Printf("Server is ready!")

	osutil.HandleInterrupts()

	stopped := node.StopNotify()
	errc := node.Err()

	tt := time.NewTicker(60 * time.Minute)

	select {
	case <-tt.C:
		log.Printf("BYE~~~~~\n")
	case lerr := <-errc:
		// fatal out on listener errors
		log.Fatal(lerr)
	case <-stopped:
	}

	osutil.Exit(0)

}
