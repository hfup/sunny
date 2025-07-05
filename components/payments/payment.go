package payments


type PayOptionInf interface {
	GetOrderId() string // 订单id
	PayFee() int64 // 支付金额 单位分
	GetPayUserId() string // 获取用户id
	GetOption() any // 获取其他参数
}


type RefundOptionInf interface {
	PayOptionInf
	GetRefundFee() int64 // 退款金额 单位分
}

// 支付结果接口
type PayresultInf interface {
	GetOrderId() string // 订单id
	GetThirdOrderId() string // 第三方支付
	GetPayTime() int64 // 时间戳 支付时间
	GetPayUserId() string // 支付用户id
}

type PaymentInf interface {
	PayType() string // 返回支付类型
	// 预支付 创建支付参数 返回支付参数
	PrePay(option PayOptionInf,result any) error
	// 验证支付结果
	// 参数：
	//  - opt 验证参数
	// 返回：
	//  - PayresultInf 支付结果
	//  - error 错误
	VerifyPay(opt any) (PayresultInf,error)
	// 退款
	Refund(option RefundOptionInf) error
}

// 支付管理
type PaymentMangerInf interface {
	AddPayment(payment PaymentInf) error
	GetPayment(paymentType string) PaymentInf
	RemovePayment(paymentType string) error

	PrePay(option PayOptionInf,result any) error
	Refund(option RefundOptionInf) error
}