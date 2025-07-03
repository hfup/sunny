package mqs

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"github.com/google/uuid"
	"github.com/streadway/amqp"
	"github.com/hfup/sunny/types"
)

const (
	DefaultExchange     = "default"        // 默认交换机名称
	DefaultExchangeType = "topic"        // 默认交换机类型
)

// RabbitMQ 配置选项
type RabbitMQOptions struct {
	URL             string        `json:"url"`              // 连接地址
	Durable         bool          `json:"durable"`          // 是否持久化
	AutoDelete      bool          `json:"auto_delete"`      // 是否自动删除
	Internal        bool          `json:"internal"`         // 是否内部交换机
	NoWait          bool          `json:"no_wait"`          // 是否等待确认
	MaxRetries      int           `json:"max_retries"`      // 最大重试次数
	RetryInterval   time.Duration `json:"retry_interval"`   // 重试间隔
	ReconnectDelay  time.Duration `json:"reconnect_delay"`  // 重连延迟
	ChannelPoolSize int           `json:"channel_pool_size"` // Channel 池大小
}

// RabbitMQ 队列配置选项
type RabbitMQQueueOptions struct {
	Durable    bool            `json:"durable"`     // 持久化
	AutoDelete bool            `json:"auto_delete"` // 自动删除
	Exclusive  bool            `json:"exclusive"`   // 排他性
	NoWait     bool            `json:"no_wait"`     // 不等待
	Args       map[string]any  `json:"args"`        // 其他参数
}

// RabbitMQ 消费者配置选项
type RabbitMQConsumerOptions struct {
	RabbitMQQueueOptions
	AutoAck   bool   `json:"auto_ack"`   // 自动确认
	Consumer  string `json:"consumer"`   // 消费者标签
	NoLocal   bool   `json:"no_local"`   // 不接收本地消息
	Exclusive bool   `json:"exclusive"`  // 排他性
}

// RabbitMQ 生产者配置选项
type RabbitMQProducerOptions struct {
	Mandatory bool `json:"mandatory"` // 强制路由
	Immediate bool `json:"immediate"` // 立即投递
}

// ChannelPool Channel 池管理
type ChannelPool struct {
	conn     *amqp.Connection // 连接
	channels chan *amqp.Channel // Channel 池
	maxSize  int              // 池最大大小
	mu       sync.Mutex       // 锁
	closed   bool             // 是否已关闭
}

// NewChannelPool 创建 Channel 池
// 参数:
//   - conn: *amqp.Connection RabbitMQ 连接
//   - maxSize: int 池的最大大小
// 返回:
//   - *ChannelPool Channel 池
func NewChannelPool(conn *amqp.Connection, maxSize int) *ChannelPool {
	return &ChannelPool{
		conn:     conn,
		channels: make(chan *amqp.Channel, maxSize),
		maxSize:  maxSize,
	}
}

// Get 从池中获取 Channel
// 返回:
//   - *amqp.Channel 通道
//   - error 错误信息
func (p *ChannelPool) Get() (*amqp.Channel, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.closed {
		return nil, fmt.Errorf("channel 池已关闭")
	}
	
	// 尝试从池中获取
	select {
	case ch := <-p.channels:
		// 检查 Channel 是否还可用，如果不可用则创建新的
		return ch, nil
	default:
		// 池为空，创建新的 Channel
		return p.createChannel()
	}
}

// Put 将 Channel 放回池中
// 参数:
//   - ch: *amqp.Channel 通道
func (p *ChannelPool) Put(ch *amqp.Channel) {
	if ch == nil {
		return
	}
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.closed {
		ch.Close()
		return
	}
	
	// 尝试放回池中
	select {
	case p.channels <- ch:
		// 成功放回池中
	default:
		// 池已满，关闭 Channel
		ch.Close()
	}
}

// createChannel 创建新的 Channel
// 返回:
//   - *amqp.Channel 通道
//   - error 错误信息
func (p *ChannelPool) createChannel() (*amqp.Channel, error) {
	return p.conn.Channel()
}

// Close 关闭 Channel 池
func (p *ChannelPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.closed {
		return
	}
	
	p.closed = true
	close(p.channels)
	
	// 关闭池中所有 Channel
	for ch := range p.channels {
		ch.Close()
	}
}

