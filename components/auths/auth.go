package auths

import (
	"context"
)

// 选项
type LoginOptionInf interface {
	GetLoginType() string // 登录类型
	GetUserId() // 登陆用户身份
	GetOption() any // 获取登陆其他参数
}

// 认证器
type Authenticator interface {
	Login(ctx context.Context,option LoginOptionInf,result any) error
	Logout(ctx context.Context,option LoginOptionInf) error
	// 验证登陆信息 返回信息，错误
	// 参数：
	//  - ctx 上下文
	//  - option 验证选项
	//  - result 获取验证的用户信息
	// 返回：
	//  - error 错误
	Verify(ctx context.Context,option LoginOptionInf,result any) error // 验证登陆信息 返回信息，错误
}

// 权限管理器
type AuthManagerInf interface {
	GetAuthenticator(loginType string) (Authenticator,error) // 获取认证器
	AddAuthenticator(authenticator Authenticator) error // 添加认证器
	RemoveAuthenticator(loginType string) error // 删除认证器
	Login(ctx context.Context,option LoginOptionInf,result any) error // 登录
	Logout(ctx context.Context,option LoginOptionInf) error // 登出
	Verify(ctx context.Context,option LoginOptionInf,result any) error // 验证登陆信息 返回信息，错误
}



