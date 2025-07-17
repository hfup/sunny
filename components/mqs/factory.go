package mqs

import (
	"fmt"
	"github.com/hfup/sunny/types"
	"time"
)

// CreateMqManager 根据类型创建消息队列管理器
// 参数:
//   - mqType: MqType 消息队列类型
//   - options: any 配置选项
//   - failedStore: FailedMessageStore 失败消息存储(可选)
// 返回:
//   - MqManagerInf 消息队列管理器接口
//   - error 错误信息
func CreateMqManager(opt *types.MqConfig, failedStore FailedMessageStore) (MqManagerInf, error) {
	switch opt.Type {
	case "rabbitmq":
		url := fmt.Sprintf("amqp://%s:%s@%s:%d/",opt.RabbitMQ.Username,opt.RabbitMQ.Password,opt.RabbitMQ.Host,opt.RabbitMQ.Port)
		return NewRabbitMqManager(&RabbitMQOptions{
		URL: url,
		ChannelPoolSize: opt.RabbitMQ.ChannelPoolSize,
		MaxRetries: opt.RabbitMQ.MaxRetries,
		RetryInterval: time.Duration(opt.RabbitMQ.RetryInterval) * time.Second, // 重试间隔
		ReconnectDelay: time.Duration(opt.RabbitMQ.ReconnectDelay) * time.Second, // 重连延迟
		},failedStore),nil
	case "kafka":
		return NewKafkaManager(&KafkaOptions{
			Brokers: opt.Kafka.Brokers,
			MaxRetries: opt.Kafka.MaxRetries,
			RetryInterval: time.Duration(opt.Kafka.RetryInterval) * time.Second, // 重试间隔
			ReconnectDelay: time.Duration(opt.Kafka.ReconnectDelay) * time.Second, // 重连延迟
			SecurityProtocol: opt.Kafka.SecurityProtocol,
		},failedStore), nil	
	default:
		return nil, fmt.Errorf("不支持的消息队列类型: %s", opt.Type)
	}
}

// CreateConsumerByType 根据管理器类型创建消费者
// 参数:
//   - manager: MqManagerInf 消息队列管理器
//   - topic: string 主题
//   - queue: string 队列
//   - options: any 配置选项
// 返回:
//   - ConsumerInf 消费者接口
//   - error 错误信息
func CreateConsumerByType(manager MqManagerInf, topic, queue string, options any) (ConsumerInf, error) {
	switch m := manager.(type) {
	case *RabbitMqManager:
		rabbitOptions, ok := options.(*RabbitMQConsumerOptions)
		if !ok {
			return nil, fmt.Errorf("rabbitMQ 消费者配置选项类型错误，期望 *RabbitMQConsumerOptions")
		}
		return m.CreateConsumer(topic, queue, rabbitOptions)
		
	case *KafkaManager:
		kafkaOptions, ok := options.(*KafkaConsumerOptions)
		if !ok {
			return nil, fmt.Errorf("kafka 消费者配置选项类型错误，期望 *KafkaConsumerOptions")
		}
		return m.CreateConsumer(topic, queue, kafkaOptions)
		
	default:
		return nil, fmt.Errorf("不支持的消息队列管理器类型")
	}
}

// CreateProducerByType 根据管理器类型创建生产者
// 参数:
//   - manager: MqManagerInf 消息队列管理器
//   - topic: string 主题
//   - options: any 配置选项
// 返回:
//   - ProducerInf 生产者接口
//   - error 错误信息
func CreateProducerByType(manager MqManagerInf, topic string, options any) (ProducerInf, error) {
	switch m := manager.(type) {
	case *RabbitMqManager:
		rabbitOptions, ok := options.(*RabbitMQProducerOptions)
		if !ok {
			return nil, fmt.Errorf("rabbitMQ 生产者配置选项类型错误，期望 *RabbitMQProducerOptions")
		}
		return m.CreateProducer(topic, rabbitOptions)
		
	case *KafkaManager:
		kafkaOptions, ok := options.(*KafkaProducerOptions)
		if !ok {
			return nil, fmt.Errorf("kafka 生产者配置选项类型错误，期望 *KafkaProducerOptions")
		}
		return m.CreateProducer(topic, kafkaOptions)
		
	default:
		return nil, fmt.Errorf("不支持的消息队列管理器类型")
	}
}

// GetDefaultFailedStore 获取默认的失败消息存储
// 返回:
//   - FailedMessageStore 失败消息存储接口
func GetDefaultFailedStore() FailedMessageStore {
	return NewMemoryFailedMessageStore()
}

// GetFileFailedStore 获取基于文件的失败消息存储
// 参数:
//   - filePath: string 存储文件路径
// 返回:
//   - FailedMessageStore 失败消息存储接口
//   - error 错误信息
func GetFileFailedStore(filePath string) (FailedMessageStore, error) {
	return NewFileFailedMessageStore(filePath)
}

