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

import (
	"context"
	"fmt"
	"time"

	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/pkg/capnslog"
	"github.com/catyguan/csf/storage4si"
)

var (
	plog = capnslog.NewPackageLogger("github.com/catyguan/csf", "service-master_slave")
)

type MasterAPI interface {
	Begin(ctx context.Context) (uint64, error)
	End(ctx context.Context, sid uint64) error
	LastSnapshot(ctx context.Context, sid uint64) (uint64, []byte, error)
	Process(ctx context.Context, sid uint64) ([]*corepb.Request, error)
}

type MasterConfig struct {
	Storage       storage4si.Storage
	QuerySize     int
	PullWaitTime  time.Duration
	SessionExpire time.Duration
}

func NewMasterConfig() *MasterConfig {
	r := new(MasterConfig)
	r.QuerySize = 16
	r.PullWaitTime = 1 * time.Second
	r.SessionExpire = 5 * time.Second
	return r
}

func DefaultMasterServiceName(s string) string {
	return fmt.Sprintf("%s#master", s)
}
