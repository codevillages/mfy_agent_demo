# 配置清单 / 所需输入
- 配置文件：`etc/<svc>.yaml`（REST/RPC/KQ/Job），至少包含 `Name`、`Host/Port` 或 `ListenOn`、`Timeout`、`Log`、`Telemetry`、`Redis/Mysql`。
- 环境变量：`APP_ENV`(`dev`/`staging`/`prod`)、`LOG_LEVEL`（覆盖配置）、`TRACE_ENDPOINT`、`HTTP_PORT`、密钥类（DB/Redis/AccessSecret）。
- 运行要求：Go 1.19+；容器 liveness/readiness；`GOMAXPROCS`=CPU 核；必须有退出信号钩子。
- 配置模板：`examples/rest/etc/user-api.yaml`、`examples/rpc/etc/user-rpc.yaml`。

## 推荐默认
- `RestConf`: `Host: 0.0.0.0`，`Port: 8080`，`Timeout: 2000ms`，`CpuThreshold: 900`，`MaxBytes: 1MiB`，`Verbose: false`。
- `RpcServerConf`: `ListenOn: 0.0.0.0:8081`，`Timeout: 2000ms`，`CpuThreshold: 900`，`StrictControl: true`。
- `Log`: `Mode: console`（线上可 `file`），`Encoding: json`，`Level: info`。
- `Redis`: `Host` 必填，`Type: node|cluster|sentinel`；`Pass` 走密钥；`TLS` 按需。
- `Mysql`: `DataSource` 必填；`MaxOpenConns: 80`、`MaxIdleConns: 20`、`ConnMaxLifetime: 1h`。
- `Telemetry`: `Endpoint`（OTLP/Zipkin），`Batcher: otlp`，`Sampler: 1`（上线按流量降采样）。
- `Prometheus`: `Host: 0.0.0.0`，`Port: 9091`（独立端口）。

## 字段说明
- `RestConf`：`Timeout` 单次请求超时；`MaxConns` 并发连接；`MaxBytes` 请求体大小；`CpuThreshold` CPU 保护（>1000 关闭）。
- `RpcServerConf`：`Timeout`、`CpuThreshold`、`StrictControl`(true 防过载)，`Auth`(AccessSecret/AccessExpire)。
- `ClientConf`：RPC 客户端 `Timeout`/`Retry`(Attempts、Interval、Timeout)；`Etcd.Hosts` 服务发现；`App`/`Token` 鉴权。
- `Telemetry`：`Endpoint`、`Sampler`（0.1~1）、`Resource` 标签（`service.name`=`Name`）。
- `Prometheus`：暴露 metrics 地址，Path `/metrics`。
- `RedisConf`：`Type`、`Pass`、`TLS`、`NonBlock`。
- `SqlConf`：`DataSource`、`MaxOpenConns`、`MaxIdleConns`、`ConnMaxLifetime`、`ConnMaxIdleTime`。

## KQ（Kafka）示例
```yaml
KqConf:
  Name: user-consumer
  Brokers: ["kafka:9092"]
  Group: user-group
  Topic: user-topic
  Offset: first  # or last
  Conns: 4
  Consumers: 8
  Processors: 16
```
