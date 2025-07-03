package mqs

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"github.com/google/uuid"
	"github.com/IBM/sarama"
	"github.com/hfup/sunny/types"
)

// Kafka 配置选项
type KafkaOptions struct {
	Brokers         []string      `json:"brokers"`          // Kafka 集群地址
	MaxRetries      int           `json:"max_retries"`      // 最大重试次数
	RetryInterval   time.Duration `json:"retry_interval"`   // 重试间隔
	ReconnectDelay  time.Duration `json:"reconnect_delay"`  // 重连延迟
	SecurityProtocol string       `json:"security_protocol"` // 安全协议
	SASLMechanism   string        `json:"sasl_mechanism"`   // SASL 机制
	SASLUsername    string        `json:"sasl_username"`    // SASL 用户名
	SASLPassword    string        `json:"sasl_password"`    // SASL 密码
}

// Kafka 消费者配置选项
type KafkaConsumerOptions struct {
	GroupID          string            `json:"group_id"`           // 消费者组ID
	AutoOffsetReset  string            `json:"auto_offset_reset"`  // 自动偏移重置策略
	EnableAutoCommit bool              `json:"enable_auto_commit"` // 自动提交
	SessionTimeout   time.Duration     `json:"session_timeout"`    // 会话超时
	HeartbeatInterval time.Duration    `json:"heartbeat_interval"` // 心跳间隔
	FetchMinBytes    int32             `json:"fetch_min_bytes"`    // 最小拉取字节数
	FetchMaxBytes    int32             `json:"fetch_max_bytes"`    // 最大拉取字节数
	MaxWaitTime      time.Duration     `json:"max_wait_time"`      // 最大等待时间
	Partition        int32             `json:"partition"`          // 分区
	Offset           int64             `json:"offset"`             // 偏移量
	Metadata         map[string]string `json:"metadata"`           // 元数据
}

// Kafka 生产者配置选项
type KafkaProducerOptions struct {
	Acks             string        `json:"acks"`              // 确认模式
	Retries          int           `json:"retries"`           // 重试次数
	BatchSize        int           `json:"batch_size"`        // 批处理大小
	Linger           time.Duration `json:"linger_ms"`         // 延迟时间
	BufferMemory     int64         `json:"buffer_memory"`     // 缓冲内存
	CompressionType  string        `json:"compression_type"`  // 压缩类型
	MaxRequestSize   int32         `json:"max_request_size"`  // 最大请求大小
	RequestTimeout   time.Duration `json:"request_timeout"`   // 请求超时
	Partition        int32         `json:"partition"`         // 分区
	Key              []byte        `json:"key"`               // 消息键
	Headers          map[string][]byte `json:"headers"`       // 消息头
}

// KafkaManager 是 Kafka 消息队列管理器
// 实现了 MqManagerInf 接口
type KafkaManager struct {
	options         *KafkaOptions       // 配置选项
	client          sarama.Client       // Kafka 客户端
	producer        sarama.SyncProducer // 同步生产者
	consumers       []ConsumerInf       // 消费者列表
	producers       []ProducerInf       // 生产者列表
	topics          []TopicInf          // 主题列表
	failedStore     FailedMessageStore  // 失败消息存储
	isConnected     bool                // 连接状态
	reconnectTicker *time.Ticker        // 重连定时器
	mu              sync.RWMutex        // 读写锁
	ctx             context.Context     // 上下文
	cancel          context.CancelFunc  // 取消函数
}

// KafkaConsumer Kafka 消费者
type KafkaConsumer struct {
	topic         string                 // 主题
	queue         string                 // 队列(对应Kafka分区)
	options       *KafkaConsumerOptions  // 配置选项
	handler       ConsumerHandlerFunc    // 处理函数
	manager       *KafkaManager          // 管理器引用
	consumerGroup sarama.ConsumerGroup   // 消费者组
	done          chan error             // 完成通道
}

// KafkaProducer Kafka 生产者
type KafkaProducer struct {
	topic   string                 // 主题
	options *KafkaProducerOptions  // 配置选项
	manager *KafkaManager          // 管理器引用
}

