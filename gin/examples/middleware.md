# Gin v1.7.7 中间件配置示例

## 基础链路：RequestID + 日志 + Panic Recovery
```go
r := gin.New()
r.Use(
    requestIDMiddleware(),        // 生成/透传 X-Request-ID
    logMiddleware(),              // 结构化访问日志，输出到 pkg/log
    gin.RecoveryWithWriter(log.Writer()), // 捕获 panic，写入日志
)
```

### RequestID
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

### 访问日志（避免默认 Logger）
```go
func logMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        cost := time.Since(start)
        logger := log.FromContext(c.Request.Context()).
            With(log.String("request_id", c.GetString("request_id")))
        logger.Info("http request",
            log.String("method", c.Request.Method),
            log.String("path", c.FullPath()),
            log.Int("status", c.Writer.Status()),
            log.Duration("latency", cost),
            log.String("client_ip", c.ClientIP()),
        )
    }
}
```

## CORS
```go
func corsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*") // 生产建议配置域名白名单
        c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Request-ID")
        c.Header("Access-Control-Expose-Headers", "X-Request-ID")
        if c.Request.Method == http.MethodOptions { c.Status(http.StatusNoContent); return }
        c.Next()
    }
}
```

## 限流（令牌桶示例）
```go
func ratelimitMiddleware(limit int, interval time.Duration) gin.HandlerFunc {
    limiter := rate.NewLimiter(rate.Every(interval), limit)
    return func(c *gin.Context) {
        if !limiter.Allow() {
            resp.Error(c, errs.Wrap(errs.ErrTooManyRequests, errs.CodeTooManyRequests, "rate limited"))
            c.Abort()
            return
        }
        c.Next()
    }
}
```
> 依赖 `golang.org/x/time/rate`。

## 指标与 Tracing
```go
import (
    "github.com/gin-gonic/gin"
    "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

r := gin.New()
r.Use(
    requestIDMiddleware(),
    otelgin.Middleware("my-service"), // 自动埋点 http.* 标签
    metricsMiddleware(),              // 统一 QPS/latency/status 指标
    logMiddleware(),
    gin.RecoveryWithWriter(log.Writer()),
)
```
`metricsMiddleware` 可按业务封装，记录 `method/path/status/duration_ms`；暴露 `/debug/metrics` 供 Prometheus 抓取。

### 指标中间件示例（Prometheus）
```go
var (
    httpLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "http_server_latency_ms", Buckets: []float64{5, 10, 25, 50, 100, 250, 500, 1000}},
        []string{"route", "method", "status"},
    )
    httpQPS = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "http_server_requests_total"},
        []string{"route", "method", "status"},
    )
)

func metricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        route := c.FullPath()
        if route == "" { route = c.Request.URL.Path }
        status := strconv.Itoa(c.Writer.Status())
        httpQPS.WithLabelValues(route, c.Request.Method, status).Inc()
        httpLatency.WithLabelValues(route, c.Request.Method, status).Observe(float64(time.Since(start).Milliseconds()))
    }
}
```

> 依赖 `github.com/prometheus/client_golang/prometheus`。

### Tracing 补充标签
```go
func traceAttrsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        span := trace.SpanFromContext(c.Request.Context())
        if span != nil && span.SpanContext().IsValid() {
            span.SetAttributes(
                attribute.String("http.client_ip", c.ClientIP()),
                attribute.String("http.route", c.FullPath()),
            )
        }
        c.Next()
    }
}
```
> 依赖 `go.opentelemetry.io/otel/trace` 与 `go.opentelemetry.io/otel/attribute`。

## 安全相关中间件
- **鉴权/鉴别**：在版本组或资源组挂载 `authMiddleware()`，校验 Token/签名，失败返回 401/403。
- **安全头**：设置 `X-Content-Type-Options: nosniff`、`X-Frame-Options: DENY`、`Content-Security-Policy` 等。
- **请求体大小**：`http.MaxBytesReader` 限制最大 body。

## 统一错误处理中间件
```go
func errorMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        if len(c.Errors) == 0 { return }
        resp.Error(c, errs.Wrap(c.Errors.Last(), errs.CodeInternal, "gin error"))
    }
}
```

