# PubSub 返回约定

- `Subscribe`：返回 `*redis.PubSub`；订阅失败返回错误。
- `PubSub.Channel()`：返回 `<-chan *redis.Message`；当调用 `Close()` 或 ctx 取消时通道关闭。
- `ReceiveMessage`: 阻塞直到收到消息或发生错误；超时需用带超时的 context。
- `Publish`: 返回 `int64`，表示成功接收的订阅者数量；无订阅者返回 0。
