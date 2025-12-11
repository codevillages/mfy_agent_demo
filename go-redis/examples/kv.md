# KV 读写与未命中处理

```go
ctx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
defer cancel()

if err := rdb.Set(ctx, "k1", "v1", 10*time.Minute).Err(); err != nil {
    return err
}

val, err := rdb.Get(ctx, "k1").Result()
switch {
case errors.Is(err, redis.Nil):
    // cache miss
case err != nil:
    return err
default:
    _ = val
}
```
