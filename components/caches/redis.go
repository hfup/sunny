package caches

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisCache Redis缓存实现
type RedisCache[T any] struct {
	client        redis.UniversalClient
	prefix        string        // 前缀
	ttl           time.Duration // 过期时间
}

// NewRedisCache 创建新的Redis缓存实例
func NewRedisCache[T any](client *redis.Client, prefix string, ttl time.Duration) *RedisCache[T] {
	return &RedisCache[T]{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}
}

// NewRedisClusterCache 创建新的Redis集群缓存实例
func NewRedisClusterCache[T any](client redis.UniversalClient, prefix string, ttl time.Duration) *RedisCache[T] {
	return &RedisCache[T]{
		client:        client,
		prefix:        prefix,
		ttl:           ttl,
	}
}

// NewRedisCacheWithOptions 使用选项创建Redis缓存实例
func NewRedisCacheWithOptions[T any](options *redis.Options, prefix string, ttl time.Duration) *RedisCache[T] {
	client := redis.NewClient(options)
	return NewRedisCache[T](client, prefix, ttl)
}

// NewRedisClusterCacheWithOptions 使用集群选项创建Redis集群缓存实例
func NewRedisClusterCacheWithOptions[T any](options *redis.ClusterOptions, prefix string, ttl time.Duration) *RedisCache[T] {
	clusterClient := redis.NewClusterClient(options)
	return NewRedisClusterCache[T](clusterClient, prefix, ttl)
}

// getKey 获取带前缀的键名
func (r *RedisCache[T]) getKey(key string) string {
	if r.prefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", r.prefix, key)
}

// Get 获取缓存
func (r *RedisCache[T]) Get(ctx context.Context, key string) (T, error) {
	var zero T
	fullKey := r.getKey(key)
	
	result, err := r.client.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return zero, fmt.Errorf("key not found: %s", key)
		}
		return zero, fmt.Errorf("failed to get cache: %w", err)
	}
	
	var value T
	if err := json.Unmarshal([]byte(result), &value); err != nil {
		return zero, fmt.Errorf("failed to unmarshal cache value: %w", err)
	}
	
	return value, nil
}

// Set 设置缓存
func (r *RedisCache[T]) Set(ctx context.Context, key string, value T) error {
	fullKey := r.getKey(key)
	
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}
	
	if err := r.client.Set(ctx, fullKey, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}
	
	return nil
}

// Delete 删除键
func (r *RedisCache[T]) Delete(ctx context.Context, key string) error {
	fullKey := r.getKey(key)
	
	if err := r.client.Del(ctx, fullKey).Err(); err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}
	
	return nil
}

// Clean 清除所有缓存
func (r *RedisCache[T]) Clean(ctx context.Context) error {
	pattern := r.getKey("*")
	
	script := `
		local keys = redis.call('KEYS', ARGV[1])
		if #keys > 0 then
			return redis.call('DEL', unpack(keys))
		end
		return 0
	`
	
	_, err := r.client.Eval(ctx, script, []string{}, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to clean cache: %w", err)
	}
	
	return nil
}

// Close 关闭Redis连接
func (r *RedisCache[T]) Close() error {
	if r.client == nil {
		return nil
	}
	return r.client.Close()
}

// Ping 检查Redis连接状态
func (r *RedisCache[T]) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// SetTTL 设置键的过期时间
func (r *RedisCache[T]) SetTTL(ctx context.Context, key string, ttl time.Duration) error {
	fullKey := r.getKey(key)
	
	if err := r.client.Expire(ctx, fullKey, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set TTL: %w", err)
	}
	
	return nil
}

// GetTTL 获取键的剩余过期时间
func (r *RedisCache[T]) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := r.getKey(key)
	
	ttl, err := r.client.TTL(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}
	
	return ttl, nil
}

// Exists 检查键是否存在
func (r *RedisCache[T]) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := r.getKey(key)
	
	count, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	
	return count > 0, nil
}
