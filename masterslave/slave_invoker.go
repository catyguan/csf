// Copyright 2016 The CSF Authors
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

package masterslave

import "github.com/catyguan/csf/core"

type SlaveInvoker struct {
	sn string
	si core.ServiceInvoker
}

func NewSlaveAPI(serviceName string, si core.ServiceInvoker) SlaveAPI {
	return &SlaveInvoker{sn: serviceName, si: si}
}

// BEGIN: 业务

// END: 业务
