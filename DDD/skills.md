# @mfycommon/DDD v1.0 — DDD Skills 指南

## 标题/目的/适用场景
- 名称：`@mfycommon/DDD` v1.0，面向中大型业务服务的领域建模与演进。
- 推荐用在：需要清晰业务边界、复杂规则、可演化的核心域；需要事件驱动、可插拔外部依赖时。
- 替代方案：简单 CRUD/薄服务时优先直接用三层架构或轻量脚手架；原型验证可用快速 API 框架避免过度设计。
- 不适用：数据同步/报表批处理为主、无清晰领域语义、单次脚本类任务。

## 所需输入
- 环境变量：`APP_ENV`(default: `dev`)、`LOG_LEVEL`(default: `info`)、`HTTP_PORT`(default: `8080`)。
- 基础设施：`DB_DSN`/`READONLY_DB_DSN`、`CACHE_ADDR`、`EVENT_BUS_URL`(Kafka/NATS)、`TRACE_ENDPOINT`，缺失时禁止启用对应依赖。
- 资源限制推荐：DB 连接池 `max_open=100`、`max_idle=20`；HTTP server `read_timeout=5s`、`write_timeout=10s`；全局超时 `context.WithTimeout` 默认 3s。
- 日志：通过 `pkg/log` 获取全局 logger；请求需携带 `request_id`，默认从上下文提取。

## 流程/工作流程
1. **划定界限上下文**：为每个核心域建目录 `internal/<bounded_context>`，定义领域语言词典，禁止跨上下文直接调用实体。
2. **聚合建模**：实体/值对象定义在 `domain/`，聚合根负责不变量检查；仓储接口在 `domain/repository.go`。
3. **应用服务编排**：在 `app/service` 层暴露用例方法，只调用聚合根行为+领域服务，禁止直接操作 DAO。
4. **领域事件**：定义事件结构体 `domain/event/*.go`，发布使用 `event.Bus`（封装于 `pkg/eventbus`）；订阅处理器放在 `app/handler`，确保幂等。
5. **基础设施实现**：仓储适配器放 `infra/repository`，使用事务模板 `pkg/db.WithTx(ctx, func(tx *sql.Tx) error { ... })`；缓存放 `infra/cache`。
6. **日志链路**：使用 `pkg/log.FromContext(ctx)` 获取 logger；必须带 `request_id`/`biz` 字段；退出时调用 `log.Flush()`。
7. **接口层**：HTTP/RPC handler 仅负责参数校验与调用应用服务；错误统一经 `pkg/errs` 转换为返回码。
8. **关闭与资源回收**：在 `cmd/<svc>/main.go` 的进程钩子里依次停止 handler、flush 日志、关闭连接池/producer。

### 关键指令示例
```go
logger := log.FromContext(ctx).With(log.String("request_id", reqID))
order, err := svc.PlaceOrder(ctx, req)
if err != nil {
    return nil, errs.Wrap(err, errs.CodeBiz, "place order failed")
}
return orderDTO.From(order), nil
```

## 何时使用该技能
- 需要长期演进的核心业务域，要求清晰边界、规则可迭代。
- 需要事件驱动的跨上下文协作或异步解耦。
- 需要将易变的外部依赖与稳定的领域模型解耦时。
- 需要可测试、可替换的基础设施实现时。

## 输出格式
- 日志字段：必须包含 `ts`、`level`、`request_id`、`biz`、`event`；敏感字段（手机号/身份证）脱敏后写入。
- 返回约定：应用层返回 DTO，错误使用 `pkg/errs` 包装，外露 `code`+`message`；`nil` 资源（如 Redis miss）返回 `errs.NotFound`。
- 领域事件：`event_name` 使用 `<context>.<aggregate>.<action>`，payload 只含业务字段，不含基础设施细节。

