# 错误处理总览

- `redis.Nil`：表示未命中/不存在，不计为错误；业务自行分支处理。
- `context.DeadlineExceeded` / `context.Canceled`：超时/取消，应优化超时配置或上层超时；不要重试非幂等操作。
- 网络类（`i/o timeout`, `connection refused`, `EOF`）：归类为可重试，但需结合 MaxRetries 与幂等性；持续失败要告警。
- 连接池耗尽（`context deadline exceeded` 于 `PoolTimeout`）：提升池/超时前先排查泄漏或慢查询。
- 服务器错误（`LOADING`, `READONLY`, `CLUSTERDOWN`）：大多暂时性，可按内置重试策略；持续出现需告警。
- 业务逻辑错误：不要重试（如 Lua 返回自定义错误、CAS 失败等），直接返回给业务。
- 日志必要字段：`request_id, redis_addr, cmd, key/keys, elapsed_ms, retry_count, err_type, err`.
