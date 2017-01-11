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
package storage4si

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"github.com/catyguan/csf/core/corepb"
	"github.com/catyguan/csf/masterslave"
)

type MemoryStorage struct {
	maxSize int

	index uint64
	head  int
	size  int
	ents  []RequestEntry

	snapIndex uint64
	snapData  []byte
	llist     Listeners
	mu        sync.Mutex
}

func NewMemoryStorage(maxSize int) Storage {
	r := &MemoryStorage{
		maxSize:   maxSize,
		index:     0,
		head:      0,
		size:      0,
		ents:      make([]RequestEntry, maxSize),
		snapIndex: 0,
	}
	return r
}

func (this *MemoryStorage) impl() {
	_ = Storage(this)
	_ = masterslave.MasterNode(this)
	_ = Rebuildable(this)
}

func (this *MemoryStorage) SaveRequest(idx uint64, req *corepb.Request) (uint64, error) {
	this.mu.Lock()
	defer this.mu.Unlock()

	if idx == 0 {
		idx = this.index + 1
	}
	if this.snapIndex != 0 && idx < this.snapIndex {
		return 0, ErrConflict
	}
	if this.size > 0 {
		// check overwrite idx
		h, t := this.indexRange()
		tru := true
		if idx <= h {
			this.head = 0
			this.size = 0
		} else if idx < t {
			_, pos := this.seek(idx)
			if this.ents[pos].Index != idx {
				pos = this.next(pos, 1)
			}
			this.size = this.distance(this.head, pos)
			nt := make([]RequestEntry, this.maxSize)
			p := this.head
			for i := 0; i < this.maxSize; i++ {
				if p == pos {
					break
				}
				nt[i] = this.ents[p]
				p = this.next(p, 1)
			}
			this.ents = nt
			this.head = 0
		} else {
			tru = false
		}
		if tru {
			this.llist.OnTruncate(idx)
		}
	}

	var e *RequestEntry
	if this.size < this.maxSize {
		e = &this.ents[this.size]
		this.size++
	} else {
		e = &this.ents[this.head]
		this.head = this.next(this.head, 1)
	}
	e.Index = idx
	e.Request = req
	this.index = idx
	this.llist.OnSaveRequest(idx, req)
	return this.index, nil
}

type cursorOfMemoryStorage struct {
	start uint64
}

func (this *cursorOfMemoryStorage) String() string {
	return fmt.Sprintf("Index:%d", this.start)
}

func (this *MemoryStorage) BeginLoad(start uint64) (interface{}, error) {
	if start == 0 {
		start = 1
	}
	return &cursorOfMemoryStorage{start: start}, nil
}

func (this *MemoryStorage) LoadRequest(c interface{}, size int, lis StorageListener) (uint64, []*corepb.Request, error) {
	return this.doLoadRequest(c, size, lis, nil)
}

func (this *MemoryStorage) doLoadRequest(c interface{}, size int, lis StorageListener, f masterslave.MasterFollower) (uint64, []*corepb.Request, error) {
	this.mu.Lock()
	defer this.mu.Unlock()

	cc := c.(*cursorOfMemoryStorage)
	ok, pos := this.seek(cc.start)
	if !ok {
		if lis != nil {
			this.doAddListener(lis)
		}
		if f != nil {
			this.doAddListener(&MasterFollowerListener{Follower: f})
		}
		return 0, nil, nil
	}
	ll := uint64(0)
	r := make([]*corepb.Request, 0, size)
	t := this.tail()
	for {
		if len(r) >= size {
			break
		}
		e := &this.ents[pos]
		if e.Index >= cc.start {
			r = append(r, e.Request)
			ll = e.Index
		}
		if pos == t {
			break
		}
		pos = this.next(pos, 1)
	}
	if len(r) == 0 {
		r = nil
		if lis != nil {
			this.doAddListener(lis)
		}
		if f != nil {
			this.doAddListener(&MasterFollowerListener{Follower: f})
		}
	}
	cc.start = ll + 1
	return ll, r, nil
}

func (this *MemoryStorage) EndLoad(c interface{}) error {
	return nil
}

func (this *MemoryStorage) SaveSnapshot(idx uint64, r io.Reader) error {
	this.mu.Lock()
	defer this.mu.Unlock()

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return this.doSaveSnapshot(idx, data)
}

