# Set / Sorted Set 返回约定

## Set
- `SAdd`: 返回 `int64` 新增元素数量；已存在元素不计入。
- `SMembers`: 返回 `[]string`；key 不存在返回空 slice（`len=0`）。
- `SIsMember`: 返回 `bool`；key 不存在返回 `false`。
- `SRem`: 返回 `int64` 删除的元素数量；不存在返回 0。

## Sorted Set
- `ZAdd`: 返回 `int64` 新增成员数量（更新分数不计入）。
- `ZRange`/`ZRevRange`: 返回 `[]string`；key 不存在返回空 slice。
- `ZRangeWithScores`: 返回 `[]Z`, 其中 `Z{Member any; Score float64}`；key 不存在返回空 slice。
- `ZScore`: 成员存在返回 `float64`；成员不存在或 key 不存在返回 `redis.Nil`。
- `ZRem`: 返回 `int64` 删除数量；不存在返回 0。
