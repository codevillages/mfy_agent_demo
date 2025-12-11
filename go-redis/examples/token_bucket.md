# 令牌桶限流（Lua 原子校验）

> 在 Redis 中存储剩余令牌数，使用 Lua 保证发放与过期原子性。

```go
var tokenBucket = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local refill_interval = tonumber(ARGV[2]) -- 毫秒
local refill_tokens = tonumber(ARGV[3])
local capacity = tonumber(ARGV[4])
local requested = tonumber(ARGV[5])

local bucket = redis.call("HMGET", key, "tokens", "ts")
local tokens = tonumber(bucket[1])
local ts = tonumber(bucket[2])

if tokens == nil then
    tokens = capacity
    ts = now
end

local delta = math.max(0, now - ts)
local add = math.floor(delta / refill_interval) * refill_tokens
tokens = math.min(capacity, tokens + add)
ts = ts + math.floor(delta / refill_interval) * refill_interval

if tokens < requested then
    redis.call("HMSET", key, "tokens", tokens, "ts", ts)
    redis.call("PEXPIRE", key, refill_interval * 2)
    return 0
end

tokens = tokens - requested
redis.call("HMSET", key, "tokens", tokens, "ts", now)
redis.call("PEXPIRE", key, refill_interval * 2)
return 1
`)

func Allow(ctx context.Context, rdb *redis.Client, key string, capacity, refillTokens int64, refillInterval time.Duration, n int64) (bool, error) {
    now := time.Now().UnixMilli()
    res, err := tokenBucket.Run(ctx, rdb, []string{key},
        now,
        refillInterval.Milliseconds(),
        refillTokens,
        capacity,
        n,
    ).Int()
    if err != nil {
        return false, err
    }
    return res == 1, nil
}
```

用法：
```go
ok, err := Allow(ctx, rdb, "bucket:api:v1", 100, 10, time.Second, 1)
if err != nil {
    // 上报错误
}
if !ok {
    return errors.New("rate limited")
}
// 继续处理请求
```

提示：
- key 建议包含业务名和调用方标识，避免冲突。
- 根据延迟与吞吐调整容量与补充速率；高可用限流推荐配合本地缓存/熔断。*** End Patch ***!
