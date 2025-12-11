# @mfycommon/gin v1.7.7 — Gin Skills 指南

## 标题/目的/适用场景
- 名称：`@mfycommon/gin`（Gin v1.7.7），用于构建高性能 HTTP API 服务。
- 推荐用在：中小型 API/网关/回调服务，需要路由分组、中间件链、快速参数绑定与 JSON 渲染的场景。
- 替代方案：超轻接口可用 `net/http` 直接实现；需要强约束/全栈生成可用 go-zero/kratos；复杂领域建模用 `@mfycommon/DDD`。
- 不适用：长连接/推送优先的场景（选 gRPC/WebSocket 专用框架）；极简脚本或一次性任务。

## 目录/索引（按需读取）
- `skills.md`：总体规范、关键步骤、输出与限制。
- `examples/router.md`：路由分组/版本化、参数绑定、REST 约定、静态/调试路由、调试路由访问控制、路由级中间件。
- `examples/middleware.md`：RequestID、日志、Recovery、CORS（含白名单加载）、令牌桶限流、Tracing/指标标签、指标命名+告警、JWT 鉴权/回调签名实现、安全响应头、统一错误。
- `examples/ops.md`：上下文超时、并发/重试、优雅关闭、请求体限制、debug 隔离、NoRoute/NoMethod。
- `examples/response.md`：成功/失败响应、文件下载、流式/SSE、重定向、Cookie、错误码映射。
- `examples/testing.md`：链路单测/基准、mock 下游、带中间件的集成测试。

## 所需输入（配置/参数/环境）
- 必备：`HTTP_PORT`（default `8080`）、`GIN_MODE`(`release`/`debug`/`test`，推荐 prod 用 `release`)、`LOG_LEVEL`(`info`)、`APP_ENV`。
- 超时：`read_timeout=5s`、`write_timeout=10s`、`idle_timeout=60s`；全局 `context.WithTimeout`（入口 2–3s）。
- 中间件：统一引入 `request_id`、日志、恢复、访问控制、限流；禁止裸用 `gin.Default()` 的 Logger 输出到 stdout（改写到 `pkg/log`）。
- 连接数/池：与下游依赖一致（DB/Redis）；HTTP Server `MaxHeaderBytes` 推荐 `1<<20`。
- 观测：Trace 端点 `TRACE_ENDPOINT`，Prometheus 采集端点（若有）；日志写入由 `pkg/log`。
- TLS（可选）：`TLS_CERT_FILE`、`TLS_KEY_FILE`；无则监听 HTTP。

## 流程/工作流程
1. **设置模式与引擎**：`gin.SetMode(gin.ReleaseMode)`，使用 `gin.New()` 创建 Engine，禁止直接用全局 `gin.Default()` 以便自定义日志/恢复。
2. **注入日志与 RequestID**：从 `pkg/log` 获取 `zap.Logger`，中间件中读取/生成 `request_id`（从 header `X-Request-ID`，无则生成），写回响应头。
3. **注册基础中间件**：`RecoveryWithWriter(log.Writer())` 捕获 panic；统一日志中间件记录 `method/path/status/latency/request_id`；可选接入 `otelgin.Middleware` 透传 `trace_id`。
4. **路由组织**：使用 `router := r.Group("/api")` 分组；公共中间件挂在 Group 上；版本化路由 `v1 := router.Group("/v1")`。
5. **参数绑定与校验**：使用 `ShouldBindJSON/Query/Uri` 与 `binding` tag；必要时加 `validate` tag（自定义校验器注册到 `binding.Validator`）。
6. **业务调用与超时控制**：在 handler 内使用 `ctx := c.Request.Context()`，如需下游调用加 `context.WithTimeout`；禁止长阻塞导致 goroutine 泄漏。
7. **统一响应**：成功 `c.JSON(http.StatusOK, data)`；错误通过自研封装 `resp.Error(c, err)` / `errs.Wrap` 输出统一 `code/message/request_id`。
8. **静态/文件上传（可选）**：静态资源用 `StaticFS`；文件上传使用 `FormFile` 并限制大小 `MaxMultipartMemory`（默认 32MiB）。
9. **启动与优雅关闭**：使用 `http.Server{Handler: r,...}`；`srv.ListenAndServe()` + `Shutdown(ctx)`（带 5–10s 超时）释放连接，退出前 `log.Flush()`。

