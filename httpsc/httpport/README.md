# Http服务端口

## 启动

smux := core.NewServiceMux()
...

hmux := http.NewServeMux()

pcfg := &httpport.Config{}
pcfg.Addr = ":8086"
pcfg.Host = "" // 配置虚拟主机

port := httpport.NewPort(pcfg)
port.BuildHttpMux(hmux, "/service", smux, nil)
port.BuildHttpMux(hmux, "/peer", pmux, nil)
port.BuildHttpMux(hmux, "/admin", amux, nil)

err0 := port.StartServe(hmux)
if err0 != nil {
	fmt.Printf("start fail - %v", err0)
	return
}
port.Run()
defer port.Stop()

