package databases

import (
	"context"
	"errors"

	"github.com/go-redis/redis/v8"
	"github.com/hfup/sunny/types"
	"github.com/sirupsen/logrus"
)

var (
	ErrRedisClientNotFound = errors.New("redis client not found")
)

type RedisRouterFunc func(key string) (redis.UniversalClient, error)

type RedisClientManagerInf interface {
	types.RunAbleInf
	GetClientFromKey(key string) (redis.UniversalClient, error)
	SetRouterHandler(routerHandler RedisRouterFunc) // 设置路由方法
}

type LocalRedisClientManager struct {
	redisConfigs    []*types.RedisInfo
	redisRouterFunc RedisRouterFunc
	redisMap        map[string]redis.UniversalClient
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

// Run 启动 redis 客户端管理器
func (l *LocalRedisClientManager) Run(ctx context.Context, app any) error {
	l.redisMap = make(map[string]redis.UniversalClient)
	if len(l.redisConfigs) == 0 {
		logrus.Warn("redis config is empty")
		return nil
	}
	isSingle := len(l.redisConfigs) == 1
	for _, redisConfig := range l.redisConfigs {
		if isSingle && redisConfig.AreaKey == "" {
			redisConfig.AreaKey = "default"
		}
		if redisConfig.AreaKey == "" {
			logrus.Warn("redis config key is empty")
			continue
		}
		var client redis.UniversalClient
		var err error

		if redisConfig.IsCluster {
			client, err = RedisClusterConnect(redisConfig)
		} else {
			client, err = RedisConnect(redisConfig)
		}

		if err != nil {
			logrus.Error("redis connect error: " + err.Error())
			return err
		}
		// 存储客户端到 map 中
		l.redisMap[redisConfig.AreaKey] = client
	}
	return nil
}

func (l *LocalRedisClientManager) Description() string {
	return "本地 Redis 客户端管理器"
}


// 获取 redis 客户端
func (l *LocalRedisClientManager) GetClientFromKey(key string) (redis.UniversalClient, error) {
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
