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
	"fmt"
	"strconv"
	"strings"

	"github.com/catyguan/csf/core"
)

func init() {
	core.RegisterServiceLocationBuilder("sign", ServiceLocaltionBuilder4Sign)
}

func ServiceLocaltionBuilder4Sign(sl *core.ServiceLocation, typ, s string) error {
	slist := strings.SplitN(s, ",", 2)
	if len(slist) != 2 {
		return fmt.Errorf("invalid sign format 'sign:sign_content,invoker_loc'")
	}
	splist := strings.Split(slist[0], ";")
	key := splist[0]
	signType := uint16(SIGN_REQUEST)
	if len(splist) > 1 {
		i, errP := strconv.ParseInt(splist[1], 0, 16)
		if errP != nil {
			return fmt.Errorf("signType invalid - %v", errP)
		}
		signType = uint16(i)
	}
	signAll := false
	if len(splist) > 2 {
		v, errP := strconv.ParseBool(splist[2])
		if errP != nil {
			return fmt.Errorf("signAll invalid - %v", errP)
		}
		signAll = v
	}

	s2 := slist[1]
	ssl, errS := core.ParseLocation(s2)
	if errS != nil {
		return errS
	}
	sl.Copy(ssl)

	sc, ok := sl.Invoker.(*core.ServiceChannel)
	if !ok {
		sc = core.NewServiceChannel()
		sc.Sink(sl.Invoker)
		sl.Invoker = sc
	}

	sign := NewSign(key, signType, signAll)
	sc.Begin(sign)
	return nil
}
