# 常见坑 / FAQ（按严重度）
- 高：未开启 RequestID/Telemetry 导致排障困难—务必在 server.Use 注入中间件，配置 Telemetry.Endpoint。
- 高：上下文无超时或过长导致连接泄漏—handler/logic 强制 `context.WithTimeout`；DB/Redis/外部调用都要用 ctx。
- 高：错误直透内部实现—用统一错误码/`httpx.ErrorCtx` 映射；RPC 错误用 status code。
- 中：goctl 生成的 model 未开启 cache—生成时加 `-cache true` 并配置 `CacheRedis`。
- 中：配置热更新误用—go-zero 未内置热更，修改配置需重启。
- 中：限流/熔断误配—`CpuThreshold` 过低会过早拒绝；`MaxBytes` 过小导致大请求被拒。
- 低：未关闭 server/依赖—`defer server.Stop(); svcCtx.Close(); logx.Close()`。
