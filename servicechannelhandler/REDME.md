# ServiceChannelHandler

## Blocker

schblocker模块

对服务路径的访问控制

NewBlocker() *Blocker

Blocker.Accept(path string) 允许访问
Blocker.Refuse(path string) 拒绝访问
Blocker.ReadOnly() 只接受QUERY类型请求

path=* 表示全部路径
规则处理按照先设置先判断，如果符合即生效

## Log

schlog模块

对请求和响应做日志显示

NewLogger(n string)

## Sign

schsign模块

对请求和响应做数据签名处理

NewSign(key string, signType uint16, signAll bool)
	signType
		SIGN_REQUEST         = 0x01  请求数据签名
		SIGN_RESPONSE        = 0x02  响应数据签名
		SIGN_REQUEST_VERIFY  = 0x04  检验请求数据签名
		SIGN_RESPONSE_VERIFY = 0x08  检验响应数据签名

	signAll
		是否把整个数据进行签名，否的话只签名请求数据体或响应数据体