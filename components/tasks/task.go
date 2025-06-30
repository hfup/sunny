package tasks

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/hfup/sunny"
	"github.com/sirupsen/logrus"
)


// TaskInf 任务接口
type TaskInf interface {
	sunny.RunAbleInf
	TaskId() string                // 任务id
	TaskName() string              // 任务名称
	IsContinue() bool              // 是否持续执行 
	Stop() error                   // 停止任务
	Continue() error               // 继续任务
	IsCycle() bool                 // 是否循环执行
	CycleTime() time.Duration      // 循环时间
	GetExecuteTime() time.Time     // 获取执行时间
}

// TaskManagerInf 任务管理器接口
type TaskManagerInf interface {
	sunny.SubServiceInf
	AddTask(task TaskInf) error
	RemoveTask(taskId string) error
	GetTask(taskId string) (TaskInf, error)
	GetAllTasks() []TaskInf
	GetTaskCount() int
	PauseTask(taskId string) error   // 暂停任务
	ResumeTask(taskId string) error // 恢复任务
}

// TaskInitHandler 任务初始化处理函数
type TaskInitHandler func(ctx context.Context) ([]TaskInf, error)



// 任务包装器
type TaskWrapper struct {
	Task TaskInf
	IsRunning bool
}

type TaskManger struct {
	sunny.SubServiceInf
	taskInitHandler TaskInitHandler
	taskMap map[string]TaskWrapper
	taskLock sync.RWMutex
}

func NewTaskManger(taskInitHandler TaskInitHandler) *TaskManger {
	return &TaskManger{
		taskInitHandler: taskInitHandler,
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
	t.taskMap[task.TaskId()] = TaskWrapper{
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

	if _, ok := t.taskMap[taskId]; !ok {
		return errors.New("task not found")
	}

	t.taskMap[taskId].Task.Stop()
	return nil
}


// 恢复任务
func (t *TaskManger) ResumeTask(taskId string) error {
	if taskId == "" {
		return errors.New("task id is empty")
	}
	t.taskLock.Lock()
	defer t.taskLock.Unlock()

	if _, ok := t.taskMap[taskId]; !ok {
		return errors.New("task not found")
	}

	t.taskMap[taskId].Task.Continue()
	return nil
}

// 启动任务
func (t *TaskManger) Start(ctx context.Context,args any,resultChan chan<- sunny.Result) error {
	taskList, err := t.taskInitHandler(ctx)
	if err != nil {
		return err
	}

	for _, task := range taskList {
		err = t.AddTask(task)
		if err != nil {
			return err
		}
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// 检查任务
			for _, task := range t.taskMap {
				if task.IsRunning {
					continue
				}
				
				
			}
		}
	}
}

func (t *TaskManger) IsErrorStop() bool {
	return false
}

func (t *TaskManger) ServiceName() string {
	return "task_manger"
}