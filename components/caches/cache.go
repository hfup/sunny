package caches

import "context"

// CacheInterface 缓存接口
// 参数:
//   - T: 泛型类型参数，表示缓存值的类型
type CacheInf[T any] interface {
	Get(ctx context.Context, key string) (T, error)     // 获取缓存
	Set(ctx context.Context, key string, value T) error // 设置缓存
	Delete(ctx context.Context, key string) error       // 删除键
	Clean(ctx context.Context) error                    // 清除所有缓存
}

// LoadFunc 回调函数类型，用于加载数据
type LoadFunc[T any] func(ctx context.Context, key string) (T, error)
