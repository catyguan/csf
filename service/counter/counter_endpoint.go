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

package counter

import "sync"

type counterEP struct {
	data map[string]uint64
	mu   sync.RWMutex
}

func (this *counterEP) GetValue(key string) uint64 {
	this.mu.RLock()
	defer this.mu.RUnlock()
	if v, ok := this.data[key]; ok {
		return v
	}
	return 0
}

func (this *counterEP) AddValue(key string, val uint64) uint64 {
	if val == 0 {
		val = 1
	}
	this.mu.Lock()
	defer this.mu.Unlock()

	if this.data == nil {
		this.data = make(map[string]uint64)
	}
	if v, ok := this.data[key]; ok {
		this.data[key] = v + val
		return v + val
	} else {
		this.data[key] = val
		return val
	}
}
