package caches

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// LoadFunc 回调函数类型，用于加载数据
type LoadFunc[T any] func(ctx context.Context, key string) (T, error)

// cacheItem 缓存项
type cacheItem[T any] struct {
	value     T
	expiredAt time.Time
}

// isExpired 检查是否过期
func (item *cacheItem[T]) isExpired() bool {
	return !item.expiredAt.IsZero() && time.Now().After(item.expiredAt)
}

// LocalCache 本地缓存实现
type LocalCache[T any] struct {
	data     sync.Map
	loadFunc LoadFunc[T] // 只读字段，创建后不可修改
	ttl      time.Duration
	stopChan chan struct{}
	once     sync.Once
	sf       singleflight.Group // 单飞机制，防止并发重复加载
}

// NewLocalCache 创建本地缓存实例
func NewLocalCache[T any](loadFunc LoadFunc[T], ttl time.Duration) *LocalCache[T] {
	cache := &LocalCache[T]{
		loadFunc: loadFunc,
		ttl:      ttl,
		stopChan: make(chan struct{}),
	}

	// 启动清理协程
	cache.startCleanup()

	return cache
}

// NewLocalCacheWithoutLoader 创建无回调函数的本地缓存实例
func NewLocalCacheWithoutLoader[T any](ttl time.Duration) *LocalCache[T] {
	cache := &LocalCache[T]{
		ttl:      ttl,
		stopChan: make(chan struct{}),
	}

	// 启动清理协程
	cache.startCleanup()

	return cache
}

// Get 获取缓存
// 参数:
//   - ctx: context.Context 上下文
//   - key: string 缓存键
//
// 返回:
//   - T 缓存值
//   - error 错误信息
func (l *LocalCache[T]) Get(ctx context.Context, key string) (T, error) {
	var zero T

	// 从缓存中获取
	if value, exists := l.data.Load(key); exists {
		item := value.(*cacheItem[T])
		if !item.isExpired() {
			return item.value, nil
		}
		// 过期则删除
		l.data.Delete(key)
	}

	// 缓存不存在或已过期，使用回调函数加载
	if l.loadFunc != nil {
		// 使用单飞机制防止并发重复加载
		result, err, _ := l.sf.Do(key, func() (interface{}, error) {
			// 再次检查缓存，防止在等待期间其他协程已加载
			if value, exists := l.data.Load(key); exists {
				item := value.(*cacheItem[T])
				if !item.isExpired() {
					return item.value, nil
				}
				// 过期则删除
				l.data.Delete(key)
			}

			// 执行回调函数加载数据
			value, err := l.loadFunc(ctx, key)
			if err != nil {
				return zero, fmt.Errorf("failed to load data: %w", err)
			}

			// 加载成功后存入缓存
			if err := l.Set(ctx, key, value); err != nil {
				return zero, fmt.Errorf("failed to cache loaded data: %w", err)
			}

			return value, nil
		})

		if err != nil {
			return zero, err
		}

		// 安全的类型断言
		if typedResult, ok := result.(T); ok {
			return typedResult, nil
		}
		return zero, fmt.Errorf("type assertion failed for key: %s", key)
	}

	return zero, fmt.Errorf("key not found: %s", key)
}

// Set 设置缓存
// 参数:
//   - ctx: context.Context 上下文
//   - key: string 缓存键
//   - value: T 缓存值
//
// 返回:
//   - error 错误信息
func (l *LocalCache[T]) Set(ctx context.Context, key string, value T) error {
	var expiredAt time.Time
	if l.ttl > 0 {
		expiredAt = time.Now().Add(l.ttl)
	}

	item := &cacheItem[T]{
		value:     value,
		expiredAt: expiredAt,
	}

	l.data.Store(key, item)
	return nil
}

// Delete 删除键
// 参数:
//   - ctx: context.Context 上下文
//   - key: string 缓存键
//
// 返回:
//   - error 错误信息
func (l *LocalCache[T]) Delete(ctx context.Context, key string) error {
	l.data.Delete(key)
	return nil
}

// Clean 清除所有缓存
// 参数:
//   - ctx: context.Context 上下文
//
// 返回:
//   - error 错误信息
func (l *LocalCache[T]) Clean(ctx context.Context) error {
	l.data.Range(func(key, value interface{}) bool {
		l.data.Delete(key)
		return true
	})
	return nil
}

// startCleanup 启动清理协程
func (l *LocalCache[T]) startCleanup() {
	go func() {
		ticker := time.NewTicker(time.Minute * 5) // 每5分钟清理一次
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				l.cleanup()
			case <-l.stopChan:
				return
			}
		}
	}()
}

// cleanup 清理过期项
func (l *LocalCache[T]) cleanup() {
	l.data.Range(func(key, value interface{}) bool {
		item := value.(*cacheItem[T])
		if item.isExpired() {
			l.data.Delete(key)
		}
		return true
	})
}

// Close 关闭缓存
func (l *LocalCache[T]) Close() error {
	l.once.Do(func() {
		close(l.stopChan)
	})
	return nil
}

// Size 获取缓存大小
func (l *LocalCache[T]) Size() int {
	count := 0
	l.data.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// Keys 获取所有键
func (l *LocalCache[T]) Keys() []string {
	var keys []string
	l.data.Range(func(key, value interface{}) bool {
		if keyStr, ok := key.(string); ok {
			item := value.(*cacheItem[T])
			if !item.isExpired() {
				keys = append(keys, keyStr)
			}
		}
		return true
	})
	return keys
}

// Exists 检查键是否存在
func (l *LocalCache[T]) Exists(ctx context.Context, key string) (bool, error) {
	if value, exists := l.data.Load(key); exists {
		item := value.(*cacheItem[T])
		if !item.isExpired() {
			return true, nil
		}
		// 过期则删除
		l.data.Delete(key)
	}
	return false, nil
}

// GetWithTTL 获取缓存值和剩余TTL
func (l *LocalCache[T]) GetWithTTL(ctx context.Context, key string) (T, time.Duration, error) {
	var zero T

	if value, exists := l.data.Load(key); exists {
		item := value.(*cacheItem[T])
		if !item.isExpired() {
			var remainingTTL time.Duration
			if !item.expiredAt.IsZero() {
				remainingTTL = time.Until(item.expiredAt)
				if remainingTTL < 0 {
					remainingTTL = 0
				}
			}
			return item.value, remainingTTL, nil
		}
		// 过期则删除
		l.data.Delete(key)
	}

	return zero, 0, fmt.Errorf("key not found: %s", key)
}