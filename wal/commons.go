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
	"errors"
	"hash/crc32"

	"github.com/catyguan/csf/pkg/capnslog"
)

var (
	WALVersion          uint8  = 1
	DefaultMaxBlockSize uint64 = 64 * 1024 * 1024
	MaxRecordSize       int64  = (0xFFFFFFFF - sizeofLogIndex)

	plog = capnslog.NewPackageLogger("github.com/catyguan/csf", "wal")

	ErrLogIndexError  = errors.New("wal: logindex format error")
	ErrLogHeaderError = errors.New("wal: logheader format error")
	ErrFileNotFound   = errors.New("wal: file not found")
	ErrCRCMismatch    = errors.New("wal: crc mismatch")
	ErrClosed         = errors.New("closed")
	crcTable          = crc32.MakeTable(crc32.Castagnoli)
)

type Entry struct {
	Index uint64
	Data  []byte
}

type Cursor interface {
	// Read Entry
	Read() (*Entry, error)

	// Close Cursor
	Close()
}

type Follow interface {
	EntryCh() <-chan *Entry
	Error() error
	Close()
}

type Result struct {
	Index uint64
	Err   error
	data  interface{}
}

// WriteAheadLogger is the primary interface
type WAL interface {
	// Append
	Append(ents []Entry, sync bool) <-chan Result

	// Sync blocks
	Sync() <-chan Result

	// Close flushes and cleanly closes the log
	Close()

	// Reset destructively clears out any pending data in the log
	Reset() <-chan Result

	// Truncate to index
	Truncate(idx uint64) <-chan Result

	// Index returns the last index
	LastIndex() uint64

	// GetCursor returns a Cursor at the specified index
	GetCursor(idx uint64) (Cursor, error)

	AddListener(lis WALListener) (uint64, error)

	RemoveListener(lis WALListener)
}

type WALListener interface {
	OnReset()

	OnTruncate(idx uint64)

	OnAppendEntry(ents []Entry)

	OnClose()
}
