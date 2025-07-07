package sequences

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/go-redis/redis/v8"
)

// 基于Redis的id生成器
type RedisSequence struct {
	client   redis.UniversalClient
	key      string // Redis中存储计数器的key
	strLen   int    // 字符串长度
	maxValue int64  // 最大值
}

// NewRedisSequence 创建新的Redis序列号生成器
// 参数:
//   - client: Redis客户端
//   - key: Redis中存储计数器的key
//   - strLen: 生成字符串的长度
//
// 返回:
//   - *RedisSequence: Redis序列号生成器实例
//   - error: 错误信息
func NewRedisSequence(client redis.UniversalClient, key string, strLen int) *RedisSequence {
	maxValue := int64(math.Pow10(strLen) - 1)
	return &RedisSequence{
		client:   client,
		key:      key,
		strLen:   strLen,
		maxValue: maxValue,
	}
}

// Next 生成下一个id
// 返回:
//   - string: 下一个id
//   - error: 错误信息
func (r *RedisSequence) Next() (string, error) {
	ctx := context.Background()

	// 使用Lua脚本确保原子操作
	luaScript := `
		local key = KEYS[1]
		local maxValue = tonumber(ARGV[1])
		
		-- 获取当前值并递增
		local current = redis.call('INCR', key)
		
		-- 如果超过最大值，重置为1
		if current > maxValue then
			redis.call('SET', key, 1)
			current = 1
		end
		
		return current
	`

	// 执行Lua脚本
	result, err := r.client.Eval(ctx, luaScript, []string{r.key}, r.maxValue).Result()
	if err != nil {
		return "", fmt.Errorf("Redis操作失败: %w", err)
	}

	// 将结果转换为int64
	current, ok := result.(int64)
	if !ok {
		return "", fmt.Errorf("Redis返回值类型错误")
	}

	// 格式化为指定长度的字符串
	return fmt.Sprintf("%0*d", r.strLen, current), nil
}

// Reset 重置计数器
// 返回:
//   - error: 错误信息
func (r *RedisSequence) Reset() error {
	ctx := context.Background()
	err := r.client.Set(ctx, r.key, 0, 0).Err()
	if err != nil {
		return fmt.Errorf("重置计数器失败: %w", err)
	}
	return nil
}

// GetCurrent 获取当前计数器值
// 返回:
//   - int64: 当前计数器值
//   - error: 错误信息
func (r *RedisSequence) GetCurrent() (int64, error) {
	ctx := context.Background()
	result, err := r.client.Get(ctx, r.key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil // key不存在，返回0
		}
		return 0, fmt.Errorf("获取当前值失败: %w", err)
	}

	current, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("解析当前值失败: %w", err)
	}

	return current, nil
}
