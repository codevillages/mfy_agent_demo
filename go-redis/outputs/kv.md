# KV 返回约定

- `Get`: 存在返回 `string` 与 `nil` 错误；不存在返回 `""` + `redis.Nil`；错误需判空。
- `MGet`: 返回 `[]interface{}`，对应值为 `string` 或 `nil`（未命中）。整体错误只在网络/序列化等异常时出现。
- `Set`: 返回 `string` `"OK"`；错误表示失败。常用 `Err()` 判断。
- `SetNX`: 返回 `bool` 是否成功；不存在写入成功返回 `true`，否则 `false`；错误为异常。
- `Exists`: 返回 `int64` 计数，表示存在的 key 数量；不存在时为 `0`，无 `redis.Nil`。
- `Del`: 返回 `int64` 删除的 key 数；不存在返回 `0`。
- `TTL`/`PTTL`: 返回 `time.Duration`；不存在返回 `-2`；存在但无过期返回 `-1`。
- `Expire`/`PExpire`: 返回 `bool`，成功设置过期返回 `true`，不存在返回 `false`。
- `Incr`/`Decr`: 返回 `int64` 新值；不存在会先按 0 递增。
