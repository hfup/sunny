package databases

import (
	"context"
	"errors"

	"github.com/go-redis/redis/v8"
	"github.com/hfup/sunny/types"
)

var (
	ErrRedisClientNotFound = errors.New("redis client not found")
)


type RedisRouterFunc func(key string) (redis.UniversalClient, error)

type RedisClientManagerInf interface {
	types.SubServiceInf
	GetClientFromKey(key string) (redis.UniversalClient, error)
	SetRouterHandler(routerHandler RedisRouterFunc) // 设置路由方法
}

type LocalRedisClientManager struct {
	redisConfigs []*types.RedisConfig
	redisRouterFunc RedisRouterFunc
	redisMap map[string]redis.UniversalClient

	defaultClient redis.UniversalClient
}


func NewLocalRedisClientManager(opt []*types.RedisConfig) *LocalRedisClientManager {
	return &LocalRedisClientManager{
		redisConfigs: opt,
		redisMap: make(map[string]redis.UniversalClient),
	}
}

func (l *LocalRedisClientManager) SetRouterHandler(routerHandler RedisRouterFunc) {
	l.redisRouterFunc = routerHandler
}


// 配置
func (l *LocalRedisClientManager) Start(ctx context.Context, args any, resultChan chan<- types.Result[any]) {
	l.redisMap = make(map[string]redis.UniversalClient)
	
	for _, redisConfig := range l.redisConfigs {
		var client redis.UniversalClient
		var err error
		
		if redisConfig.IsCluster {
			client, err = RedisClusterConnect(redisConfig)
		} else {
			client, err = RedisConnect(redisConfig)
		}
		
		if err != nil {
			resultChan <- types.Result[any]{
				ErrCode: 1,
				Message: "redis connect error: " + err.Error(),
			}
			return
		}
		
		// 存储客户端到 map 中
		l.redisMap[redisConfig.DbId] = client
		
		if redisConfig.DbId == "default" {
			l.defaultClient = client
		}
	}

	resultChan <- types.Result[any]{
		ErrCode: 0,
		Message: "redis client manager start success",
	}
}

// 获取 redis 客户端
func (l *LocalRedisClientManager) GetClientFromKey(key string) (redis.UniversalClient, error) {
	if key == "default" {
		return l.defaultClient, nil
	}
	if l.redisRouterFunc != nil {
		return l.redisRouterFunc(key)
	}	
	client, ok := l.redisMap[key]
	if !ok {
		return nil, ErrRedisClientNotFound
	}
	return client, nil
}

// 遇到错误 终止整个程序执行
func (l *LocalRedisClientManager) IsErrorStop() bool {
	return true
}

// ServiceName 返回服务名称
func (l *LocalRedisClientManager) ServiceName() string {
	return "LocalRedisClientManager"
}



// 单机连接
func RedisConnect(redisConfig *types.RedisConfig) (redis.UniversalClient, error) {
	if len(redisConfig.Addrs) == 0 {
		return nil,errors.New("redis 配置信息不存在")
	}
	options := &redis.Options{
		Addr:         redisConfig.Addrs[0],
		Password:     redisConfig.Password,
		DB:           redisConfig.DB,
		PoolSize:     redisConfig.PoolSize,
		MinIdleConns: redisConfig.MinIdleConns,
	}
	
	client := redis.NewClient(options)
	// 测试连接
	if err := client.Ping(client.Context()).Err(); err != nil {
		return nil, err
	}
	return client, nil
}

// 集群连接
func RedisClusterConnect(redisConfig *types.RedisConfig) (redis.UniversalClient, error) {
	options := &redis.ClusterOptions{
		Addrs:        redisConfig.Addrs,
		Password:     redisConfig.Password,
		PoolSize:     redisConfig.PoolSize,
		MinIdleConns: redisConfig.MinIdleConns,
	}
	
	client := redis.NewClusterClient(options)
	// 测试连接
	if err := client.Ping(client.Context()).Err(); err != nil {
		return nil, err
	}
	return client, nil
}