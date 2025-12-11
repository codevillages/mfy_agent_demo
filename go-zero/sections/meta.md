# 版本 / 依赖 / Owner
- 框架：go-zero v1.8.2；Go 1.19+；goctl v1.8.2（保持一致）。
- 主要依赖：`github.com/zeromicro/go-zero/rest`、`.../zrpc`、`.../core/logx`、`.../core/conf`、`.../core/stores/sqlx`、`.../core/stores/redis`、`.../core/trace`、`.../core/breaker`、`.../core/queue/kq`。
- 目录约定：`etc/`（配置）、`cmd/<svc>`（入口）、`internal/config`、`internal/svc`、`internal/handler`、`internal/logic`、`internal/types`、`internal/repo|model`、`internal/middleware`。
- 最后更新时间：2024-05-xx。
- Owner：go-zero 小组（@架构负责人），评审人：服务域负责人；变更需双人 Review。