## CORS 白名单（按 Host 过滤）
```go
func corsWhitelistMiddleware(allowed map[string]struct{}) gin.HandlerFunc {
    return func(c *gin.Context) {
        origin := c.GetHeader("Origin")
        if origin != "" {
            if _, ok := allowed[origin]; ok {
                c.Header("Access-Control-Allow-Origin", origin)
                c.Header("Vary", "Origin")
            }
        }
        c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
        c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Request-ID")
        c.Header("Access-Control-Expose-Headers", "X-Request-ID")
        if c.Request.Method == http.MethodOptions { c.Status(http.StatusNoContent); return }
        c.Next()
    }
}
```

### 白名单加载示例
```go
func loadCORSWhitelist() map[string]struct{} {
    origins := strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") // 配置中心/ENV
    m := make(map[string]struct{}, len(origins))
    for _, o := range origins {
        o = strings.TrimSpace(o)
        if o != "" { m[o] = struct{}{} }
    }
    return m
}
```
> 可用 `atomic.Value` 持有 map，实现动态刷新；更新后重新创建中间件或读最新 map。

## 鉴权/签名示例
### Bearer/JWT
```go
func authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
        if token == "" { resp.Error(c, errs.Wrap(errs.ErrUnauthorized, errs.CodeUnauthorized, "missing token")); c.Abort(); return }
        claims, err := verifyJWT(token) // 见下方实现
        if err != nil { resp.Error(c, errs.Wrap(errs.ErrUnauthorized, errs.CodeUnauthorized, "invalid token")); c.Abort(); return }
        c.Set("user_id", claims.UserID)
        c.Next()
    }
}
```

// JWT 校验（HS256 示例）
type Claims struct {
    UserID string `json:"uid"`
    jwt.RegisteredClaims
}

func verifyJWT(tokenStr string) (*Claims, error) {
    parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
    token, err := parser.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(os.Getenv("JWT_SECRET")), nil
    })
    if err != nil { return nil, err }
    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid { return nil, errs.ErrUnauthorized }
    // 容忍 60s 时钟偏移
    if claims.VerifyExpiresAt(time.Now(), true) == false { return nil, errs.ErrUnauthorized }
    if claims.VerifyIssuedAt(time.Now(), true) == false { return nil, errs.ErrUnauthorized }
    return claims, nil
}
> 依赖 `github.com/golang-jwt/jwt/v4`，`JWT_SECRET` 从密钥管理注入。

### 回调签名（HMAC）
```go
func signatureMiddleware(secret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        body, _ := io.ReadAll(c.Request.Body)
        c.Request.Body = io.NopCloser(bytes.NewBuffer(body)) // 复用 body
        sig := c.GetHeader("X-Signature")
        expected := calcHMAC(body, secret)
        if !hmac.Equal([]byte(sig), []byte(expected)) {
            resp.Error(c, errs.Wrap(errs.ErrForbidden, errs.CodeForbidden, "bad signature"))
            c.Abort()
            return
        }
        c.Next()
    }
}
```

## 安全响应头
```go
func securityHeadersMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("Referrer-Policy", "no-referrer")
        c.Header("Content-Security-Policy", "default-src 'self'")
        c.Next()
    }
}
```

## 中间件挂载建议
- 全局：RequestID、访问日志、Recovery、Tracing、基础指标。
- 读接口组：可轻量限流、缓存控制。
- 写接口组：鉴权、严格限流/熔断、审计日志。
- 调试路由：拆到 `/debug/*`，避免和线上主流量混用。

## 指标命名与告警建议
- 命名示例：`http_server_requests_total{route,method,status}`、`http_server_latency_ms_bucket{route,method,status,le}`。
- 告警参考：p99 > 300ms 连续 5m；5xx 比例 > 1% 连续 5m；拒绝/限流计数暴增（`http_server_rejected_total`）；`http_server_timeout_total` 持续升高。
- 标签控制：`route` 用 `FullPath`，未知路由回退 `URL.Path`；避免高基数参数落入标签。
