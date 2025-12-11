# go-redis v9.0.5 使用手册

## 标题 / 目的 / 适用场景
- 目的：在 Go 服务中通过 `github.com/redis/go-redis/v9` 访问 Redis（单机 / Sentinel / Cluster），提供团队统一的配置、调用、可观测与安全规则。
- 适用：高性能 KV/队列/发布订阅、需要连接池与自动重试、需要接入指标与链路追踪的场景。
- 替代：只需本地缓存可用 ristretto；需要多副本防穿透可用更高级缓存框架；云厂商特性可直接用官方 SDK。
- 不适用：需要强一致分布式锁（需自建 Redlock）、极端内存敏感但无需持久化、或需要事务强隔离的场景。

## 所需输入（配置与推荐值）
- 地址：`Addr`（单机）、`Addrs`（Cluster）、`MasterName + SentinelAddrs`（Sentinel）；默认 `localhost:6379`，推荐走配置中心/服务发现。
- 认证：`Password`（密钥管理注入），开启 ACL 时还需 `Username`；禁止在代码/日志中出现明文。
- DB：`DB=0` 默认；业务隔离建议独立 DB 或实例。
- 连接池：`PoolSize` 默认 `10*GOMAXPROCS`，常用 50–200；`MinIdleConns` 10–20；`PoolTimeout` 推荐 1s。
- 超时与重试：`DialTimeout/ReadTimeout/WriteTimeout` 500ms–1s；`MaxRetries` 默认 3，通常 1–5；`MinRetryBackoff/MaxRetryBackoff` 8ms/200ms。
- TLS：跨机房或云上明文风险时必须开启 `TLSConfig`。
- 上下文：所有调用必须带 `context.Context`（含 request_id/trace_id），并设置 cancel/timeout。
- 关闭：进程退出前调用 `Close()`；PubSub 需单独 `Close()`。

## 何时使用
- 需要 Redis 读写 KV/哈希/集合/列表/发布订阅。
- 需要复用连接池、自动重试、并接入链路追踪或指标时。
- 需要流水线减少 RTT、事务 CAS、或 Sentinel/Cluster 自动路由时。
- 需要轻量分布式锁（SET NX + 有效期 + 安全释放）时。
- 需要基于 Redis 的令牌桶/限流时。

## 流程 / 步骤
1. 从配置中心读取地址、认证、池参数、超时、重试；禁止硬编码敏感信息。
2. 构造 `redis.Options` / `redis.ClusterOptions` / `redis.FailoverOptions`，填入推荐参数。
3. 创建客户端 `redis.NewClient` / `redis.NewClusterClient` / `redis.NewFailoverClient`。
4. 接入可观测性：调用 `redisotel.InstrumentTracing(client)` 与 `redisotel.InstrumentMetrics(client)`；如需日志 Hook，实现 `redis.Hook`。
5. 健康检查：`Ping` 一次，失败则阻断启动并告警。
6. 业务调用：所有命令使用带超时的 context；`redis.Nil` 视为未命中，不按错误处理。
7. 错误与重试：依赖 go-redis 内置重试；业务层仅对幂等操作追加重试，禁止无限重试。
8. 清理：进程退出钩子中 `Close()`；PubSub 或自建连接务必关闭。

## 输出格式 / 错误处理
- 读取结果：使用 `Cmd.Result()` / `Cmd.Val()`；错误通过 `Err()` 判断。
- 缓存未命中：`err == redis.Nil` -> 按未命中分支处理，不报警。
- 其他错误：记录 `request_id, redis_addr, cmd, key, elapsed_ms, retry_count, err`；必要时告警。
- 返回细节索引：详见 `outputs/` 子目录，覆盖 KV、Hash、List、Set/ZSet、PubSub 的返回与边界值：
  - `outputs/kv.md`：Get/MGet/Set/Exists/Del/TTL 等返回与未命中规则
  - `outputs/hash.md`：HGet/HGetAll/HMGet/HSet 等返回与空值
  - `outputs/list.md`：LPop/LRange/LLen 等返回与空列表
  - `outputs/set_zset.md`：S*/Z* 返回、空集合与 `redis.Nil` 区分
  - `outputs/pubsub.md`：订阅、发布返回、通道关闭语义
