package mqs

import (
	"context"
	"log"
	"time"
	"github.com/hfup/sunny/types"
	"fmt"
)

// ExampleUsage 使用示例
func ExampleUsage() {
	ctx := context.Background()
	
	// === RabbitMQ 使用示例 ===
	
	// 1. 创建 RabbitMQ 管理器
	rabbitOptions := DefaultRabbitMQOptions("amqp://guest:guest@localhost:5672/")
	failedStore := GetDefaultFailedStore()
	
	rabbitManager, err := CreateMqManager(MqTypeRabbitMQ, rabbitOptions, failedStore)
	if err != nil {
		log.Printf("创建 RabbitMQ 管理器失败: %v", err)
		return
	}
	
	// 2. 启动管理器
	resultChan := make(chan types.Result[any], 1)
	go rabbitManager.Start(ctx, nil, resultChan)
	
	result := <-resultChan
	if result.ErrCode != 0 {
		log.Printf("启动 RabbitMQ 管理器失败: %s", result.Message)
		return
	}
	log.Printf("RabbitMQ 管理器启动成功: %s", result.Message)
	
	// 3. 创建消费者 (快速方法)
	// false = 手动确认，消息消费失败会重发，保证消息不丢失
	consumer := NewRabbitMQConsumer(rabbitManager.(*RabbitMqManager), "test.topic", "test_queue", false)
	
	// 4. 绑定消息处理函数
	consumer.BindHandler(func(ctx context.Context, msg []byte) error {
		log.Printf("收到消息: %s", string(msg))
		return nil
	})
	
	// 5. 开始消费
	if err := consumer.Consume(ctx); err != nil {
		log.Printf("开始消费失败: %v", err)
		return
	}
	
	// 6. 创建生产者 (快速方法)
	producer := NewRabbitMQProducer(rabbitManager.(*RabbitMqManager), "test.topic")
	
	// 7. 发送消息
	if err := producer.Publish(ctx, []byte("Hello RabbitMQ!")); err != nil {
		log.Printf("发送消息失败: %v", err)
		return
	}
	log.Printf("消息发送成功")
	
	// === Kafka 使用示例 ===
	
	// 1. 创建 Kafka 管理器
	kafkaOptions := DefaultKafkaOptions([]string{"localhost:9092"})
	
	kafkaManager, err := CreateMqManager(MqTypeKafka, kafkaOptions, failedStore)
	if err != nil {
		log.Printf("创建 Kafka 管理器失败: %v", err)
		return
	}
	
	// 2. 启动管理器
	resultChan2 := make(chan types.Result[any], 1)
	go kafkaManager.Start(ctx, nil, resultChan2)
	
	result2 := <-resultChan2
	if result2.ErrCode != 0 {
		log.Printf("启动 Kafka 管理器失败: %s", result2.Message)
		return
	}
	log.Printf("Kafka 管理器启动成功: %s", result2.Message)
	
	// 3. 创建 Kafka 消费者 (快速方法)
	// true = 自动提交偏移量，消息被消费后立即提交偏移量
	kafkaConsumer := NewKafkaConsumer(kafkaManager.(*KafkaManager), "test_topic", "test_group", true)
	
	// 4. 绑定消息处理函数
	kafkaConsumer.BindHandler(func(ctx context.Context, msg []byte) error {
		log.Printf("Kafka 收到消息: %s", string(msg))
		return nil
	})
	
	// 5. 开始消费
	if err := kafkaConsumer.Consume(ctx); err != nil {
		log.Printf("Kafka 开始消费失败: %v", err)
		return
	}
	
	// 6. 创建 Kafka 生产者 (快速方法)
	kafkaProducer := NewKafkaProducer(kafkaManager.(*KafkaManager), "test_topic")
	
	// 7. 发送消息
	if err := kafkaProducer.Publish(ctx, []byte("Hello Kafka!")); err != nil {
		log.Printf("Kafka 发送消息失败: %v", err)
		return
	}
	log.Printf("Kafka 消息发送成功")
	
	// 等待一段时间处理消息
	time.Sleep(5 * time.Second)
	
	// 8. 关闭管理器
	rabbitManager.Close()
	kafkaManager.Close()
	
	log.Printf("示例执行完成")
}

