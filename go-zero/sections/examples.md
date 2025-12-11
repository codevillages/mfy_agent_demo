# 示例索引（高频用例）
- 最小 REST：`examples/rest/main.go` + `examples/rest/etc/user-api.yaml`
- 最小 RPC：`examples/rpc/main.go` + `examples/rpc/etc/user-rpc.yaml`
- RPC 客户端调用：见下方片段
- Handler -> Logic -> Repo：使用 goctl 生成目录，handler 仅校验与调用 logic，repo 访问 DB/Redis
- SQLX + 模型缓存、Redis 操作、中间件、熔断/限流、KQ 消费者、Graceful、定时任务、JWT 鉴权：片段如下

```go
// RPC 客户端调用
cli := zrpc.MustNewClient(c.TargetRpc)
userCli := pb.NewUserClient(cli.Conn())
ctx, cancel := context.WithTimeout(r.Context(), time.Millisecond*time.Duration(c.TargetRpc.Timeout))
defer cancel()
resp, err := userCli.GetUser(ctx, &pb.UserReq{Id: req.Id})
```

```go
// SQLX + 模型缓存
db := sqlx.NewMysql(c.Mysql.DataSource)
cache := sqlc.NewCache(c.CacheRedis, c.CacheExpire)
userModel := model.NewUserModel(db, cache)
u, err := userModel.FindOne(ctx, id)
```

```go
// Redis
rdb := redis.MustNewRedis(c.Redis)
v, err := rdb.GetCtx(ctx, key)
if err == redis.Nil { return nil } // miss
```

```go
// 中间件启用 request_id/metrics
server.Use(middleware.RequestID())
server.Use(middleware.Metrics())
```

```go
// 熔断/限流包装外部调用
err := breaker.DoWithAcceptable("pay", func() error { return payCli.Pay(ctx, req) },
    breaker.Acceptable(func(err error) bool { return errors.Is(err, errs.Biz) }))
limiter := tokenlimit.NewTokenLimiter(100, time.Second)
if !limiter.Allow() { return errs.TooManyRequests }
```

```go
// KQ 消费者（Kafka）
q := kq.MustNewQueue(c.KqConf, kq.WithHandler(func(ctx context.Context, v []byte) error { return handleMsg(ctx, v) }))
defer q.Stop(); q.Start()
```

```go
// Graceful 收尾
defer func() { server.Stop(); rpc.Stop(); svcCtx.Close(); logx.Close(); tp.Shutdown(context.Background()) }()
```

```go
// 定时任务（cron-like）
timer := timex.NewScheduler()
timer.AddFunc("@every 1m", func() { _ = job.Do(ctx) })
defer timer.Stop()
```

```go
// JWT 鉴权
server.Use(auth.NewAuthorizer(c.Auth.AccessSecret))
```
