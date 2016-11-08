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

// Package httptypes defines how etcd's HTTP API entities are serialized to and
// deserialized from JSON.
package clusterapi

import "encoding/json"

type apiMember struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	PeerURLs   []string `json:"peerURLs"`
	ClientURLs []string `json:"clientURLs"`
}

type apiMemberCollection []apiMember

func (c *apiMemberCollection) MarshalJSON() ([]byte, error) {
	d := struct {
		Members []apiMember `json:"members"`
	}{
		Members: []Member(*c),
	}

	return json.Marshal(d)
}
