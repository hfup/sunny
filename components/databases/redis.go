package databases

import "github.com/go-redis/redis/v8"


// 单机连接
func RedisConnect(redisConfig *redis.Options) (redis.UniversalClient, error) {
	client := redis.NewClient(redisConfig)
	// 测试连接
	if err := client.Ping(client.Context()).Err(); err != nil {
		return nil, err
	}
	return client, nil
}

// 集群连接
func RedisClusterConnect(redisConfig *redis.ClusterOptions) (redis.UniversalClient, error) {
	client := redis.NewClusterClient(redisConfig)
	// 测试连接
	if err := client.Ping(client.Context()).Err(); err != nil {
		return nil, err
	}
	return client, nil
}