## 何时使用该技能
- 需要快速交付 RESTful/JSON API。
- 需要路由分组、中间件链路、便捷的绑定与渲染。
- 需要接入日志/指标/Tracing 且团队已有 `pkg/log`、`pkg/errs`、`pkg/resp`。
- 需要与现有 HTTP 生态（net/http middleware、Prometheus、otel）兼容时。

## 输出格式
- 日志字段：`ts`、`level`、`request_id`、`trace_id`（可选）、`method`、`path`、`status`、`latency_ms`、`client_ip`；禁止记录请求体中的敏感字段。
- 错误返回：`code` + `message` + `request_id`，内部错误统一 `pkg/errs`；`context.DeadlineExceeded` 映射为超时码；`redis.Nil`/空资源映射 `NotFound`。
- 响应规范：JSON，成功 200，创建资源用 201；分页字段 `page/page_size/total`；下载/文件需设置正确 `Content-Type` 与 `Content-Disposition`。

## 示例（最小可运行 + 高频用例）
> 子目录索引（按需展开）：  
> - `examples/router.md`：路由/分组/版本、URI+Query 绑定、REST 约定、静态资源、pprof/Prometheus 挂载、路由级中间件。  
> - `examples/middleware.md`：RequestID、访问日志、Recovery、CORS、令牌桶限流、Tracing/指标、鉴权/安全头、统一错误。  
> - `examples/ops.md`：Handler 超时、并发 errgroup、幂等重试、优雅关闭、请求体大小限制、debug 路由隔离、自定义 404/405。  
> - `examples/response.md`：统一成功/失败响应、文件下载、流式/SSE、Redirect/Cookie、错误码映射。  
> - `examples/testing.md`：`gin.TestMode`、链路单测、带中间件的集成、mock 下游、基准测试。

1. **初始化与启动（优雅关闭）**
```go
func main() {
    log.Init(log.Config{Level: os.Getenv("LOG_LEVEL")})
    gin.SetMode(gin.ReleaseMode)
    r := gin.New()
    r.Use(requestIDMiddleware(), logMiddleware(), gin.RecoveryWithWriter(log.Writer()))

    v1 := r.Group("/api/v1")
    v1.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"pong": true}) })

    srv := &http.Server{Addr: ":" + getenv("HTTP_PORT", "8080"), Handler: r, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second}
    go func() { _ = srv.ListenAndServe() }()
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    _ = srv.Shutdown(ctx)
    log.Flush()
}
```
2. **请求 ID 中间件**
```go
func requestIDMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        rid := c.GetHeader("X-Request-ID")
        if rid == "" { rid = uuid.New().String() }
        c.Set("request_id", rid)
        c.Writer.Header().Set("X-Request-ID", rid)
        c.Next()
    }
}
```
3. **日志中间件（接入 pkg/log）**
```go
func logMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        cost := time.Since(start)
        log.FromContext(c.Request.Context()).
            With(log.String("request_id", c.GetString("request_id"))).
            Info("http request",
                log.String("method", c.Request.Method),
                log.String("path", c.FullPath()),
                log.Int("status", c.Writer.Status()),
                log.Duration("latency", cost),
                log.String("client_ip", c.ClientIP()),
            )
    }
}
```
4. **参数绑定与校验**
```go
type CreateReq struct {
    Name  string `json:"name" binding:"required,min=1,max=50"`
    Email string `json:"email" binding:"required,email"`
}

func (h *Handler) Create(c *gin.Context) {
    var req CreateReq
    if err := c.ShouldBindJSON(&req); err != nil {
        resp.Error(c, errs.Wrap(err, errs.CodeBadRequest, "invalid params"))
        return
    }
    ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
    defer cancel()
    if err := h.svc.Create(ctx, req); err != nil { resp.Error(c, err); return }
    resp.JSON(c, http.StatusCreated, gin.H{"ok": true})
}
```
5. **分组路由与中间件**
```go
r := gin.New()
api := r.Group("/api", requestIDMiddleware(), logMiddleware())
v1 := api.Group("/v1", authMiddleware())
v1.GET("/users/:id", h.GetUser)
v1.POST("/users", h.Create)
```
6. **文件上传（限制大小）**
```go
r.MaxMultipartMemory = 8 << 20 // 8 MiB
r.POST("/upload", func(c *gin.Context) {
    file, err := c.FormFile("file")
    if err != nil { resp.Error(c, errs.Wrap(err, errs.CodeBadRequest, "missing file")); return }
    dst := filepath.Join(os.TempDir(), file.Filename)
    _ = c.SaveUploadedFile(file, dst)
    c.JSON(http.StatusOK, gin.H{"saved": dst})
})
```
7. **接入 OpenTelemetry Trace**
```go
import "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

r := gin.New()
r.Use(otelgin.Middleware("my-service"), requestIDMiddleware(), logMiddleware(), gin.RecoveryWithWriter(log.Writer()))
```
8. **自定义错误处理中间件（可选）**
```go
func errMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        if len(c.Errors) > 0 {
            resp.Error(c, errs.Wrap(c.Errors.Last(), errs.CodeInternal, "gin error"))
        }
    }
}
```

