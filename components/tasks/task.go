package tasks

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/hfup/sunny/types"
	"github.com/sirupsen/logrus"
)


// TaskInf 任务接口
type TaskInf interface {
	Run(ctx context.Context) error // 执行任务
	TaskId() string                // 任务id
	TaskName() string              // 任务名称
	IsContinue() bool              // 是否持续执行 
	GetExecuteTime() time.Time     // 获取执行时间
}

// TaskManagerInf 任务管理器接口
type TaskManagerInf interface {
	types.SubServiceInf
	AddTask(task TaskInf) error
	RemoveTask(taskId string) error
	GetTask(taskId string) (TaskInf, error)
	GetAllTasks() []TaskInf
	GetTaskCount() int
	PauseTask(taskId string) error   // 暂停任务
	ResumeTask(taskId string) error // 恢复任务
	BindTaskInitHandler(taskInitHandler TaskInitHandler) error // 绑定任务初始化处理函数
}

// TaskInitHandler 任务初始化处理函数
type TaskInitHandler func(ctx context.Context) ([]TaskInf, error)



// 任务包装器
type TaskWrapper struct {
	Task TaskInf
	IsRunning bool
	IsStop bool
}

type TaskManger struct {
	interval time.Duration // 任务执行间隔
	types.SubServiceInf
	taskInitHandler TaskInitHandler // 任务初始化处理函数 从持久化层中获取任务 重启服务的时候
	taskMap map[string]*TaskWrapper
	taskLock sync.RWMutex
}

func NewTaskManger(interval time.Duration,taskInitHandler TaskInitHandler) *TaskManger {
	return &TaskManger{
		taskInitHandler: taskInitHandler,
		interval: interval,
	}
}


// 添加任务
func (t *TaskManger) AddTask(task TaskInf) error {
	if task == nil {
		return errors.New("task is nil")
	}
	if task.TaskId() == "" {
		return errors.New("task id is empty")
	}
	t.taskLock.Lock()
	defer t.taskLock.Unlock()
	if _, ok := t.taskMap[task.TaskId()]; ok {
		return errors.New("task already exists")
	}
	t.taskMap[task.TaskId()] = &TaskWrapper{
		Task: task,
		IsRunning: false,
	}
	return nil
}

func (t *TaskManger) RemoveTask(taskId string) error {
	if taskId == "" {
		return errors.New("task id is empty")
	}
	t.taskLock.Lock()
	defer t.taskLock.Unlock()

	if _, ok := t.taskMap[taskId]; !ok {
		return errors.New("task not found")
	}
	delete(t.taskMap, taskId)
	return nil
}


func (t *TaskManger) GetTask(taskId string) (TaskInf, error) {
	if taskId == "" {
		return nil, errors.New("task id is empty")
	}
	t.taskLock.RLock()
	defer t.taskLock.RUnlock()

	task, ok := t.taskMap[taskId]
	if !ok {
		return nil, errors.New("task not found")
	}
	return task.Task, nil
}

// 获取所有任务
func (t *TaskManger) GetAllTasks() []TaskInf {
	t.taskLock.RLock()
	defer t.taskLock.RUnlock()

	taskList := make([]TaskInf, 0)
	for _, task := range t.taskMap {
		taskList = append(taskList, task.Task)
	}
	return taskList
}

// 获取任务数量
func (t *TaskManger) GetTaskCount() int {
	t.taskLock.RLock()
	defer t.taskLock.RUnlock()
	return len(t.taskMap)
}


// 暂停任务
func (t *TaskManger) PauseTask(taskId string) error {
	if taskId == "" {
		return errors.New("task id is empty")
	}
	t.taskLock.Lock()
	defer t.taskLock.Unlock()

	val,ok := t.taskMap[taskId]
	if !ok {
		return errors.New("task not found")
	}
	val.IsStop = true
	return nil
}


// 恢复任务
func (t *TaskManger) ResumeTask(taskId string) error {
	if taskId == "" {
		return errors.New("task id is empty")
	}
	t.taskLock.Lock()
	defer t.taskLock.Unlock()

	val,ok := t.taskMap[taskId]
	if !ok {
		return errors.New("task not found")
	}
	val.IsStop = false
	return nil
}

// 启动任务
func (t *TaskManger) Start(ctx context.Context,args any,resultChan chan<- types.Result[any]) {
	if t.taskInitHandler != nil {
		taskList, err := t.taskInitHandler(ctx) // 初始化任务 
		if err != nil {
			logrus.Errorf("task init error: %v", err)
			resultChan <- types.Result[any]{
				ErrCode: 1,
				Message: "task manager start error;task init error: " + err.Error(),
			}
			return 
		}
		for _, task := range taskList {
			err = t.AddTask(task)
			if err != nil {
				logrus.Errorf("task add error: %v", err)
				resultChan <- types.Result[any]{
					ErrCode: 1,
					Message: "task manager start error;task add error: " + err.Error(),
				}
				return 
			}
		}
	}

	if t.interval == 0 {
		t.interval = 5 * time.Second
	}
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	resultChan <- types.Result[any]{
		ErrCode: 0,
		Message: "task manager start success",
	}

	for {
		select {
		case <-ctx.Done():
			return 
		case <-ticker.C:
			// 检查任务
			for _, task := range t.taskMap {
				if task.IsRunning { // 如果任务正在运行，则跳过
					continue
				}
				if task.IsStop { // 如果任务被暂停，则跳过
					continue
				}
				if task.Task.GetExecuteTime().Before(time.Now()) {
					task.IsRunning = true

					go func() {
						defer func() {
							// 如果任务执行失败，获取panic
							if r := recover(); r != nil {
								logrus.Errorf("task %s run panic: %v", task.Task.TaskId(), r)
							}
							task.IsRunning = false
						}()
						err := task.Task.Run(ctx)
						if err != nil {
							logrus.Errorf("task %s run error: %v", task.Task.TaskId(), err)
						}
					}()
				}
			}
		}
	}
}

func (t *TaskManger) IsErrorStop() bool {
	return false
}

func (t *TaskManger) ServiceName() string {
	return "Task任务管理器"
}

// 绑定任务初始化处理函数
// 参数:
//   - taskInitHandler: 任务初始化处理函数
// 返回:
//   - error 错误信息
func (t *TaskManger) BindTaskInitHandler(taskInitHandler TaskInitHandler) error {
	if taskInitHandler == nil {
		return errors.New("task init handler is nil")
	}
	t.taskInitHandler = taskInitHandler
	return nil
}