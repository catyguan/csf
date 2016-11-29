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

package wal

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/catyguan/csf/pkg/pbutil"
	"github.com/catyguan/csf/raft/raftpb"
)

type snapHeader struct {
	state     raftpb.HardState
	confState raftpb.ConfState
	index     uint64
	term      uint64
}

func (this *snapHeader) String() string {
	s := ""
	s += fmt.Sprintf("state: %s, ", this.state.String())
	s += fmt.Sprintf("confState: %s, ", this.confState.String())
	s += fmt.Sprintf("index: %v, ", this.index)
	s += fmt.Sprintf("term: %v", this.term)

	return s
}

func (this *logCoder) WriteSnap(w io.Writer, lr *snapHeader) error {
	bm := make([]byte, 8*2)
	binary.LittleEndian.PutUint64(bm, lr.index)
	binary.LittleEndian.PutUint64(bm[:8], lr.term)

	bs := [][]byte{}
	bs = append(bs, pbutil.MustMarshal(&lr.state))
	bs = append(bs, pbutil.MustMarshal(&lr.confState))
	bs = append(bs, bm)
	buf := make([]byte, 4)
	for _, b := range bs {
		binary.LittleEndian.PutUint32(buf[0:], uint32(len(b)))
		_, err := w.Write(buf)
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *logCoder) ReadSnap(r io.Reader) (*snapHeader, error) {
	bs := [][]byte{nil, nil, nil}
	buf := make([]byte, 4)
	for i, _ := range bs {
		n, err := io.ReadFull(r, buf)
		if err != nil {
			return nil, err
		}
		if n != 4 {
			return nil, io.ErrUnexpectedEOF
		}
		l := binary.LittleEndian.Uint32(buf[0:])
		b := make([]byte, l)
		n, err = io.ReadFull(r, b)
		if err != nil {
			return nil, err
		}
		if uint32(n) != l {
			return nil, io.ErrUnexpectedEOF
		}
		bs[i] = b
	}

	lr := &snapHeader{}
	pbutil.MustUnmarshal(&lr.state, bs[0])
	pbutil.MustUnmarshal(&lr.confState, bs[1])
	lr.index = binary.LittleEndian.Uint64(bs[2])
	lr.term = binary.LittleEndian.Uint64(bs[2][:8])

	return lr, nil
}
