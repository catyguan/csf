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

package storage4si0admin

import (
	"context"
	"fmt"

	"github.com/catyguan/csf/pkg/capnslog"
)

var (
	plog = capnslog.NewPackageLogger("github.com/catyguan/csf", "storage4si0admin")
)

type AdminAPI interface {
	MakeSnapshot(ctx context.Context) (uint64, error)
}

func DefaultAdminServiceName(s string) string {
	return fmt.Sprintf("%s#admin", s)
}
