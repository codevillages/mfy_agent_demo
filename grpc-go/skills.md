# @mfycommon/grpc-go v1.71.1 — Skills

## 标题 / 目的 / 适用场景
- 库名+版本：`@mfycommon/grpc-go` v1.71.1，用于 gRPC 服务端/客户端（Go）开发、治理、可观测与安全封装。
- 推荐用在：内部微服务 RPC、需要双向流/流式大数据传输、需要内置拦截器链（超时、重试、限流、trace、metrics）。
- 替代方案：HTTP/REST 用 go-zero/REST；轻量内部调用可用简单 HTTP；极端性能自研协议。
- 不适用：强实时硬件协议、浏览器直连、需要 GraphQL 等非 gRPC 场景。

## 所需输入
- proto：定义 service/message，需开启 `go_package`。
- 监听与客户端配置：`ListenAddr`（server）、`Target`/`Endpoints`（client）；可使用服务发现（如 etcd/dns）。
- TLS/鉴权：服务端/客户端证书路径，或禁用（内网可明文但不推荐）；JWT/自定义 token。
- 超时：服务端 `context.WithTimeout` 包裹业务；客户端 `grpc.Dial` 时设置 `DefaultCallOptions`/`WithTimeout`（每次 call）。
- 重试：客户端拦截器配置（幂等请求）重试次数/退避；服务端避免长时阻塞。
- 并发/连接：客户端 `MaxConn`、`MinConn`（如使用连接池封装），keepalive 参数。
- 日志：`pkg/log` 的 zap logger；拦截器注入 `request_id`/`trace_id`。
- 环境变量：`GRPC_LISTEN_ADDR`、`GRPC_TARGET`、`GRPC_TLS_CERT/KEY/CA`、`GRPC_TIMEOUT_MS`。

## 流程 / 工作流程
1) 编写 proto：定义 service、消息、错误码枚举；`option go_package = "path/to/pb;pb"`。
2) 生成代码：`protoc --go_out=. --go-grpc_out=. *.proto`（锁定 v1.71.1 生成的接口）。
3) 服务端：
   - 从配置加载 ListenAddr/TLS/超时/日志。
   - 构建拦截器链：Recover -> Logging -> Metrics -> Tracing -> Auth -> Timeout/Validate。
   - `grpc.NewServer(opts...)`，注册服务实现。
   - 健康检查：注册 `grpc/health` 服务；暴露状态。
   - 优雅关闭：监听信号，`GracefulStop()`，关闭日志。
4) 客户端：
   - `grpc.DialContext(ctx, target, opts...)`，含：TLS creds 或 insecure；拦截器（retry/metrics/logging/tracing/auth）。
   - 每次调用传入带超时/trace 的 ctx。
   - 重试仅用于幂等请求，次数≤3，指数退避。
5) 日志与可观测：
   - 拦截器记录 `method`, `peer`, `code`, `duration_ms`, `req_id`, `trace_id`。
   - Metrics：QPS/latency/error_rate 按 method/status 维度。
6) 收尾：服务端 `GracefulStop`；客户端 `Close()`；日志 flush。

### 服务端示例（精简）
```go
lis, _ := net.Listen("tcp", os.Getenv("GRPC_LISTEN_ADDR"))
serverOpts := []grpc.ServerOption{
    grpc.ChainUnaryInterceptor(
        interceptors.Recover(),
        interceptors.Logging(pkglog.L()),
        interceptors.Metrics(),
        interceptors.Tracing(),
        interceptors.Auth(jwtValidator),
        interceptors.Timeout(time.Millisecond*1500),
    ),
}
if useTLS {
    creds, _ := credentials.NewServerTLSFromFile(cert, key)
    serverOpts = append(serverOpts, grpc.Creds(creds))
}
grpcServer := grpc.NewServer(serverOpts...)
pb.RegisterUserServer(grpcServer, userSvc)
healthSrv := health.NewServer()
healthgrpc.RegisterHealthServer(grpcServer, healthSrv)
go func() { <-ctx.Done(); grpcServer.GracefulStop() }()
log.Printf("listening %s", lis.Addr())
grpcServer.Serve(lis)
```

### 客户端示例（精简）
```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second); defer cancel()
dialOpts := []grpc.DialOption{
    grpc.WithTransportCredentials(insecure.NewCredentials()), // 或 TLS creds
    grpc.WithChainUnaryInterceptor(
        interceptors.ClientTracing(),
        interceptors.ClientLogging(pkglog.L()),
        interceptors.ClientMetrics(),
        interceptors.Retry(3, backoff.Exponential(50*time.Millisecond)),
    ),
}
conn, err := grpc.DialContext(ctx, os.Getenv("GRPC_TARGET"), dialOpts...)
defer conn.Close()
cli := pb.NewUserClient(conn)
resp, err := cli.GetUser(ctx, &pb.GetUserReq{Id: id})
```

