package sunny

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"

	"github.com/hfup/sunny/components/auths"
	"github.com/hfup/sunny/components/databases"
	"github.com/hfup/sunny/components/mqs"
	"github.com/hfup/sunny/types"
	"github.com/hfup/sunny/utils"
	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

	"google.golang.org/grpc"
	"gorm.io/gorm"

	"fmt"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)
	
var (
	sunnyShared *Sunny
	onceInitSunnyShared sync.Once
)

func GetApp() *Sunny{
	if sunnyShared == nil {
		onceInitSunnyShared.Do(func() {
			sunnyShared = &Sunny{
				Engine: gin.Default(),
				roles: make(map[string]RoleInf),
				groups: make(map[string]GroupInf),
				pathRoles: make(map[string]string),
				multiRoleHandlers: make(map[string]MultiRoleHandler),
				rolesBeforeHandlers: make(map[string][]ActionHandlerWithOrder),
				rolesAfterHandlers: make(map[string][]ActionHandlerWithOrder),
				subServices: make([]types.SubServiceInf,0),
				syncRunAbles: make([]types.RunAbleInf,0),
				asyncRunAbles: make([]types.RunAbleInf,0),
				grpcServices: make([]types.RegisterGrpcServiceInf,0),
				grpcServerInterceptorHandler: nil,
			}
		})
	}
	return sunnyShared
}


type MultiRoleHandler func(groupLabel, actionLabel string) string

// Sunny 主服务
type Sunny struct {
	*gin.Engine
	serviceMark string // 服务标识 用于区分不同的服务
	configPath string // 配置文件路径
	activeEnv string // 激活环境 dev,test prod
	runPath string // 运行路径
	config *types.Config // 配置

	roles map[string]RoleInf // 角色
	groups map[string]GroupInf // 组

	pathRoles map[string]string // 路径与角色映射
	multiRoleHandlers map[string]MultiRoleHandler // 多角色处理器  存在同一个路径，但是有多个角色

	rolesBeforeHandlers map[string][]ActionHandlerWithOrder // 角色前置处理器
	rolesAfterHandlers map[string][]ActionHandlerWithOrder // 角色后置处理器

	subServices []types.SubServiceInf
	redisClient redis.UniversalClient // redis 客户端

	redisManager databases.RedisClientManagerInf // redis 管理器
	databaseClientManager databases.DatabaseClientMangerInf // 数据库管理器
	mqsManager mqs.MqManagerInf // mq 管理器
	mqFailStore mqs.FailedMessageStore // mq 失败消息存储


	grpcServices             []types.RegisterGrpcServiceInf
	grpcServerInterceptorHandler grpc.UnaryServerInterceptor // grpc 服务拦截器



	consumerFactories []mqs.ConsumerFactory // 消费者工厂 要延迟处理
	producerFactories []mqs.ProducerFactory // 生产者工厂 要延迟处理



	jwtKeyManager *auths.JwtKeyManager // jwt key 管理器

	remoteResourceManager *RemoteResourceManager // 远程资源管理器


	// 同步执行的 RunAble
	syncRunAbles []types.RunAbleInf
	asyncRunAbles []types.RunAbleInf
	subSrvSuccessCount int // 启动成功子服务数量
	subSrvCount int // 子服务数量
	errSrvCount int // 启动失败子服务数量
}

// 设置服务标识	
// 参数：
//  - serviceMark 服务标识
// 返回：
//  - 错误
func (s *Sunny) SetServiceMark(serviceMark string) {
	s.serviceMark = serviceMark
}

// 设置 redis 管理器
// 参数：
//  - redisClientManager redis 管理器
// 返回：
//  - 错误
func (s *Sunny) SetRedisClientManager(redisClientManager databases.RedisClientManagerInf) {
	s.redisManager = redisClientManager
}

// 设置数据库管理器
// 参数：
//  - databaseClientManager 数据库管理器
// 返回：
//  - 错误
func (s *Sunny) SetDatabaseClientManager(databaseClientManager databases.DatabaseClientMangerInf) {
	s.databaseClientManager = databaseClientManager
}