// NewKafkaConsumer 快速创建 Kafka 消费者
// 参数:
//   - manager: *KafkaManager 管理器
//   - topic: string 主题
//   - groupID: string 消费者组ID
//   - enableAutoCommit: bool 是否自动提交偏移量，true 自动提交，消息一旦被消费就提交偏移量，
//                       false 手动提交，需要在消息处理成功后才提交偏移量
// 返回:
//   - *KafkaConsumer 消费者实例
func NewKafkaConsumer(manager *KafkaManager, topic, groupID string, enableAutoCommit bool) *KafkaConsumer {
	options := DefaultKafkaConsumerOptions(groupID)
	options.EnableAutoCommit = enableAutoCommit
	return &KafkaConsumer{
		topic:   topic,
		queue:   "0", // 默认分区
		options: options,
		manager: manager,
		done:    make(chan error),
	}
}

// NewKafkaConsumerWithOptions 使用自定义配置创建 Kafka 消费者
// 参数:
//   - manager: *KafkaManager 管理器
//   - topic: string 主题
//   - queue: string 队列(分区)
//   - options: *KafkaConsumerOptions 配置选项
// 返回:
//   - *KafkaConsumer 消费者实例
func NewKafkaConsumerWithOptions(manager *KafkaManager, topic, queue string, options *KafkaConsumerOptions) *KafkaConsumer {
	return &KafkaConsumer{
		topic:   topic,
		queue:   queue,
		options: options,
		manager: manager,
		done:    make(chan error),
	}
}

// NewKafkaProducer 快速创建 Kafka 生产者
// 参数:
//   - manager: *KafkaManager 管理器
//   - topic: string 主题
// 返回:
//   - *KafkaProducer 生产者实例
func NewKafkaProducer(manager *KafkaManager, topic string) *KafkaProducer {
	return &KafkaProducer{
		topic:   topic,
		options: DefaultKafkaProducerOptions(),
		manager: manager,
	}
}

// NewKafkaProducerWithOptions 使用自定义配置创建 Kafka 生产者
// 参数:
//   - manager: *KafkaManager 管理器
//   - topic: string 主题
//   - options: *KafkaProducerOptions 配置选项
// 返回:
//   - *KafkaProducer 生产者实例
func NewKafkaProducerWithOptions(manager *KafkaManager, topic string, options *KafkaProducerOptions) *KafkaProducer {
	return &KafkaProducer{
		topic:   topic,
		options: options,
		manager: manager,
	}
}

// NewKafkaManager 创建 Kafka 管理器
// 参数:
//   - options: *KafkaOptions Kafka 配置选项
//   - failedStore: FailedMessageStore 失败消息存储(可选)
// 返回:
//   - *KafkaManager 管理器实例
func NewKafkaManager(options *KafkaOptions, failedStore FailedMessageStore) *KafkaManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	if options.MaxRetries <= 0 {
		options.MaxRetries = 3 // 默认重试3次
	}
	if options.RetryInterval <= 0 {
		options.RetryInterval = 5 * time.Second // 默认重试间隔5秒
	}
	if options.ReconnectDelay <= 0 {
		options.ReconnectDelay = 10 * time.Second // 默认重连延迟10秒
	}
	
	return &KafkaManager{
		options:     options,
		consumers:   make([]ConsumerInf, 0),
		producers:   make([]ProducerInf, 0),
		topics:      make([]TopicInf, 0),
		failedStore: failedStore,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start 启动 Kafka 管理器
// 参数:
//   - ctx: context.Context 上下文
//   - args: any 启动参数
//   - resultChan: chan<- types.Result[any] 结果通道
func (k *KafkaManager) Start(ctx context.Context, args any, resultChan chan<- types.Result[any]) {
	if err := k.connect(); err != nil {
		resultChan <- types.Result[any]{
			ErrCode: 1,
			Message: fmt.Sprintf("Kafka 连接失败: %v", err),
		}
		return
	}
	
	// 启动重连监控
	go k.startReconnectMonitor()
	
	// 重发失败消息
	if k.failedStore != nil {
		go k.resendFailedMessages()
	}
	
	resultChan <- types.Result[any]{
		ErrCode: 0,
		Message: "Kafka 管理器启动成功",
	}
}

// connect 建立连接
// 返回:
//   - error 错误信息
func (k *KafkaManager) connect() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	config.Producer.RequiredAcks = sarama.WaitForAll // 等待所有副本确认
	config.Producer.Retry.Max = k.options.MaxRetries
	config.Producer.Return.Successes = true
	config.Consumer.Return.Errors = true
	
	// 安全配置
	if k.options.SecurityProtocol != "" {
		switch k.options.SecurityProtocol {
		case "SASL_PLAINTEXT", "SASL_SSL":
			config.Net.SASL.Enable = true
			config.Net.SASL.Mechanism = sarama.SASLMechanism(k.options.SASLMechanism)
			config.Net.SASL.User = k.options.SASLUsername
			config.Net.SASL.Password = k.options.SASLPassword
		}
	}
	
	var err error
	k.client, err = sarama.NewClient(k.options.Brokers, config)
	if err != nil {
		return fmt.Errorf("创建 kafka 客户端失败: %w", err)
	}
	
	k.producer, err = sarama.NewSyncProducerFromClient(k.client)
	if err != nil {
		k.client.Close()
		return fmt.Errorf("创建 kafka 生产者失败: %w", err)
	}
	
	k.isConnected = true
	log.Printf("Kafka 连接成功: %v", k.options.Brokers)
	return nil
}

