# 关闭与清理

```go
func CloseRedis(cli *redis.Client) {
    if cli != nil {
        _ = cli.Close()
    }
}
```