// RabbitMqManager 是 RabbitMQ 消息队列管理器
// 实现了 MqManagerInf 接口
// 交换机 全局使用一个，交换机类型 topic，通过 router_key 来确定 不同 主题
// 队列申明 ，消费者声明 ，生产者声明 参数 解释 放在 字段后面 例如  xx1 // 持久化
// 实现 重连机制
// 当 发送消息失败的时候，需要持久化 发送失败的消息，等下次连接成功的时候 发送
type RabbitMqManager struct {
	options         *RabbitMQOptions     // 配置选项
	conn            *amqp.Connection     // 连接
	channelPool     *ChannelPool         // Channel 池
	consumers       []ConsumerInf        // 消费者列表
	producers       []ProducerInf        // 生产者列表
	topics          []TopicInf           // 主题列表
	failedStore     FailedMessageStore   // 失败消息存储
	isConnected     bool                 // 连接状态
	reconnectTicker *time.Ticker         // 重连定时器
	mu              sync.RWMutex         // 读写锁
	ctx             context.Context      // 上下文
	cancel          context.CancelFunc   // 取消函数
}

// RabbitMQConsumer RabbitMQ 消费者
type RabbitMQConsumer struct {
	topic    string                  // 主题
	queue    string                  // 队列
	options  *RabbitMQConsumerOptions // 配置选项
	handler  ConsumerHandlerFunc     // 处理函数
	manager  *RabbitMqManager        // 管理器引用
	channel  *amqp.Channel           // 专用通道
	delivery <-chan amqp.Delivery    // 消息通道
	done     chan error              // 完成通道
}

// RabbitMQProducer RabbitMQ 生产者
type RabbitMQProducer struct {
	topic   string                   // 主题
	options *RabbitMQProducerOptions // 配置选项
	manager *RabbitMqManager         // 管理器引用
}

// NewRabbitMQConsumer 快速创建 RabbitMQ 消费者
// 参数:
//   - manager: *RabbitMqManager 管理器
//   - topic: string 主题
//   - queue: string 队列
//   - autoAsk: 消息自动确认，为true时 消息一旦投递就被认为已消费，不会重发，即使消费失败，
//           为false时 需要手动确认，消息消费失败后，会重发，直到消费成功为止
// 返回:
//   - *RabbitMQConsumer 消费者实例
func NewRabbitMQConsumer(manager *RabbitMqManager, topic, queue string,autoAsk bool) *RabbitMQConsumer {
	return &RabbitMQConsumer{
		topic:   topic,
		queue:   queue,
		options: DefaultRabbitMQConsumerOptions(autoAsk),
		manager: manager,
		done:    make(chan error),
	}
}

// NewRabbitMQConsumerWithOptions 使用自定义配置创建 RabbitMQ 消费者
// 参数:
//   - manager: *RabbitMqManager 管理器
//   - topic: string 主题
//   - queue: string 队列
//   - options: *RabbitMQConsumerOptions 配置选项
// 返回:
//   - *RabbitMQConsumer 消费者实例
func NewRabbitMQConsumerWithOptions(manager *RabbitMqManager, topic, queue string, options *RabbitMQConsumerOptions) *RabbitMQConsumer {
	return &RabbitMQConsumer{
		topic:   topic,
		queue:   queue,
		options: options,
		manager: manager,
		done:    make(chan error),
	}
}

// NewRabbitMQProducer 快速创建 RabbitMQ 生产者
// 参数:
//   - manager: *RabbitMqManager 管理器
//   - topic: string 主题
// 返回:
//   - *RabbitMQProducer 生产者实例
func NewRabbitMQProducer(manager *RabbitMqManager, topic string) *RabbitMQProducer {
	return &RabbitMQProducer{
		topic:   topic,
		options: DefaultRabbitMQProducerOptions(),
		manager: manager,
	}
}

// NewRabbitMQProducerWithOptions 使用自定义配置创建 RabbitMQ 生产者
// 参数:
//   - manager: *RabbitMqManager 管理器
//   - topic: string 主题
//   - options: *RabbitMQProducerOptions 配置选项
// 返回:
//   - *RabbitMQProducer 生产者实例
func NewRabbitMQProducerWithOptions(manager *RabbitMqManager, topic string, options *RabbitMQProducerOptions) *RabbitMQProducer {
	return &RabbitMQProducer{
		topic:   topic,
		options: options,
		manager: manager,
	}
}

