# 输出格式 / 错误 / 治理与限制

## 输出与错误
- 日志字段：`timestamp`、`level`、`caller`、`reqId`、`traceId`、`spanId`、`method`/`path` 或 `rpc`、`duration_ms`、`code`、`peer`；敏感字段脱敏。
- REST 响应：统一 `{"code":<int>,"message":<string>,"data":<obj|null>}`；分页 `page`、`pageSize`、`total` 固定。
- RPC：使用 gRPC status code。
- Redis 未命中：`redis.Nil` 视为未命中，不报警。

### 错误码建议
- `0` 成功；`400xx` 参数/鉴权；`500xx` 内部错误；`504xx` 依赖超时；`429xx` 限流/熔断。
- REST 用 `httpx.ErrorCtx` 映射；RPC 用 `status.Errorf`，网关再映射。

## 治理开关
- 超时：`Timeout`（毫秒）；handler/logic 再包 `context.WithTimeout`。
- 限流：`RestConf.MaxConns/MaxBytes`；业务限流用 `tokenlimit` 或网关。
- 熔断：`breaker.DoWithAcceptable` 或 RPC 默认熔断；`acceptable` 过滤业务错误。
- 重试：RPC 客户端 `Retry`；业务重试仅限幂等操作，≤3 次，指数退避。
- 中间件：RequestID、Recover、Prometheus、Tracing、Auth/JWT。
- Graceful：信号监听，顺序关闭 server/RPC -> 依赖 -> 日志 -> tracer。

## 限制与安全
- Handler 禁止直接操作 DB/Redis；依赖通过 ServiceContext 注入。
- 配置禁止明文秘钥；`Pass`/`AccessSecret` 用密钥管理或 env。
- 性能红线：单实例 QPS > 2k 需压测与限流；`Timeout` 不低于 200ms；连接池耗尽必须拒绝/降级。
- 安全：日志/trace/metrics 禁止泄露 Token/密码/完整身份证/手机号；需要脱敏。
- 并发：logic 中 goroutine 必须受 ctx 控制；禁止阻塞长连接占用 worker。
- 不可用 API：禁止绕过 logx、直接 new logger/DB/Redis；禁止在 handler 直接写 Response 不经 httpx。
