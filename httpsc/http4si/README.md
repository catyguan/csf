# HttpInvoker

## 使用

cfg := &Config{}
cfg.URL = ...
cfg.ExcecuteTimeout = 10 * time.Second
si, err := NewHttpServiceInvoker(cfg, nil)

## Converter

用于自定义请求和响应的处理过程

type Converter interface {
	BuildRequest(url string, creq *corepb.ChannelRequest) (*http.Request, error)

	HandleResponse(resp *http.Response) (*corepb.ChannelResponse, error)
}

## LocBuilder

http://[host]:[port]/path?SERVICE=[serviceName]&PARAMS...

