# 配置与所需输入
- 连接串：`DSN` 必填（禁止硬编码密码，走密钥/环境变量）。
- 连接池：`MaxOpenConns` 推荐 80、`MaxIdleConns` 20、`ConnMaxLifetime` 1h、`ConnMaxIdleTime` 10m；高并发按压测调优。
- 日志：使用自研 `pkg/log`；`Logger` 需注入 gorm 的自定义 logger（支持 request_id/trace_id）。
- 超时：所有操作包裹 `context.WithTimeout`（读 1s、写 3s）；必须使用 `WithContext(ctx)`。
- 迁移：`AutoMigrate` 需显式开启且仅在部署阶段/启动开关下运行。
- 事务：使用 `db.Transaction(func(tx *gorm.DB) error { ... })` 或 `Begin/Commit/Rollback`；上下文透传。
- 软删除：统一使用 `gorm.DeletedAt`，字段名 `deleted_at`。
- 时间字段：`created_at`、`updated_at` 使用 `gorm.Model` 或自定义标签。
- 环境变量：`MYSQL_DSN`、`LOG_LEVEL`、`APP_ENV`。
- 推荐默认 Logger 级别：prod 用 `Warn`，dev 用 `Info`，禁用 `Colorful`。

## 推荐默认与 DSN 模板
- DSN 示例：`user:pass@tcp(127.0.0.1:3306)/db?charset=utf8mb4&parseTime=True&loc=Local&timeout=1s&readTimeout=1s&writeTimeout=2s`
- 必选参数：`parseTime=True`、`charset=utf8mb4`、合理的读/写超时；可按需指定 `loc=Asia%2FShanghai`。
- 连接池：`MaxOpenConns=80`、`MaxIdleConns=20`、`ConnMaxLifetime=1h`、`ConnMaxIdleTime=10m`。
- Logger：`SlowThreshold=200ms`；`IgnoreRecordNotFoundError=true`；prod `Warn`，dev `Info`；禁用彩色输出。
- 自动迁移开关：`ENABLE_AUTO_MIGRATE`（默认 false），生产需审批。
- 时区：统一 `loc=Local`（或 `Asia/Shanghai`），避免时间偏移。

## 字段说明
- DSN 组成：用户/密码、host:port、dbname、charset、parseTime、loc、超时。
- Pool：打开/空闲连接数、生命周期/空闲回收。
- Logger：级别、慢阈值、忽略 not found、是否彩色输出。
- Retry：gorm 不自带重试；业务侧仅对幂等读加有限重试。

## 启动校验清单
- 检查 `DSN` 非空且 `parseTime=True`。
- `db.DB().Ping()` 成功。
- 池参数已设置（非默认 0）。
- Logger 已注入自定义（包含 request_id/trace_id）。
