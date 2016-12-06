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
package core

import (
	"context"
	"io"

	"github.com/catyguan/csf/core/corepb"
)

type ServiceInvoker interface {
	InvokeRequest(ctx context.Context, req *corepb.Request) (*corepb.Response, error)
}

type CoreService interface {
	// 检验请求数据是否正确（可Apply）
	// 返回 bool - 该请求需要保存; error - 错误
	VerifyRequest(ctx context.Context, req *corepb.Request) (bool, error)

	// 应用服务请求(需要保证处理能成功)
	ApplyRequest(ctx context.Context, req *corepb.Request) (*corepb.Response, error)

	// 创建服务状态快照
	CreateSnapshot(ctx context.Context, w io.Writer) error

	// 恢复服务状态快照
	ApplySnapshot(ctx context.Context, r io.Reader) error
}

type ServiceChannel interface {
	SendRequest(ctx context.Context, creq *corepb.ChannelRequest) (<-chan *corepb.ChannelResponse, error)
}

type ServiceChannelHandler interface {
	SendRequest(ctx context.Context, nextChannel ServiceChannel, creq *corepb.ChannelRequest) (<-chan *corepb.ChannelResponse, error)
}
