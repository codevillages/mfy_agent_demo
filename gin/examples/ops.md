# Gin v1.7.7 运维/超时/关闭示例

## 全局超时与下游调用
```go
func (h *Handler) GetOrder(c *gin.Context) {
    ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
    defer cancel()
    order, err := h.svc.Get(ctx, c.Param("id"))
    if err != nil { resp.Error(c, err); return }
    resp.JSON(c, http.StatusOK, order)
}
```

## 并发调用 + 错误聚合
```go
g, ctx := errgroup.WithContext(c.Request.Context())
var a, b Result
g.Go(func() error { var err error; a, err = h.svc.A(ctx); return err })
g.Go(func() error { var err error; b, err = h.svc.B(ctx); return err })
if err := g.Wait(); err != nil { resp.Error(c, err); return }
resp.JSON(c, http.StatusOK, gin.H{"a": a, "b": b})
```

## 可控重试（幂等操作）
```go
err := retry.Do(ctx, 3, func() error {
    return h.repo.Save(ctx, obj)
}, retry.WithBackoff(time.Millisecond*50, time.Millisecond*200))
if err != nil { resp.Error(c, errs.Wrap(err, errs.CodeInternal, "save failed")) }
```

## 优雅关闭（HTTP + Background）
```go
srv := &http.Server{
    Addr:         ":" + getenv("HTTP_PORT", "8080"),
    Handler:      r,
    ReadTimeout:  5 * time.Second,
    WriteTimeout: 10 * time.Second,
    IdleTimeout:  60 * time.Second,
}
go func() {
    if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        log.L().Error("server start failed", log.Error(err))
    }
}()
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
_ = srv.Shutdown(ctx) // 等待连接完成
log.Flush()
```

## 请求体大小/上传限制
```go
// 全局限制
r.Use(func(c *gin.Context) {
    c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20) // 5MB
    c.Next()
})
r.MaxMultipartMemory = 8 << 20
```

## Debug/内省路由隔离
```go
debug := r.Group("/debug")
debug.Use(authMiddleware()) // 生产限制访问
debug.GET("/metrics", gin.WrapH(promhttp.Handler()))
debug.GET("/pprof/*any", gin.WrapH(http.DefaultServeMux))
```

## 自定义 404 / 405 响应
```go
r.NoRoute(func(c *gin.Context) {
    resp.Error(c, errs.Wrap(errs.ErrNotFound, errs.CodeNotFound, "route not found"))
})
r.NoMethod(func(c *gin.Context) {
    resp.Error(c, errs.Wrap(errs.ErrMethodNotAllowed, errs.CodeMethodNotAllowed, "method not allowed"))
})
```