// IsErrorStop 是否错误停止
// 返回:
//   - bool 是否错误停止
func (k *KafkaManager) IsErrorStop() bool {
	return true // 连接失败时停止服务
}

// ServiceName 服务名称
// 返回:
//   - string 服务名称
func (k *KafkaManager) ServiceName() string {
	return "Kafka Manager"
}

// CreateConsumer 创建消费者
// 参数:
//   - topic: string 主题
//   - queue: string 队列(对应分区)
//   - opt: any 配置选项
// 返回:
//   - ConsumerInf 消费者接口
//   - error 错误信息
func (k *KafkaManager) CreateConsumer(topic, queue string, opt any) (ConsumerInf, error) {
	options, ok := opt.(*KafkaConsumerOptions)
	if !ok {
		return nil, fmt.Errorf("配置选项类型错误，期望 *KafkaConsumerOptions")
	}
	
	consumer := &KafkaConsumer{
		topic:   topic,
		queue:   queue,
		options: options,
		manager: k,
		done:    make(chan error),
	}
	
	return consumer, nil
}

// CreateProducer 创建生产者
// 参数:
//   - topic: string 主题
//   - opt: any 配置选项
// 返回:
//   - ProducerInf 生产者接口
//   - error 错误信息
func (k *KafkaManager) CreateProducer(topic string, opt any) (ProducerInf, error) {
	options, ok := opt.(*KafkaProducerOptions)
	if !ok {
		return nil, fmt.Errorf("配置选项类型错误，期望 *KafkaProducerOptions")
	}
	
	producer := &KafkaProducer{
		topic:   topic,
		options: options,
		manager: k,
	}
	
	return producer, nil
}

// BindConsumer 绑定消费者
// 参数:
//   - consumers: ...ConsumerInf 消费者列表
// 返回:
//   - error 错误信息
func (k *KafkaManager) BindConsumer(consumers ...ConsumerInf) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	
	k.consumers = append(k.consumers, consumers...)
	return nil
}

// BindProducer 绑定生产者
// 参数:
//   - producers: ...ProducerInf 生产者列表
// 返回:
//   - error 错误信息
func (k *KafkaManager) BindProducer(producers ...ProducerInf) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	
	k.producers = append(k.producers, producers...)
	return nil
}

// BindTopic 绑定主题
// 参数:
//   - topics: ...TopicInf 主题列表
// 返回:
//   - error 错误信息
func (k *KafkaManager) BindTopic(topics ...TopicInf) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	
	k.topics = append(k.topics, topics...)
	return nil
}

