# @mfycommon/MVC v1.0 — MVC Skills 指南

## 标题/目的/适用场景
- 名称：`@mfycommon/MVC` v1.0，面向中小型 API/Web 服务的分层组织与快速交付。
- 推荐用在：CRUD 为主、页面/接口渲染分工清晰、需要快速落地且团队已有 MVC 习惯的服务。
- 替代方案：复杂领域演进用 `@mfycommon/DDD`；仅数据搬运/批处理用作业脚本或 ETL 工具。
- 不适用：跨上下文强一致、事件驱动主导的核心域；高复杂规则需要聚合不变量的场景。

## 所需输入
- 环境变量：`APP_ENV`(default: `dev`)、`LOG_LEVEL`(`info`)、`HTTP_PORT`(`8080`)。
- 基础设施：`DB_DSN`、`CACHE_ADDR`、`TRACE_ENDPOINT`，缺失对应依赖应降级或拒用。
- 池/超时推荐：DB 连接池 `max_open=80`、`max_idle=20`；HTTP server `read_timeout=5s`、`write_timeout=10s`；默认 `context.WithTimeout` 2s。
- 日志：从 `pkg/log` 获取全局 logger，禁止直接 new；请求需带 `request_id`。

## 流程/工作流程
1. **路由定义**：在 `router` 注册 HTTP 路由，指向 controller 方法；统一中间件（鉴权/trace/log）放路由层。
2. **Controller 层**：负责参数校验、绑定 DTO、调用 service；禁止直接访问 DAO。
3. **Service 层**：封装业务逻辑，调用 repository/cache/external API；聚合事务在 service 层。
4. **Repository/DAO 层**：`infra/dao` 或 `repo` 封装 SQL/缓存调用；使用 `pkg/db`、`pkg/cache`。
5. **视图/响应**：REST 返回 JSON DTO，使用 `pkg/resp` 统一编码；错误经 `pkg/errs` 转换。
6. **日志链路**：`log.FromContext(ctx)` 获取 logger，必须包含 `request_id`、`route` 字段；进程退出调用 `log.Flush()`。
7. **收尾**：在 `cmd/<svc>/main.go` 的退出钩子里依次关闭 HTTP server、DB、cache、flush log。

### 关键指令示例
```go
logger := log.FromContext(ctx).With(log.String("request_id", rid))
user, err := svc.GetUser(ctx, req.ID)
if err != nil { return resp.Error(w, errs.Wrap(err, errs.CodeBiz, "get user")) }
return resp.JSON(w, userDTO.From(user))
```

## 何时使用该技能
- 需要快速上线的 CRUD/查询型接口。
- 业务复杂度低到中等，团队已有 MVC 约定。
- 前后端分离但接口层仍需轻量逻辑的场景。

## 输出格式
- 日志字段：`ts`、`level`、`request_id`、`route`、`status`；敏感字段脱敏。
- 错误：统一 `pkg/errs`，外露 `code`/`message`；Redis `nil` 返回映射为 `errs.NotFound`。
- 响应：`resp.JSON(w, data)`，错误使用 `resp.Error(w, err)`；分页字段统一 `page`、`page_size`、`total`。

## 示例（最小可运行 + 常用包装）
1. 初始化服务
```go
func main() {
    ctx := context.Background()
    log.Init(log.Config{Level: os.Getenv("LOG_LEVEL")})
    db := db.New(os.Getenv("DB_DSN"))
    cache := cache.New(os.Getenv("CACHE_ADDR"))
    r := router.New()
    userRepo := repo.NewUser(db, cache)
    userSvc := service.NewUser(userRepo)
    controller.BindUser(r, userSvc)
    http.Run(ctx, os.Getenv("HTTP_PORT"), r)
    defer func() { log.Flush(); cache.Close(); db.Close() }()
}
```
2. Controller 处理请求
```go
func (c *UserController) Get(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    id := mux.Vars(r)["id"]
    user, err := c.svc.Get(ctx, id)
    if err != nil { resp.Error(w, errs.Wrap(err, errs.CodeBiz, "get user")); return }
    resp.JSON(w, user)
}
```
3. Service 调用 Repository
```go
func (s *UserService) Get(ctx context.Context, id string) (*User, error) {
    ctx, cancel := context.WithTimeout(ctx, 2*time.Second); defer cancel()
    return s.repo.FindByID(ctx, id)
}
```
4. Repository 访问 DB/Cache
```go
func (r *UserRepo) FindByID(ctx context.Context, id string) (*User, error) {
    if user, err := r.cache.Get(ctx, id); err == nil { return user, nil }
    var u User
    if err := r.db.GetContext(ctx, &u, "SELECT * FROM users WHERE id=?", id); err != nil { return nil, err }
    _ = r.cache.Set(ctx, id, &u, time.Minute)
    return &u, nil
}
```
5. 错误与重试
```go
err := retry.Do(ctx, 2, func() error { return r.db.ExecContext(ctx, sql) })
if errors.Is(err, errs.Transient) { metrics.Inc("db.write.retry_exhausted") }
```
6. 并发/连接管理
```go
g, ctx := errgroup.WithContext(ctx)
g.Go(func() error { return svc.A(ctx, reqA) })
g.Go(func() error { return svc.B(ctx, reqB) })
if err := g.Wait(); err != nil { return err }
```
7. 收尾清理
```go
defer func() { cache.Close(); db.Close(); log.Flush() }()
```

## 限制条件与安全规则
- Controller 禁止直接访问 DB/缓存；必须经 service/repo。
- 禁止在 repo 返回 ORM 原始模型给 controller，统一 DTO。
- 性能红线：单实例 QPS 超 2k 需压测；DB pool 耗尽需限流/拒绝；请求超时默认 2s。
- 安全：日志/响应中敏感信息脱敏，禁止输出秘钥/Token。
- 超时/重试：读 1s、写 3s；重试不超过 3 次且总时长 < 10s；幂等操作需幂等键。
- 不可用 API：禁止直接 new logger/DB client，统一使用 `pkg/log`、`pkg/db`。

## 常见坑/FAQ（按严重度）
- 高：Controller 做业务判断过多导致耦合—将规则下沉到 service。
- 高：未统一错误码导致前端解析混乱—使用 `pkg/errs` + `resp.Error`。
- 中：缓存穿透/击穿—空值写缓存+随机过期；热点加互斥锁。
- 中：路由分散—集中在 `router` 层，避免多处散落。
- 低：缺少请求超时—service 层统一加 `context.WithTimeout`。

## 可观测性/诊断
- Metrics：路由级 QPS/latency/error_rate；DB/缓存操作 `latency`、`hit_rate`；重试耗尽计数。
- Tracing：请求入口创建 span，注入 `trace_id`；DB/缓存调用打上 `db.op`、`cache.op` 标签。
- 慢日志：接口超过 300ms 打 warn，包含 `request_id`、`route`、`status`、`duration_ms`。

## 版本与依赖
- 语言/框架：Go 1.21+；日志 `pkg/log`；错误 `pkg/errs`；HTTP `router`/`http` 封装；DB `pkg/db`；缓存 `pkg/cache`。
- 外部服务：MySQL 5.7+/8.0+、Redis 6+。
- 目录约定：`cmd/<svc>`(入口)、`router`、`controller`、`service`、`repo`/`infra/dao`、`pkg/*`(基础设施)。

## 更新记录 / Owner
- 最后更新时间：2024-05-xx（首次落地）。
- Owner：MVC 规范小组（@架构负责人），评审人：服务域负责人；变更需双人 Review。
