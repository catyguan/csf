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
	"io"

	"github.com/catyguan/csf/core/corepb"
)

type RequestEntry struct {
	Index   uint64
	Request *corepb.Request
}

type Storage interface {
	// 保存请求 index:ActionIndex, 为0采用Storage内部编号
	SaveRequest(idx uint64, req *corepb.Request) (uint64, error)

	// 请求数据加载游标
	// 返回游标
	BeginLoad(start uint64) (interface{}, error)

	// 加载请求 [start, start+size), 如果没有新数据并提供listener，则自动监听
	// 返回 uint64-响应结果集最后的数据的index; []*Request- 响应结果, nil表示没有数据了;error;
	LoadRequest(cursor interface{}, size int, lis StorageListener) (uint64, []*corepb.Request, error)

	// 关闭数据加载游标
	EndLoad(cursor interface{}) error

	// 保存快照, index:快照对应的服务状态Index（包含该Index的结果）
	SaveSnapshot(idx uint64, r io.Reader) error

	// 加载快照
	LoadLastSnapshot() (uint64, io.Reader, error)

	// 添加Listener，返回添加成功时候的LastIndex
	AddListener(lis StorageListener) uint64

	RemoveListener(lis StorageListener)
}

type Rebuildable interface {
	ApplySnapshot(idx uint64, data []byte) error
}

type StorageListener interface {
	OnReset()

	OnTruncate(idx uint64)

	OnSaveRequest(idx uint64, req *corepb.Request)

	OnSaveSanepshot(idx uint64)

	OnClose()
}
