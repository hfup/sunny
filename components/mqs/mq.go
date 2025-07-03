package mqs

import (
	"context"
	"time"

	"github.com/hfup/sunny/types"
)

// MQ类型枚举 (保留，用于类型检查)
type MqType string

const (
	MqTypeRabbitMQ MqType = "rabbitmq"
	MqTypeKafka    MqType = "kafka"
)

// 消息主题
type TopicInf interface {
	Topic() string
}

// 消息队列
type QueuesInf interface {
	Queues() string
}


type ConsumerHandlerFunc func(ctx context.Context, msg []byte) error

// 消费者
type ConsumerInf interface {
	Consume(ctx context.Context) error // 消费消息
	BindHandler(handler ConsumerHandlerFunc) // 绑定处理函数
	ChannelId() string // topic + queues
}


// 生产者
type ProducerInf interface {
	TopicInf
	Publish(ctx context.Context, msg []byte) error // 发布消息
}


// 消息队列管理器
type MqManagerInf interface {
	types.SubServiceInf
	// 创建消费者
	// 参数
	// - topic: 主题
	// - queue: 队列
	// - opt: 选项 [配置选项]
	// 返回
	// - 消费者
	// - 错误
	CreateConsumer(topic,queue string,opt any) (ConsumerInf,error)
	// 参数
	// - topic: 主题
	// - opt: 选项 [配置选项]
	// 返回
	// - 消费者
	// - 错误
	CreateProducer(topic string,opt any) (ProducerInf,error)
	
	
	// 绑定消费者
	BindConsumer(consumers ...ConsumerInf) error
	BindProducer(producers ...ProducerInf) error
	BindTopic(topics ...TopicInf) error

	Publish(ctx context.Context,topic string,msg []byte) error
	Close() error // 关闭mq
	
	// 重连相关
	IsConnected() bool // 检查连接状态
	Reconnect() error  // 手动重连
}


// 消息结构
type Message struct {
	ID        string            `json:"id"`         // 消息ID
	Topic     string            `json:"topic"`      // 主题
	Queue     string            `json:"queue"`      // 队列
	Body      []byte            `json:"body"`       // 消息体
	Headers   map[string]any    `json:"headers"`    // 消息头
	Timestamp time.Time         `json:"timestamp"`  // 时间戳
	RetryCount int              `json:"retry_count"` // 重试次数
}

// 失败消息存储接口
type FailedMessageStore interface {
	// Store 存储失败的消息
	// 参数:
	//   - msg: *Message 失败的消息
	// 返回:
	//   - error 错误信息
	Store(msg *Message) error
	
	// Retrieve 检索所有失败的消息
	// 返回:
	//   - []*Message 失败消息列表
	//   - error 错误信息  
	Retrieve() ([]*Message, error)
	
	// Remove 移除已成功重发的消息
	// 参数:
	//   - id: string 消息ID
	// 返回:
	//   - error 错误信息
	Remove(id string) error
	
	// Clear 清空所有失败消息
	// 返回:
	//   - error 错误信息
	Clear() error
}