// ExampleRabbitMQOnly 仅使用 RabbitMQ 的简化示例
func ExampleRabbitMQOnly() {
	ctx := context.Background()
	
	// 创建配置
	options := &RabbitMQOptions{
		URL:     "amqp://guest:guest@localhost:5672/",
		Durable: true,
	}
	
	// 创建管理器
	manager := NewRabbitMqManager(options, GetDefaultFailedStore())
	
	// 启动
	resultChan := make(chan types.Result[any], 1)
	go manager.Start(ctx, nil, resultChan)
	<-resultChan
	
	// 创建消费者
	consumerOpts := &RabbitMQConsumerOptions{
		RabbitMQQueueOptions: RabbitMQQueueOptions{
			Durable: true,
		},
		AutoAck: false,
	}
	
	consumer, _ := manager.CreateConsumer("order.created", "order_queue", consumerOpts)
	consumer.BindHandler(func(ctx context.Context, msg []byte) error {
		log.Printf("处理订单: %s", string(msg))
		return nil
	})
	consumer.Consume(ctx)
	
	// 创建生产者发送消息
	producerOpts := &RabbitMQProducerOptions{}
	producer, _ := manager.CreateProducer("order.created", producerOpts)
	producer.Publish(ctx, []byte(`{"order_id": "12345", "amount": 100}`))
	
	// 通过管理器直接发送消息
	manager.Publish(ctx, "user.registered", []byte(`{"user_id": "user123"}`))
	
	time.Sleep(2 * time.Second)
	manager.Close()
}

// ExampleKafkaOnly 仅使用 Kafka 的简化示例
func ExampleKafkaOnly() {
	ctx := context.Background()
	
	// 创建配置
	options := &KafkaOptions{
		Brokers: []string{"localhost:9092"},
	}
	
	// 创建管理器
	manager := NewKafkaManager(options, GetDefaultFailedStore())
	
	// 启动
	resultChan := make(chan types.Result[any], 1)
	go manager.Start(ctx, nil, resultChan)
	<-resultChan
	
	// 创建消费者
	consumerOpts := &KafkaConsumerOptions{
		GroupID:         "my_group",
		AutoOffsetReset: "earliest",
	}
	
	consumer, _ := manager.CreateConsumer("events", "0", consumerOpts)
	consumer.BindHandler(func(ctx context.Context, msg []byte) error {
		log.Printf("处理事件: %s", string(msg))
		return nil
	})
	consumer.Consume(ctx)
	
	// 创建生产者发送消息
	producerOpts := &KafkaProducerOptions{
		Acks: "all",
	}
	producer, _ := manager.CreateProducer("events", producerOpts)
	producer.Publish(ctx, []byte(`{"event": "user_login", "user_id": "123"}`))
	
	// 通过管理器直接发送消息
	manager.Publish(ctx, "notifications", []byte(`{"type": "email", "to": "user@example.com"}`))
	
	time.Sleep(2 * time.Second)
	manager.Close()
}

// ExampleQuickStart 快速开始示例 - 展示新的 New 函数
func ExampleQuickStart() {
	ctx := context.Background()
	
	// === RabbitMQ 快速开始 ===
	
	// 1. 创建管理器
	options := DefaultRabbitMQOptions("amqp://localhost:5672")
	manager := NewRabbitMqManager(options, GetDefaultFailedStore())
	
	// 2. 启动管理器
	resultChan := make(chan types.Result[any], 1)
	go manager.Start(ctx, nil, resultChan)
	<-resultChan
	
	// 3. 快速创建消费者和生产者
	// false = 手动确认，确保消息处理成功
	consumer := NewRabbitMQConsumer(manager, "order.created", "order_service", false)
	producer := NewRabbitMQProducer(manager, "order.created")
	
	// 4. 绑定处理函数并开始消费
	consumer.BindHandler(func(ctx context.Context, msg []byte) error {
		log.Printf("处理订单: %s", string(msg))
		return nil
	})
	consumer.Consume(ctx)
	
	// 5. 发送消息
	producer.Publish(ctx, []byte(`{"order_id": "12345", "status": "created"}`))
	
	// === Kafka 快速开始 ===
	
	kafkaOptions := DefaultKafkaOptions([]string{"localhost:9092"})
	kafkaManager := NewKafkaManager(kafkaOptions, GetDefaultFailedStore())
	
	resultChan2 := make(chan types.Result[any], 1)
	go kafkaManager.Start(ctx, nil, resultChan2)
	<-resultChan2
	
	// 快速创建
	// false = 手动提交偏移量，确保消息处理成功后才提交
	kafkaConsumer := NewKafkaConsumer(kafkaManager, "events", "event_service", false)
	kafkaProducer := NewKafkaProducer(kafkaManager, "events")
	
	kafkaConsumer.BindHandler(func(ctx context.Context, msg []byte) error {
		log.Printf("处理事件: %s", string(msg))
		return nil
	})
	kafkaConsumer.Consume(ctx)
	
	kafkaProducer.Publish(ctx, []byte(`{"event": "user_registered", "user_id": "user123"}`))
	
	time.Sleep(2 * time.Second)
	manager.Close()
	kafkaManager.Close()
}

