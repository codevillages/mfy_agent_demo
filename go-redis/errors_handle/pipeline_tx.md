# Pipeline / 事务错误处理

- Pipeline：`Pipelined`/`TxPipelined` 返回的 `err` 汇总命令执行错误；内部每个 `Cmd.Err()` 也需检查（例如部分命令 `redis.Nil`）。
- Watch 事务：`Watch` 函数返回的错误包含 CAS 冲突（`redis.TxFailedErr`），应视为可重试的业务冲突，而非系统故障。
- Lua 脚本：`Run` 返回的错误可能是脚本自定义错误（`ERR ...`），属业务错误，不应重试；若为网络类则按网络错误策略。
- Key 未命中：事务/流水线里的 `Get` 未命中仍返回 `redis.Nil`，需区分未命中与真正错误。
