package limits

import (
	"context"
	"errors"
	"sync"
	"time"
	"crypto/rand"
	"encoding/hex"
	"github.com/go-redis/redis/v8"
)


// UniLimiterLockerInf 单机限流器接口
// 参数:
//   - DoWithTryLock: 尝试获取锁并执行函数，如果锁被占用则立即返回错误  例如 用户自己操作余额 只存在 1种情况
//   - DoWithWaitLock: 等待获取锁并执行函数，如果锁被占用则等待 例如 后台新增/逻辑奖励余额 需要串行
type UniLimiterLockerInf interface {
	DoWithTryLock(ctx context.Context,key string,fn func() error) error
	DoWithWaitLock(ctx context.Context,key string,fn func() error) error
}


var (
	ErrLockBusy = errors.New("lock busy")
)


type UniLimiterLocker struct {
	muMap sync.Map // key -> *sync.Mutex
}


func (l *UniLimiterLocker) getMutex(key string) (*sync.Mutex,bool){
    m, ok := l.muMap.LoadOrStore(key,&sync.Mutex{})
    return m.(*sync.Mutex),ok
}

// 用来执行函数，如果锁被占用，则返回错误
// 参数：
//  - ctx: 上下文
//  - key: 锁的key
//  - fn: 执行函数
// 返回：
//  - 错误
func (l *UniLimiterLocker) DoWithTryLock(ctx context.Context,key string,fn func() error) error {
	mu,ok:= l.getMutex(key)
	if ok{
		return ErrLockBusy
	}
	mu.Lock()
	defer func ()  {
		mu.Unlock()
		l.Release(key)
	}()
	
	err := fn()
	if err != nil{
		return err
	}
	return nil
}

// 用来执行函数，如果锁被占用，则等待获取锁
// 参数：
//  - ctx: 上下文
//  - key: 锁的key
//  - fn: 执行函数
// 返回：
//  - 错误
func (l *UniLimiterLocker) DoWithWaitLock(ctx context.Context,key string,fn func() error) error {
	mu,_:= l.getMutex(key)
	mu.Lock()
	defer func() {
		mu.Unlock()
		l.Release(key)
	}()
	err := fn()
	if err != nil{
		return err
	}
	return nil
}


func (l *UniLimiterLocker) Release(key string){
	l.muMap.Delete(key)
}

// RedisLimiterLocker Redis分布式锁实现
type RedisLimiterLocker struct {
	rdb redis.UniversalClient
	defaultTTL time.Duration // 锁的默认过期时间
}




// NewRedisLimiterLocker 创建Redis分布式锁实例
// 参数:
//   - rdb: Redis客户端
//   - defaultTTL: 默认锁过期时间
// 返回:
//   - *RedisLimiterLocker Redis分布式锁实例
func NewRedisLimiterLocker(rdb redis.UniversalClient, defaultTTL time.Duration) *RedisLimiterLocker {
	if defaultTTL <= 0 {
		defaultTTL = 30 * time.Second // 默认30秒
	}
	return &RedisLimiterLocker{
		rdb: rdb,
		defaultTTL: defaultTTL,
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

// DoWithTryLock 尝试获取锁并执行函数，如果锁被占用则立即返回错误
// 参数:
//   - ctx: 上下文
//   - key: 锁的key
//   - fn: 执行函数
// 返回:
//   - error 错误信息
func (r *RedisLimiterLocker) DoWithTryLock(ctx context.Context, key string, fn func() error) error {
	return r.doWithTryLock(ctx, key, r.generateLockValue(), fn)
}

// DoWithTryLockWithValue 允许自定义锁值的版本（高级用法）
// 参数:
//   - ctx: 上下文
//   - key: 锁的key
//   - lockValue: 自定义锁值（必须唯一）
//   - fn: 执行函数
// 返回:
//   - error 错误信息
func (r *RedisLimiterLocker) DoWithTryLockWithValue(ctx context.Context, key string, lockValue string, fn func() error) error {
	return r.doWithTryLock(ctx, key, lockValue, fn)
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
//   - key: 锁的key
//   - fn: 执行函数
// 返回:
//   - error 错误信息
func (r *RedisLimiterLocker) DoWithWaitLock(ctx context.Context, key string, fn func() error) error {
	lockValue := r.generateLockValue()
	
	// 循环尝试获取锁
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// 尝试获取锁
		success, err := r.rdb.SetNX(ctx, key, lockValue, r.defaultTTL).Result()
		if err != nil {
			return err
		}
		
		if success {
			// 成功获取锁，执行函数
			defer r.releaseLock(ctx, key, lockValue)
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



