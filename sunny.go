package sunny

import "context"

// Result 结果
type Result struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
}


// Sunny 主服务
type Sunny struct {
	subServices []SubServiceInf
	// 同步执行的 RunAble
	syncRunAbles []RunAbleInf
	asyncRunAbles []RunAbleInf
	requiredSubSrvSuccessCount int // 必须成功启动的子服务数量
	currentSubSrvSuccessCount int // 当前成功启动的子服务数量
}

// 添加子服务
// 参数：
//  - srvs 子服务
// 返回：
//  - 错误	
func (s *Sunny) AddSubServices(srvs ...SubServiceInf) error{
	for _,srv := range srvs{
		if srv.IsErrorStop(){
			s.requiredSubSrvSuccessCount += 1
		}
		s.subServices = append(s.subServices,srv)
	}
	return nil
}

// 添加同步执行的 RunAble
// 参数：
//  - srvs 同步执行的 RunAble
// 返回：
//  - 错误
func (s *Sunny) AddSyncRunAbles(srvs ...RunAbleInf) error{
	s.syncRunAbles = append(s.syncRunAbles,srvs...)
	return nil
}

// 添加异步执行的 RunAble
// 参数：
//  - srvs 异步执行的 RunAble
// 返回：
//  - 错误
func (s *Sunny) AddAsyncRunAbles(srvs ...RunAbleInf) error{
	s.asyncRunAbles = append(s.asyncRunAbles,srvs...)
	return nil
}


// Start 启动 Sunny
// 参数：
//  - ctx 上下文
//  - args 参数
// 返回：
//  - 错误
func (s *Sunny) Start(ctx context.Context,args ...string) error{

	return nil
}