## 何时使用该技能
- 服务间需要低延迟、强类型接口、双向流/流式能力。
- 需要统一的链路追踪、日志、指标、重试/超时治理。
- 需要健康检查、负载均衡/服务发现（DNS/etcd）能力。

## 输出格式 / 错误处理
- gRPC Status：使用 `status.Errorf(codes.Code, "message")`；业务错误映射为自定义 codes（如 `codes.InvalidArgument` / `codes.NotFound` / `codes.Aborted` / `codes.DeadlineExceeded`）。
- 日志字段：`method`、`code`、`peer`、`duration_ms`、`req_id`、`trace_id`、`msg`；敏感字段脱敏。
- 空响应约定：返回空 message（非 nil）+ `nil` error；不要返回 `nil, nil`。
- 重试：仅对幂等方法；`codes.Unavailable/DeadlineExceeded` 可重试；`InvalidArgument` 等不可重试。
- 超时：客户端 ctx 超时 -> `DeadlineExceeded`；服务端应尽早返回并记录。
- 流式：对端正常结束返回 `io.EOF`；需区分与错误。

## 示例（常见用例）
1. 服务端最小运行 + 拦截器链（上方示例）
2. 客户端 Dial + 重试 + 超时
3. TLS 双向认证（配置 `credentials.NewTLS` / `NewClientTLSFromFile`）
4. Unary 拦截器：logging/metrics/tracing/recover/auth/timeout
5. Stream 拦截器：同上对应流式版本
6. 健康检查：`grpc/health` 注册与探针
7. 负载均衡：使用 `dns:///service:port` 或 `grpc.WithDefaultServiceConfig` 配置 round_robin
8. 压缩：`grpc.UseCompressor(gzip.Name)` 或在 call 级别指定
9. Metadata 透传：在 ctx 注入 request_id/trace_id，并在拦截器写入/读取
10. 并发限制/限流：在服务端拦截器实现 token bucket；超过直接返回 `ResourceExhausted`
11. 流式示例：Server-stream/Client-stream/Bidi，含超时和最大消息控制
12. 断线重连：客户端 WithBlock+Backoff 重试 Dial；或自定义 balancer 重试策略
13. Deadline Propagation：从 HTTP/gateway ctx 提取 deadline，传递到 gRPC ctx
14. 自定义错误映射：业务错误 -> gRPC code + detail，客户端转换为业务码
15. 观测集成：Prometheus 指标导出 + OpenTelemetry tracing
16. CI 校验：`buf lint/format` 或 proto lint，生成代码版本锁定

## 限制条件与安全规则
- 禁止在非幂等方法上开启自动重试。
- 禁止服务端长时间阻塞 handler；需合理超时与取消。
- TLS 证书必须正确加载；禁止线上明文敏感流量。
- Metadata 不得传递敏感数据（token 脱敏/短期有效）；日志不可落 token。
- 流式接口需控制消息大小与速率；设置 `MaxRecvMsgSize/MaxSendMsgSize`。
- Health/探针：必须注册 `grpc/health`；未就绪时返回 `NOT_SERVING`。
- 负载均衡配置需与服务发现一致；避免客户端直连硬编码。

## 常见坑 / FAQ
- 忘记 ctx 超时：导致 client hang；必须每次调用设置超时。
- 重试配置错误：对非幂等接口重试导致重复写；需限制方法。
- 拦截器顺序混乱：Recover 应最前，Auth/业务前；Logging/Tracing 需包含 request_id/trace_id。
- 健康检查未注册：探针失败导致被摘除。
- TLS CN/SAN 不匹配：Dial 失败；需正确配置根证书与目标名称。
- 消息过大未设置限制：需配置 `MaxRecvMsgSize/MaxSendMsgSize`。
- 未处理流式 io.EOF：需区分正常结束与错误。
- 不同版本 proto/go-grpc 插件不匹配：需锁定版本。
- Metadata 大小/数量过多导致头过大：限制 header 大小。