- 错误处理索引：详见 `errors_handle/` 子目录，覆盖常见错误的分类与处理建议：
  - `errors_handle/overview.md`：`redis.Nil`、超时、网络、池耗尽、服务器与业务错误分类及日志字段
  - `errors_handle/network_timeout.md`：超时与网络类错误配置、重试策略
  - `errors_handle/pipeline_tx.md`：Pipeline/事务/Lua 错误检查与重试
  - `errors_handle/cluster_pubsub.md`：Cluster（MOVED/CLUSTERDOWN/READONLY）与 PubSub 订阅错误处理
- 日志示例：
```go
logger.Info("redis failed",
    zap.String("request_id", rid),
    zap.String("redis_addr", addr),
    zap.String("cmd", "GET"),
    zap.String("key", "k1"),
    zap.Duration("elapsed", cost),
    zap.Int("retry_count", retries),
    zap.Error(err),
)
```

## 示例索引（按需查看）
- 初始化（单机）：`examples/init.md`
- 配置化 + 可观测性接入：`examples/config_observability.md`
- KV 读写与未命中处理：`examples/kv.md`
- Pipeline 减少 RTT：`examples/pipeline.md`
- 事务 CAS：`examples/txn_cas.md`
- 发布订阅：`examples/pubsub.md`
- 错误与超时处理：`examples/error_timeout.md`
- 关闭与清理：`examples/cleanup.md`
- 单元测试 Mock（miniredis）：`examples/mock.md`
- 分布式锁（安全释放）：`examples/lock.md`
- 令牌桶限流：`examples/token_bucket.md`

> 如需新增示例，按主题在 `examples/` 新建 `*.md` 并在此列表补充说明。

## 限制与安全规则
- 禁止在生产执行 `FLUSHALL/FLUSHDB`、大步长 `SCAN` 或全量 `KEYS *`。
- 连接池与 QPS：PoolSize 不应无上限放大；单实例 5–10w QPS 需监控 CPU/网络与延迟。
- 重试：仅对临时性网络错误重试；业务错误不重试；所有命令必须设置超时。
- 敏感信息：密码/完整连接串不得出现在日志；地址可脱敏到 host。
- 大对象：避免存储 >1MB Value，优先压缩或拆分。

## 常见坑 / FAQ
- 忘记处理 `redis.Nil`：未命中被当成错误，导致误报警。
- 未设置 context：使用 `context.Background()` 造成阻塞和 goroutine 泄漏。
- Pipeline 错误只在尾部检查：务必检查 `Pipelined` 返回的 err。
- PubSub 未关闭：连接泄漏；订阅退出要 `Close()`。
- Cluster 读从库：需要设置 `ReadOnly=true` 才会路由到从节点。

## 可观测性 / 诊断
- 链路追踪：`redisotel.InstrumentTracing(client)` 自动生成 span，包含命令、键数量、耗时。
- 指标：`redisotel.InstrumentMetrics(client)` 暴露命令耗时、连接池状态，可抓取到 Prometheus。
- 慢日志/错误日志字段：`request_id, cmd, key, elapsed_ms, retry_count, err`；对超时/连接错误单独计数。
- 自定义埋点：实现 `redis.Hook` 可在命令前后写日志或上报指标。

## 版本与依赖
- 版本：go-redis v9.0.5（兼容 v9.x），需 Go 1.20+。
- 依赖：Redis 实例（单机/哨兵/Cluster）；可选 `go.opentelemetry.io/contrib/instrumentation/github.com/redis/go-redis/redisotel`。
- 内部封装：如已有公共封装，优先复用配置加载、Hook 注入、统一日志。

## 更新记录 / Owner
- 最后更新时间：2024-05
- 维护人/评审人：请补充团队 Redis 组件 Owner/架构组联系人。
