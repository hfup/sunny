package limits

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// UniLimiterLockerInf 单机限流器接口
// 参数:
//   - DoWithTryLock: 尝试获取锁并执行函数，如果锁被占用则立即返回错误  例如 用户自己操作余额 只存在 1种情况
//   - DoWithWaitLock: 等待获取锁并执行函数，如果锁被占用则等待 例如 后台新增/逻辑奖励余额 需要串行
type UniLimiterLockerInf interface {
	DoWithTryLock(ctx context.Context, key string, fn func() error) error
	DoWithWaitLock(ctx context.Context, key string, fn func() error) error
}

var (
	ErrLockBusy = errors.New("lock busy")
)

// mutexRef 带引用计数的互斥锁
type mutexRef struct {
	mu       *sync.Mutex
	refCount int32
}

type UniLimiterLocker struct {
	muMap sync.Map   // key -> *mutexRef
	mux   sync.Mutex // 保护引用计数操作
}

func NewUniLimiterLocker() *UniLimiterLocker {
	return &UniLimiterLocker{
		muMap: sync.Map{},
	}
}

// getMutex 获取互斥锁并增加引用计数
// 参数:
//   - key: 锁的key
//
// 返回:
//   - *sync.Mutex 互斥锁
//   - bool 是否已存在（true表示已存在，false表示新创建）
func (l *UniLimiterLocker) getMutex(key string) (*sync.Mutex, bool) {
	l.mux.Lock()
	defer l.mux.Unlock()

	if val, ok := l.muMap.Load(key); ok {
		ref := val.(*mutexRef)
		ref.refCount++
		return ref.mu, true
	}

	// 创建新的互斥锁
	ref := &mutexRef{
		mu:       &sync.Mutex{},
		refCount: 1,
	}
	l.muMap.Store(key, ref)
	return ref.mu, false
}

// releaseMutex 释放互斥锁并减少引用计数
// 参数:
//   - key: 锁的key
func (l *UniLimiterLocker) releaseMutex(key string) {
	l.mux.Lock()
	defer l.mux.Unlock()

	if val, ok := l.muMap.Load(key); ok {
		ref := val.(*mutexRef)
		ref.refCount--
		if ref.refCount <= 0 {
			l.muMap.Delete(key)
		}
	}
}

// 用来执行函数，如果锁被占用，则返回错误
// 参数：
//   - ctx: 上下文
//   - key: 锁的key
//   - fn: 执行函数
//
// 返回：
//   - 错误
func (l *UniLimiterLocker) DoWithTryLock(ctx context.Context, key string, fn func() error) error {
	mu, _ := l.getMutex(key)

	// 尝试获取锁，如果获取失败则立即返回错误
	if !mu.TryLock() {
		l.releaseMutex(key) // 减少引用计数
		return ErrLockBusy
	}

	defer func() {
		mu.Unlock()
		l.releaseMutex(key)
	}()

	err := fn()
	if err != nil {
		return err
	}
	return nil
}

// 用来执行函数，如果锁被占用，则等待获取锁
// 参数：
//   - ctx: 上下文
//   - key: 锁的key
//   - fn: 执行函数
//
// 返回：
//   - 错误
func (l *UniLimiterLocker) DoWithWaitLock(ctx context.Context, key string, fn func() error) error {
	mu, _ := l.getMutex(key)
	mu.Lock()
	defer func() {
		mu.Unlock()
		l.releaseMutex(key)
	}()
	err := fn()
	if err != nil {
		return err
	}
	return nil
}

// RedisLimiterLocker Redis分布式锁实现
type RedisLimiterLocker struct {
	rdb        redis.UniversalClient
	defaultTTL time.Duration // 锁的默认过期时间
	prefix     string        // 锁的前缀，用于区分不同业务场景
}

