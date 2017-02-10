# raft4si 采用一致性算法实现的服务集群容器

## RaftServiceContainer

特点:
* 对写请求进行raft算法进行一致性处理
* WAL存储机制
* 快照机制
* 支持重启Recover
* 支持纯内存运作模式
* 支持MasterSlave模式(未完成)

### 使用

s := NewXXXService

cfg := raft4si.NewConfig()
// cfg.MemoryMode = true
cfg.WALDir = ...
cfg.BlockRollSize = 16 * 1024
cfg.Symbol = "tcserver3"
cfg.NodeID = lport
cfg.InitPeers = peers
cfg.SnapCount = 16
cfg.NumberOfCatchUpEntries = 16
cfg.MemberSeq = 10000
cfg.AccessCode = acccessCode

si := raft4si.NewRaftServiceContainer(s, cfg)

errS := si.Run()
if errS != nil {
	fmt.Printf("run RaftServiceContainer fail - %v", errS)
	return
}
defer si.Close()

### Admin服务接口

as := raft4si0admin.NewAdminService(si)
sc := core.NewServiceChannel()
sc.Next(schlog.NewLogger("RAFT_ADMIN"))
sc.Next(schsign.NewSign("123456", schsign.SIGN_REQUEST_VERIFY|schsign.SIGN_RESPONSE, false))
sc.Sink(core.NewSimpleServiceContainer(as))
amux.AddInvoker(raft4si0admin.DefaultAdminServiceName("service_name_xxxx"), sc)

MakeSnapshot 立即启动快照功能
