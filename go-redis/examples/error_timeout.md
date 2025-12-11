# 错误与超时处理

```go
ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
defer cancel()

if err := rdb.Set(ctx, "k", "v", 0).Err(); err != nil {
    // 记录/告警，关注是否为超时或连接错误
}
```