## 限制条件与安全规则
- 禁止在生产使用默认控制台 Logger；日志必须走 `pkg/log` 并脱敏敏感字段（手机号、身份证、Token）。
- 性能/资源：单实例 QPS 超 5k 前需压测；`MaxMultipartMemory` 合理设置避免内存暴涨；避免在中间件做重 CPU/IO。
- 重试/超时：Handler 内必须设置 context 超时；对外部依赖的重试上限 3 次，总时长 < 10s；禁止在非幂等操作上盲目重试。
- API 使用：不要在 handler 内启动无限 goroutine；不要直接操作全局默认路由器；跨域需显式 CORS 中间件。
- 安全：默认拒绝未认证的写接口；开启 HTTPS 时校验证书；禁止反射路由导致的 Path Traversal。

## 常见坑/FAQ（按严重度）
- 高：遗漏 `Shutdown` 导致连接未释放—使用 `http.Server` + `Shutdown`。
- 高：未设置超时，业务阻塞导致 goroutine 堆积—入口统一 `context.WithTimeout`。
- 中：使用 `c.FullPath()` 为空（未命名路由）导致指标打点缺 label—路由需命名或用 `c.Request.URL.Path`。
- 中：`ShouldBind` 默认允许未知字段—使用 `binding.Validator` 配置或自定义校验拒绝未知字段。
- 低：请求体过大导致 OOM—设置 `MaxMultipartMemory`/`http.MaxBytesReader`。
- 低：忘记设置 `X-Request-ID` 回传，排障困难—在中间件写回响应头。

## 可观测性/诊断
- Metrics：为路由/方法记录 `qps/latency/error_rate`；暴露 Prometheus 端点时过滤掉健康检查；池/队列指标与下游一致。
- Tracing：`otelgin.Middleware` 自动创建 span，包含 `http.method`、`http.route`、`http.status_code`；透传 `trace_id` 到下游。
- 日志：慢请求 >300ms 打 warn，字段含 `request_id`、`route`、`status`、`duration_ms`、`client_ip`；panic 由 `RecoveryWithWriter` 记录。
- Debug：开启 `GIN_MODE=debug` 仅用于本地；线上禁止。

## 版本与依赖
- 版本：Gin v1.7.7；Go 1.18+（建议 1.20/1.21）。
- 依赖：`net/http` 标准库；可选 `go.opentelemetry.io/contrib/.../otelgin`；团队封装 `pkg/log`、`pkg/errs`、`pkg/resp`。
- 内部封装路径：`@mfycommon/gin`（本文档），结合服务目录 `cmd/<svc>` + `internal/handler`/`router`。

## 更新记录 / Owner
- 最后更新时间：2024-05-xx（首版）。
- Owner：Gin 组件维护人（@架构/中间件负责人），评审人：服务线负责人；变更需双人 Review。
