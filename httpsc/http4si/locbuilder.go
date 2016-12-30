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
	"net/url"
	"time"

	"github.com/catyguan/csf/core"
)

func init() {
	core.RegisterServiceLocationBuilder("http4si", ServiceLocaltionBuilder4HttpInvoker)
	core.RegisterServiceLocationBuilder("http", ServiceLocaltionBuilder4HttpInvoker)
	core.RegisterServiceLocationBuilder("https", ServiceLocaltionBuilder4HttpInvoker)
}

func ServiceLocaltionBuilder4HttpInvoker(sl *core.ServiceLocation, typ, s string) error {
	if typ == "http4si" {
		typ = "http"
	}
	uls := typ + ":" + s
	ul, err0 := url.Parse(uls)
	if err0 != nil {
		return err0
	}

	qs := ul.Query()
	for k, _ := range qs {
		sl.Params[k] = qs.Get(k)
	}
	ul.RawQuery = ""
	sl.ServiceName = sl.GetParam("SERVICE")
	newone := ul.String()

	si, err := CreateServiceInvokerByLocation(newone, sl)
	if err != nil {
		return err
	}
	sl.Invoker = si

	return nil
}

func CreateServiceInvokerByLocation(u string, sl *core.ServiceLocation) (*HttpServiceInvoker, error) {
	dt := time.Duration(0)
	et := time.Duration(0)
	if du, err := time.ParseDuration(sl.GetParam("DIAL")); err == nil {
		dt = du
	}
	if du, err := time.ParseDuration(sl.GetParam("EXEC")); err == nil {
		et = du
	}
	host := sl.GetParam("HOST")
	cfg := &Config{
		URL:             u,
		Host:            host,
		DialTimeout:     dt,
		ExcecuteTimeout: et,
	}
	return NewHttpServiceInvoker(cfg, nil)
}
