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

// Package setupnode defines a csf sample app.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/httpsc/httpport"
	"github.com/catyguan/csf/pkg/osutil"
	"github.com/catyguan/csf/service/counter"
)

func main1() {

	s := counter.NewCounterService()
	si := core.NewLockerServiceInvoker(s, nil)

	pcfg := &httpport.Config{}
	pcfg.Addr = ":8086"
	pcfg.Host = ""

	mux := core.NewServiceMux()
	mux.AddInvoker(counter.SERVICE_NAME, si)

	port := httpport.NewPort(pcfg)
	err0 := port.Start("/counter", mux, nil)
	if err0 != nil {
		fmt.Printf("start fail - %v", err0)
		return
	}
	port.Run()
	defer port.Stop()

	log.Printf("Server is ready!")

	osutil.HandleInterrupts()

	tt := time.NewTicker(60 * time.Minute)

	select {
	case <-tt.C:
		log.Printf("BYE~~~~~\n")
	}

	osutil.Exit(0)
}
