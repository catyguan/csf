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
package httpport

import (
	"io/ioutil"
	"net/http"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/capnslog"
)

var (
	plog = capnslog.NewPackageLogger("github.com/catyguan/csf", "httpport")
)

type DefaultConverter struct {
}

func (this *DefaultConverter) BuildRequest(hreq *http.Request) (*corepb.ChannelRequest, error) {
	defer hreq.Body.Close()
	b, err := ioutil.ReadAll(hreq.Body)
	if err != nil {
		return nil, err
	}
	r := &corepb.ChannelRequest{}
	err = r.Unmarshal(b)
	if err != nil {
		return nil, err
	}
	core.CommonHeaders.SetRemoteAddr(r, hreq.RemoteAddr)
	return r, nil
}

func (this *DefaultConverter) WriteResponse(w http.ResponseWriter, resp *corepb.ChannelResponse) error {
	data, err := resp.Marshal()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/protobuf")
	w.Write(data)
	return nil
}