// 设置 mq 管理器
// 参数：
//  - mqManager mq 管理器
// 返回：
//  - 错误
func (s *Sunny) SetMqManager(mqManager mqs.MqManagerInf) {
	s.mqsManager = mqManager
}

// 设置 mq 失败消息存储
// 参数：
//  - mqFailStore mq 失败消息存储
// 返回：
//  - 错误
func (s *Sunny) SetMqFailStore(mqFailStore mqs.FailedMessageStore) {
	s.mqFailStore = mqFailStore
}

// 设置远程资源管理器
// 参数：
//  - remoteResourceManager 远程资源管理器
// 返回：
//  - 错误
func (s *Sunny) SetRemoteResourceManager(remoteResourceManager *RemoteResourceManager) {
	s.remoteResourceManager = remoteResourceManager
}


// 初始化
// 参数：
//  - configPath 配置文件路径
//  - activeEnv 激活环境 dev,test prod
// 返回：
//  - 错误
func (s *Sunny) Init(configPath,activeEnv string) error{
	s.configPath = configPath
	s.activeEnv = activeEnv

	runPath,err := os.Getwd()
	if err != nil{
		return err
	}
	s.runPath = runPath

	// 读取配置文件
	if s.configPath == ""{
		s.configPath = s.runPath + "/resources/"
	}
	logrus.Info("config path: ", s.configPath)
	err = s.loadConfig()
	if err != nil{
		panic("load config file error: " + err.Error())
	}
	logrus.Info("load config file success")

	if len(s.config.WebRoutes) > 0 {
		
		s.initWebRoutes(s.config.WebRoutes)
	}

	// 初始化数据库
	if len(s.config.Databases) > 0{
		databaseClientManager := databases.NewLocalDatabaseClientManager(s.config.Databases)
		s.AddSubServices(databaseClientManager)
		s.databaseClientManager = databaseClientManager
	}

	// 初始化远程资源管理器
	if s.remoteResourceManager != nil{
		err = s.remoteResourceManager.Init(context.TODO(),s)
		if err != nil{
			return err
		}
	}
	// 初始化mq
	if s.config.Mq != nil{
		// 默认 失败 放内存
		switch s.config.Mq.Type {
		case "rabbitmq":
			url := fmt.Sprintf("amqp://%s:%s@%s:%d/",s.config.Mq.RabbitMQ.Username,s.config.Mq.RabbitMQ.Password,s.config.Mq.RabbitMQ.Host,s.config.Mq.RabbitMQ.Port)
			if s.mqFailStore == nil{
				s.mqFailStore = mqs.NewMemoryFailedMessageStore()
			}
			s.mqsManager = mqs.NewRabbitMqManager(&mqs.RabbitMQOptions{
				URL: url,
				ChannelPoolSize: s.config.Mq.RabbitMQ.ChannelPoolSize,
				MaxRetries: s.config.Mq.RabbitMQ.MaxRetries,
				RetryInterval: time.Duration(s.config.Mq.RabbitMQ.RetryInterval) * time.Second, // 重试间隔
				ReconnectDelay: time.Duration(s.config.Mq.RabbitMQ.ReconnectDelay) * time.Second, // 重连延迟
			},s.mqFailStore)

			s.AddSubServices(s.mqsManager)
		case "kafka":
			if s.mqFailStore == nil{
				s.mqFailStore = mqs.NewMemoryFailedMessageStore()
			}
			s.mqsManager = mqs.NewKafkaManager(&mqs.KafkaOptions{
				Brokers: s.config.Mq.Kafka.Brokers,
				MaxRetries: s.config.Mq.Kafka.MaxRetries,
				RetryInterval: time.Duration(s.config.Mq.Kafka.RetryInterval) * time.Second, // 重试间隔
				ReconnectDelay: time.Duration(s.config.Mq.Kafka.ReconnectDelay) * time.Second, // 重连延迟
				SecurityProtocol: s.config.Mq.Kafka.SecurityProtocol,
			},s.mqFailStore)
			s.AddSubServices(s.mqsManager)
		default:
			logrus.Warn("mq config type not support")
		}
	}


	return nil
}


