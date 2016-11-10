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

package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"golang.org/x/net/context"
)

var (
	defaultMembersPrefix = "/csf/members"
	defaultLeaderSuffix  = "leader"
)

type Member struct {
	// ID is the unique identifier of this Member.
	ID string `json:"id"`

	// Name is a human-readable, non-unique identifier of this Member.
	Name string `json:"name"`

	// PeerURLs represents the HTTP(S) endpoints this Member uses to
	// participate in etcd's consensus protocol.
	PeerURLs []string `json:"peerURLs"`

	// ClientURLs represents the HTTP(S) endpoints on which this Member
	// serves it's client-facing APIs.
	ClientURLs []string `json:"clientURLs"`
}

type memberCollection []Member

func (c *memberCollection) UnmarshalJSON(data []byte) error {
	d := struct {
		Members []Member
	}{}

	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}

	if d.Members == nil {
		*c = make([]Member, 0)
		return nil
	}

	*c = d.Members
	return nil
}

// NewMembersAPI constructs a new MembersAPI that uses HTTP to
// interact with etcd's membership API.
func NewMembersAPI(c Client) MembersAPI {
	return &httpMembersAPI{
		client: c,
	}
}

type MembersAPI interface {
	// List enumerates the current cluster membership.
	List(ctx context.Context) ([]Member, error)

	// Leader gets current leader of the cluster
	Leader(ctx context.Context) (*Member, error)
}

type httpMembersAPI struct {
	client httpClient
}

func (m *httpMembersAPI) List(ctx context.Context) ([]Member, error) {
	req := &membersAPIActionList{}
	resp, body, err := m.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	if err := assertStatusCode(resp.StatusCode, http.StatusOK); err != nil {
		return nil, err
	}

	var mCollection memberCollection
	if err := json.Unmarshal(body, &mCollection); err != nil {
		return nil, err
	}

	return []Member(mCollection), nil
}

func (m *httpMembersAPI) Leader(ctx context.Context) (*Member, error) {
	req := &membersAPIActionLeader{}
	resp, body, err := m.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	if err := assertStatusCode(resp.StatusCode, http.StatusOK); err != nil {
		return nil, err
	}

	var leader Member
	if err := json.Unmarshal(body, &leader); err != nil {
		return nil, err
	}

	return &leader, nil
}

type membersAPIActionList struct{}

func (l *membersAPIActionList) HTTPRequest(ep url.URL) *http.Request {
	u := membersURL(ep)
	// fmt.Printf("sending to %v\n", u)
	req, _ := http.NewRequest("GET", u.String(), nil)
	return req
}

func assertStatusCode(got int, want ...int) (err error) {
	for _, w := range want {
		if w == got {
			return nil
		}
	}
	return fmt.Errorf("unexpected status code %d", got)
}

type membersAPIActionLeader struct{}

func (l *membersAPIActionLeader) HTTPRequest(ep url.URL) *http.Request {
	u := membersURL(ep)
	u.Path = path.Join(u.Path, defaultLeaderSuffix)
	// fmt.Printf("sending to %v\n", u)
	req, _ := http.NewRequest("GET", u.String(), nil)
	return req
}

// v2MembersURL add the necessary path to the provided endpoint
// to route requests to the default v2 members API.
func membersURL(ep url.URL) *url.URL {
	ep.Path = path.Join(ep.Path, defaultMembersPrefix)
	return &ep
}

type membersError struct {
	Message string `json:"message"`
	Code    int    `json:"-"`
}

func (e membersError) Error() string {
	return e.Message
}
