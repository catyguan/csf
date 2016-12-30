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
package schsign

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/catyguan/csf/core"
	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/pbutil"
)

var (
	HEADER_SIGN = "SIGN"

	ErrVerifyRequestFail = errors.New("request sign verify fail")
	errVFDetail1         = "request sign verify fail, want=%s, got=%s"

	ErrVerifyResponseFail = errors.New("response sign verify fail")
	errVFDetail2          = "response sign verify fail, want=%s, got=%s"
)

const (
	SIGN_REQUEST         = 0x01
	SIGN_RESPONSE        = 0x02
	SIGN_REQUEST_VERIFY  = 0x04
	SIGN_RESPONSE_VERIFY = 0x08
)

type Sign struct {
	key        string
	signType   uint16
	signAll    bool
	errDetail  bool
	headerName string
}

func NewSign(key string, signType uint16, signAll bool) *Sign {
	r := &Sign{key: key, signType: signType, signAll: signAll, headerName: HEADER_SIGN}
	return r
}

func (this *Sign) impl() {
	_ = core.ServiceChannelHandler(this)
}

func (this *Sign) SetHeaderName(n string) *Sign {
	this.headerName = n
	return this
}

func (this *Sign) ErrorShowDetail(v bool) *Sign {
	this.errDetail = v
	return this
}

func (this *Sign) DoSignRequest(creq *corepb.ChannelRequest) []byte {
	data := creq.Data
	if this.signAll {
		data = pbutil.MustMarshal(&creq.Request)
	}
	h := md5.New()
	h.Write([]byte(this.key))
	h.Write(data)
	return h.Sum(nil)
}

func (this *Sign) DoSignResponse(cresp *corepb.ChannelResponse) []byte {
	data := cresp.Data
	if this.signAll {
		data = pbutil.MustMarshal(&cresp.Response)
	}
	h := md5.New()
	h.Write([]byte(this.key))
	h.Write(data)
	return h.Sum(nil)
}

func (this *Sign) HandleRequest(ctx context.Context, si core.ServiceInvoker, creq *corepb.ChannelRequest) (*corepb.ChannelResponse, error) {
	if this.key == "" {
		// skip sign
		return si.InvokeRequest(ctx, creq)
	}
	var err error
	var cresp *corepb.ChannelResponse
	if (this.signType & SIGN_REQUEST) > 0 {
		ldata := this.DoSignRequest(creq)
		creq.SetHeader(this.headerName, ldata)
	} else if (this.signType & SIGN_REQUEST_VERIFY) > 0 {
		ldata := this.DoSignRequest(creq)
		h := creq.GetHeaderInfo(this.headerName)
		if h == nil {
			return nil, ErrVerifyRequestFail
		}
		if !bytes.Equal(ldata, h.Data) {
			if this.errDetail {
				return nil, fmt.Errorf(errVFDetail1, hex.EncodeToString(ldata), hex.EncodeToString(h.Data))
			}
			return nil, ErrVerifyRequestFail
		}
	}

	cresp, err = si.InvokeRequest(ctx, creq)
	if err != nil {
		return nil, err
	}

	if (this.signType & SIGN_RESPONSE) > 0 {
		ldata := this.DoSignResponse(cresp)
		cresp.SetHeader(this.headerName, ldata)
	} else if (this.signType & SIGN_RESPONSE_VERIFY) > 0 {
		ldata := this.DoSignResponse(cresp)
		h := creq.GetHeaderInfo(this.headerName)
		if h == nil {
			return nil, ErrVerifyResponseFail
		}
		if !bytes.Equal(ldata, h.Data) {
			if this.errDetail {
				return nil, fmt.Errorf(errVFDetail2, hex.EncodeToString(ldata), hex.EncodeToString(h.Data))
			}
			return nil, ErrVerifyResponseFail
		}

	}
	return cresp, nil
}