// ExampleAutoAck 展示自动确认和手动确认的区别
func ExampleAutoAck() {
	ctx := context.Background()
	
	options := DefaultRabbitMQOptions("amqp://localhost:5672")
	manager := NewRabbitMqManager(options, GetDefaultFailedStore())
	
	resultChan := make(chan types.Result[any], 1)
	go manager.Start(ctx, nil, resultChan)
	<-resultChan
	
	// 手动确认消费者 - 消息处理失败会重发
	manualConsumer := NewRabbitMQConsumer(manager, "test.manual", "manual_queue", false)
	manualConsumer.BindHandler(func(ctx context.Context, msg []byte) error {
		log.Printf("手动确认消费者收到消息: %s", string(msg))
		// 如果返回错误，消息会被重发
		return nil // 成功处理，消息被确认
	})
	manualConsumer.Consume(ctx)
	
	// 自动确认消费者 - 消息一旦投递就被确认，不管是否处理成功
	autoConsumer := NewRabbitMQConsumer(manager, "test.auto", "auto_queue", true)
	autoConsumer.BindHandler(func(ctx context.Context, msg []byte) error {
		log.Printf("自动确认消费者收到消息: %s", string(msg))
		// 即使返回错误，消息也不会重发，因为已经自动确认了
		return fmt.Errorf("处理失败，但消息不会重发")
	})
	autoConsumer.Consume(ctx)
	
	// 发送测试消息
	producer := NewRabbitMQProducer(manager, "test.manual")
	producer.Publish(ctx, []byte("手动确认测试消息"))
	
	producer2 := NewRabbitMQProducer(manager, "test.auto")
	producer2.Publish(ctx, []byte("自动确认测试消息"))
	
	time.Sleep(3 * time.Second)
	manager.Close()
}

// ExampleFileStore 展示文件存储的使用
func ExampleFileStore() {
	ctx := context.Background()
	
	// 使用文件存储来保存失败消息
	fileStore, err := GetFileFailedStore("./data/rabbitmq_failed_messages.json")
	if err != nil {
		log.Printf("创建文件存储失败: %v", err)
		return
	}
	
	// 创建使用文件存储的 RabbitMQ 管理器
	options := DefaultRabbitMQOptions("amqp://localhost:5672")
	manager := NewRabbitMqManager(options, fileStore)
	
	resultChan := make(chan types.Result[any], 1)
	go manager.Start(ctx, nil, resultChan)
	result := <-resultChan
	
	if result.ErrCode != 0 {
		log.Printf("启动失败: %s", result.Message)
		return
	}
	
	log.Printf("RabbitMQ 管理器启动成功，使用文件存储: %s", result.Message)
	
	// 创建生产者和消费者
	producer := NewRabbitMQProducer(manager, "test.file.store")
	consumer := NewRabbitMQConsumer(manager, "test.file.store", "file_queue", false)
	
	// 绑定消费者处理函数
	consumer.BindHandler(func(ctx context.Context, msg []byte) error {
		log.Printf("处理消息: %s", string(msg))
		return nil
	})
	
	// 启动消费者
	if err := consumer.Consume(ctx); err != nil {
		log.Printf("启动消费者失败: %v", err)
		return
	}
	
	// 发送一些测试消息
	for i := 0; i < 5; i++ {
		msg := fmt.Sprintf(`{"id": %d, "content": "测试消息 %d"}`, i, i)
		if err := producer.Publish(ctx, []byte(msg)); err != nil {
			log.Printf("发送消息失败: %v", err)
		} else {
			log.Printf("发送消息成功: %s", msg)
		}
	}
	
	// 展示文件存储的额外功能
	if fileStore, ok := fileStore.(*FileFailedMessageStore); ok {
		// 获取失败消息数量
		count, err := fileStore.Count()
		if err != nil {
			log.Printf("获取失败消息数量失败: %v", err)
		} else {
			log.Printf("当前失败消息数量: %d", count)
		}
		
		// 展示文件路径
		log.Printf("失败消息存储文件路径: %s", fileStore.GetFilePath())
		
		// 创建备份
		backupPath := "./data/backup_failed_messages.json"
		if err := fileStore.Backup(backupPath); err != nil {
			log.Printf("创建备份失败: %v", err)
		} else {
			log.Printf("备份创建成功: %s", backupPath)
		}
	}
	
	time.Sleep(2 * time.Second)
	manager.Close()
	
	log.Printf("示例完成，失败消息已持久化到文件")
}

