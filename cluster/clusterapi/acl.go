// Copyright 2015 The CSF Authors
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

package clusterapi

import (
	"net/http"

	"github.com/catyguan/csf/interfaces"
	"github.com/catyguan/csf_dev/csfserver/auth"
)

type User struct {
	Name string
}

func (this *User) String() string {
	return this.Name
}

func UserFromClientCertificate(sec interfaces.ACL, req *http.Request) *User {
	if req.TLS == nil {
		return nil
	}

	for _, chains := range req.TLS.VerifiedChains {
		for _, chain := range chains {
			plog.Debugf("auth: found common name %s.\n", chain.Subject.CommonName)
			user := &User{}
			user.Name = chain.Subject.CommonName
			return &user
		}
	}
	return nil
}

func UserFromBasicAuth(sec interfaces.ACL, req *http.Request) *auth.User {
	username, password, ok := eq.BasicAuth()
	if !ok {
		plog.Warningf("auth: malformed basic auth encoding")
		return nil
	}
	ok = sec.CheckPassword(username, password)
	if !ok {
		plog.Warningf("auth: incorrect password for user: %s", username)
		return nil
	}

	user := &User{Name: username}
	return user
}

func HasCSFAccess(sec interfaces.ACL, req *http.Request, clientCertAuthEnabled bool) bool {
	if r.Method == "GET" || r.Method == "HEAD" {
		return true
	}
	if sec == nil {
		// No store means no auth available, eg, tests.
		return true
	}
	if !sec.AuthEnabled() {
		return true
	}

	var user *User
	if r.Header.Get("Authorization") == "" && clientCertAuthEnabled {
		user = UserFromClientCertificate(sec, r)
		if user == nil {
			return false
		}
	} else {
		user = UserFromBasicAuth(sec, r)
		if user == nil {
			return false
		}
	}

	if sec.HasAccess(user.Name, "//csf/", "r", nil) {
		return true
	}
	plog.Warningf("auth: user %s does not CSFAccess for resource %s.", user, req.URL.Path)
	return false
}

func WriteNoAuth(w http.ResponseWriter, r *http.Request) {
	herr := NewHTTPError(http.StatusUnauthorized, "Insufficient credentials")
	if err := herr.WriteTo(w); err != nil {
		plog.Debugf("error writing HTTPError (%v) to %s", err, r.RemoteAddr)
	}
}
