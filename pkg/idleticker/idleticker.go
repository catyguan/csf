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

package idleticker

import (
	"errors"
	"sync"
	"time"
)

type Ticker struct {
	C      <-chan time.Time // The channel on which the ticks are delivered.
	atime  time.Time
	du     time.Duration
	closec chan interface{}
	f      func()
	mu     sync.Mutex
}

func NewTicker(d time.Duration) *Ticker {
	if d <= 0 {
		panic(errors.New("non-positive interval for NewTicker"))
	}
	// Give the channel a 1-element time buffer.
	// If the client falls behind while reading, we drop ticks
	// on the floor until the client catches up.
	c := make(chan time.Time, 1)
	t := &Ticker{
		C:      c,
		du:     d,
		closec: make(chan interface{}, 0),
	}

	go t.run(c)

	return t
}

func (this *Ticker) OnIdle(f func()) {
	this.f = f
}

func (this *Ticker) run(c chan time.Time) {
	this.mu.Lock()
	this.atime = time.Now()
	this.mu.Unlock()
	for {
		this.mu.Lock()
		n := time.Now()
		d := n.Sub(this.atime)
		this.mu.Unlock()
		if d >= this.du {
			if this.f != nil {
				this.f()
			}
			select {
			case c <- n:
			default:
			}
			this.mu.Lock()
			this.atime = time.Now()
			this.mu.Unlock()
		} else {
			ti := time.NewTimer(this.du - d)
			// fmt.Printf("---- wait %v, %v\n", n, this.du-d)
			select {
			case <-ti.C:
			case <-this.closec:
				ti.Stop()
				return
			}
		}
	}
}

func (this *Ticker) Stop() {
	close(this.closec)
}

func (this *Ticker) Reset() {
	this.mu.Lock()
	this.atime = time.Now()
	this.mu.Unlock()
}