// ExampleCompareStores 对比内存存储和文件存储
func ExampleCompareStores() {
	
	log.Printf("=== 内存存储 vs 文件存储对比 ===")
	
	// 1. 内存存储示例
	log.Printf("1. 内存存储（程序重启后丢失）")
	memoryStore := GetDefaultFailedStore()
	
	// 模拟添加失败消息
	testMsg := &Message{
		ID:    "test-001",
		Topic: "test.memory",
		Body:  []byte("内存存储测试消息"),
	}
	
	if err := memoryStore.Store(testMsg); err != nil {
		log.Printf("内存存储失败: %v", err)
	} else {
		log.Printf("内存存储成功")
	}
	
	messages, _ := memoryStore.Retrieve()
	log.Printf("内存中的失败消息数量: %d", len(messages))
	
	// 2. 文件存储示例
	log.Printf("2. 文件存储（持久化，程序重启后保留）")
	fileStore, err := GetFileFailedStore("./data/compare_test.json")
	if err != nil {
		log.Printf("创建文件存储失败: %v", err)
		return
	}
	
	testMsg2 := &Message{
		ID:    "test-002",
		Topic: "test.file",
		Body:  []byte("文件存储测试消息"),
	}
	
	if err := fileStore.Store(testMsg2); err != nil {
		log.Printf("文件存储失败: %v", err)
	} else {
		log.Printf("文件存储成功")
	}
	
	messages2, _ := fileStore.Retrieve()
	log.Printf("文件中的失败消息数量: %d", len(messages2))
	
	// 3. 性能对比（简单测试）
	log.Printf("3. 性能对比")
	
	// 内存存储性能测试
	start := time.Now()
	for i := 0; i < 1000; i++ {
		msg := &Message{
			ID:    fmt.Sprintf("memory-%d", i),
			Topic: "perf.test",
			Body:  []byte(fmt.Sprintf("性能测试消息 %d", i)),
		}
		memoryStore.Store(msg)
	}
	memoryDuration := time.Since(start)
	log.Printf("内存存储 1000 条消息耗时: %v", memoryDuration)
	
	// 文件存储性能测试
	start = time.Now()
	for i := 0; i < 100; i++ { // 文件存储较慢，测试较少数量
		msg := &Message{
			ID:    fmt.Sprintf("file-%d", i),
			Topic: "perf.test",
			Body:  []byte(fmt.Sprintf("性能测试消息 %d", i)),
		}
		fileStore.Store(msg)
	}
	fileDuration := time.Since(start)
	log.Printf("文件存储 100 条消息耗时: %v", fileDuration)
	
	// 清理测试数据
	memoryStore.Clear()
	fileStore.Clear()
	
	log.Printf("=== 对比结论 ===")
	log.Printf("内存存储: 速度快，但程序重启后丢失")
	log.Printf("文件存储: 速度较慢，但数据持久化")
	log.Printf("建议: 生产环境使用文件存储，开发环境可使用内存存储")
}

// ExampleKafkaAutoCommit 展示 Kafka 自动提交和手动提交的区别
func ExampleKafkaAutoCommit() {
	ctx := context.Background()
	
	options := DefaultKafkaOptions([]string{"localhost:9092"})
	manager := NewKafkaManager(options, GetDefaultFailedStore())
	
	resultChan := make(chan types.Result[any], 1)
	go manager.Start(ctx, nil, resultChan)
	<-resultChan
	
	// 手动提交消费者 - 消息处理失败不会提交偏移量，消息会被重新消费
	manualConsumer := NewKafkaConsumer(manager, "test.manual", "manual_group", false)
	manualConsumer.BindHandler(func(ctx context.Context, msg []byte) error {
		log.Printf("手动提交消费者收到消息: %s", string(msg))
		// 如果返回错误，偏移量不会提交，消息会被重新消费
		return nil // 成功处理，偏移量被提交
	})
	manualConsumer.Consume(ctx)
	
	// 自动提交消费者 - 消息一旦被读取就提交偏移量，不管是否处理成功
	autoConsumer := NewKafkaConsumer(manager, "test.auto", "auto_group", true)
	autoConsumer.BindHandler(func(ctx context.Context, msg []byte) error {
		log.Printf("自动提交消费者收到消息: %s", string(msg))
		// 即使返回错误，偏移量也会自动提交，消息不会重新消费
		return fmt.Errorf("处理失败，但偏移量已自动提交")
	})
	autoConsumer.Consume(ctx)
	
	// 发送测试消息
	producer := NewKafkaProducer(manager, "test.manual")
	producer.Publish(ctx, []byte("手动提交测试消息"))
	
	producer2 := NewKafkaProducer(manager, "test.auto")
	producer2.Publish(ctx, []byte("自动提交测试消息"))
	
	time.Sleep(3 * time.Second)
	manager.Close()
}