// Publish 发布消息
// 参数:
//   - ctx: context.Context 上下文
//   - topic: string 主题
//   - msg: []byte 消息体
// 返回:
//   - error 错误信息
func (k *KafkaManager) Publish(ctx context.Context, topic string, msg []byte) error {
	k.mu.RLock()
	defer k.mu.RUnlock()
	
	if !k.isConnected {
		// 连接断开时，保存失败消息
		if k.failedStore != nil {
			failedMsg := &Message{
				ID:        uuid.New().String(),
				Topic:     topic,
				Body:      msg,
				Timestamp: time.Now(),
				RetryCount: 0,
			}
			if err := k.failedStore.Store(failedMsg); err != nil {
				log.Printf("保存失败消息出错: %v", err)
			}
		}
		return fmt.Errorf("kafka 连接已断开")
	}
	
	message := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(msg),
	}
	
	_, _, err := k.producer.SendMessage(message)
	if err != nil {
		// 发送失败，保存失败消息
		if k.failedStore != nil {
			failedMsg := &Message{
				ID:        uuid.New().String(),
				Topic:     topic,
				Body:      msg,
				Timestamp: time.Now(),
				RetryCount: 0,
			}
			if storeErr := k.failedStore.Store(failedMsg); storeErr != nil {
				log.Printf("保存失败消息出错: %v", storeErr)
			}
		}
		return fmt.Errorf("发送消息失败: %w", err)
	}
	
	return nil
}

// Close 关闭连接
// 返回:
//   - error 错误信息
func (k *KafkaManager) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	
	k.cancel() // 取消上下文
	
	if k.reconnectTicker != nil {
		k.reconnectTicker.Stop()
	}
	
	if k.producer != nil {
		k.producer.Close()
	}
	
	if k.client != nil {
		k.client.Close()
	}
	
	k.isConnected = false
	return nil
}

// IsConnected 检查连接状态
// 返回:
//   - bool 连接状态
func (k *KafkaManager) IsConnected() bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.isConnected
}

// Reconnect 手动重连
// 返回:
//   - error 错误信息
func (k *KafkaManager) Reconnect() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	
	if k.isConnected {
		return nil // 已经连接
	}
	
	// 关闭旧连接
	if k.producer != nil {
		k.producer.Close()
	}
	if k.client != nil {
		k.client.Close()
	}
	
	// 重新连接
	return k.connect()
}

// startReconnectMonitor 启动重连监控
func (k *KafkaManager) startReconnectMonitor() {
	k.reconnectTicker = time.NewTicker(k.options.ReconnectDelay)
	defer k.reconnectTicker.Stop()
	
	for {
		select {
		case <-k.ctx.Done():
			return
		case <-k.reconnectTicker.C:
			if !k.isConnected {
				log.Printf("尝试重连 Kafka...")
				if err := k.Reconnect(); err != nil {
					log.Printf("重连失败: %v", err)
				} else {
					log.Printf("重连成功")
				}
			}
		}
	}
}

// resendFailedMessages 重发失败消息
func (k *KafkaManager) resendFailedMessages() {
	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()
	
	for {
		select {
		case <-k.ctx.Done():
			return
		case <-ticker.C:
			if !k.isConnected {
				continue
			}
			
			messages, err := k.failedStore.Retrieve()
			if err != nil {
				log.Printf("获取失败消息出错: %v", err)
				continue
			}
			
			for _, msg := range messages {
				if msg.RetryCount >= k.options.MaxRetries {
					log.Printf("消息重试次数超限，放弃重发: %s", msg.ID)
					k.failedStore.Remove(msg.ID)
					continue
				}
				
				err := k.Publish(k.ctx, msg.Topic, msg.Body)
				if err != nil {
					msg.RetryCount++
					k.failedStore.Store(msg) // 更新重试次数
					log.Printf("重发消息失败: %v", err)
				} else {
					k.failedStore.Remove(msg.ID)
					log.Printf("重发消息成功: %s", msg.ID)
				}
			}
		}
	}
}

// === KafkaConsumer 实现 ===

// ConsumerGroupHandler 实现 sarama.ConsumerGroupHandler 接口
type ConsumerGroupHandler struct {
	consumer *KafkaConsumer
}

// Setup 设置消费者组
func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup 清理消费者组
func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 消费消息
func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}
			
			// 处理消息
			if err := h.consumer.handler(session.Context(), message.Value); err != nil {
				log.Printf("处理消息失败: %v", err)
				// 如果是手动提交模式且消息处理失败，不提交偏移量
				if !h.consumer.options.EnableAutoCommit {
					// 消息处理失败，不标记消息（不提交偏移量）
					continue
				}
			} else {
				// 消息处理成功
				if !h.consumer.options.EnableAutoCommit {
					// 手动提交模式，标记消息为已处理
					session.MarkMessage(message, "")
					session.Commit() // 手动提交偏移量
				}
				// 自动提交模式下，Kafka 会自动提交偏移量，无需手动操作
			}
			
		case <-session.Context().Done():
			return nil
		}
	}
}

