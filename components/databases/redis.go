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

type RedisRouterFunc func(key string) (string, error) // key -> areaKey -> redis.UniversalClient


// Redis 客户端管理器接口
type RedisClientManagerInf interface {
	types.RunAbleInf
	GetClientFromKey(key string) (redis.UniversalClient, error)
	SetRouterHandler(routerHandler RedisRouterFunc) // 设置路由方法
	AddRedisConfigs(redisConfigs []*types.RedisInfo) // 添加 redis 配置
	IsDebug() bool // 是否开启调试
}

type RedisClientManager struct {
	redisConfigs    []*types.RedisInfo
	redisRouterFunc RedisRouterFunc
	redisMap        map[string]redis.UniversalClient
	isDebug bool
}

// 创建 redis 客户端管理器
// 参数：
//   - opt 配置
// 返回：
//   - redis 客户端管理器
func NewRedisClientManager(isDebug bool) *RedisClientManager {
	return &RedisClientManager{
		redisConfigs: make([]*types.RedisInfo, 0),
		redisMap:     make(map[string]redis.UniversalClient),
		isDebug: isDebug,
	}
}

func (l *RedisClientManager) IsDebug() bool {
	return l.isDebug
}

// 设置路由处理函数
// 参数：
//   - routerHandler 路由处理函数
// 返回：
//   - 错误
func (l *RedisClientManager) SetRouterHandler(routerHandler RedisRouterFunc) {
	l.redisRouterFunc = routerHandler
}

// 添加 redis 配置
// 参数：
//   - redisConfigs 配置
// 返回：
//   - 错误
func (l *RedisClientManager) AddRedisConfigs(redisConfigs []*types.RedisInfo) {
	l.redisConfigs = append(l.redisConfigs, redisConfigs...)
}

// 启动 redis 客户端管理器
// 参数：
//   - ctx 上下文
//   - app 应用
// 返回：
//   - 错误
func (l *RedisClientManager) Run(ctx context.Context, app any) error {
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
		l.redisMap[redisConfig.AreaKey] = client
	}
	return nil
}


// 返回描述
// 返回：
//   - 描述
func (l *RedisClientManager) Description() string {
	return "Redis 客户端管理器"
}


// 获取 redis 客户端
func (l *RedisClientManager) GetClientFromKey(key string) (redis.UniversalClient, error) {
	if l.redisRouterFunc != nil {
		areaKey, err := l.redisRouterFunc(key)
		if err != nil {
			return nil, err
		}
		credis,ok := l.redisMap[areaKey]
		if !ok {
			return nil, ErrRedisClientNotFound
		}
		return credis, nil
	}
	client, ok := l.redisMap[key]
	if !ok {
		return nil, ErrRedisClientNotFound
	}
	return client, nil
}