// NewRabbitMqManager 创建 RabbitMQ 管理器
// 参数:
//   - options: *RabbitMQOptions RabbitMQ 配置选项
//   - failedStore: FailedMessageStore 失败消息存储(可选)
// 返回:
//   - *RabbitMqManager 管理器实例
func NewRabbitMqManager(options *RabbitMQOptions, failedStore FailedMessageStore) *RabbitMqManager {
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
	if options.ChannelPoolSize <= 0 {
		options.ChannelPoolSize = 10 // 默认 Channel 池大小为 10
	}
	
	return &RabbitMqManager{
		options:     options,
		consumers:   make([]ConsumerInf, 0),
		producers:   make([]ProducerInf, 0),
		topics:      make([]TopicInf, 0),
		failedStore: failedStore,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start 启动 RabbitMQ 管理器
// 参数:
//   - ctx: context.Context 上下文
//   - args: any 启动参数
//   - resultChan: chan<- types.Result[any] 结果通道
func (r *RabbitMqManager) Start(ctx context.Context, args any, resultChan chan<- types.Result[any]) {
	if err := r.connect(); err != nil {
		resultChan <- types.Result[any]{
			ErrCode: 1,
			Message: fmt.Sprintf("RabbitMQ 连接失败: %v", err),
		}
		return
	}
	
	// 启动重连监控
	go r.startReconnectMonitor()
	
	// 重发失败消息
	if r.failedStore != nil {
		go r.resendFailedMessages()
	}
	
	resultChan <- types.Result[any]{
		ErrCode: 0,
		Message: "RabbitMQ 管理器启动成功",
	}
}

// connect 建立连接
// 返回:
//   - error 错误信息
func (r *RabbitMqManager) connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	var err error
	r.conn, err = amqp.Dial(r.options.URL)
	if err != nil {
		return fmt.Errorf("连接 RabbitMQ 失败: %w", err)
	}
	
	// 创建 Channel 池
	r.channelPool = NewChannelPool(r.conn, r.options.ChannelPoolSize)
	
	// 获取一个 Channel 用于声明交换机
	setupCh, err := r.channelPool.Get()
	if err != nil {
		r.conn.Close()
		return fmt.Errorf("创建通道失败: %w", err)
	}
	defer r.channelPool.Put(setupCh)
	
	// 声明交换机
	err = setupCh.ExchangeDeclare(
		DefaultExchange,        // 交换机名称
		DefaultExchangeType,    // 交换机类型
		r.options.Durable,      // 持久化
		r.options.AutoDelete,   // 自动删除
		false,                  // 内部交换机
		r.options.NoWait,       // 不等待
		nil,                    // 参数
	)
	if err != nil {
		r.channelPool.Close()
		r.conn.Close()
		return fmt.Errorf("声明交换机失败: %w", err)
	}
	
	r.isConnected = true
	log.Printf("RabbitMQ 连接成功: %s", r.options.URL)
	return nil
}

// IsErrorStop 是否错误停止
// 返回:
//   - bool 是否错误停止
func (r *RabbitMqManager) IsErrorStop() bool {
	return true // 连接失败时停止服务
}

// ServiceName 服务名称
// 返回:
//   - string 服务名称
func (r *RabbitMqManager) ServiceName() string {
	return "RabbitMQ Manager"
}

// CreateConsumer 创建消费者
// 参数:
//   - topic: string 主题
//   - queue: string 队列
//   - opt: any 配置选项
// 返回:
//   - ConsumerInf 消费者接口
//   - error 错误信息
func (r *RabbitMqManager) CreateConsumer(topic, queue string, opt any) (ConsumerInf, error) {
	options, ok := opt.(*RabbitMQConsumerOptions)
	if !ok {
		return nil, fmt.Errorf("配置选项类型错误，期望 *RabbitMQConsumerOptions")
	}
	
	consumer := &RabbitMQConsumer{
		topic:   topic,
		queue:   queue,
		options: options,
		manager: r,
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
func (r *RabbitMqManager) CreateProducer(topic string, opt any) (ProducerInf, error) {
	options, ok := opt.(*RabbitMQProducerOptions)
	if !ok {
		return nil, fmt.Errorf("配置选项类型错误，期望 *RabbitMQProducerOptions")
	}
	
	producer := &RabbitMQProducer{
		topic:   topic,
		options: options,
		manager: r,
	}
	
	return producer, nil
}

// BindConsumer 绑定消费者
// 参数:
//   - consumers: ...ConsumerInf 消费者列表
// 返回:
//   - error 错误信息
func (r *RabbitMqManager) BindConsumer(consumers ...ConsumerInf) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.consumers = append(r.consumers, consumers...)
	return nil
}

// BindProducer 绑定生产者
// 参数:
//   - producers: ...ProducerInf 生产者列表
// 返回:
//   - error 错误信息
func (r *RabbitMqManager) BindProducer(producers ...ProducerInf) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.producers = append(r.producers, producers...)
	return nil
}

// BindTopic 绑定主题
// 参数:
//   - topics: ...TopicInf 主题列表
// 返回:
//   - error 错误信息
func (r *RabbitMqManager) BindTopic(topics ...TopicInf) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.topics = append(r.topics, topics...)
	return nil
}