## 示例（最小可运行 + 常用包装）
1. 初始化服务
```go
func main() {
    ctx := context.Background()
    log.Init(log.Config{Level: os.Getenv("LOG_LEVEL")})
    db := db.New(os.Getenv("DB_DSN"))
    bus := eventbus.New(os.Getenv("EVENT_BUS_URL"))
    svc := service.NewOrder(db, bus)
    http.Run(ctx, svc)
    defer func() { log.Flush(); bus.Close(); db.Close() }()
}
```
2. 创建聚合并持久化
```go
order := domain.NewOrder(userID, items)
if err := orderRepo.Save(ctx, order); err != nil { return err }
```
3. 处理错误/重试
```go
err = retry.Do(ctx, 3, func() error { return orderRepo.Save(ctx, order) })
if errors.Is(err, errs.Transient) { metrics.Inc("order.save.retry_exhausted") }
```
4. 并发与连接管理
```go
g, ctx := errgroup.WithContext(ctx)
g.Go(func() error { return inventorySvc.Reserve(ctx, order) })
g.Go(func() error { return paymentSvc.Authorize(ctx, order) })
if err := g.Wait(); err != nil { return errs.Wrap(err, errs.CodeBiz, "order flow") }
```
5. 发布/消费领域事件
```go
bus.Publish(ctx, domain.OrderCreatedEvent{OrderID: order.ID})
bus.Subscribe("order.created", handler.OnOrderCreated)
```
6. 读写分离
```go
order, err := orderQueryRepo.FindByID(ctx, readOnlyDB, id)
```
7. 事务包裹
```go
return db.WithTx(ctx, func(tx *sql.Tx) error {
    if err := orderRepo.SaveTx(ctx, tx, order); err != nil { return err }
    return outboxRepo.SaveTx(ctx, tx, domain.NewOutbox(order))
})
```
8. 收尾清理
```go
defer func() { cache.Close(); bus.Close(); log.Flush() }()
```

## 限制条件与安全规则
- 禁止跨聚合直接修改对方内部状态；只能通过领域服务/事件。
- 禁止在应用层绕过聚合根校验直接改数据库。
- 性能红线：单实例 QPS 过 2k 前必须压测；连接池溢出需降级或拒绝；重试总时长不超 15s。
- 安全：日志/事件中必须脱敏敏感信息；禁止记录秘钥、Token、完整身份证/手机号。
- 超时/重试：读 1s、写 3s、外部依赖重试 3 次带指数退避；幂等键必填。
- 不可用 API：禁止直接 new logger/DB 客户端，统一从 `pkg/log`、`pkg/db` 获取。

## 常见坑/FAQ（按严重度）
- 高：聚合过大导致事务长锁—拆分子聚合并通过事件保持最终一致。
- 高：事件消费非幂等—引入幂等键或去重表，消费前先检查 processed 标记。
- 中：仓储泄漏基础设施类型到领域层—用接口+DTO 转换隔离。
- 中：错误码不统一—所有错误通过 `pkg/errs` 包装并映射。
- 低：读写分离下读到旧数据—读接口允许指定 `force_primary`，或延迟读。

## 可观测性/诊断
- Metrics：为用例入口、仓储、外部依赖记录 `latency`、`qps`、`error_rate`；事件消费记录 `lag`。
- Tracing：在 handler 入口开始 span，跨上下文透传 `trace_id`；事件发布/消费打上 `event.name`、`aggregate` 标签。
- 慢日志：对超过 500ms 的应用服务和 100ms 的仓储操作输出 warn，携带 `request_id`、`aggregate`、`op`、`duration_ms`。

## 版本与依赖
- 语言/框架：Go 1.21+；日志 `pkg/log`（zap 封装）；错误 `pkg/errs`；事件总线 `pkg/eventbus`；DB `pkg/db`。
- 兼容的外部服务：MySQL 5.7+/8.0+、Redis 6+、Kafka 2.8+/NATS 2.9+。
- 内部封装路径：`mfycommon/DDD`（规范）、`pkg/*`（基础设施）、`internal/<bounded_context>`（领域实现）。

## 更新记录 / Owner
- 最后更新时间：2024-05-xx（首次落地）。
- Owner：DDD 小组（@架构负责人），评审人：服务域负责人；变更需经双人 Review。
