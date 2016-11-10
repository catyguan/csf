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

package interfaces

import (
	"net/http"
)

type User struct {
	Name string
}

func (this *User) String() string {
	return this.Name
}

func UserFromClientCertificate(sec ACL, req *http.Request) *User {
	if req.TLS == nil {
		return nil
	}

	for _, chains := range req.TLS.VerifiedChains {
		for _, chain := range chains {
			plog.Debugf("auth: found common name %s.\n", chain.Subject.CommonName)
			user := &User{}
			user.Name = chain.Subject.CommonName
			return user
		}
	}
	return nil
}

func UserFromBasicAuth(sec ACL, req *http.Request) *User {
	username, password, ok := req.BasicAuth()
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

func CheckCSFAccess(sm ServiceManager, req *http.Request) bool {
	return hasCSFAccess(sm.GetACL(), req, sm.ClientCertAuthEnabled())
}

func hasCSFAccess(sec ACL, req *http.Request, clientCertAuthEnabled bool) bool {
	if sec == nil {
		// No store means no auth available, eg, tests.
		return true
	}
	if req.Method == "HEAD" {
		return true
	}
	if !sec.AuthEnabled() {
		return true
	}

	var user *User
	if req.Header.Get("Authorization") == "" && clientCertAuthEnabled {
		user = UserFromClientCertificate(sec, req)
		if user == nil {
			return false
		}
	} else {
		user = UserFromBasicAuth(sec, req)
		if user == nil {
			return false
		}
	}

	if sec.HasAccess(user.Name, "//csf/", "r", "") {
		return true
	}
	plog.Warningf("auth: user %s does not CSFAccess for resource %s.", user, req.URL.Path)
	return false
}

func CheckAccess(sm ServiceManager, req *http.Request, res, act, content string) bool {
	sec := sm.GetACL()
	if sec == nil {
		// No store means no auth available, eg, tests.
		return true
	}
	if req.Method == "HEAD" {
		return true
	}
	if !sec.AuthEnabled() {
		return true
	}

	var user *User
	if req.Header.Get("Authorization") == "" && sm.ClientCertAuthEnabled() {
		user = UserFromClientCertificate(sec, req)
		if user == nil {
			return false
		}
	} else {
		user = UserFromBasicAuth(sec, req)
		if user == nil {
			return false
		}
	}

	if sec.HasAccess(user.Name, res, act, content) {
		return true
	}
	plog.Warningf("auth: user %s does not for resource %s:%s:%s", user, res, act, content)
	return false
}

func WriteNoAuth(w http.ResponseWriter, r *http.Request) {
	herr := NewHTTPError(http.StatusUnauthorized, "Insufficient credentials")
	if err := herr.WriteTo(w); err != nil {
		plog.Debugf("error writing HTTPError (%v) to %s", err, r.RemoteAddr)
	}
}
