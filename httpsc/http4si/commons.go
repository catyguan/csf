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
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/catyguan/csf/core/corepb"
)

type DefaultConverter struct {
}

func (this *DefaultConverter) BuildRequest(url string, creq *corepb.ChannelRequest) (*http.Request, error) {

	data, err0 := creq.Marshal()
	if err0 != nil {
		return nil, err0
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CSF/HttpServiceInvoker")
	req.Header.Set("Content-Type", "application/protobuf")

	return req, nil
}

func (this *DefaultConverter) HandleResponse(resp *http.Response) (*corepb.ChannelResponse, error) {
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error:(%d), %s", resp.StatusCode, string(data))
	}

	r := &corepb.ChannelResponse{}
	err1 := r.Unmarshal(data)
	if err1 != nil {
		return nil, err1
	}

	return r, nil
}
