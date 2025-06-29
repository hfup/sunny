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
	
}


func (s *Sunny) AddSubServices(srvs ...SubServiceInf) error{
	s.subServices = append(s.subServices,srvs...)
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