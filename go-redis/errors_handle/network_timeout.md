# 超时与网络错误处理

- 配置：设置 `DialTimeout/ReadTimeout/WriteTimeout`（建议 500ms–1s），`PoolTimeout`（建议 1s），`MaxRetries`（1–5）配合 `MinRetryBackoff/MaxRetryBackoff`。
- 分类：
  - `i/o timeout`/`net.Error.Timeout`: 视为暂时性，可在幂等操作上重试。
  - `connection refused`/`no route to host`/`EOF`: 网络不可达或服务未启动，需告警。
  - `context deadline exceeded`：通常是上层超时或连接池等待超时，优先排查池耗尽或慢命令。
- 建议：
  - 只对幂等命令重试；非幂等（如 `INCR` 视场景可接受）需谨慎。
  - 设置调用级超时，避免调用挂起；在高延迟场景适当放宽 `ReadTimeout`/`WriteTimeout`，但不应无限大。
  - 监控：超时/连接错误计数、重试次数、平均耗时；超过阈值触发告警。
