# 单元测试中的 Mock（使用 miniredis）

> 通过 `github.com/alicebob/miniredis/v2` 启动内存 Redis，避免依赖真实实例；兼容 go-redis v9。

```go
import (
    "context"
    "testing"
    "time"

    "github.com/alicebob/miniredis/v2"
    "github.com/redis/go-redis/v9"
)

func TestWithMockRedis(t *testing.T) {
    mr, err := miniredis.Run()
    if err != nil {
        t.Fatal(err)
    }
    defer mr.Close()

    rdb := redis.NewClient(&redis.Options{
        Addr: mr.Addr(),
        DB:   0,
    })

    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    if err := rdb.Set(ctx, "k1", "v1", 0).Err(); err != nil {
        t.Fatal(err)
    }
    got, err := rdb.Get(ctx, "k1").Result()
    if err != nil {
        t.Fatal(err)
    }
    if got != "v1" {
        t.Fatalf("want v1, got %s", got)
    }
}
```

提示：
- miniredis 支持基本命令与部分过期语义，但不覆盖集群/订阅等高级功能。
- 对 Pipeline/事务的测试，可复用同一实例；需要订阅/流式特性的测试应使用真实 Redis 并打标签区分。*** End Patch�__(/*! End Patch line missing?) Wait we must follow grammar. Need "*** End Patch". ensure newline. Let's craft again carefully with proper patch.♪