// 加载配置文件
func (s *Sunny) loadConfig() error{
	configer := &types.Config{}
	if s.configPath == ""{
		s.configPath = s.runPath + "/resources/"
	}
	if !strings.HasSuffix(s.configPath, "/") {
		s.configPath += "/"
	}

	baseConfigFile := s.configPath + "application.yaml"
	err := utils.ReadYamlFile(baseConfigFile, configer)
	if err != nil {
		return err
	}

	activeEnv := configer.ActiveEnv
	if s.activeEnv != ""{
		activeEnv = s.activeEnv // 如果传入了激活环境，则使用传入的激活环境
	}

	if activeEnv != ""{
		envConfigFile := s.configPath + "application-" + activeEnv + ".yaml"
		err = utils.ReadYamlFile(envConfigFile, configer)
		if err != nil {
			return err
		}
	}

	s.config = configer	
	return nil
}


func (r *Sunny) initWebRoutes(routes []*types.WebRouterInfo){
	ginHandlerFunc := func(c *gin.Context) {
		fullPath := c.FullPath()
		paths := strings.Split(fullPath, "/")
		if len(paths) < 3 {
			c.JSON(200, gin.H{"err_code": -1, "message": "path not found"})
			c.Abort()
			return
		}

		findPath := "/" + paths[1]

		groupLabel := c.Param("group")
		actionLabel := c.Param("action")
		roleLabel := r.pathRoles[findPath]

		// if multi role handler exists
		if r.multiRoleHandlers[findPath] != nil {
			roleLabel = r.multiRoleHandlers[findPath](groupLabel, actionLabel)
		}

		ctx := &Context{
			Context:     c,
			roleLabel:   roleLabel,
			groupLabel:  groupLabel,
			actionLabel: actionLabel,
			actionNext:  true,
		}

		roleInfo, ok := r.roles[roleLabel]
		if !ok {
			c.JSON(200, gin.H{"err_code": -1, "message": "role not found"})
			c.Abort()
			return
		}

		roleInfo.RunBefore(ctx)
		if !ctx.actionNext {
			return
		}
		groupKey := roleLabel + ":" + groupLabel
		groupInfo, ok := r.groups[groupKey]
		if !ok {
			c.JSON(200, gin.H{"err_code": -1, "message": "group not found .."})
			return
		}
		groupInfo.Call(ctx)
		roleInfo.RunAfter(ctx)
	}

	// 初始化路由
	for _, route := range routes {
		r.pathRoles[route.Path] = route.Role
		// check role label
		if _, ok := r.roles[route.Role]; !ok {
			// 如果角色不存在 则 created
			r.roles[route.Role] = NewRole(route.Role, route.RoleDesc)
		}

		// 注入角色前置处理器
		if handlers, ok := r.rolesBeforeHandlers[route.Role]; ok {
			for _, handler := range handlers {
				r.roles[route.Role].UseBefore(handler.Order, handler.ActionHandlerFunc)
			}
		}

		// 注入角色后置处理器
		if handlers, ok := r.rolesAfterHandlers[route.Role]; ok {
			for _, handler := range handlers {
				r.roles[route.Role].UseAfter(handler.Order, handler.ActionHandlerFunc)
			}
		}

		r.GET(route.Path+"/:group/:action", ginHandlerFunc)  // 注册GET方法
		r.POST(route.Path+"/:group/:action", ginHandlerFunc) // 注册POST方法
	}
}

// 添加子服务
// 参数：
//  - srvs 子服务
// 返回：
//  - 错误	
func (s *Sunny) AddSubServices(srvs ...types.SubServiceInf) error{
	for _,srv := range srvs{
		s.subServices = append(s.subServices,srv)
		s.subSrvCount += 1
	}
	return nil
}

