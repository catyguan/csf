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
	"io"
	"io/ioutil"

	"github.com/catyguan/csf/core/corepb"
)

type memoryEntry struct {
	index   uint64
	request *corepb.Request
}

type MemoryStorage struct {
	maxSize int

	index uint64
	head  int
	size  int
	ents  []memoryEntry

	snapIndex uint64
	snapData  []byte
}

func NewMemoryStorage(maxSize int) Storage {
	r := &MemoryStorage{
		maxSize:   maxSize,
		index:     0,
		head:      0,
		size:      0,
		ents:      make([]memoryEntry, maxSize),
		snapIndex: 0,
	}
	return r
}

func (this *MemoryStorage) SaveRequest(idx uint64, req *corepb.Request) (uint64, error) {
	if idx == 0 {
		idx = this.index + 1
	}
	if this.snapIndex != 0 && idx < this.snapIndex {
		return 0, ErrConflict
	}
	if this.size > 0 {
		// check overwrite idx
		h, t := this.indexRange()
		if idx <= h {
			this.head = 0
			this.size = 0
		} else if idx < t {
			_, pos := this.seek(idx)
			if this.ents[pos].index != idx {
				pos = this.next(pos, 1)
			}
			this.size = this.distance(this.head, pos)
			nt := make([]memoryEntry, this.maxSize)
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
		}
	}

	var e *memoryEntry
	if this.size < this.maxSize {
		e = &this.ents[this.size]
		this.size++
	} else {
		e = &this.ents[this.head]
		this.head = this.next(this.head, 1)
	}
	e.index = idx
	e.request = req
	this.index = idx
	return this.index, nil
}

func (this *MemoryStorage) LoadRequest(start uint64, size int) (uint64, []*corepb.Request, error) {
	ok, pos := this.seek(start)
	if !ok {
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
		if e.index >= start {
			r = append(r, e.request)
			ll = e.index
		}
		pos = this.next(pos, 1)
		if pos == t {
			break
		}
	}
	if len(r) == 0 {
		r = nil
	}
	return ll, r, nil
}

func (this *MemoryStorage) SaveSnapshot(idx uint64, r io.Reader) error {
	if idx < this.snapIndex {
		plog.Warningf("skip older snapshot(%v), now(%v)", idx, this.snapIndex)
		return nil
	}

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	this.snapData = data
	this.snapIndex = idx
	return nil
}

func (this *MemoryStorage) LoadLastSnapshot() (uint64, io.Reader, error) {
	return this.snapIndex, bytes.NewBuffer(this.snapData), nil
}

func (this *MemoryStorage) Reset() {
	this.size = 0
	this.head = 0
	this.index = 0
	this.snapData = nil
	this.snapIndex = 0
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
	return this.ents[this.head].index, this.ents[this.tail()].index
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

		i1 := this.ents[pos].index
		if i1 == idx {
			return true, pos
		}
		if i1 > idx {
			pp := this.prev(pos, 1)
			i2 := this.ents[pp].index
			if i2 > idx {
				pos = this.prev(pos, step)
			} else {
				return true, pp
			}
		} else {
			i2 := this.ents[this.next(pos, 1)].index
			if i2 > idx {
				return true, pos
			} else {
				pos = this.next(pos, step)
			}
		}
	}

}

func (this *MemoryStorage) Count() int {
	return this.size
}
