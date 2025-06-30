package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/hfup/sunny/utils"
)

// ExampleTask 示例任务实现
type ExampleTask struct {
	id           string
	name         string
	executeTime  time.Time
	priority     int
	maxRetries   int
	retryDelay   time.Duration
	isContinue   bool
	data         map[string]any
}

// NewExampleTask 创建示例任务
// 参数:
//   - name: string 任务名称
//   - executeTime: time.Time 执行时间
//   - priority: int 优先级
//   - isContinue: bool 是否持续执行
// 返回:
//   - *ExampleTask 示例任务
func NewExampleTask(name string, executeTime time.Time, priority int, isContinue bool) *ExampleTask {
	return &ExampleTask{
		id:          utils.RandStr(16, false),
		name:        name,
		executeTime: executeTime,
		priority:    priority,
		maxRetries:  3,
		retryDelay:  5 * time.Second,
		isContinue:  isContinue,
		data:        make(map[string]any),
	}
}

// Run 执行任务
// 参数:
//   - ctx: context.Context 上下文
//   - args: any 参数
// 返回:
//   - error 错误信息
func (t *ExampleTask) Run(ctx context.Context, args any) error {
	fmt.Printf("执行任务: %s (ID: %s)\n", t.name, t.id)
	
	// 模拟任务执行
	select {
	case <-time.After(2 * time.Second):
		fmt.Printf("任务完成: %s\n", t.name)
		return nil
	case <-ctx.Done():
		fmt.Printf("任务被取消: %s\n", t.name)
		return ctx.Err()
	}
}

// TaskId 获取任务ID
// 返回:
//   - string 任务ID
func (t *ExampleTask) TaskId() string {
	return t.id
}

// TaskName 获取任务名称
// 返回:
//   - string 任务名称
func (t *ExampleTask) TaskName() string {
	return t.name
}

// IsContinue 是否继续执行
// 返回:
//   - bool 是否继续执行
func (t *ExampleTask) IsContinue() bool {
	return t.isContinue
}

// GetExecuteTime 获取执行时间
// 返回:
//   - time.Time 执行时间
func (t *ExampleTask) GetExecuteTime() time.Time {
	return t.executeTime
}

// GetPriority 获取优先级
// 返回:
//   - int 优先级
func (t *ExampleTask) GetPriority() int {
	return t.priority
}

// GetMaxRetries 获取最大重试次数
// 返回:
//   - int 最大重试次数
func (t *ExampleTask) GetMaxRetries() int {
	return t.maxRetries
}

// GetRetryDelay 获取重试延迟
// 返回:
//   - time.Duration 重试延迟
func (t *ExampleTask) GetRetryDelay() time.Duration {
	return t.retryDelay
}

// SetData 设置任务数据
// 参数:
//   - key: string 键
//   - value: any 值
func (t *ExampleTask) SetData(key string, value any) {
	t.data[key] = value
}

// GetData 获取任务数据
// 参数:
//   - key: string 键
// 返回:
//   - any 值
//   - bool 是否存在
func (t *ExampleTask) GetData(key string) (any, bool) {
	value, exists := t.data[key]
	return value, exists
}

// PeriodicTask 周期性任务
type PeriodicTask struct {
	*ExampleTask
	interval time.Duration
}

// NewPeriodicTask 创建周期性任务
// 参数:
//   - name: string 任务名称
//   - interval: time.Duration 执行间隔
//   - priority: int 优先级
// 返回:
//   - *PeriodicTask 周期性任务
func NewPeriodicTask(name string, interval time.Duration, priority int) *PeriodicTask {
	return &PeriodicTask{
		ExampleTask: NewExampleTask(name, time.Now(), priority, true),
		interval:    interval,
	}
}

// GetExecuteTime 获取下次执行时间
// 返回:
//   - time.Time 下次执行时间
func (t *PeriodicTask) GetExecuteTime() time.Time {
	return time.Now().Add(t.interval)
}