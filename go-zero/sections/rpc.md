# RPC 指南（服务端/客户端）

## 服务端示例
```go
func main() {
    // 1) 配置 + 日志 + 追踪
    var c config.Config
    conf.MustLoad("etc/user-rpc.yaml", &c)
    logx.MustSetup(c.Log)
    tp, _ := telemetry.Setup(c.Telemetry)
    defer tp.Shutdown(context.Background())

    // 2) 依赖
    svcCtx := svc.NewServiceContext(c)
    defer svcCtx.Close()

    // 3) 构建 RPC server，注册 pb 服务；默认超时/熔断
    s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
        pb.RegisterUserServer(grpcServer, server.NewUserServer(svcCtx))
    })
    // 自定义拦截器/错误映射/鉴权
    // s.AddUnaryInterceptors(myInterceptor)
    // TLS: grpc_tls := credentials.NewServerTLSFromFile(certFile, keyFile); s.SetServerOption(grpc.Creds(grpc_tls))

    // 4) 优雅关闭
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-stop
        s.Stop()
        svcCtx.Close()
        logx.Close()
        tp.Shutdown(context.Background())
    }()
    logx.Infof("Starting rpc server at %s", c.ListenOn)
    s.Start()
}
```

## 客户端调用
```go
cli := zrpc.MustNewClient(c.TargetRpc) // 或 MustNewClientWithTarget("dns:///svc:8081")
userCli := pb.NewUserClient(cli.Conn())
ctx, cancel := context.WithTimeout(r.Context(), time.Millisecond*time.Duration(c.TargetRpc.Timeout))
defer cancel()
resp, err := userCli.GetUser(ctx, &pb.UserReq{Id: req.Id})
```

## 拦截器要点
- Trace/metrics/auth/recover 必开，业务需要可追加统一错误映射。
- 客户端 `Retry` 仅用于幂等调用；非幂等禁止自动重试。
- TLS/Etcd/直连按部署配置选择；`StrictControl` 打开以防过载。
