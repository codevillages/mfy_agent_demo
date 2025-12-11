# 初始化（单机）

```go
rdb := redis.NewClient(&redis.Options{
    Addr:         "localhost:6379",
    Password:     "", // 从密钥服务获取
    DB:           0,
    PoolSize:     100,
    MinIdleConns: 10,
    DialTimeout:  500 * time.Millisecond,
    ReadTimeout:  500 * time.Millisecond,
    WriteTimeout: 500 * time.Millisecond,
    MaxRetries:   3,
})
ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()
if err := rdb.Ping(ctx).Err(); err != nil { panic(err) }
defer rdb.Close()
```
