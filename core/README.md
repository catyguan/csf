# CSF 基础概念和模块

## 请求和响应数据 (corepb)

Header —— 头数据
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

## 服务实现

4层结构：
	API —— 服务接口定义
	Invoker —— 也称为Client，业务参数编码ChannelRequest，再把ChannelResponse解码为业务响应
	Service —— 把ChannelRequest解码为业务参数，再把业务响应编码为ChannelResponse
	EndPoint —— 具体实现业务

服务调用流程：
	Invoker -> (ServiceInvoker) -> ... -> (ServicePort) -> (ServiceContainer) -> Service -> EndPoint
	           { --- ServiceChannel ---                                     }
括号内的是服务调用通道框架提供的实现

## 基本ServiceContainer实现

SimpleServiceContainer
	简单的服务容器，直接把调用传递给CoreService对象，多线程不安全

LockerServiceContainer
	具备锁功能的服务容器，通过锁技术处理多线程并发情况
	缺省实现采用读写锁，读操作使用读锁，写操作使用写锁

SingleThreadServiceContainer
	单工作协程的服务容器，通过队列方式处理请求

ServiceMux
	用于把多个服务进行集合的服务容器，可根据ServiceName和ServicePath进行请求调度

## 服务定位符

服务定位符为一个字符串，使用特定格式进行编写，可以定位和构建具体服务的服务调用器（ServiceInvoker）
	sl, err := ParseLocation(str)
	if err!=nil {
		...
	}
	invoker := sl.Invoker

具体的服务定位符说明请查阅
	http -- httpsc/http4si模块说明

## 常用技巧

### RemoteAddr (远程IP)

使用 REMOTE_ADDR 头
CommonHeaders.SetRemoteAddr
CommonHeaders.GetRemoteAddr

### ErrorTrace的实现

使用 ERROR_TRACE 头
CommonHeaders.AddErrorTrace
CommonHeaders.GetErrorTrace