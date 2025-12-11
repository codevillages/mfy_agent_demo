# Hash 返回约定

- `HGet`: key 与 field 存在返回 `string`；field 不存在返回 `redis.Nil`；key 不存在同样 `redis.Nil`。
- `HGetAll`: 返回 `map[string]string`；key 不存在返回空 map（长度 0），无 `redis.Nil`。
- `HMGet`: 返回 `[]interface{}`，每个元素是 `string` 或 `nil`（field 不存在）；不会返回 `redis.Nil`。
- `HSet`: 返回 `int64`，表示新增 field 数量（更新已有 field 返回 0）。
- `HDel`: 返回 `int64`，删除的 field 数；不存在返回 0。
- `HLen`: 返回 `int64` field 数；不存在返回 0。
