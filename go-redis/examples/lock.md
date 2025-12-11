# 分布式锁（SET NX + 安全释放）

> 适用于轻量互斥（非强一致），必须设置过期时间并在释放时校验锁值，避免误删他人锁。

```go
var unlockScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end`)

func AcquireLock(ctx context.Context, rdb *redis.Client, key, value string, ttl time.Duration) (bool, error) {
    ok, err := rdb.SetNX(ctx, key, value, ttl).Result()
    return ok, err
}

func ReleaseLock(ctx context.Context, rdb *redis.Client, key, value string) error {
    _, err := unlockScript.Run(ctx, rdb, []string{key}, value).Result()
    return err
}
```

用法：
```go
id := uuid.NewString()
ok, err := AcquireLock(ctx, rdb, "lock:order:123", id, 5*time.Second)
if err != nil || !ok {
    return errors.New("lock busy")
}
defer ReleaseLock(ctx, rdb, "lock:order:123", id)
// 执行业务
```

提示：
- 过期时间必须小于业务最长执行时间，或实现续约逻辑。
- 不满足强一致锁需求时，应使用 Redlock 或专用锁服务。*** End Patch तैयार need ensure grammar with End Patch newline. We'll reapply patch properly.`
