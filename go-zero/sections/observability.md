# 可观测性 / 调试 / 压测

## Metrics
- 开启 go-zero 内置 Prometheus（`Prometheus` 节）；关注 `qps`、`latency`、`error_rate`、`breaker`/`limit`。
- DB/Redis/外部依赖自定义 histogram；记录 `op`、`status`、`retry_count`。

## Tracing
- 配置 `Telemetry.Endpoint`；REST/RPC 自动注入 trace/span。
- 外部依赖打 tag：`db.op`、`cache.op`、`peer.service`。
- 采样率：本地 1，线上按流量降采样（0.1~0.3）。

## 慢日志
- 阈值：REST >300ms、RPC >200ms、DB >100ms、Redis >50ms 打 warn。
- 字段：`reqId`、`traceId`、`route`/`method`、`op`、`duration_ms`。

## 调试/测试/压测
- 本地：`LOG_LEVEL=debug`、`Verbose=true`；可替换外部依赖为 stub。
- 单测：logic 用 fake repo；handler 用 httpx；RPC 用 grpctest + fake client。
- 压测前：调高 `MaxConns`/`MaxBytes`、连接池，关闭 debug 日志；确保 RequestID/trace 打开。
- 压测观测：Prometheus 采集 qps/latency/error_rate；慢日志阈值设为 SLA 的 2-3 倍。
