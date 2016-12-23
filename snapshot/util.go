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

package snapshot

import (
	"errors"
	"os"
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
