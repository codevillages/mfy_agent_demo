# 事务 CAS

```go
err := rdb.Watch(ctx, func(tx *redis.Tx) error {
    v, err := tx.Get(ctx, "stock").Int()
    if err != nil && err != redis.Nil {
        return err
    }
    if v <= 0 {
        return errors.New("sold out")
    }
    _, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
        pipe.Decr(ctx, "stock")
        return nil
    })
    return err
}, "stock")
if err != nil {
    // handle
}
```
