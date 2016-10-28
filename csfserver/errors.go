// Copyright 2015 The etcd Authors
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

package csfserver

import (
	"errors"
	"fmt"
)

var (
	ErrUnknownMethod              = errors.New("csfserver: unknown method")
	ErrStopped                    = errors.New("csfserver: server stopped")
	ErrCanceled                   = errors.New("csfserver: request cancelled")
	ErrTimeout                    = errors.New("csfserver: request timed out")
	ErrTimeoutDueToLeaderFail     = errors.New("csfserver: request timed out, possibly due to previous leader failure")
	ErrTimeoutDueToConnectionLost = errors.New("csfserver: request timed out, possibly due to connection lost")
	ErrTimeoutLeaderTransfer      = errors.New("csfserver: request timed out, leader transfer took too long")
	ErrNotEnoughStartedMembers    = errors.New("csfserver: re-configuration failed due to not enough started members")
	ErrNoLeader                   = errors.New("csfserver: no leader")
	ErrRequestTooLarge            = errors.New("csfserver: request is too large")
	ErrNoSpace                    = errors.New("csfserver: no space")
	ErrInvalidAuthToken           = errors.New("csfserver: invalid auth token")
	ErrTooManyRequests            = errors.New("csfserver: too many requests")
	ErrUnhealthy                  = errors.New("csfserver: unhealthy cluster")
)

type DiscoveryError struct {
	Op  string
	Err error
}

func (e DiscoveryError) Error() string {
	return fmt.Sprintf("failed to %s discovery cluster (%v)", e.Op, e.Err)
}