// Publish 发布消息
// 参数:
//   - ctx: context.Context 上下文
//   - topic: string 主题
//   - msg: []byte 消息体
// 返回:
//   - error 错误信息
func (r *RabbitMqManager) Publish(ctx context.Context, topic string, msg []byte) error {
	r.mu.RLock()
	isConnected := r.isConnected
	channelPool := r.channelPool
	r.mu.RUnlock()
	
	if !isConnected {
		// 连接断开时，保存失败消息
		if r.failedStore != nil {
			failedMsg := &Message{
				ID:        uuid.New().String(),
				Topic:     topic,
				Body:      msg,
				Timestamp: time.Now(),
				RetryCount: 0,
			}
			if err := r.failedStore.Store(failedMsg); err != nil {
				log.Printf("保存失败消息出错: %v", err)
			}
		}
		return fmt.Errorf("RabbitMQ 连接已断开")
	}
	
	// 从池中获取 Channel
	ch, err := channelPool.Get()
	if err != nil {
		return fmt.Errorf("获取 channel 失败: %w", err)
	}
	defer channelPool.Put(ch)
	
	// 发布消息
	err = ch.Publish(
		DefaultExchange,    // 交换机
		topic,              // 路由键
		false,              // mandatory
		false,              // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msg,
			Timestamp:   time.Now(),
		},
	)
	
	if err != nil {
		// 发送失败，保存失败消息
		if r.failedStore != nil {
			failedMsg := &Message{
				ID:        uuid.New().String(),
				Topic:     topic,
				Body:      msg,
				Timestamp: time.Now(),
				RetryCount: 0,
			}
			if storeErr := r.failedStore.Store(failedMsg); storeErr != nil {
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
func (r *RabbitMqManager) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.cancel() // 取消上下文
	
	if r.reconnectTicker != nil {
		r.reconnectTicker.Stop()
	}
	
	if r.channelPool != nil {
		r.channelPool.Close()
	}
	
	if r.conn != nil {
		r.conn.Close()
	}
	
	r.isConnected = false
	return nil
}

// IsConnected 检查连接状态
// 返回:
//   - bool 连接状态
func (r *RabbitMqManager) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isConnected
}

// Reconnect 手动重连
// 返回:
//   - error 错误信息
func (r *RabbitMqManager) Reconnect() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.isConnected {
		return nil // 已经连接
	}
	
	// 关闭旧连接
	if r.channelPool != nil {
		r.channelPool.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
	
	// 重新连接
	return r.connect()
}

// startReconnectMonitor 启动重连监控
func (r *RabbitMqManager) startReconnectMonitor() {
	r.reconnectTicker = time.NewTicker(r.options.ReconnectDelay)
	defer r.reconnectTicker.Stop()
	
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-r.reconnectTicker.C:
			if !r.isConnected {
				log.Printf("尝试重连 RabbitMQ...")
				if err := r.Reconnect(); err != nil {
					log.Printf("重连失败: %v", err)
				} else {
					log.Printf("重连成功")
				}
			}
		}
	}
}

// resendFailedMessages 重发失败消息
func (r *RabbitMqManager) resendFailedMessages() {
	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()
	
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			if !r.isConnected {
				continue
			}
			
			messages, err := r.failedStore.Retrieve()
			if err != nil {
				log.Printf("获取失败消息出错: %v", err)
				continue
			}
			
			for _, msg := range messages {
				if msg.RetryCount >= r.options.MaxRetries {
					log.Printf("消息重试次数超限，放弃重发: %s", msg.ID)
					r.failedStore.Remove(msg.ID)
					continue
				}
				
				err := r.Publish(r.ctx, msg.Topic, msg.Body)
				if err != nil {
					msg.RetryCount++
					r.failedStore.Store(msg) // 更新重试次数
					log.Printf("重发消息失败: %v", err)
				} else {
					r.failedStore.Remove(msg.ID)
					log.Printf("重发消息成功: %s", msg.ID)
				}
			}
		}
	}
}

// === RabbitMQConsumer 实现 ===

// Topic 获取主题
// 返回:
//   - string 主题
func (c *RabbitMQConsumer) Topic() string {
	return c.topic
}

// Queues 获取队列
// 返回:
//   - string 队列
func (c *RabbitMQConsumer) Queues() string {
	return c.queue
}

