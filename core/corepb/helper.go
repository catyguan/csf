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
package corepb

import "errors"

func NewQueryRequest(serviceName, servicePath string, data []byte) *Request {
	r := new(Request)
	r.Info = &RequestInfo{
		ID:          0,
		Type:        RequestType_QUERY,
		ServiceName: serviceName,
		ServicePath: servicePath,
	}
	r.Data = data
	return r
}

func NewExecuteRequest(serviceName, servicePath string, data []byte) *Request {
	r := new(Request)
	r.Info = &RequestInfo{
		ID:          0,
		Type:        RequestType_EXECUTE,
		ServiceName: serviceName,
		ServicePath: servicePath,
	}
	r.Data = data
	return r
}

func NewMessageRequest(serviceName, servicePath string, data []byte) *Request {
	r := new(Request)
	r.Info = &RequestInfo{
		ID:          0,
		Type:        RequestType_MESSAGE,
		ServiceName: serviceName,
		ServicePath: servicePath,
	}
	r.Data = data
	return r
}

func HandleError(resp *Response, err error) error {
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	return resp.ToError()
}

func (this *Response) ToError() error {
	if this.Error == "" {
		return nil
	}
	return errors.New(this.Error)
}

func (this *Request) CreateResponse(data []byte) *Response {
	r := new(Response)
	r.Info = &ResponseInfo{
		RequestID: this.Info.ID,
	}
	r.Data = data
	return r
}

func (this *Request) IsQueryType() bool {
	if this.Info == nil {
		return false
	}
	return this.Info.Type == RequestType_QUERY
}

func (this *Request) IsExecuteType() bool {
	if this.Info == nil {
		return false
	}
	return this.Info.Type == RequestType_EXECUTE
}

func (this *Request) IsMessageType() bool {
	if this.Info == nil {
		return false
	}
	return this.Info.Type == RequestType_MESSAGE
}

func HandleChannelError(cresp *ChannelResponse, err error) error {
	if err != nil {
		return err
	}
	if cresp == nil {
		return nil
	}
	if cresp.Response == nil {
		return nil
	}
	return cresp.Response.ToError()
}

func (this *ChannelResponse) ToError() error {
	if this.Response == nil {
		return nil
	}
	return this.Response.ToError()
}
