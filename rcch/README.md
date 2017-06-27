# 远程通讯通道 (Remote Communication Channel)

## 目标

* 集群内部使用（有限节点之间的通讯模式）
	* C/S模式
	* TCP Socket Stream模式
	* Request/Response交互模式
* 支持上层构建RPC
	* Message/ACK
	* Server Push
* 支持通道模式的组件扩展

## 使用

### Client

构建客户端对象
```go
cfg := &Config{}
cfg.PeerHost = "127.0.0.1:1060"
cfg.ConnectionTimeout = 3 * time.Second
cfg.ExcecuteTimeout = 10 * time.Second
cl, err := rcchclient.NewClient(cfg)
```

## Converter

用于自定义请求和响应的处理过程

type Converter interface {
	BuildRequest(url string, creq *corepb.ChannelRequest) (*http.Request, error)

	HandleResponse(resp *http.Response) (*corepb.ChannelResponse, error)
}

## LocBuilder

http://[host]:[port]/path?SERVICE=[serviceName]&PARAMS...

