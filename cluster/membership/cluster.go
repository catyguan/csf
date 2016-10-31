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

package membership

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/catyguan/csf/pkg/types"
	"github.com/catyguan/csf/semver"
)

// RaftCluster is a list of Members that belong to the same raft cluster
type RaftCluster struct {
	id    types.ID
	name  string
	token string

	sync.Mutex // guards the fields below
	version    *semver.Version
	members    map[types.ID]*Member
	// removed contains the ids of removed members in the cluster.
	// removed id cannot be reused.
	removed map[types.ID]bool
}

func NewClusterFromMembers(name, token string, membs []*Member, version string) *RaftCluster {
	c := newCluster()
	c.name = name
	c.token = token
	ver, _ := semver.NewVersion(version)
	c.version = ver

	for _, m := range membs {
		c.members[m.ID] = m
	}

	var b []byte
	b = append(b, []byte(name)...)
	b = append(b, []byte("-")...)
	b = append(b, []byte(token)...)
	hash := sha1.Sum(b)
	c.id = types.ID(binary.BigEndian.Uint64(hash[:8]))

	return c
}

func newCluster() *RaftCluster {
	return &RaftCluster{
		members: make(map[types.ID]*Member),
		removed: make(map[types.ID]bool),
	}
}

func (c *RaftCluster) ID() types.ID { return c.id }

func (c *RaftCluster) Members() []*Member {
	c.Lock()
	defer c.Unlock()
	var ms MembersByID
	for _, m := range c.members {
		ms = append(ms, m.Clone())
	}
	sort.Sort(ms)
	return []*Member(ms)
}

func (c *RaftCluster) Member(id types.ID) *Member {
	c.Lock()
	defer c.Unlock()
	m := c.members[id]
	if m != nil {
		return m.Clone()
	}
	plog.Panicf("not found member id=%v", id)
	return nil
}

// MemberByName returns a Member with the given name if exists.
// If more than one member has the given name, it will panic.
func (c *RaftCluster) MemberByName(name string) *Member {
	c.Lock()
	defer c.Unlock()
	var memb *Member
	for _, m := range c.members {
		if m.Name == name {
			if memb != nil {
				plog.Panicf("two members with the given name %q exist", name)
			}
			memb = m
		}
	}
	return memb.Clone()
}

func (c *RaftCluster) MemberIDs() []types.ID {
	c.Lock()
	defer c.Unlock()
	var ids []types.ID
	for _, m := range c.members {
		ids = append(ids, m.ID)
	}
	sort.Sort(types.IDSlice(ids))
	return ids
}

func (c *RaftCluster) IsIDRemoved(id types.ID) bool {
	c.Lock()
	defer c.Unlock()
	return c.removed[id]
}

// PeerURLs returns a list of all peer addresses.
// The returned list is sorted in ascending lexicographical order.
func (c *RaftCluster) PeerURLs() types.URLs {
	c.Lock()
	defer c.Unlock()
	urls := make(types.URLs, 0)
	for _, p := range c.members {
		urls = append(urls, p.PeerURLs...)
	}
	urls.Sort()
	return urls
}

// ClientURLs returns a list of all client addresses.
// The returned list is sorted in ascending lexicographical order.
func (c *RaftCluster) ClientURLs() types.URLs {
	c.Lock()
	defer c.Unlock()
	urls := make(types.URLs, 0)
	for _, p := range c.members {
		urls = append(urls, p.ClientURLs...)
	}
	urls.Sort()
	return urls
}

func (c *RaftCluster) String() string {
	c.Lock()
	defer c.Unlock()
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "{ClusterID:%s ", c.id)
	fmt.Fprintf(b, "{ClusterName:%s ", c.name)
	var ms []string
	for _, m := range c.members {
		ms = append(ms, fmt.Sprintf("%v:%v", m.ID, m.Name))
	}
	fmt.Fprintf(b, "Members:[%s] ", strings.Join(ms, " "))
	var ids []string
	for id := range c.removed {
		ids = append(ids, fmt.Sprintf("%s", id))
	}
	fmt.Fprintf(b, "RemovedMemberIDs:[%s]}", strings.Join(ids, " "))
	return b.String()
}

func (c *RaftCluster) SetID(id types.ID) { c.id = id }

func (c *RaftCluster) Version() *semver.Version {
	c.Lock()
	defer c.Unlock()
	if c.version == nil {
		return nil
	}
	return semver.Must(semver.NewVersion(c.version.String()))
}
