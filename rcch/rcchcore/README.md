# RCCH 基础模块

## 请求和响应数据

Header —— 头数据 (PBHeader)
	string name —— 头的名称
	string value —— 头的字符串数值
	bytes data —— 头的数据体

Request —— 服务处理的请求数据
	uint64 ID —— 请求编号,0表示无,中间件如果修改该编号需要保证响应数据也需对应修改
	string service_name —— 请求处理的服务名称
	string service_path —— 请求处理的服务路径
	RequestType type —— 请求类型
		UNKNOW = 0 ： 不明确，部分处理程序会对这类请求采用错误处理，所以不建议使用
		QUERY = 1 ： 查询请求，一般是指读操作
		EXECUTE = 2 ： 执行请求，一般是指写操作
		MESSAGE = 3 ： 消息请求，表示该请求不期望响应
	bytes data —— 请求数据体

ChannelRequest —— 服务调用通道处理的请求数据
	包含Request的所有数据
	list<Header> 请求头数据，相同名字的头以最后的为最新

Response —— 服务处理的响应数据
	uint64 RequestID —— 响应对应的请求编号
	uint64 ActionIndex —— 处理的序号，用于跟踪调试
	bytes data —— 响应数据体
	string error —— 错误信息

ChannelResponse —— 服务调用通道处理响应数据
	包含Response的所有数据
	list<Header> 响应头数据

## 通讯编码实现

请求
```
[FrameSize:32bit, not include this]
[RequestSize:varuint64]
[PBRequestInfo]
[HeadersSize:varuint64]
[PBHeaders]
[PayloadSize:varuint64]
[Payload:...]
```

响应
```
[FrameSize:32bit, not include this]
[ResponseSize:varuint64]
[PBResponseInfo]
[HeadersSize:varuint64]
[PBHeaders]
[PayloadSize:varuint64]
[Payload:...]
```