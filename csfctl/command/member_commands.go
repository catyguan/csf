// Copyright 2015 The etcd Authors
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

package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli"
)

func NewMemberCommand() cli.Command {
	return cli.Command{
		Name:  "member",
		Usage: "member list",
		Subcommands: []cli.Command{
			{
				Name:      "list",
				Usage:     "enumerate existing cluster members",
				ArgsUsage: " ",
				Action:    actionMemberList,
			},
		},
	}
}

func actionMemberList(c *cli.Context) error {
	if len(c.Args()) != 0 {
		fmt.Fprintln(os.Stderr, "No arguments accepted")
		os.Exit(1)
	}
	mAPI := mustNewMembersAPI(c)
	ctx, cancel := ContextWithTotalTimeout(c)
	defer cancel()

	members, err := mAPI.List(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	leader, err := mAPI.Leader(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to get leader: ", err)
		os.Exit(1)
	}

	for _, m := range members {
		isLeader := false
		if m.ID == leader.ID {
			isLeader = true
		}
		if len(m.Name) == 0 {
			fmt.Printf("%s[unstarted]: peerURLs=%s\n", m.ID, strings.Join(m.PeerURLs, ","))
		} else {
			fmt.Printf("%s: name=%s peerURLs=%s clientURLs=%s isLeader=%v\n", m.ID, m.Name, strings.Join(m.PeerURLs, ","), strings.Join(m.ClientURLs, ","), isLeader)
		}
	}

	return nil
}
