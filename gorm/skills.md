# @mfycommon/gorm v1.22.4 — Skills

## 标题 / 目的 / 适用场景
- 库名+版本：`@mfycommon/gorm` v1.22.4，用于 MySQL ORM 访问、迁移、事务、查询与可观测性。
- 推荐用在：中小型业务表的 CRUD、查询、事务、软删除、审计字段统一；需要链路日志/trace/metrics。
- 替代方案：极端性能或精细 SQL 优化用 `database/sql` + `sqlx`；纯 KV/缓存不必用 ORM。
- 不适用：需要跨数据库特性、极端性能/锁控制且 ORM 抽象有损的场景。

## 所需输入
- DSN：`user:pass@tcp(host:port)/db?charset=utf8mb4&parseTime=True&loc=Local&timeout=1s&readTimeout=1s&writeTimeout=2s`（密码走密钥/环境变量）。
- 连接池：`MaxOpenConns` 80、`MaxIdleConns` 20、`ConnMaxLifetime` 1h、`ConnMaxIdleTime` 10m（按压测调整）。
- 日志：从 `pkg/log` 获取 zap.Logger，注入 gorm logger；prod 级别 `Warn`，慢阈值 200ms，忽略 not found。
- 超时：读 1s、写 3s，所有调用必须 `WithContext(context.WithTimeout(...))`。
- 开关：`ENABLE_AUTO_MIGRATE` 默认 false（仅 dev/staging），生产需审批。
- 环境变量：`MYSQL_DSN`、`LOG_LEVEL`、`APP_ENV`、`ENABLE_AUTO_MIGRATE`。

## 流程 / 工作流程
1) 从 `pkg/log` 获取 logger，构建 gorm logger（带 request_id/trace_id，慢阈值 200ms）。
2) `gorm.Open(mysql.New(mysql.Config{DSN: ...}), &gorm.Config{Logger: gormLogger, DisableForeignKeyConstraintWhenMigrating: true, NamingStrategy: schema.NamingStrategy{SingularTable:false}})`;
3) 设置池：`sqlDB.SetMaxOpenConns/SetMaxIdleConns/SetConnMaxLifetime/SetConnMaxIdleTime`；
4) 受控迁移：`if ENABLE_AUTO_MIGRATE { db.WithContext(ctx).AutoMigrate(&User{}, ...) }`;
5) 业务查询/更新均 `db.WithContext(ctx)` + `context.WithTimeout`；更新必须带 where；
6) 事务：`db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { ... })`，事务内避免外部 I/O；
7) 收尾：进程退出钩子关闭 `sqlDB.Close()`，flush 日志。

### 关键代码片段
```go
zapLogger := pkglog.FromContext(ctx)
gormLogger := gormzap.New(zapLogger, gormzap.Config{
    SlowThreshold: 200 * time.Millisecond,
    LogLevel:      logger.Warn,
    IgnoreRecordNotFoundError: true,
})
db, _ := gorm.Open(mysql.New(mysql.Config{DSN: os.Getenv("MYSQL_DSN")}),
    &gorm.Config{Logger: gormLogger, DisableForeignKeyConstraintWhenMigrating: true})
sqlDB, _ := db.DB()
sqlDB.SetMaxOpenConns(80); sqlDB.SetMaxIdleConns(20)
sqlDB.SetConnMaxLifetime(time.Hour); sqlDB.SetConnMaxIdleTime(10*time.Minute)
if os.Getenv("ENABLE_AUTO_MIGRATE") == "true" { _ = db.WithContext(ctx).AutoMigrate(&User{}) }
ctxQ, cancel := context.WithTimeout(ctx, time.Second); defer cancel()
var u User; err := db.WithContext(ctxQ).First(&u, "id=?", id).Error
```

## 何时使用该技能
- 需要标准化的 MySQL ORM 访问、软删除、审计字段、迁移、事务与链路日志。
- 需要结合请求上下文的超时/trace/request_id 贯穿 DB 操作。
- 需要内置慢查询日志与指标观测。

## 输出格式 / 错误处理
- 日志字段：`request_id`、`trace_id`、`db.addr`、`sql`（脱敏/截断）、`rows`、`elapsed_ms`、`err`；参数可 hash/截断防泄露。
- 返回模式（仓储层）：
  - 单条：`func FindUser(ctx, id) (*User, error)`；未命中返回 `(nil, gorm.ErrRecordNotFound)`，业务转换为 NotFound。
  - 多条：`func List(ctx, filter) ([]User, error)`；无结果返回 `[]User{}`+`nil`（不返回 not found）。
  - 计数/聚合：返回基本类型+`error`，未命中返回零值+`nil`。
  - 分页：返回 `items []T`、`total int64`、`error`；空列表时 `total=0`，`items=[]`。