## 可观测性 / 诊断
- Metrics：QPS/latency/error_rate 按 method/code；流式记录消息数与字节数。
- Tracing：Unary/Stream 拦截器注入/生成 span；标签包含 `rpc.system=grpc`、`rpc.service`、`rpc.method`、`net.peer.name`。
- 慢日志：>200ms Warn（按 SLA 调整）；记录 req_id/trace_id/method/duration/code/peer。
- 诊断：`DeadlineExceeded` → 查超时/队列；`ResourceExhausted` → 限流/池耗尽；`Unavailable` → 连接/负载均衡。
- 健康探针：可用 gRPC healthz CLI；状态与服务内部依赖（DB/Cache）联动。
- Prometheus：导出方法级指标，分客户端/服务端；流式统计 per message。
- 追踪采样：本地 1，线上按流量降采样；跨服务透传 traceparent/baggage。

## 附：关键配置与代码片段
- TLS 服务端
```go
creds, _ := credentials.NewServerTLSFromFile("server.crt", "server.key")
serverOpts = append(serverOpts, grpc.Creds(creds))
```
- TLS 客户端（验证 SAN）
```go
creds, _ := credentials.NewClientTLSFromFile("ca.crt", "server.domain")
grpc.WithTransportCredentials(creds)
```
- 负载均衡（round_robin + service config）
```go
sc := `{"loadBalancingPolicy":"round_robin"}`
conn, _ := grpc.DialContext(ctx, "dns:///user.grpc.svc:8081",
    grpc.WithDefaultServiceConfig(sc),
    grpc.WithTransportCredentials(insecure.NewCredentials()),
    grpc.WithChainUnaryInterceptor(...),
)
```
- 客户端重试策略（幂等方法）
```json
{
  "methodConfig": [{
    "name": [{"service": "user.User", "method": "Get"}],
    "retryPolicy": {
      "maxAttempts": 3,
      "initialBackoff": "0.05s",
      "maxBackoff": "0.2s",
      "backoffMultiplier": 2.0,
      "retryableStatusCodes": ["UNAVAILABLE","DEADLINE_EXCEEDED"]
    }
  }]
}
```
- Unary 拦截器骨架（Logging + Trace + Recover）
```go
func Logging(logger *zap.Logger) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
        start := time.Now()
        defer func() {
            logger.Info("grpc",
                zap.String("method", info.FullMethod),
                zap.Duration("duration", time.Since(start)),
                zap.Error(err),
                zap.String("req_id", requestid.FromContext(ctx)),
                zap.String("trace_id", traceid.FromContext(ctx)),
            )
        }()
        return handler(ctx, req)
    }
}
```
- Deadline 透传（HTTP -> gRPC）
```go
// 从 HTTP ctx 取 deadline，创建 gRPC ctx
deadline, ok := httpCtx.Deadline()
grpcCtx := httpCtx
if ok {
    var cancel context.CancelFunc
    grpcCtx, cancel = context.WithDeadline(httpCtx, deadline)
    defer cancel()
}
resp, err := cli.GetUser(grpcCtx, req)
```
- 流式（server-stream）示例
```go
func (s *UserServer) List(req *pb.ListReq, stream pb.User_ListServer) error {
    ctx, cancel := context.WithTimeout(stream.Context(), 3*time.Second); defer cancel()
    for _, u := range users {
        if err := stream.Send(&pb.UserResp{Id: u.Id, Name: u.Name}); err != nil { return err }
        if ctx.Err() != nil { return ctx.Err() }
    }
    return nil
}
```
- Metadata 透传 request_id/trace_id
```go
md := metadata.Pairs("x-request-id", rid, "x-trace-id", tid)
ctx = metadata.NewOutgoingContext(ctx, md)
resp, err := cli.GetUser(ctx, req)
```
- 限流拦截器（token bucket，超出返回 ResourceExhausted）
```go
type RateLimiter struct { tb *ratelimit.Limiter }
func (l *RateLimiter) Unary() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        if !l.tb.Allow() { return nil, status.Error(codes.ResourceExhausted, "rate limited") }
        return handler(ctx, req)
    }
}
```
- 客户端连接 Keepalive 建议
```go
ka := keepalive.ClientParameters{
    Time: 30 * time.Second, Timeout: 10 * time.Second, PermitWithoutStream: true,
}
grpc.WithKeepaliveParams(ka)
```
- 服务端 MaxRecv/Send 限制
```go
serverOpts = append(serverOpts,
    grpc.MaxRecvMsgSize(4<<20), // 4MB
    grpc.MaxSendMsgSize(4<<20),
)
```
- mTLS（服务端 + 客户端）
```go
// server
creds, _ := tls.LoadX509KeyPair("server.crt", "server.key")
caCert, _ := os.ReadFile("ca.crt")
caPool := x509.NewCertPool(); caPool.AppendCertsFromPEM(caCert)
srvCreds := credentials.NewTLS(&tls.Config{
    Certificates: []tls.Certificate{creds},
    ClientCAs:    caPool,
    ClientAuth:   tls.RequireAndVerifyClientCert,
})
grpc.NewServer(grpc.Creds(srvCreds))
// client
clientCreds := credentials.NewClientTLSFromFile("ca.crt", "server.domain")
grpc.Dial(target, grpc.WithTransportCredentials(clientCreds))
```
- Auth 拦截器（JWT from metadata）
```go
func Auth(validate func(token string) error) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        md, _ := metadata.FromIncomingContext(ctx)
        token := ""
        if vals := md.Get("authorization"); len(vals) > 0 { token = strings.TrimPrefix(vals[0], "Bearer ") }
        if err := validate(token); err != nil { return nil, status.Error(codes.Unauthenticated, "invalid token") }
        return handler(ctx, req)
    }
}
```
- grpc-gateway（REST -> gRPC）骨架
```go
gwMux := runtime.NewServeMux()
creds := insecure.NewCredentials() // or TLS
opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}
pb.RegisterUserHandlerFromEndpoint(ctx, gwMux, target, opts)
http.ListenAndServe(":8080", gwMux)
```
- Status detail（业务码映射）
```go
st := status.New(codes.Aborted, "order conflict")
st, _ = st.WithDetails(&errdetails.ErrorInfo{
    Reason: "ORDER_VERSION_MISMATCH",
    Metadata: map[string]string{"order_id": id},
})
return nil, st.Err()
```
- 双向流拦截器骨架（logging + trace）
```go
func StreamLogging(logger *zap.Logger) grpc.StreamServerInterceptor {
    return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
        start := time.Now()
        err := handler(srv, ss)
        logger.Info("grpc_stream",
            zap.String("method", info.FullMethod),
            zap.Bool("client_stream", info.IsClientStream),
            zap.Bool("server_stream", info.IsServerStream),
            zap.Duration("duration", time.Since(start)),
            zap.Error(err),
            zap.String("req_id", requestid.FromContext(ss.Context())),
            zap.String("trace_id", traceid.FromContext(ss.Context())),
        )
        return err
    }
}
```
- Service Config 组合（LB + Retry）示例
```json
{
  "loadBalancingPolicy": "round_robin",
  "methodConfig": [{
    "name": [{"service": "user.User", "method": "Get"}],
    "retryPolicy": {
      "maxAttempts": 3,
      "initialBackoff": "0.05s",
      "maxBackoff": "0.5s",
      "backoffMultiplier": 2.0,
      "retryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED"]
    },
    "timeout": "2s"
  }]
}
```
- CI 锁定生成版本（示例脚本）
```bash
protoc --version            # 应为 3.x/21.x 与生成器兼容
protoc-gen-go --version     # 固定版本，与 go.mod 对齐
protoc-gen-go-grpc --version
protoc --go_out=. --go-grpc_out=. api.proto
git diff --exit-code api.pb.go api_grpc.pb.go  # 确保无漂移
```
- proto lint（buf 示例）
```yaml
version: v1
lint:
  use:
    - DEFAULT
breaking:
  use:
    - FILE
```

