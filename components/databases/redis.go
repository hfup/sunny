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
	redisConfigs    []*types.RedisInfo
	redisRouterFunc RedisRouterFunc
	redisMap        map[string]redis.UniversalClient

	defaultClient redis.UniversalClient
}

func NewLocalRedisClientManager(opt []*types.RedisInfo) *LocalRedisClientManager {
	return &LocalRedisClientManager{
		redisConfigs: opt,
		redisMap:     make(map[string]redis.UniversalClient),
	}
}

func (l *LocalRedisClientManager) SetRouterHandler(routerHandler RedisRouterFunc) {
	l.redisRouterFunc = routerHandler
}

// 配置
func (l *LocalRedisClientManager) Start(ctx context.Context, args any, resultChan chan<- types.Result[any]) {
	l.redisMap = make(map[string]redis.UniversalClient)

	// 遍历配置 创建 redis 客户端
	for _, redisConfig := range l.redisConfigs {
		if redisConfig.AreaKey == "" {
			resultChan <- types.Result[any]{
				ErrCode: 1,
				Message: "redis config key is empty, key: ",
			}
			return
		}
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
		l.redisMap[redisConfig.AreaKey] = client

		if redisConfig.AreaKey == "default" {
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
	return "local Redis 客户端管理器"
}
