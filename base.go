package sunny

import "context"	


// 子服务接口
type SubServiceInf interface {
	Start(ctx context.Context,app *Sunny,resultChan chan<- Result)
	IsErrorStop() bool // 如果返回 true, 如果启动 服务 code 不为 0 则终止整个服务启动
	ServiceName() string // 服务名称
}


// 可运行接口  
type RunAbleInf interface {
	Run(ctx context.Context,args any) error
}