func (this *MemoryStorage) doSaveSnapshot(idx uint64, data []byte) error {
	if idx < this.snapIndex {
		plog.Warningf("skip older snapshot(%v), now(%v)", idx, this.snapIndex)
		return nil
	}
	this.snapData = data
	this.snapIndex = idx
	this.llist.OnSaveSnapshot(idx)
	return nil
}

func (this *MemoryStorage) ApplySnapshot(idx uint64, data []byte) error {
	this.Reset()
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.doSaveSnapshot(idx, data)
}

func (this *MemoryStorage) LoadLastSnapshot() (uint64, io.Reader, error) {
	this.mu.Lock()
	defer this.mu.Unlock()

	return this.snapIndex, bytes.NewBuffer(this.snapData), nil
}

func (this *MemoryStorage) AddListener(lis StorageListener) uint64 {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.doAddListener(lis)
}

func (this *MemoryStorage) doAddListener(lis StorageListener) uint64 {
	this.llist.Add(lis)
	return this.index
}

func (this *MemoryStorage) RemoveListener(lis StorageListener) {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.llist.Remove(lis)
}

func (this *MemoryStorage) Reset() {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.size = 0
	this.head = 0
	this.index = 0
	this.snapData = nil
	this.snapIndex = 0
	this.llist.OnReset()
}

func (this *MemoryStorage) tail() int {
	if this.size == 0 {
		return 0
	}
	if this.size < this.maxSize {
		return this.size - 1
	} else {
		return this.prev(this.head, 1)
	}
}

func (this *MemoryStorage) next(i int, s int) int {
	i += s
	if i >= this.maxSize {
		return i - this.maxSize
	}
	return i
}

func (this *MemoryStorage) prev(i int, s int) int {
	i -= s
	if i < 0 {
		return this.maxSize + i
	}
	return i
}

func (this *MemoryStorage) distance(s, e int) int {
	if e >= s {
		return e - s
	}
	return e + this.maxSize - s
}

func (this *MemoryStorage) indexRange() (uint64, uint64) {
	return this.ents[this.head].Index, this.ents[this.tail()].Index
}

func (this *MemoryStorage) seek(idx uint64) (bool, int) {
	h, t := this.indexRange()
	if idx == 0 {
		return false, 0
	}
	if idx < h || t < idx {
		// plog.Infof("%v %v %v - %v, %v", idx, h.index, t.index, this.head, this.tail())
		return false, 0
	}
	step := this.size / 2
	if step < 1 {
		step = 1
	}
	pos := this.next(this.head, step)
	for {
		if step > 1 {
			step = step / 2
		}

		i1 := this.ents[pos].Index
		if i1 == idx {
			return true, pos
		}
		if i1 > idx {
			pp := this.prev(pos, 1)
			i2 := this.ents[pp].Index
			if i2 > idx {
				pos = this.prev(pos, step)
			} else {
				return true, pp
			}
		} else {
			i2 := this.ents[this.next(pos, 1)].Index
			if i2 > idx {
				return true, pos
			} else {
				pos = this.next(pos, step)
			}
		}
	}

}

func (this *MemoryStorage) Count() int {
	this.mu.Lock()
	defer this.mu.Unlock()
	return this.size
}

func (this *MemoryStorage) MasterLoadLastSnapshot() (uint64, []byte, error) {
	lidx, r, err := this.LoadLastSnapshot()
	if err != nil {
		return 0, nil, err
	}
	if r == nil {
		return lidx, nil, nil
	}
	b, err2 := ioutil.ReadAll(r)
	return lidx, b, err2
}

func (this *MemoryStorage) MasterBeginLoad(idx uint64) (interface{}, error) {
	return this.BeginLoad(idx)
}

func (this *MemoryStorage) MasterLoadRequest(c interface{}, size int, f masterslave.MasterFollower) ([]*corepb.Request, error) {
	_, r1, r2 := this.doLoadRequest(c, size, nil, f)
	return r1, r2
}

func (this *MemoryStorage) MasterEndLoad(c interface{}) {
	this.EndLoad(c)
}

func (this *MemoryStorage) RemoveFollower(f masterslave.MasterFollower) {
	this.RemoveListener(&MasterFollowerListener{Follower: f})
}