// 添加同步执行的 RunAble
// 参数：
//  - srvs 同步执行的 RunAble
// 返回：
//  - 错误
func (s *Sunny) AddSyncRunAbles(srvs ...types.RunAbleInf) error{
	s.syncRunAbles = append(s.syncRunAbles,srvs...)
	return nil
}

// 添加异步执行的 RunAble
// 参数：
//  - srvs 异步执行的 RunAble
// 返回：
//  - 错误
func (s *Sunny) AddAsyncRunAbles(srvs ...types.RunAbleInf) error{
	s.asyncRunAbles = append(s.asyncRunAbles,srvs...)
	return nil
}


// Start 启动 Sunny
// 参数：
//  - ctx 上下文
//  - args 参数 0 是 激活的环境 1 是 配置文件路径
// 返回：
//  - 错误
func (s *Sunny) Start(ctx context.Context,args ...string) error{
	configPath:=""
	activeEnv:=""
	if len(args) > 0 {
		activeEnv = args[0]
	}
	if len(args) > 1 {
		configPath = args[1]
	}
	err := s.Init(configPath,activeEnv) // 初始化
	if err != nil{
		return err
	}

	// 绑定主题 生产者 消费者 
	if s.mqsManager != nil{
		for _,factory := range s.consumerFactories{
			consumer,err := factory(s.mqsManager)
			if err != nil{
				logrus.WithFields(logrus.Fields{
					"err_message": err.Error(),
				}).Error("consumer factory create error")
			}
			s.mqsManager.BindConsumer(consumer)
		}
		for _,factory := range s.producerFactories{
			producer,err := factory(s.mqsManager)
			if err != nil{
				logrus.WithFields(logrus.Fields{
					"err_message": err.Error(),
				}).Error("producer factory create error")
			}
			s.mqsManager.BindProducer(producer)
		}
	}

	cldCtx, cldCancel := context.WithCancel(ctx) // 创建一个上下文 用于取消
	defer cldCancel()

	// 启动子服务
	if len(s.subServices) > 0 {
		for _,srv := range s.subServices{
			resultChan := make(chan types.Result[any])
			go srv.Start(cldCtx,s,resultChan)
			result := <-resultChan // 等待子服务启动完成
			if result.ErrCode != 0 {
				logrus.WithFields(logrus.Fields{
					"service_name": srv.ServiceName(),
					"err_code": result.ErrCode,
					"err_message": result.Message,
				}).Error("sub service start error")
				s.errSrvCount += 1
				if srv.IsErrorStop(){
					return errors.New(result.Message)
				}
			}else{
				logrus.WithFields(logrus.Fields{
					"service_name": srv.ServiceName(),
					"message": result.Message,
				}).Info("sub service start success")
				s.subSrvSuccessCount += 1
			}
		}
	}

	// 需要同步执行的 runAble
	if len(s.syncRunAbles) > 0 {
		for _,runAble := range s.syncRunAbles{
			err=runAble.Run(cldCtx,s)
			if err != nil{
				logrus.WithFields(logrus.Fields{
					"err_message": err.Error(),
					"tip":runAble.Description(),
				}).Error("sync run able run error")
				return err
			}
		}
	}

	// 需要异步执行的 runAble
	if len(s.asyncRunAbles) > 0 {
		for _,runAble := range s.asyncRunAbles{
			go func (app *Sunny)  {
				err=runAble.Run(cldCtx,app)
				if err != nil{
					logrus.WithFields(logrus.Fields{
						"err_message": err.Error(),
						"tip":runAble.Description(),
					}).Error("async run able run error")
				}
			}(s)
		}
	}


	// 开启服务
	if len(s.config.Services) > 0 {
		var grpcServer *grpc.Server
		startHttpServices := make([]*http.Server, 0)
		for _, service := range s.config.Services {
			if service.Protocol == "http" {
				httpSrv := &http.Server{
					Addr:    fmt.Sprintf(":%d", service.Port),
					Handler: s,
				}
				startHttpServices = append(startHttpServices, httpSrv)
				go func() {
					logrus.Infof("http server start at port: %d", service.Port)
					if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
						logrus.Fatalf("listen: %s\n", err)
					}
				}()
			}
			if service.Protocol == "https" {
				httpsSrv := &http.Server{
					Addr:    fmt.Sprintf(":%d", service.Port),
					Handler: s,
				}
				startHttpServices = append(startHttpServices, httpsSrv)
				if service.CertPemPath == "" || service.KeyPemPath == "" {
					logrus.WithFields(logrus.Fields{"service": service}).Error("https server cert pem or cert key is empty")
					return errors.New("https server cert pem or cert key is empty")
				}
				go func() {
					logrus.Infof("https server start at port: %d", service.Port)
					if err := httpsSrv.ListenAndServeTLS(service.CertPemPath, service.KeyPemPath); err != nil && err != http.ErrServerClosed {
						logrus.Fatalf("listen: %s\n", err)
					}
				}()
			}

			if service.Protocol == "grpc" {
				lis, err := net.Listen("tcp", fmt.Sprintf(":%d", service.Port))
				if err != nil {
					logrus.WithFields(logrus.Fields{"err": err, "service": service}).Error("grpc server listen error")
					return errors.New("grpc server listen error")
				}
				serviceOpts := make([]grpc.ServerOption, 0)
				if s.grpcServerInterceptorHandler != nil {
					serviceOpts = append(serviceOpts, grpc.UnaryInterceptor(s.grpcServerInterceptorHandler)) // 绑定拦截器
				}
				grpcServer = grpc.NewServer(serviceOpts...)

				for _, service := range s.grpcServices {
					service.RegisterGrpcService(grpcServer)
				}

				go func() {
					logrus.Infof("grpc server start at port: %d", service.Port)
					if err := grpcServer.Serve(lis); err != nil {
						logrus.Fatalf("grpc server serve error: %v", err)
					}
				}()
			}
		}
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		cctx, cancel := context.WithTimeout(ctx, 1*time.Second) // 超时为了处理未完成的请求给的最大时间 如果该时间内未完成则强制关闭
		defer cancel()
		for _, srv := range startHttpServices {
			if err := srv.Shutdown(cctx); err != nil {
				logrus.Errorf("HTTP server shutdown error: %v", err)
			}
		}
		if grpcServer != nil {
			grpcServer.GracefulStop()
		}
		<-cctx.Done()
		logrus.Info("server shutdown success")
	}


	return nil
}


