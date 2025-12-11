# 发布订阅

```go
sub := rdb.Subscribe(ctx, "topic")
defer sub.Close()

for msg := range sub.Channel() {
    log.Printf("topic=%s payload=%s", msg.Channel, msg.Payload)
}
```
