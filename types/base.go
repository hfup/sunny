package types

import (
	"context"
	"google.golang.org/grpc"
)


// Result 结果
type Result[T any] struct {
	ErrCode     int    `json:"err_code"`
	Message     string `json:"message"`
	Data        T      `json:"data"`
}

// Dict 字典
type Dict map[string]any



// 子服务接口
type SubServiceInf interface {
	Start(ctx context.Context,args any,resultChan chan<- Result[any])
	IsErrorStop() bool // 如果返回 true, 如果启动 服务 err_code 不为 0 则终止整个服务启动
	ServiceName() string // 服务名称
}

// 可运行接口  
type RunAbleInf interface {
	Run(ctx context.Context,args any) error
}

// 注册 grpc 服务接口
type RegisterGrpcServiceInf interface {
	RegisterGrpcService(server *grpc.Server)
}