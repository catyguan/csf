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

// Package raftservertc defines a test app for raftserver.
package main

import (
	"flag"
	"log"

	"github.com/catyguan/csf/wal"
)

var (
	testDir string = "c:\\tmp"
)

func main() {
	wal.SegmentSizeBytes = 16 * 1024

	var configFile string
	flag.StringVar(&configFile, "C", "", "Path to the server configuration file")
	flag.Parse()

	// do_Test_LeaderDisconn()
	// do_Test_Simple()
	do_Test_WAL_Simple()

	log.Printf("BYE~~~~~\n")
}