// GetFileFailedStoreWithDefault 获取基于文件的失败消息存储(使用默认路径)
// 返回:
//   - FailedMessageStore 失败消息存储接口
//   - error 错误信息
func GetFileFailedStoreWithDefault() (FailedMessageStore, error) {
	return NewFileFailedMessageStore("./data/failed_messages.json")
}

// DefaultRabbitMQOptions 获取默认的 RabbitMQ 配置
// 参数:
//   - url: string 连接地址
// 返回:
//   - *RabbitMQOptions 配置选项
func DefaultRabbitMQOptions(url string) *RabbitMQOptions {
	return &RabbitMQOptions{
		URL:             url,               // 连接地址
		Durable:         true,              // 持久化
		AutoDelete:      false,             // 不自动删除
		Internal:        false,             // 非内部交换机
		NoWait:          false,             // 等待确认
		MaxRetries:      3,                 // 最大重试3次
		RetryInterval:   5 * time.Second,   // 5秒重试间隔
		ReconnectDelay:  10 * time.Second,  // 10秒重连延迟
		ChannelPoolSize: 10,                // 默认 Channel 池大小为 10
	}
}

// DefaultKafkaOptions 获取默认的 Kafka 配置
// 参数:
//   - brokers: []string Kafka 集群地址
// 返回:
//   - *KafkaOptions 配置选项
func DefaultKafkaOptions(brokers []string) *KafkaOptions {
	return &KafkaOptions{
		Brokers:       brokers,
		MaxRetries:    3,                 // 最大重试3次
		RetryInterval: 5 * time.Second,   // 5秒重试间隔
		ReconnectDelay: 10 * time.Second, // 10秒重连延迟
	}
}

// DefaultRabbitMQConsumerOptions 获取默认的 RabbitMQ 消费者配置
// 参数:
//   - autoAck: bool 是否自动确认 true 自动确认，消息一旦投递就被认为已消费，不会重发，即使消费失败
// 返回:
//   - *RabbitMQConsumerOptions 配置选项
func DefaultRabbitMQConsumerOptions(autoAck bool) *RabbitMQConsumerOptions {
	return &RabbitMQConsumerOptions{
		RabbitMQQueueOptions: RabbitMQQueueOptions{
			Durable:    true,  // 持久化
			AutoDelete: false, // 不自动删除
			Exclusive:  false, // 非排他性
			NoWait:     false, // 等待确认
		},
		AutoAck:   autoAck, // 是否自动确认
		Consumer:  "",    // 自动生成消费者标签
		NoLocal:   false, // 接收本地消息
		Exclusive: false, // 非排他性
	}
}

// DefaultKafkaConsumerOptions 获取默认的 Kafka 消费者配置
// 参数:
//   - groupID: string 消费者组ID
// 返回:
//   - *KafkaConsumerOptions 配置选项
func DefaultKafkaConsumerOptions(groupID string) *KafkaConsumerOptions {
	return &KafkaConsumerOptions{
		GroupID:           groupID,
		AutoOffsetReset:   "latest",          // 从最新偏移开始
		EnableAutoCommit:  true,              // 自动提交偏移量（消息被读取后立即提交，不管处理是否成功）
		SessionTimeout:    30 * time.Second,  // 30秒会话超时
		HeartbeatInterval: 3 * time.Second,   // 3秒心跳间隔
		FetchMinBytes:     1,                 // 最小拉取1字节
		FetchMaxBytes:     1048576,           // 最大拉取1MB
		MaxWaitTime:       250 * time.Millisecond, // 最大等待250毫秒
		Partition:         -1,                // 自动分配分区
		Offset:            -1,                // 自动偏移
	}
}

// DefaultRabbitMQProducerOptions 获取默认的 RabbitMQ 生产者配置
// 返回:
//   - *RabbitMQProducerOptions 配置选项
func DefaultRabbitMQProducerOptions() *RabbitMQProducerOptions {
	return &RabbitMQProducerOptions{
		Mandatory: false, // 非强制路由
		Immediate: false, // 非立即投递
	}
}

// DefaultKafkaProducerOptions 获取默认的 Kafka 生产者配置
// 返回:
//   - *KafkaProducerOptions 配置选项
func DefaultKafkaProducerOptions() *KafkaProducerOptions {
	return &KafkaProducerOptions{
		Acks:            "all",           // 等待所有副本确认
		Retries:         3,               // 重试3次
		BatchSize:       16384,           // 16KB批处理大小
		Linger:          0,               // 不延迟
		BufferMemory:    33554432,        // 32MB缓冲内存
		CompressionType: "none",          // 不压缩
		MaxRequestSize:  1048576,         // 1MB最大请求大小
		RequestTimeout:  30 * time.Second, // 30秒请求超时
		Partition:       -1,              // 自动分配分区
	}
}