// NewRedisLimiterLocker 创建Redis分布式锁实例
// 参数:
//   - rdb: Redis客户端
//   - defaultTTL: 默认锁过期时间
//   - prefix: 锁的前缀，用于区分不同业务场景（如 "user:balance", "user:points"）
//
// 返回:
//   - *RedisLimiterLocker Redis分布式锁实例
func NewRedisLimiterLocker(rdb redis.UniversalClient, defaultTTL time.Duration, prefix string) *RedisLimiterLocker {
	if defaultTTL <= 0 {
		defaultTTL = 30 * time.Second // 默认30秒
	}
	if prefix == "" {
		prefix = "lock" // 默认前缀
	}
	return &RedisLimiterLocker{
		rdb:        rdb,
		defaultTTL: defaultTTL,
		prefix:     prefix,
	}
}

// generateLockValue 生成锁的唯一值
// 返回:
//   - string 唯一值
func (r *RedisLimiterLocker) generateLockValue() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// formatKey 格式化锁的key，添加前缀
// 参数:
//   - key: 原始key
//
// 返回:
//   - string 格式化后的key
func (r *RedisLimiterLocker) formatKey(key string) string {
	return r.prefix + ":" + key
}

// DoWithTryLock 尝试获取锁并执行函数，如果锁被占用则立即返回错误
// 参数:
//   - ctx: 上下文
//   - key: 锁的key（不包含前缀）
//   - fn: 执行函数
//
// 返回:
//   - error 错误信息
func (r *RedisLimiterLocker) DoWithTryLock(ctx context.Context, key string, fn func() error) error {
	return r.doWithTryLock(ctx, r.formatKey(key), r.generateLockValue(), fn)
}

// DoWithTryLockWithValue 允许自定义锁值的版本（高级用法）
// 参数:
//   - ctx: 上下文
//   - key: 锁的key（不包含前缀）
//   - lockValue: 自定义锁值（必须唯一）
//   - fn: 执行函数
//
// 返回:
//   - error 错误信息
func (r *RedisLimiterLocker) DoWithTryLockWithValue(ctx context.Context, key string, lockValue string, fn func() error) error {
	return r.doWithTryLock(ctx, r.formatKey(key), lockValue, fn)
}

// doWithTryLock 内部实现
func (r *RedisLimiterLocker) doWithTryLock(ctx context.Context, key string, lockValue string, fn func() error) error {

	// 使用 SET NX EX 命令尝试获取锁
	success, err := r.rdb.SetNX(ctx, key, lockValue, r.defaultTTL).Result()
	if err != nil {
		return err
	}

	// 如果获取锁失败，说明锁被占用
	if !success {
		return ErrLockBusy
	}

	// 确保在函数执行完成后释放锁
	defer r.releaseLock(ctx, key, lockValue)

	// 执行业务函数
	return fn()
}

// releaseLock 释放锁（使用Lua脚本确保原子性）key value 相等 则删除 防止误删
// 参数:
//   - ctx: 上下文
//   - key: 锁的key
//   - value: 锁的值
func (r *RedisLimiterLocker) releaseLock(ctx context.Context, key, value string) {
	luaScript := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	r.rdb.Eval(ctx, luaScript, []string{key}, value)
}

// DoWithWaitLock 等待获取锁并执行函数，如果锁被占用则等待
// 参数:
//   - ctx: 上下文
//   - key: 锁的key（不包含前缀）
//   - fn: 执行函数
//
// 返回:
//   - error 错误信息
func (r *RedisLimiterLocker) DoWithWaitLock(ctx context.Context, key string, fn func() error) error {
	lockValue := r.generateLockValue()
	fullKey := r.formatKey(key)

	// 循环尝试获取锁
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 尝试获取锁
		success, err := r.rdb.SetNX(ctx, fullKey, lockValue, r.defaultTTL).Result()
		if err != nil {
			return err
		}

		if success {
			// 成功获取锁，执行函数
			defer r.releaseLock(ctx, fullKey, lockValue)
			return fn()
		}
		// 锁被占用，等待一段时间后重试
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond): // 等待300ms后重试
		}
	}
}