- 查询：
  - 找到：返回实体/DTO。
  - 未命中：`err == gorm.ErrRecordNotFound`，不记错误日志，转换为业务 NotFound。
  - 其他错误：包装并记录，附 `rows=0`、SQL、耗时。
- 新增：
  - 成功：`RowsAffected > 0`，返回主键或实体；日志记录 `rows`、耗时。
  - 失败：返回错误，记录 SQL/参数（脱敏）。
- 更新：
  - 成功：`RowsAffected > 0`；为 0 时判定条件未命中或版本冲突。
  - 乐观锁失败：`err == gorm.ErrOptimisticLockingFailure`，映射业务冲突；可有限重试。
- 删除：
  - 软删成功：`RowsAffected > 0`；硬删用 `Unscoped()`。
  - 未命中：`RowsAffected == 0`，可视为未删除，不记错误。
- 批量：
  - `CreateInBatches` 返回插入数量；超大批需分批。
- 原生 SQL：
  - `Raw.Scan`/`Exec` 必检查 `err` 和 `RowsAffected`；避免空 `IN ()`。
- 空/极端条件：
  - 空切片过滤：直接返回空结果，避免生成无效 SQL。
  - 可选条件为空时不拼 where，保持最小 SQL。
- 错误映射建议：
  - NotFound：`gorm.ErrRecordNotFound` -> 业务 `NotFound`。
  - Conflict：`ErrOptimisticLockingFailure` -> 业务冲突。
  - Timeout：`context.DeadlineExceeded` -> 504/依赖超时。
  - Validation/参数：在仓储前处理，禁止拼接不可信 SQL。

## 示例（按需加载，覆盖常见场景）
1. 初始化 + 迁移开关：`sections/examples.md` 片段 1
2. 查询 + 预加载 + 字段选择：片段 2
3. 条件更新 / 零值更新：片段 3、12
4. 乐观锁：片段 4
5. Upsert（唯一键冲突更新）：片段 5
6. 事务模板：片段 6
7. 原生 SQL + 扫描：片段 7
8. 批量插入：片段 8
9. 软删除与恢复：片段 9
10. 并发控制（悲观锁）：片段 10
11. 分页 Scope：片段 11
12. 局部字段更新避免零值丢失：片段 12

## 限制条件与安全规则
- 禁止无 where 的更新/删除；禁止拼接不可信 SQL。
- 禁止未带 ctx/超时的 DB 调用；事务内禁止外部 I/O。
- 生产禁止随意 AutoMigrate，大表需评估/审批。
- 敏感数据日志需脱敏；禁止记录明文密码/身份证/手机号。
- 池参数必须显式设置；慢查询阈值≥200ms，超出需告警/优化。

## 常见坑 / FAQ
- 忘记 `WithContext` 或无超时导致连接泄漏。
- `Save` 覆盖零值：需 `Select` 指定字段或 `UpdateColumn`。
- `ErrRecordNotFound` 当错误处理：应分支为未命中。
- 时区/loc 未设导致时间偏移：DSN 必带 `parseTime=True&loc=Local`。
- 预加载拉全字段：`Preload` 配合 `Select`。
- 忽略 `RowsAffected` 导致误以为成功。

## 可观测性 / 诊断
- Metrics：记录 `latency`、`qps`、`error_rate`、`rows` 按 `op/table/status` 打标签。
- Tracing：`WithContext` 透传 trace，gorm 插件/回调在 span 记录 SQL（脱敏/截断）。
- 慢日志：>200ms Warn；字段含 `request_id`、`trace_id`、`op`、`table`、`rows`、`duration_ms`。
- 诊断：`context deadline exceeded` 检查池/超时；锁等待查看 `INNODB_TRX`/`LOCKS`。

## 版本与依赖
- gorm v1.22.4；Go 1.17+（建议 1.19+）；MySQL 5.7+/8.0+。
- 依赖：`gorm.io/gorm`、`gorm.io/driver/mysql`；团队 `pkg/log`。
- 目录：`mfycommon/gorm/sections/*` 为细节规范与示例。

## 更新记录 / Owner
- 最后更新时间：2024-05-xx。
- Owner：数据库规范小组（@架构负责人），评审人：服务域负责人；变更需双人 Review。
