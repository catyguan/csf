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
	"net/url"
	"strconv"
	"strings"
)

// writeError logs and writes the given Error to the ResponseWriter
// If Error is an etcdErr, it is rendered to the ResponseWriter
// Otherwise, it is assumed to be a StatusInternalServerError
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}
	switch e := err.(type) {
	// case *stdError.Error:
	// 	e.WriteTo(w)
	case *HTTPError:
		if et := e.WriteTo(w); et != nil {
			plog.Debugf("error writing HTTPError (%v) to %s", et, r.RemoteAddr)
		}
	// case auth.Error:
	// 	herr := NewHTTPError(e.HTTPStatus(), e.Error())
	// 	if et := herr.WriteTo(w); et != nil {
	// 		plog.Debugf("error writing HTTPError (%v) to %s", et, r.RemoteAddr)
	// 	}
	default:
		switch err {
		default:
			plog.Warningf("got unexpected response error (%v)", err)
		}
		herr := NewHTTPError(http.StatusInternalServerError, "Internal Server Error")
		if et := herr.WriteTo(w); et != nil {
			plog.Debugf("error writing HTTPError (%v) to %s", et, r.RemoteAddr)
		}
	}
}

// allowMethod verifies that the given method is one of the allowed methods,
// and if not, it writes an error to w.  A boolean is returned indicating
// whether or not the method is allowed.
func AllowMethod(w http.ResponseWriter, m string, ms ...string) bool {
	for _, meth := range ms {
		if m == meth {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(ms, ","))
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	return false
}

// getUint64 extracts a uint64 by the given key from a Form. If the key does
// not exist in the form, 0 is returned. If the key exists but the value is
// badly formed, an error is returned. If multiple values are present only the
// first is considered.
func GetUint64(form url.Values, key string) (i uint64, err error) {
	if vals, ok := form[key]; ok {
		i, err = strconv.ParseUint(vals[0], 10, 64)
	}
	return
}

// getBool extracts a bool by the given key from a Form. If the key does not
// exist in the form, false is returned. If the key exists but the value is
// badly formed, an error is returned. If multiple values are present only the
// first is considered.
func GetBool(form url.Values, key string) (b bool, err error) {
	if vals, ok := form[key]; ok {
		b, err = strconv.ParseBool(vals[0])
	}
	return
}

// trimPrefix removes a given prefix and any slash following the prefix
// e.g.: trimPrefix("foo", "foo") == trimPrefix("foo/", "foo") == ""
func TrimPrefix(p, prefix string) (s string) {
	s = strings.TrimPrefix(p, prefix)
	s = strings.TrimPrefix(s, "/")
	return
}
