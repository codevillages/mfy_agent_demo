# 高频示例

1) 初始化 + AutoMigrate（受开关控制）
```go
func InitDB(ctx context.Context) (*gorm.DB, func() error, error) {
    zapLogger := pkglog.FromContext(ctx)
    gormLogger := gormzap.New(zapLogger, gormzap.Config{
        SlowThreshold: 200 * time.Millisecond,
        LogLevel:      logger.Warn,
        IgnoreRecordNotFoundError: true,
    })
    db, err := gorm.Open(mysql.Open(os.Getenv("MYSQL_DSN")), &gorm.Config{
        Logger:                                   gormLogger,
        DisableForeignKeyConstraintWhenMigrating: true,
    })
    if err != nil { return nil, nil, err }
    sqlDB, _ := db.DB()
    sqlDB.SetMaxOpenConns(80); sqlDB.SetMaxIdleConns(20)
    sqlDB.SetConnMaxLifetime(time.Hour); sqlDB.SetConnMaxIdleTime(10*time.Minute)
    if os.Getenv("ENABLE_AUTO_MIGRATE") == "true" {
        if err := db.WithContext(ctx).AutoMigrate(&User{}); err != nil { return nil, nil, err }
    }
    return db, sqlDB.Close, nil
}
```

2) 查询 + 预加载 + 字段选择
```go
var u User
err := db.WithContext(ctx).
    Select("id,name,email").
    Preload("Profile", func(db *gorm.DB) *gorm.DB { return db.Select("user_id,age") }).
    First(&u, "id = ?", id).Error
```

3) 条件更新（避免零值被忽略用 `Select` 或 `UpdateColumn`）
```go
err := db.WithContext(ctx).
    Model(&User{}).Where("id=?", id).
    Select("nickname", "status").
    Updates(map[string]interface{}{"nickname": nick, "status": status}).Error
```

4) 乐观锁
```go
type Order struct {
    ID      int64
    Status  string
    Version int64 `gorm:"version"`
}
err := db.WithContext(ctx).Model(&Order{}).
    Where("id = ? AND version = ?", id, ver).
    Updates(map[string]interface{}{"status": "paid"}).Error
if errors.Is(err, gorm.ErrOptimisticLockingFailure) { /* 重试或返回业务冲突 */ }
```

5) Upsert（有唯一键冲突时更新）
```go
err := db.WithContext(ctx).
    Clauses(clause.OnConflict{
        Columns:   []clause.Column{{Name: "email"}},
        DoUpdates: clause.AssignmentColumns([]string{"name", "updated_at"}),
    }).
    Create(&User{Name: name, Email: email}).Error
```

6) 事务模板
```go
err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&Order{...}).Error; err != nil { return err }
    if err := tx.Exec("update stock set count = count-1 where id=? and count>0", sid).Error; err != nil { return err }
    return nil
})
```

7) 原生 SQL + 扫描
```go
type Res struct{ ID int64; Total int64 }
var out []Res
err := db.WithContext(ctx).Raw(`select user_id as id, sum(amount) as total from orders where status=? group by user_id limit ?`, "paid", 100).Scan(&out).Error
```

8) 批量插入（分批）
```go
batch := make([]User, 0, 100)
// fill batch...
err := db.WithContext(ctx).CreateInBatches(batch, 100).Error
```

9) 软删除与恢复
```go
if err := db.WithContext(ctx).Delete(&User{}, id).Error; err != nil { ... }          // 软删
if err := db.Unscoped().WithContext(ctx).Model(&User{}).Where("id=?", id).Update("deleted_at", nil).Error; err != nil { ... } // 恢复
```

10) 并发控制（悲观锁示例）
```go
err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    var u User
    if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, "id=?", id).Error; err != nil { return err }
    u.Balance -= amt
    return tx.Save(&u).Error
})
```

11) 分页 Scope
```go
func Paginate(page, size int) func(db *gorm.DB) *gorm.DB {
    if page < 1 { page = 1 }
    if size <= 0 || size > 200 { size = 20 }
    offset := (page - 1) * size
    return func(db *gorm.DB) *gorm.DB { return db.Offset(offset).Limit(size) }
}
db.WithContext(ctx).Scopes(Paginate(page, size)).Find(&users)
```

12) 部分字段更新避免零值丢失
```go
err := db.WithContext(ctx).
    Model(&User{}).
    Where("id=?", id).
    Updates(map[string]interface{}{"age": age, "nickname": nick}).Error
// 若需更新为零值，使用 UpdateColumn 或 Select 指定字段
```