// ChannelId 获取通道ID
// 返回:
//   - string 通道ID
func (c *RabbitMQConsumer) ChannelId() string {
	return c.topic + ":" + c.queue
}

// BindHandler 绑定处理函数
// 参数:
//   - handler: ConsumerHandlerFunc 处理函数
func (c *RabbitMQConsumer) BindHandler(handler ConsumerHandlerFunc) {
	c.handler = handler
}

// Consume 消费消息
// 参数:
//   - ctx: context.Context 上下文
// 返回:
//   - error 错误信息
func (c *RabbitMQConsumer) Consume(ctx context.Context) error {
	if c.handler == nil {
		return fmt.Errorf("未绑定消息处理函数")
	}
	
	if !c.manager.IsConnected() {
		return fmt.Errorf("RabbitMQ 连接已断开")
	}
	
	// 为消费者获取专用 Channel
	var err error
	c.channel, err = c.manager.channelPool.Get()
	if err != nil {
		return fmt.Errorf("获取消费者 channel 失败: %w", err)
	}
	
	// 声明队列
	queue, err := c.channel.QueueDeclare(
		c.queue,                    // 队列名称
		c.options.Durable,          // 持久化
		c.options.AutoDelete,       // 自动删除
		c.options.Exclusive,        // 排他性
		c.options.NoWait,           // 不等待
		c.options.Args,             // 参数
	)
	if err != nil {
		c.manager.channelPool.Put(c.channel)
		return fmt.Errorf("声明队列失败: %w", err)
	}
	
	// 绑定队列到交换机
	err = c.channel.QueueBind(
		queue.Name,         // 队列名称
		c.topic,            // 路由键
		DefaultExchange,    // 交换机
		c.options.NoWait,   // 不等待
		nil,                // 参数
	)
	if err != nil {
		c.manager.channelPool.Put(c.channel)
		return fmt.Errorf("绑定队列失败: %w", err)
	}
	
	// 开始消费
	c.delivery, err = c.channel.Consume(
		queue.Name,           // 队列名称
		c.options.Consumer,   // 消费者标签
		c.options.AutoAck,    // 自动确认
		c.options.Exclusive,  // 排他性
		c.options.NoLocal,    // 不接收本地消息
		c.options.NoWait,     // 不等待
		nil,                  // 参数
	)
	if err != nil {
		c.manager.channelPool.Put(c.channel)
		return fmt.Errorf("开始消费失败: %w", err)
	}
	
	// 启动消费协程
	go c.consumeLoop(ctx)
	
	return nil
}

// consumeLoop 消费循环
// 参数:
//   - ctx: context.Context 上下文
func (c *RabbitMQConsumer) consumeLoop(ctx context.Context) {
	defer func() {
		// 消费结束时释放 Channel
		if c.channel != nil {
			c.manager.channelPool.Put(c.channel)
		}
	}()
	
	for {
		select {
		case <-ctx.Done():
			c.done <- ctx.Err()
			return
		case delivery, ok := <-c.delivery:
			if !ok {
				c.done <- fmt.Errorf("消费通道已关闭")
				return
			}
			
			// 处理消息
			if err := c.handler(ctx, delivery.Body); err != nil {
				log.Printf("处理消息失败: %v", err)
				if !c.options.AutoAck {
					delivery.Nack(false, true) // 拒绝并重新排队
				}
			} else {
				if !c.options.AutoAck {
					delivery.Ack(false) // 确认消息
				}
			}
		}
	}
}

// === RabbitMQProducer 实现 ===

// Topic 获取主题
// 返回:
//   - string 主题
func (p *RabbitMQProducer) Topic() string {
	return p.topic
}

// Publish 发布消息
// 参数:
//   - ctx: context.Context 上下文
//   - msg: []byte 消息体
// 返回:
//   - error 错误信息
func (p *RabbitMQProducer) Publish(ctx context.Context, msg []byte) error {
	if !p.manager.IsConnected() {
		return fmt.Errorf("RabbitMQ 连接已断开")
	}
	
	// 从池中获取 Channel
	ch, err := p.manager.channelPool.Get()
	if err != nil {
		return fmt.Errorf("获取 channel 失败: %w", err)
	}
	defer p.manager.channelPool.Put(ch)
	
	err = ch.Publish(
		DefaultExchange,     // 交换机
		p.topic,             // 路由键
		p.options.Mandatory, // 强制路由
		p.options.Immediate, // 立即投递
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msg,
			Timestamp:   time.Now(),
		},
	)
	
	if err != nil {
		return fmt.Errorf("发送消息失败: %w", err)
	}
	
	return nil
}