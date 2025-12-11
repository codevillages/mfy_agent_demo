# 中间件 / 拦截器示例

```go
// RequestID：透传/生成 request_id
func RequestID() rest.Middleware {
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            rid := r.Header.Get("X-Request-Id")
            if rid == "" { rid = xid.New().String() }
            ctx := logx.ContextWithFields(r.Context(), logx.Field("reqId", rid))
            r = r.WithContext(ctx)
            w.Header().Set("X-Request-Id", rid)
            next(w, r)
        }
    }
}

// Prometheus：记录接口耗时
func Metrics() rest.Middleware {
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            start := timex.Now()
            next(w, r)
            prometheus.Observe(r.Method, r.URL.Path, timex.Since(start))
        }
    }
}

// RPC 拦截器：统一日志/错误
func UnaryInterceptor(
    ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler) (resp interface{}, err error) {
    start := timex.Now()
    resp, err = handler(ctx, req)
    logx.WithContext(ctx).Infow("rpc",
        logx.Field("method", info.FullMethod),
        logx.Field("duration_ms", timex.Since(start).Milliseconds()),
        logx.Field("err", err),
    )
    return resp, err
}
```

推荐开启：RequestID、Recover、Prometheus、Tracing、Auth/JWT。***