## proto 规范 / 生成锁定
- `syntax = "proto3";`，显式 `go_package = "path/to/pb;pb"`。
- 方法命名使用 PascalCase；错误码枚举集中定义（如 `ErrorCode`）。
- 字段使用下划线命名，避免保留字；使用 `reserved` 保护已弃用字段号。
- 生成器版本锁定：`protoc-gen-go` / `protoc-gen-go-grpc` 与 gRPC-Go v1.71.1 兼容；在 CI 中校验版本。
- 错误映射建议：业务错误映射至 gRPC codes（如验证失败->InvalidArgument，未授权->Unauthenticated，未找到->NotFound，冲突->Aborted，限流->ResourceExhausted），客户端再转换为业务码；在 status detail 中附业务 code/message 便于前端/网关解析。

## 版本与依赖
- gRPC-Go v1.71.1；Go 1.19+。
- 依赖：`google.golang.org/grpc`（含 health、credentials、balancer、encoding/gzip）；自研拦截器在 `mfycommon/grpc-go`（假设存在）。
- 生成器：`protoc` + `protoc-gen-go` + `protoc-gen-go-grpc`（需与 v1.71.1 兼容）。

## 更新记录 / Owner
- 最后更新时间：2024-05-xx。
- Owner：RPC 规范小组（@架构负责人），评审人：服务域负责人；变更需双人 Review。
