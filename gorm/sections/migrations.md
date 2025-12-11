# 迁移与模型规范

## 模型约定
- 统一基类：
```go
type BaseModel struct {
    ID        int64          `gorm:"primaryKey"`
    CreatedAt time.Time      `gorm:"column:created_at"`
    UpdatedAt time.Time      `gorm:"column:updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}
```
- 命名：表名蛇形复数（默认），如需单数覆盖 `func (User) TableName() string { return "user" }`。
- 字段标签：`column`、`type`、`uniqueIndex`/`index`、`default`；避免省略类型导致默认长度不符。
- 软删除：使用 `gorm.DeletedAt`；硬删需 `Unscoped().Delete`。
- 时间：使用 `time.Time`，禁止 `string` 存储时间。
- 乐观锁：字段 `Version int64 "gorm:\"version\""`。

## 迁移策略
- 自动迁移仅在受控开关下执行：`ENABLE_AUTO_MIGRATE=true`（dev/staging）；生产需审批且建议使用 SQL 迁移工具。
- 禁止自动删除列/索引；`AutoMigrate` 不会删除，但也不会处理复杂变更（如字段类型修改）。
- 大表变更：使用在线 DDL（gh-ost/pt-osc）或分批迁移。
- 索引命名：`idx_<table>_<cols>`；唯一索引 `uk_<table>_<cols>`。
- 兼容性：新增非空列需带默认值；避免长事务锁表。

## 迁移示例
```go
if os.Getenv("ENABLE_AUTO_MIGRATE") == "true" {
    if err := db.WithContext(ctx).AutoMigrate(&User{}, &Order{}); err != nil {
        return err
    }
}
```

## 模型示例
```go
type User struct {
    BaseModel
    Name   string `gorm:"column:name;type:varchar(64);not null;index"`
    Email  string `gorm:"column:email;type:varchar(128);uniqueIndex"`
    Status int    `gorm:"column:status;type:tinyint(1);default:0"`
}
```
