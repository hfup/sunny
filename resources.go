package sunny

import (
	"context"
	"errors"

	"github.com/hfup/sunny/types"
	"github.com/sirupsen/logrus"
	"github.com/hfup/sunny/components/databases"
	"github.com/hfup/sunny/components/mqs"
)


type ResourcesInfo struct {
	Redis []*types.RedisInfo `yaml:"redis" json:"redis"`
	Databases []*types.DatabaseInfo `yaml:"databases" json:"databases"`
	Mq *types.MqConfig `yaml:"mq" json:"mq"`
}
 
type ResourcesHandlerFunc func(ctx context.Context,serviceMark string) (*ResourcesInfo,error)

// 远程资源获取器
type RemoteResourceManager struct {
	handler ResourcesHandlerFunc
}

// 创建远程资源获取器
// 参数：
//  - handler 资源获取器
// 返回：
//  - 远程资源获取器}
func NewRemoteResourceManager(handler ResourcesHandlerFunc) *RemoteResourceManager {
	return &RemoteResourceManager{
		handler: handler,
	}
}

// 启动远程资源获取器
// 参数：
//  - ctx 上下文
//  - args 参数
//  - resultChan 结果通道
func (r *RemoteResourceManager) Init(ctx context.Context,app *Sunny) error {
	if app.serviceMark == "" {
		return errors.New("service mark is empty")
	}
	resourcesInfo,err := r.handler(ctx,app.serviceMark)
	if err != nil{
		return err
	}
	if len(resourcesInfo.Redis) > 0{
		redisClientManager := databases.NewLocalRedisClientManager(resourcesInfo.Redis)
		app.SetRedisClientManager(redisClientManager)
		app.AddSubServices(redisClientManager)
	}
	if len(resourcesInfo.Databases) > 0{
		databaseClientManager := databases.NewLocalDatabaseClientManager(resourcesInfo.Databases)
		app.SetDatabaseClientManager(databaseClientManager)
		app.AddSubServices(databaseClientManager)
	}
	if resourcesInfo.Mq != nil{
		
	}
	return nil
}

 


