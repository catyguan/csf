# storage4si.walstorage 具有存储能力的服务容器

## WALStorage 采用WAL(WriteAheadLog)进行储存的实现

特点:
* 对写请求采用WAL模式进行存储
* 有快照机制和存储实现
* 支持重启Recover
* 支持MasterSlave模式

### 使用

// 构建存储器
scfg := walstorage.NewConfig()
scfg.Dir = dir
scfg.BlockRollSize = 16 * 1024
scfg.Symbol = "tcserver"

ms, err := walstorage.NewWALStorage(scfg)
if err != nil {
	fmt.Printf("open WALStorage fail - %v", err)
	return
}
defer ws.Close()

// 构建服务容器
// 和storage4si的相关README说明一样

### MasterSlave服务接口

// 和storage4si的相关README说明一样

### 管理接口

// 和storage4si的相关README说明一样