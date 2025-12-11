# REST 服务指南（示例代码 + 注释）
```go
func main() {
    // 1) 加载配置，敏感信息走 env/密钥服务
    var c config.Config
    conf.MustLoad("etc/user-api.yaml", &c)

    // 2) 初始化日志与追踪；关闭在 main defer
    logx.MustSetup(c.Log)
    tp, _ := telemetry.Setup(c.Telemetry)
    defer tp.Shutdown(context.Background())

    // 3) 初始化依赖，放入 ServiceContext，handler/logic 统一从 ctx 拿
    ctx := svc.NewServiceContext(c)
    defer ctx.Close() // 关闭 DB/Redis 等资源

    // 4) 构建 REST server，挂载基础中间件（RequestID/Recover/Tracing/Metrics/Auth）
    server := rest.MustNewServer(c.RestConf)
    server.Use(middleware.RequestID())
    server.Use(rest.RecoverHandler())
    server.Use(middleware.Tracing(c.Telemetry))
    server.Use(middleware.Metrics())
    // server.Use(auth.NewAuthorizer(c.Auth.AccessSecret)) // 如需 JWT

    // 5) 注册路由；生成代码里 handler.RegisterHandlers(server, ctx)
    handler.RegisterHandlers(server, ctx)
    server.AddRoute(rest.Route{
        Method: http.MethodGet,
        Path:   "/health",
        Handler: func(w http.ResponseWriter, r *http.Request) {
            httpx.OkJsonCtx(r.Context(), w, map[string]string{"status": "ok"})
        },
    })

    // 6) 启动前依赖检查：db.Ping/redis.Ping，失败直接退出

    // 7) 启动与优雅关闭
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-stop
        server.Stop()            // 优雅停机
        ctx.Close()              // 关闭依赖
        logx.Close()             // flush 日志
        tp.Shutdown(context.Background())
    }()
    logx.Infof("Starting rest server at %s:%d", c.Host, c.Port)
    server.Start()
}
```

## 关键要点
- 中间件必须注入 `request_id/trace`，确保排障链路。
- 健康检查返回快速、无外部依赖；就绪检查可在 ServiceContext 初始化时 ping。
- 关闭顺序：server -> 依赖 -> 日志/trace。
- 路由注册统一入口，避免分散。
