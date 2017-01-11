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
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/httpsc/http4si"
	"github.com/catyguan/csf/httpsc/httpport"
	"github.com/catyguan/csf/masterslave"
	"github.com/catyguan/csf/pkg/osutil"
	"github.com/catyguan/csf/service/counter"
	"github.com/catyguan/csf/servicechannelhandler/schblocker"
	"github.com/catyguan/csf/servicechannelhandler/schlog"
	"github.com/catyguan/csf/storage4si"
)

func main2() {

	var sport, mport int
	flag.IntVar(&sport, "p", 8087, "service port")
	flag.IntVar(&mport, "m", 8086, "main port")
	flag.Parse()

	storageSize := 32
	ms := storage4si.NewMemoryStorage(storageSize)
	smux := core.NewServiceMux()
	pmux := core.NewServiceMux()
	amux := core.NewServiceMux()

	var ssi *storage4si.StorageServiceContainer

	if true {

		s := counter.NewCounterService()
		cfg := storage4si.NewConfig()
		cfg.SnapCount = storageSize / 2
		cfg.Storage = ms
		cfg.Service = s
		si := storage4si.NewStorageServiceContainer(cfg)
		errS := si.Run()
		if errS != nil {
			fmt.Printf("run StorageServiceInvoker fail - %v", errS)
			return
		}
		defer si.Close()
		ssi = si

		bl := schblocker.NewBlocker().ReadOnly()

		sc := core.NewServiceChannel()
		sc.Next(schlog.NewLogger("TCSERVER"))
		sc.Next(bl)
		sc.Sink(si)

		smux.AddInvoker(counter.SERVICE_NAME, sc)
	}

	if true {

		cfg := &http4si.Config{}
		cfg.URL = fmt.Sprintf("http://localhost:%d/peer", mport)
		cfg.ExcecuteTimeout = 30 * time.Second
		si, err := http4si.NewHttpServiceInvoker(cfg, nil)
		if err != nil {
			fmt.Printf("init err - %v", err)
			return
		}

		cfg2 := masterslave.NewSlaveConfig()
		cfg2.Apply = storage4si.NewSlaveStorageContainerApply(ssi)
		cfg2.Master = masterslave.NewMasterAPI(masterslave.DefaultMasterServiceName(counter.SERVICE_NAME), si)
		service := masterslave.NewSlaveService(cfg2)
		err = service.Run()
		if err != nil {
			fmt.Printf("start slave fail - %v", err)
			return
		}
		defer service.Close()

		si2 := core.NewLockerServiceContainer(service, nil)

		sc := core.NewServiceChannel()
		sc.Next(schlog.NewLogger("SLAVE"))
		sc.Sink(si2)

		pmux.AddInvoker(masterslave.DefaultSlaveServiceName(counter.SERVICE_NAME), sc)
	}

	if true {
		cfg := masterslave.NewMasterConfig()
		cfg.Master = ms.(masterslave.MasterNode)
		service := masterslave.NewMasterService(cfg)
		si := core.NewLockerServiceContainer(service, nil)

		sc := core.NewServiceChannel()
		sc.Sink(si)

		pmux.AddInvoker(masterslave.DefaultMasterServiceName(counter.SERVICE_NAME), sc)
	}

	hmux := http.NewServeMux()

	pcfg := &httpport.Config{}
	pcfg.Addr = fmt.Sprintf(":%d", sport)
	pcfg.Host = ""

	port := httpport.NewPort(pcfg)
	port.BuildHttpMux(hmux, "/service", smux, nil)
	port.BuildHttpMux(hmux, "/peer", pmux, nil)
	port.BuildHttpMux(hmux, "/admin", amux, nil)

	err0 := port.StartServe(hmux)
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
