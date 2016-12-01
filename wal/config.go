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

import "time"

type Config struct {
	Dir              string        // WAL工作目录
	BlockRollSize    uint64        // BLOCK文件需要进行roll的大小，缺省为64M
	InitMetadata     []byte        // 初始化的Meta数据
	WALQueueSize     int           // WAL的队列大小，缺省为1000
	CursorQueueSize  int           // Cursor的队列大小，缺省为100
	IdleSyncDuration time.Duration // 进行IdleSync的检查时间间隔，缺省为30秒
}

func NewConfig() *Config {
	r := &Config{
		BlockRollSize:    DefaultMaxBlockSize,
		WALQueueSize:     1000,
		CursorQueueSize:  100,
		IdleSyncDuration: 30 * time.Second,
	}
	return r
}
