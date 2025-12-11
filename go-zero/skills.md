# @mfycommon/go-zero v1.8.2 — Skills 索引

> 用作目录，AI 按需加载子章节。详细内容见 `sections/` 与 `examples/`。

- **概览/适用场景**：`sections/overview.md`
- **配置清单**：`sections/config.md`
- **REST 指南（示例代码+注释）**：`sections/rest.md`
- **RPC 指南（服务端/客户端/TLS/拦截器）**：`sections/rpc.md`
- **中间件/拦截器示例**：`sections/middleware.md`
- **ServiceContext 组织与健康检查**：`sections/service_context.md`
- **输出/错误/治理/限制**：`sections/governance.md`
- **可观测性/调试/压测**：`sections/observability.md`
- **示例代码索引**：`sections/examples.md`
- **常见坑/FAQ**：`sections/faq.md`
- **版本/依赖/Owner**：`sections/meta.md`

入口示例：
- `sections/rest.md#L1`：REST 全流程示例，含健康检查与优雅关闭。
- `sections/rpc.md#L1`：RPC 全流程示例，含拦截器/TLS。
- `examples/rest/main.go` + `examples/rest/etc/user-api.yaml`：REST 最小可跑。
- `examples/rpc/main.go` + `examples/rpc/etc/user-rpc.yaml`：RPC 配置与入口骨架。
