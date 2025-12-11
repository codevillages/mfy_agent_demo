# 工作流程（初始化 -> 使用 -> 关闭）

```go
// 1) 构建 logger（从 pkg/log 获取全局 zap logger），注入 gorm
zapLogger := pkglog.FromContext(ctx) // 示例：自研封装
gormLogger := gormzap.New(zapLogger, gormzap.Config{
    SlowThreshold: 200 * time.Millisecond,
    LogLevel:      logger.Warn,
    IgnoreRecordNotFoundError: true,
})

// 2) 打开连接（context + DSN + 连接池）
db, err := gorm.Open(mysql.New(mysql.Config{
    DSN:                       os.Getenv("MYSQL_DSN"),
    SkipInitializeWithVersion: false,
}), &gorm.Config{
    Logger:                                   gormLogger,
    DisableForeignKeyConstraintWhenMigrating: true,
    NamingStrategy: schema.NamingStrategy{
        SingularTable: false, // 默认复数表名，需单数时改 true
    },
})
if err != nil { return err }
sqlDB, _ := db.DB()
sqlDB.SetMaxOpenConns(80)
sqlDB.SetMaxIdleConns(20)
sqlDB.SetConnMaxLifetime(time.Hour)
sqlDB.SetConnMaxIdleTime(10 * time.Minute)

// 3) 自动迁移（受开关控制）
if os.Getenv("ENABLE_AUTO_MIGRATE") == "true" {
    if err := db.WithContext(ctx).AutoMigrate(&User{}); err != nil { return err }
}

// 4) 查询/更新均使用 ctx 超时
ctx, cancel := context.WithTimeout(ctx, time.Second)
defer cancel()
var u User
if err := db.WithContext(ctx).First(&u, "id = ?", id).Error; err != nil { ... }

// 5) 事务
err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&Order{...}).Error; err != nil { return err }
    if err := tx.Model(&Stock{}).Where("id=?", sid).Update("count", gorm.Expr("count-?", 1)).Error; err != nil { return err }
    return nil
})
if err != nil { ... }

// 6) 关闭：在进程退出 hook 关闭 sql.DB
sqlDB.Close()
```

## API 选型
- 读：`First/Take/Find`；`First` 必带条件或主键。
- 写：`Create/Save/Updates/UpdateColumn`；避免 `Save` 混用零值。
- 删除：软删默认；硬删用 `Unscoped().Delete`.
- 预加载：`Preload("Assoc")`；控制字段选择避免 N+1。
- 乐观锁：字段 `version` + `gorm:"column:version;type:int;default:0;version"`。
- Upsert：`Clauses(clause.OnConflict{UpdateAll: true})` 或 `DoUpdates`。
- 原生 SQL：`db.Raw` / `db.Exec`，仍需 `WithContext`.
- Scopes：封装分页、通用过滤，例如 `db.Scopes(Paginate(page,size))`。
- 模型基类：优先嵌入 `BaseModel`（见 `migrations.md`），统一审计字段/软删除。

## 约束
- 禁止在业务层直接拼接不可信 SQL；使用参数化。
- 禁止无条件 `Updates`/`Delete`（必须带 where）。
- 禁止在查询中省略 ctx；禁止不设置连接池。
- AutoMigrate 仅在受控环境运行，不得在生产随意执行。
