# Gin v1.7.7 响应与错误包装示例

## 统一成功/失败响应（搭配 pkg/resp + pkg/errs）
```go
// 成功
resp.JSON(c, http.StatusOK, gin.H{"data": dto, "request_id": c.GetString("request_id")})
// 失败
if err != nil { resp.Error(c, errs.Wrap(err, errs.CodeInternal, "do something failed")) }
```

## 手写 JSON（无封装时）
```go
c.JSON(http.StatusOK, gin.H{
    "code":    0,
    "message": "ok",
    "data":    payload,
    "request_id": c.GetString("request_id"),
})
```

## 文件下载
```go
func download(c *gin.Context) {
    path := filepath.Join("/data/files", c.Param("name"))
    c.Header("Content-Type", "application/octet-stream")
    c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(path)))
    c.File(path)
}
```

## 流式响应（逐步写入）
```go
c.Stream(func(w io.Writer) bool {
    if chunk, ok := next(); ok {
        _, _ = w.Write(chunk)
        return true
    }
    return false
})
```

## Server-Sent Events (SSE)
```go
c.Stream(func(w io.Writer) bool {
    c.SSEvent("message", gin.H{"ts": time.Now().Unix(), "msg": "tick"})
    time.Sleep(time.Second)
    return true
})
```

## Redirect
```go
c.Redirect(http.StatusFound, "https://example.com/login")
```

## 设置 Cookie
```go
c.SetCookie("session_id", sid, 3600, "/", ".example.com", true, true) // secure + httponly
```

## 自定义错误码映射
```go
func errorMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        if len(c.Errors) == 0 { return }
        err := c.Errors.Last()
        code := errs.CodeInternal
        if errors.Is(err, context.DeadlineExceeded) { code = errs.CodeTimeout }
        resp.Error(c, errs.Wrap(err, code, "gin error"))
    }
}
```

## 404 / 405
```go
r.NoRoute(func(c *gin.Context) { resp.Error(c, errs.Wrap(errs.ErrNotFound, errs.CodeNotFound, "route not found")) })
r.NoMethod(func(c *gin.Context) { resp.Error(c, errs.Wrap(errs.ErrMethodNotAllowed, errs.CodeMethodNotAllowed, "method not allowed")) })
```
