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

	validFileExts = []string{".wal", ".tmp", ".snap", ".broken"}
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
		if _, _, err := parseWalName(name); err != nil {
			// don't complain about left over tmp files
			m := false
			for _, ext := range validFileExts {
				if strings.HasSuffix(name, ext) {
					m = true
					break
				}
			}
			if !m {
				plog.Warningf("skipped unexpected non walblock file  %v", name)
			}
			continue
		}
		wnames = append(wnames, name)
	}
	return wnames
}

func parseWalName(str string) (seq, index uint64, err error) {
	if !strings.HasSuffix(str, ".wal") {
		return 0, 0, badWalName
	}
	_, err = fmt.Sscanf(str, "%016x-%016x.wal", &seq, &index)
	return seq, index, err
}

func checkSnapNames(names []string) []string {
	snaps := []string{}
	for _, name := range names {
		if strings.HasSuffix(name, snapSuffix) {
			snaps = append(snaps, name)
		} else {
			m := false
			for _, ext := range validFileExts {
				if strings.HasSuffix(name, ext) {
					m = true
					break
				}
			}
			if !m {
				plog.Warningf("skipped unexpected non snapshot file %v", name)
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
