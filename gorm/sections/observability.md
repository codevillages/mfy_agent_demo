# 可观测性 / 诊断
- Metrics：为 DB 操作记录 `latency`、`qps`、`error_rate`、`affected_rows`；按 `op`、`table`、`status` 打标签。
- Tracing：ctx 中注入 trace，使用 `WithContext`；可接入 gorm otel 插件或自定义 callback 在 span 记录 SQL（参数脱敏/截断）。
- 慢日志：>200ms 输出 warn；字段含 `request_id`、`trace_id`、`op`、`table`、`rows`、`duration_ms`。
- MySQL 慢日志：建议开启并对齐阈值；通过 `connection id` 关联应用日志。
- 诊断：`context deadline exceeded` 检查池和超时；锁等待可查看 `INNODB_TRX`/`LOCKS`；频繁重试需评估幂等和退避。
