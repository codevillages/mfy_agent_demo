# ServiceContext 组织与健康检查

```go
type ServiceContext struct {
    Config   config.Config
    DB       sqlx.SqlConn
    Redis    *redis.Redis
    UserRepo repo.User
    PayCli   pb.PayClient // RPC client
}

func NewServiceContext(c config.Config) *ServiceContext {
    db := sqlx.NewMysql(c.Mysql.DataSource)
    rdb := redis.MustNewRedis(c.Redis)
    payCli := pb.NewPayClient(zrpc.MustNewClient(c.PayRpc).Conn())
    return &ServiceContext{
        Config:   c,
        DB:       db,
        Redis:    rdb,
        UserRepo: repo.NewUserModel(db, cache.New(c.CacheRedis, c.CacheExpire)),
        PayCli:   payCli,
    }
}

func (s *ServiceContext) Close() {
    if s.Redis != nil { s.Redis.Close() }
    logx.Close()
    // DB 为 sqlx.SqlConn，连接池由驱动管理，无需显式 Close
}
```

## 健康与就绪
- REST：`/health` 快速返回；`/ready` 可检查 DB/Redis/RPC Ping。
- RPC：实现 gRPC health check（`grpc/health/grpc_health_v1`），供探针调用。
- 启动前 Ping 关键依赖，失败退出，避免注册成功但依赖不可用。
