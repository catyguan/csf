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
	"fmt"
	"os"
	"testing"
	"time"
)

func setTimeout(d time.Duration) {
	time.AfterFunc(d, func() {
		fmt.Printf("execute timeout\n")
		os.Exit(-100)
	})
}

func TestBase(t *testing.T) {

	setTimeout(15 * time.Second)

	ti := NewTicker(2 * time.Second)
	defer ti.Stop()

	for i := 0; i < 5; i++ {
		t := <-ti.C
		fmt.Printf("active at %v\n", t)
	}

}

func TestStop(t *testing.T) {
	ti := NewTicker(2 * time.Second)
	t1 := time.After(5 * time.Second)
	t2 := time.After(10 * time.Second)

	for {
		select {
		case t := <-ti.C:
			fmt.Printf("active at %v\n", t)
		case t := <-t1:
			fmt.Printf("stop at %v\n", t)
			ti.Stop()
		case t := <-t2:
			fmt.Printf("end at %v\n", t)
			return
		}
	}
}

func TestReset(t *testing.T) {
	ti := NewTicker(2 * time.Second)
	t1 := time.After(5 * time.Second)
	t2 := time.After(12 * time.Second)

	for {
		select {
		case t := <-ti.C:
			fmt.Printf("active at %v\n", t)
		case t := <-t1:
			fmt.Printf("reset at %v\n", t)
			ti.Reset()
		case t := <-t2:
			fmt.Printf("end at %v\n", t)
			return
		}
	}
}