// 获取 redis 客户端
func (s *Sunny) GetRedisClient() redis.UniversalClient{
	return s.redisClient
}


// 设置 grpc 拦截器
// 参数：
//  - handler 拦截器
// 返回：
//  - 错误
func (s *Sunny) SetGrpcInterceptorHandler(handler grpc.UnaryServerInterceptor) {
	s.grpcServerInterceptorHandler = handler
}

// 绑定 grpc 服务
// 参数：
//  - services 服务
// 返回：
//  - 错误
func (s *Sunny) BindGrpcServices(services ...types.RegisterGrpcServiceInf) {
	s.grpcServices = append(s.grpcServices, services...)
}


// 设置 jwt key 管理器
// 参数：
//  - jwtKeyManager jwt key 管理器
// 返回：
//  - 错误
func (s *Sunny) SetJwtKeyManager(jwtKeyManager *auths.JwtKeyManager) {
	s.jwtKeyManager = jwtKeyManager
	s.AddSubServices(jwtKeyManager)
}


func (s *Sunny) AddRoleBeforeHandler(roleLabel string,order int,handler ActionHandlerFunc) {
	s.rolesBeforeHandlers[roleLabel] = append(s.rolesBeforeHandlers[roleLabel],ActionHandlerWithOrder{
		Order: order,
		ActionHandlerFunc: handler,
	})
}

