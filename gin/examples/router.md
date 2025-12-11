# Gin v1.7.7 路由/分组/静态资源示例

## 基础路由与健康检查
```go
r := gin.New()
r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
r.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"pong": true}) })
```

## 分组 + 版本化 + 公共中间件
```go
r := gin.New()
api := r.Group("/api", requestIDMiddleware(), logMiddleware())
v1 := api.Group("/v1", authMiddleware()) // 公共鉴权
{
    v1.GET("/users/:id", h.GetUser)
    v1.POST("/users", h.CreateUser)
}
v2 := api.Group("/v2") // 灰度或新版本
v2.GET("/users/:id", h.GetUserV2)
```

## 路径/查询/表单参数绑定
```go
type QueryList struct {
    Page     int    `form:"page,default=1" binding:"min=1"`
    PageSize int    `form:"page_size,default=20" binding:"min=1,max=100"`
    Keyword  string `form:"keyword"`
}
type UserURI struct {
    ID string `uri:"id" binding:"required,uuid4"`
}

func (h *Handler) List(c *gin.Context) {
    var q QueryList
    if err := c.ShouldBindQuery(&q); err != nil { resp.Error(c, errs.Wrap(err, errs.CodeBadRequest, "bad query")); return }
    resp.JSON(c, http.StatusOK, h.svc.List(c.Request.Context(), q))
}
func (h *Handler) Get(c *gin.Context) {
    var path UserURI
    if err := c.ShouldBindUri(&path); err != nil { resp.Error(c, errs.Wrap(err, errs.CodeBadRequest, "bad uri")); return }
    resp.JSON(c, http.StatusOK, h.svc.Get(c.Request.Context(), path.ID))
}
```

## REST 资源路由约定（示例）
```go
v1 := r.Group("/api/v1")
v1.GET("/orders", h.ListOrders)          // List
v1.POST("/orders", h.CreateOrder)        // Create
v1.GET("/orders/:id", h.GetOrder)        // Get
v1.PUT("/orders/:id", h.ReplaceOrder)    // Replace
v1.PATCH("/orders/:id", h.UpdateOrder)   // Partial update
v1.DELETE("/orders/:id", h.DeleteOrder)  // Delete
```

## 路由命名与 FullPath
- 优先使用 `c.FullPath()` 记录日志/指标，确保路由定义时使用参数化路径而非动态字符串。
- 对于动态注册的路由，手动设置 `route := c.FullPath(); if route == "" { route = c.Request.URL.Path }` 避免空标签。

## 静态资源与文件目录
```go
// /static/app.js => ./web/static/app.js
r.Static("/static", "./web/static")
// 用文件系统接口（例如 embed.FS）
r.StaticFS("/assets", http.FS(embedFS))
// 单文件
r.StaticFile("/favicon.ico", "./web/static/favicon.ico")
```

## 子路由挂载第三方 Handler（net/http 兼容）
```go
// 将 pprof 或 Prometheus 挂载到 /debug
debug := r.Group("/debug")
debug.Any("/metrics", gin.WrapH(promhttp.Handler()))
debug.Any("/pprof/*any", gin.WrapH(http.DefaultServeMux))
```

### /debug 访问控制（IP 白名单 + BasicAuth）
```go
allowedIPs := map[string]struct{}{"127.0.0.1": {}, "::1": {}}
debug := r.Group("/debug", func(c *gin.Context) {
    if _, ok := allowedIPs[c.ClientIP()]; !ok {
        c.Status(http.StatusForbidden); c.Abort(); return
    }
}, gin.BasicAuth(gin.Accounts{"admin": os.Getenv("DEBUG_PASSWORD")}))
debug.Any("/metrics", gin.WrapH(promhttp.Handler()))
debug.Any("/pprof/*any", gin.WrapH(http.DefaultServeMux))
```

## 路由级中间件差异化
```go
// 针对写接口追加限流
write := v1.Group("", ratelimitMiddleware(100, time.Second))
write.POST("/orders", h.CreateOrder)
write.PATCH("/orders/:id", h.UpdateOrder)
```

## CORS / OPTIONS 预检
```go
r.Use(corsMiddleware())
r.OPTIONS("/*path", func(c *gin.Context) { c.Status(http.StatusNoContent) })
```
