package databases

import (
	"fmt"
	"errors"

	"github.com/hfup/sunny/types"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"

	"github.com/go-redis/redis/v8"
)


// mysql 连接
func MysqlConnect(dataInfo *types.DatabaseInfo) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		dataInfo.User, dataInfo.Password, dataInfo.Host, dataInfo.Port, dataInfo.DbName, dataInfo.Charset)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if dataInfo.MaxIdelConns > 0 {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetMaxIdleConns(dataInfo.MaxIdelConns)
	}
	if dataInfo.MaxOpenConns > 0 {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetMaxOpenConns(dataInfo.MaxOpenConns)
	}
	if dataInfo.MaxLifetime > 0 {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetConnMaxLifetime(time.Duration(dataInfo.MaxLifetime) * time.Second)
	}
	return db, nil
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