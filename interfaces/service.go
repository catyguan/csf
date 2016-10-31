// Copyright 2016 The CSF Authors
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

// Package interfaces defines the base interface class use by csfserver and modules.
package interfaces

// 定制的集群服务接口
type Service interface {
	// 服务编号，集群系统内唯一
	ServiceID() string

	// 启动服务，可以对外提供服务
	StartService() error

	// 停止服务，不对外提供服务，用于状态同步阶段
	StopService()

	// 重置服务状态为初始化
	Reset() error

	// 应用某个数据快照
	ApplySnapshot(data []byte) error
}
