//go:build ignore
// +build ignore

// 示例 RPC 启动文件，需替换 pb 包路径与业务实现。
package main

import (
	"context"
	"flag"
	"net"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/core/trace"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// 请用 goctl 生成的 pb 代码替换此路径。
// pb "path/to/your/pb"

type Config struct {
	zrpc.RpcServerConf
	Redis     redis.RedisConf
	Mysql     sqlx.SqlConf
	Telemetry trace.Config
}

type ServiceContext struct {
	Config Config
	DB     sqlx.SqlConn
	Redis  *redis.Redis
}

func NewServiceContext(c Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		DB:     sqlx.NewMysql(c.Mysql.DataSource),
		Redis:  redis.MustNewRedis(c.Redis),
	}
}

// 示例业务实现，实际应由 goctl 生成的 server 结构体封装。
type UserServer struct {
	svcCtx *ServiceContext
}

// 示例方法，替换为实际 pb 中定义的方法。
func (s *UserServer) Ping(ctx context.Context, _ *Empty) (*Pong, error) {
	logx.WithContext(ctx).Info("rpc ping")
	return &Pong{Message: "pong"}, nil
}

// RegisterUserServer 由 goctl 生成的 pb 提供；此处示例手动注册，方便参考。
func RegisterUserServer(grpcServer *grpc.Server, srv *UserServer) {
	grpcServer.RegisterService(&grpc.ServiceDesc{
		ServiceName: "user.User",
		HandlerType: (*UserServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Ping",
				Handler: func(s interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
					in := new(Empty)
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.Ping(ctx, in)
					}
					info := &grpc.UnaryServerInfo{
						Server:     s,
						FullMethod: "/user.User/Ping",
					}
					handler := func(ctx context.Context, req interface{}) (interface{}, error) {
						return srv.Ping(ctx, req.(*Empty))
					}
					return interceptor(ctx, in, info, handler)
				},
			},
		},
		Streams:  []grpc.StreamDesc{},
		Metadata: "user.proto",
	}, srv)
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "f", "etc/user-rpc.yaml", "the config file")
	flag.Parse()

	var c Config
	conf.MustLoad(configFile, &c)
	logx.MustSetup(c.Log)

	closer := trace.StartAgent(c.Telemetry)
	defer closer.Close()

	svcCtx := NewServiceContext(c)
	defer func() {
		logx.Close()
		if svcCtx.Redis != nil {
			svcCtx.Redis.Close()
		}
	}()

	server := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		RegisterUserServer(grpcServer, &UserServer{svcCtx: svcCtx})
	})
	defer server.Stop()

	logx.Infof("Starting rpc server at %s", c.ListenOn)
	server.Start()
}

// Empty 和 Pong 仅为示例，占位用；真实项目由 proto 生成。
type Empty struct{}

type Pong struct {
	Message string
}

// dummyDecoder 提供给自定义注册时的 decode，需要真实运行时由 gRPC 框架处理。
func dummyDecoder(interface{}) error { return status.Error(0, "") }

// 保留避免未使用错误；真实使用时不需要。
var _ = net.IPv4len
