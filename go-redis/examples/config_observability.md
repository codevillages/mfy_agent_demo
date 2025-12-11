# 配置化与可观测性接入

```go
type Config struct {
    Addr         string
    Username     string
    Password     string
    DB           int
    PoolSize     int
    MinIdle      int
    MaxRetries   int
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
}

func NewRedisClient(cfg Config) (*redis.Client, error) {
    opts := &redis.Options{
        Addr:         cfg.Addr,
        Username:     cfg.Username,
        Password:     cfg.Password,
        DB:           cfg.DB,
        PoolSize:     cfg.PoolSize,
        MinIdleConns: cfg.MinIdle,
        MaxRetries:   cfg.MaxRetries,
        ReadTimeout:  cfg.ReadTimeout,
        WriteTimeout: cfg.WriteTimeout,
    }
    cli := redis.NewClient(opts)

    // 接入 OpenTelemetry
    _ = redisotel.InstrumentTracing(cli)
    _ = redisotel.InstrumentMetrics(cli)

    if err := cli.Ping(context.Background()).Err(); err != nil {
        return nil, err
    }
    return cli, nil
}
```
