package sunny

import (
	"context"
	"errors"

	"github.com/hfup/sunny/components/databases"
	"github.com/hfup/sunny/components/mqs"
	"github.com/hfup/sunny/components/storages"
	"github.com/hfup/sunny/types"

	"github.com/sirupsen/logrus"
)

type ResourcesInfo struct {
	Redis []*types.RedisInfo `yaml:"redis" json:"redis"`
	Databases []*types.DatabaseInfo `yaml:"databases" json:"databases"`
	Mq *types.MqConfig `yaml:"mq" json:"mq"`
	CloudStorage *types.CloudStorageConf `yaml:"cloud_storage" json:"cloud_storage"`
	UniqueRedis *types.RedisInfo 
	GaodeKey string `yaml:"gaode_key" json:"gaode_key"` // 高德地图key 
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


// 启动远程资源获取器 同时初始化 数据库管理器/redis管理器/mq管理器
// 参数：
//  - ctx 上下文
//  - args 参数
//  - resultChan 结果通道
func (r *RemoteResourceManager) Init(ctx context.Context,app *Sunny) error {
	if app.serviceMark == "" {
		return errors.New("service mark is empty")
	}
	//logrus.Info("当前 service mark:",app.serviceMark)
	resourcesInfo,err := r.handler(ctx,app.serviceMark)
	if err != nil{
		return err
	}
	if len(resourcesInfo.Redis) > 0{
		redisClientManager := databases.NewLocalRedisClientManager(resourcesInfo.Redis)
		app.SetRedisClientManager(redisClientManager)
		app.UseStartFunc(redisClientManager)
	}
	if len(resourcesInfo.Databases) > 0{
		databaseClientManager := databases.NewLocalDatabaseClientManager(resourcesInfo.Databases)
		if app.config.DatabaseDebug { // 数据库调试模式
			databaseClientManager.SetDebug(true)
		}
		app.SetDatabaseClientManager(databaseClientManager)
		app.UseStartFunc(databaseClientManager)
	}
	if resourcesInfo.Mq != nil{
		if app.mqFailStore == nil{
			app.mqFailStore = mqs.GetDefaultFailedStore()
		}
		mqManager,err := mqs.CreateMqManager(resourcesInfo.Mq,app.mqFailStore)
		if err != nil{
			return err
		}
		app.SetMqManager(mqManager)
		app.AddSubServices(mqManager)
	}
	if resourcesInfo.CloudStorage != nil{
		switch resourcesInfo.CloudStorage.StorageType {
		case "cos":
			cos:=storages.NewCosStorage(&types.CloudStorageConf{
				SecretId: resourcesInfo.CloudStorage.SecretId,
				SecretKey: resourcesInfo.CloudStorage.SecretKey,
				Bucket: resourcesInfo.CloudStorage.Bucket,
				Region: resourcesInfo.CloudStorage.Region,
			})
			app.SetStorager(cos)
		case "oss":
			oss:=storages.NewOssStorage(&types.CloudStorageConf{
				SecretId: resourcesInfo.CloudStorage.SecretId,
				SecretKey: resourcesInfo.CloudStorage.SecretKey,
				Bucket: resourcesInfo.CloudStorage.Bucket,
				Region: resourcesInfo.CloudStorage.Region,
			})
			app.SetStorager(oss)
		default:
			logrus.Error("cloud storage type not support")
		}
	}
	
	// 处理下 unique redis
	if resourcesInfo.UniqueRedis != nil{
		uniqueRedisClient,err := databases.RedisConnect(resourcesInfo.UniqueRedis)
		if err != nil{
			return err
		}
		app.SetUniqueRedisClient(uniqueRedisClient)
	}
	return nil
}

 


