// Copyright 2015 The csf Authors
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

// Package ctlv2 contains the main entry point for the etcdctl for v2 API.
package csfctl

import (
	"fmt"
	"os"
	"time"

	"github.com/catyguan/csf/csfctl/command"
	"github.com/catyguan/csf/version"
	"github.com/urfave/cli"
)

type Config struct {
	AppName    string
	APIVersion string
	Flags      []cli.Flag
	Commands   []cli.Command
}

func Start(cfg *Config) {
	app := cli.NewApp()
	if cfg.AppName == "" {
		app.Name = "csfctl"
	} else {
		app.Name = cfg.AppName
	}

	app.Version = version.Version
	apiver := cfg.APIVersion
	if apiver == "" {
		apiver = "unknow"
	}
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Fprintf(c.App.Writer, "csfctl version: %v\n", c.App.Version)
		fmt.Fprintf(c.App.Writer, "API version: %v\n", apiver)
	}
	app.Usage = "A simple command line client for " + app.Name + "."
	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "debug", Usage: "output cURL commands which can be used to reproduce the request"},
		cli.BoolFlag{Name: "no-sync", Usage: "don't synchronize cluster information before sending request"},
		cli.StringFlag{Name: "peers, C", Value: "", Usage: "DEPRECATED - \"--endpoints\" should be used instead"},
		cli.StringFlag{Name: "endpoint", Value: "", Usage: "DEPRECATED - \"--endpoints\" should be used instead"},
		cli.StringFlag{Name: "endpoints", Value: "", Usage: "a comma-delimited list of machine addresses in the cluster (default: \"http://127.0.0.1:2379,http://127.0.0.1:4001\")"},
		cli.StringFlag{Name: "cert-file", Value: "", Usage: "identify HTTPS client using this SSL certificate file"},
		cli.StringFlag{Name: "key-file", Value: "", Usage: "identify HTTPS client using this SSL key file"},
		cli.StringFlag{Name: "ca-file", Value: "", Usage: "verify certificates of HTTPS-enabled servers using this CA bundle"},
		cli.StringFlag{Name: "username, u", Value: "", Usage: "provide username[:password] and prompt if password is not supplied."},
		cli.DurationFlag{Name: "timeout", Value: 2 * time.Second, Usage: "connection timeout per request"},
		cli.DurationFlag{Name: "total-timeout", Value: 5 * time.Second, Usage: "timeout for the command execution (except watch)"},
	}
	if cfg.Flags != nil {
		app.Flags = append(app.Flags, cfg.Flags...)
	}

	app.Commands = []cli.Command{
		command.NewClusterHealthCommand(),
		command.NewMemberCommand(),
	}
	if cfg.Commands != nil {
		app.Commands = append(app.Commands, cfg.Commands...)
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
