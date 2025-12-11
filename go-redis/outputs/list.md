# List 返回约定

- `LPush`/`RPush`: 返回 `int64`，列表长度；key 不存在会创建并返回新长度。
- `LPop`/`RPop`: 有元素返回 `string`，空列表或 key 不存在返回 `redis.Nil`。
- `LRange`: 返回 `[]string`；空列表或 key 不存在返回空 slice（`len=0`），无 `redis.Nil`。
- `LLen`: 返回 `int64`；空列表或 key 不存在返回 0。
