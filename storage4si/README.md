# storage4si 具有存储能力的服务容器

## Storage 服务容器接口

## MemoryStorage 采用纯内存进行储存的实现

特点:
* 对写请求进行存储
* 有快照机制
* 支持MasterSlave模式

### 使用

// 构建存储器
storageSize := 1024 // 最大存储的请求数量，建议为 snapcount * 2
ms := storage4si.NewMemoryStorage(storageSize)

// 构建服务容器
cfg := storage4si.NewConfig()
cfg.SnapCount = snapcount
cfg.Storage = ms
cfg.Service = s
si := storage4si.NewStorageServiceContainer(cfg)
errS := si.Run()
if errS != nil {
	... 处理错误
	return
}
defer si.Close()

### MasterSlave服务接口

cfg := masterslave.NewMasterConfig()
cfg.Master = ms.(masterslave.MasterNode)
service := masterslave.NewMasterService(cfg)
si := core.NewSimpleServiceContainer(service)

sc := core.NewServiceChannel()
sc.Sink(si)

pmux.AddInvoker(masterslave.DefaultMasterServiceName(counter.SERVICE_NAME), sc)

### 管理接口


MakeSnapshot 立即启动快照功能

### csfctl支持