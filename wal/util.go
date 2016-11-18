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

func ExistDir(dirpath string) bool {
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

func checkSnapNames(names []string) []string {
	snaps := []string{}
	for _, name := range names {
		if strings.HasSuffix(name, snapSuffix) {
			snaps = append(snaps, name)
		} else {
			// If we find a file which is not a snapshot then check if it's
			// a vaild file. If not throw out a warning.
			if !strings.HasSuffix(name, ".tmp") &&
				!strings.HasSuffix(name, ".wal") &&
				!strings.HasSuffix(name, ".broken") {
				plog.Warningf("skipped unexpected non snapshot file %v", names)
			}
		}
	}
	return snaps
}

func blockName(id uint64) string {
	return fmt.Sprintf("%016x.wal", id)
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
