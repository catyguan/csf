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

package wal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/catyguan/csf/pkg/fileutil"
)

var (
	badWalName = errors.New("bad wal name")
)

func Exist(dirpath string) bool {
	names, err := fileutil.ReadDir(dirpath)
	if err != nil {
		return false
	}
	return len(names) != 0
}

func ExistFile(fpath string) bool {
	_, err := os.Stat(fpath)
	if err != nil {
		return false
	}
	return true
}

// searchIndex returns the last array index of names whose raft index section is
// equal to or smaller than the given index.
// The given names MUST be sorted.
func searchIndex(names []string, index uint64) (int, bool) {
	for i := len(names) - 1; i >= 0; i-- {
		name := names[i]
		curIndex, err := parseWalName(name)
		if err != nil {
			plog.Panicf("parse correct name should never fail: %v", err)
		}
		if index >= curIndex {
			return i, true
		}
	}
	return -1, false
}

func readWalNames(dirpath string) ([]string, error) {
	names, err := fileutil.ReadDir(dirpath)
	if err != nil {
		return nil, err
	}
	wnames := checkWalNames(names)
	if len(wnames) == 0 {
		return nil, ErrFileNotFound
	}
	return wnames, nil
}

func checkWalNames(names []string) []string {
	wnames := make([]string, 0)
	for _, name := range names {
		if _, err := parseWalName(name); err != nil {
			// don't complain about left over tmp files
			if !strings.HasSuffix(name, ".tmp") {
				plog.Warningf("ignored file %v in wal", name)
			}
			continue
		}
		wnames = append(wnames, name)
	}
	return wnames
}

func parseWalName(str string) (index uint64, err error) {
	if !strings.HasSuffix(str, ".wal") {
		return 0, badWalName
	}
	_, err = fmt.Sscanf(str, "%016x.wal", &index)
	return index, err
}

func walName(index uint64) string {
	return fmt.Sprintf("%016x.wal", index)
}

func allocFileSize(dir, filePath string, size int64) error {
	fpath := filepath.Join(dir, "walalloc.tmp")
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, fileutil.PrivateFileMode)
	if err != nil {
		return err
	}
	if err = fileutil.Preallocate(f, size, true); err != nil {
		f.Close()
		os.Remove(fpath)
		return err
	}
	f.Close()
	err = os.Rename(fpath, filePath)
	if err != nil {
		os.Remove(fpath)
	}
	return err
}
