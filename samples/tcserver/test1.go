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
	"os"
	"path/filepath"
	"time"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/httpsc/httpport"
	"github.com/catyguan/csf/masterslave"
	"github.com/catyguan/csf/pkg/osutil"
	"github.com/catyguan/csf/service/counter"
	"github.com/catyguan/csf/servicechannelhandler/schlog"
	"github.com/catyguan/csf/servicechannelhandler/schsign"
	"github.com/catyguan/csf/storage4si"
	"github.com/catyguan/csf/storage4si/storage4si0admin"
	"github.com/catyguan/csf/storage4si/walstorage"
)

func main1() {

	smux := core.NewServiceMux()
	pmux := core.NewServiceMux()
	amux := core.NewServiceMux()
	var ms storage4si.Storage
	var snapcount int
	var dir string

	cdir, errDir := os.Getwd()
	if errDir != nil {
		fmt.Printf("error - %s", cdir)
		return
	}

	flag.IntVar(&snapcount, "sc", 16, "snapcount")
	flag.StringVar(&dir, "d", filepath.Join(cdir, "wal"), "WAL dir")
	flag.Parse()

	if false {
		storageSize := snapcount * 2
		ms = storage4si.NewMemoryStorage(storageSize)
	}

	if true {
		scfg := walstorage.NewConfig()
		scfg.Dir = dir
		scfg.BlockRollSize = 16 * 1024
		scfg.Symbol = "tcserver"

		ws, err := walstorage.NewWALStorage(scfg)
		if err != nil {
			fmt.Printf("open WALStorage fail - %v", err)
			return
		}
		defer ws.Close()
		ms = ws
	}

	if true {
		s := counter.NewCounterService()
		cfg := storage4si.NewConfig()
		cfg.SnapCount = snapcount
		cfg.Storage = ms
		cfg.Service = s
		si := storage4si.NewStorageServiceContainer(cfg)
		errS := si.Run()
		if errS != nil {
			fmt.Printf("run StorageServiceInvoker fail - %v", errS)
			return
		}
		defer si.Close()

		sc := core.NewServiceChannel()
		sc.Next(schlog.NewLogger("TCSERVER"))
		sc.Sink(si)

		smux.AddInvoker(counter.SERVICE_NAME, sc)

		as := storage4si0admin.NewAdminService(si)
		sc2 := core.NewServiceChannel()
		sc2.Next(schlog.NewLogger("STORAGE_ADMIN"))
		sc2.Next(schsign.NewSign("123456", schsign.SIGN_REQUEST_VERIFY|schsign.SIGN_RESPONSE, false))
		sc2.Sink(core.NewSimpleServiceContainer(as))
		amux.AddInvoker(storage4si0admin.DefaultAdminServiceName(counter.SERVICE_NAME), sc2)
	}

	if true {
		cfg := masterslave.NewMasterConfig()
		cfg.Master = ms.(masterslave.MasterNode)
		service := masterslave.NewMasterService(cfg)
		si := core.NewSimpleServiceContainer(service)

		sc := core.NewServiceChannel()
		// sc.Next(schlog.NewLogger("MASTER"))
		sc.Sink(si)

		pmux.AddInvoker(masterslave.DefaultMasterServiceName(counter.SERVICE_NAME), sc)
	}

	hmux := http.NewServeMux()

	pcfg := &httpport.Config{}
	pcfg.Addr = ":8086"
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