// Topic 获取主题
// 返回:
//   - string 主题
func (c *KafkaConsumer) Topic() string {
	return c.topic
}

// Queues 获取队列
// 返回:
//   - string 队列
func (c *KafkaConsumer) Queues() string {
	return c.queue
}

// ChannelId 获取通道ID
// 返回:
//   - string 通道ID
func (c *KafkaConsumer) ChannelId() string {
	return c.topic + ":" + c.queue
}

// BindHandler 绑定处理函数
// 参数:
//   - handler: ConsumerHandlerFunc 处理函数
func (c *KafkaConsumer) BindHandler(handler ConsumerHandlerFunc) {
	c.handler = handler
}

// Consume 消费消息
// 参数:
//   - ctx: context.Context 上下文
// 返回:
//   - error 错误信息
func (c *KafkaConsumer) Consume(ctx context.Context) error {
	if c.handler == nil {
		return fmt.Errorf("未绑定消息处理函数")
	}
	
	if !c.manager.IsConnected() {
		return fmt.Errorf("kafka 连接已断开")
	}
	
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	config.Consumer.Group.Session.Timeout = c.options.SessionTimeout
	config.Consumer.Group.Heartbeat.Interval = c.options.HeartbeatInterval
	config.Consumer.Fetch.Min = c.options.FetchMinBytes
	config.Consumer.Fetch.Max = c.options.FetchMaxBytes
	config.Consumer.MaxWaitTime = c.options.MaxWaitTime
	config.Consumer.Return.Errors = true
	
	if c.options.EnableAutoCommit {
		config.Consumer.Offsets.AutoCommit.Enable = true
	}
	
	// 设置偏移重置策略
	switch c.options.AutoOffsetReset {
	case "earliest":
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	case "latest":
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
	}
	
	var err error
	c.consumerGroup, err = sarama.NewConsumerGroupFromClient(c.options.GroupID, c.manager.client)
	if err != nil {
		return fmt.Errorf("创建消费者组失败: %w", err)
	}
	
	handler := &ConsumerGroupHandler{consumer: c}
	
	// 启动消费协程
	go func() {
		for {
			select {
			case <-ctx.Done():
				c.done <- ctx.Err()
				return
			default:
				err := c.consumerGroup.Consume(ctx, []string{c.topic}, handler)
				if err != nil {
					log.Printf("消费失败: %v", err)
					c.done <- err
					return
				}
			}
		}
	}()
	
	return nil
}

// === KafkaProducer 实现 ===

// Topic 获取主题
// 返回:
//   - string 主题
func (p *KafkaProducer) Topic() string {
	return p.topic
}

// Publish 发布消息
// 参数:
//   - ctx: context.Context 上下文
//   - msg: []byte 消息体
// 返回:
//   - error 错误信息
func (p *KafkaProducer) Publish(ctx context.Context, msg []byte) error {
	if !p.manager.IsConnected() {
		return fmt.Errorf("kafka 连接已断开")
	}
	
	message := &sarama.ProducerMessage{
		Topic: p.topic,
		Value: sarama.StringEncoder(msg),
	}
	
	// 设置分区
	if p.options.Partition >= 0 {
		message.Partition = p.options.Partition
	}
	
	// 设置消息键
	if p.options.Key != nil {
		message.Key = sarama.StringEncoder(p.options.Key)
	}
	
	// 设置消息头
	if p.options.Headers != nil {
		headers := make([]sarama.RecordHeader, 0, len(p.options.Headers))
		for key, value := range p.options.Headers {
			headers = append(headers, sarama.RecordHeader{
				Key:   []byte(key),
				Value: value,
			})
		}
		message.Headers = headers
	}
	
	_, _, err := p.manager.producer.SendMessage(message)
	if err != nil {
		return fmt.Errorf("发送消息失败: %w", err)
	}
	
	return nil
}