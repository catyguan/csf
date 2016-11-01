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

package cluster

import (
	"fmt"
	"path"
	"time"
)

// VerifyBootstrap sanity-checks the initial config for bootstrap case
// and returns an error for things that should never happen.
func (c *Config) VerifyBootstrap() error {
	if err := c.verifyLocalMember(); err != nil {
		return err
	}
	done, durl := checkDuplicateURL(c.ClusterPeers)
	if done {
		return fmt.Errorf("initial cluster %s has duplicate url", durl)
	}
	return nil
}

// verifyLocalMember verifies the configured member is in configured
// cluster. If strict is set, it also verifies the configured member
// has the same peer urls as configured advertised peer urls.
func (c *Config) verifyLocalMember() error {
	if c.ClusterPeers == nil {
		return fmt.Errorf("cluster peers is nil")
	}
	m := false
	for _, peer := range c.ClusterPeers {
		if peer.Name == c.Name {
			m = true
			break
		}
	}
	// Make sure the cluster at least contains the local server.
	if !m {
		return fmt.Errorf("couldn't find local name %q in the initial cluster configuration", c.Name)
	}
	return nil
}

func (c *Config) MemberDir() string { return path.Join(c.Dir, "member") }

func (c *Config) WALDir() string {
	if c.WalDir != "" {
		return c.WalDir
	}
	return path.Join(c.MemberDir(), "wal")
}

func (c *Config) SnapDir() string { return path.Join(c.MemberDir(), "snap") }

// ReqTimeout returns timeout for request to finish.
func (c *Config) ReqTimeout() time.Duration {
	// 5s for queue waiting, computation and disk IO delay
	// + 2 * election timeout for possible leader election
	return 5*time.Second + 2*time.Duration(c.ElectionTicks())*time.Duration(c.TickMs)*time.Millisecond
}

func (c *Config) electionTimeout() time.Duration {
	return time.Duration(c.ElectionTicks()) * time.Duration(c.TickMs) * time.Millisecond
}

func (c *Config) peerDialTimeout() time.Duration {
	// 1s for queue wait and system delay
	// + one RTT, which is smaller than 1/5 election timeout
	return time.Second + time.Duration(c.ElectionTicks())*time.Duration(c.TickMs)*time.Millisecond/5
}

func (c *Config) PrintWithInitial() { c.print(true) }

func (c *Config) Print() { c.print(false) }

func (c *Config) print(initial bool) {
	plog.Infof("cluster = %s", c.ClusterName)
	plog.Infof("node = %s", c.Name)
	plog.Infof("peerURL = %s", c.LPUrls.StringSlice())

	plog.Infof("data dir = %s", c.Dir)
	plog.Infof("member dir = %s", c.MemberDir())
	if c.WalDir != "" {
		plog.Infof("WAL dir = %s", c.WalDir)
	}
	plog.Infof("heartbeat = %dms", c.TickMs)
	plog.Infof("election = %dms", c.ElectionTicks()*int(c.TickMs))
	plog.Infof("snapshot count = %d", c.SnapCount)

	// plog.Infof("client URLs = %s", c.ClientURLs)
}

func checkDuplicateURL(ps []*PeerConfig) (bool, string) {
	um := make(map[string]bool)
	for _, peer := range ps {
		if peer.PeerURL == nil {
			continue
		}
		for _, ul := range peer.PeerURL {
			u := ul.String()
			if um[u] {
				return true, u
			}
			um[u] = true
		}
	}
	return false, ""
}