func (s *Sunny) AddRoleAfterHandler(roleLabel string,order int,handler ActionHandlerFunc) {
	s.rolesAfterHandlers[roleLabel] = append(s.rolesAfterHandlers[roleLabel],ActionHandlerWithOrder{
		Order: order,
		ActionHandlerFunc: handler,
	})
}

func (s *Sunny) UseMultiRoleHandler(path string,handler MultiRoleHandler) {
	s.multiRoleHandlers[path] = handler
}

// 使用角色
// 参数：
//  - roles 角色
// 返回：
//  - 错误
func (s *Sunny) UseRoles(roles ...RoleInf) {
	for _,role := range roles{
		if _,ok := s.roles[role.RoleLabel()];ok{
			panic("role label already exists")
		}
		s.roles[role.RoleLabel()] = role
	}
}

// 使用组
// 参数：
//  - groups 组
// 返回：
//  - 错误
func (s *Sunny) UseGroups(groups ...GroupInf) {
	for _,group := range groups{
		key:=s.getGroupKey(group)
		if _,ok := s.groups[key];ok{
			panic("group label already exists")
		}
		s.groups[key] = group
	}
}

func (s *Sunny) getGroupKey(groupInfo GroupInf) string {
	return groupInfo.RoleLabel() + ":" + groupInfo.GroupLabel()
}

// 获取 grpc 客户端
// 参数：
//  - grpcSrvMark 服务标记
// 返回：
//  - grpc 客户端
//  - 错误
func (s *Sunny) GetGrpcClient(grpcSrvMark string) (grpc.ClientConnInterface,error){
	return  nil,errors.New("not implemented")
}

// 获取环境配置参数
// 参数:
// - key 
func (s *Sunny) GetEnvArgs(key string) (string,bool) {
	return "",false
}

// 获取 mq 管理器
// 返回：
//  - mq 管理器
func (s *Sunny) GetMqManager() mqs.MqManagerInf {
	return s.mqsManager
}


// 绑定 mq 管理器
// 参数：
//  - mqManager mq 管理器
// 返回：
//  - 错误
func (s *Sunny) BindMqManager(mqManager mqs.MqManagerInf) {
	s.mqsManager = mqManager
}


// 添加消费者工厂
// 参数：
//  - factory 消费者工厂
// 返回：
//  - 错误
func (s *Sunny) AddConsumerFactory(factory mqs.ConsumerFactory) {
	s.consumerFactories = append(s.consumerFactories, factory)
}

// 添加生产者工厂
// 参数：
//  - factory 生产者工厂
// 返回：
//  - 错误
func (s *Sunny) AddProducerFactory(factory mqs.ProducerFactory) {
	s.producerFactories = append(s.producerFactories, factory)
}	


// 获取数据库客户端
// 参数：
//  - key 数据库客户端标记 areaKey 区标志
// 返回：
//  - 数据库客户端
//  - 错误
func (s *Sunny) GetDBFromKey(areaKey  string) (*gorm.DB,error){
	if s.databaseClientManager == nil{
		return nil,errors.New("database client manager is not set")
	}
	return s.databaseClientManager.GetDBFromKey(areaKey)
}


// 获取默认数据库客户端
// 返回：
//  - 数据库客户端
//  - 错误
func (s *Sunny) GetDefaultDB() (*gorm.DB,error){
	if s.databaseClientManager == nil{
		return nil,errors.New("database client manager is not set")
	}
	return s.databaseClientManager.GetDBFromKey("default")
}


// 使用同步执行的 runAble
// 参数：
//  - runAbles 同步执行的 runAble
// 返回：
//  - 错误
func (s *Sunny) UseSyncRunAbles(runAbles ...types.RunAbleInf) {
	s.syncRunAbles = append(s.syncRunAbles, runAbles...)
}

// 使用异步执行的 runAble
// 参数：
//  - runAbles 异步执行的 runAble
// 返回：
//  - 错误
func (s *Sunny) UseAsyncRunAbles(runAbles ...types.RunAbleInf) {
	s.asyncRunAbles = append(s.asyncRunAbles, runAbles...)
}

