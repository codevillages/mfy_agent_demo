# Pipeline 减少 RTT

```go
_, err := rdb.Pipelined(ctx, func(p redis.Pipeliner) error {
    p.Incr(ctx, "count")
    p.Expire(ctx, "count", time.Hour)
    p.MSet(ctx, "a", 1, "b", 2)
    return nil
})
if err != nil {
    return err
}
```
