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
package http4si

import (
	"net"
	"net/http"
	"time"
)

func NewTransport(cf *Config) (*http.Transport, error) {
	info := &cf.TLSInfo
	dialtimeoutd := cf.DialTimeout
	if dialtimeoutd == 0 {
		dialtimeoutd = defaultDialTimeout
	}

	cfg, err := info.ClientConfig()
	if err != nil {
		return nil, err
	}

	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout: dialtimeoutd,
			// value taken from http.DefaultTransport
			KeepAlive: 30 * time.Second,
		}).Dial,
		// value taken from http.DefaultTransport
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     cfg,
	}
	return t, nil
}

func CreateClient(cf *Config, tr http.RoundTripper) *http.Client {
	if tr == nil {
		tr = http.DefaultTransport
	}
	return &http.Client{Transport: tr}
}
