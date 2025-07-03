package mqs

import (
	"sync"
)

// MemoryFailedMessageStore 内存中的失败消息存储实现
// 实现了 FailedMessageStore 接口
type MemoryFailedMessageStore struct {
	messages map[string]*Message // 消息存储，使用 ID 作为键
	mu       sync.RWMutex        // 读写锁
}

// NewMemoryFailedMessageStore 创建内存失败消息存储
// 返回:
//   - *MemoryFailedMessageStore 存储实例
func NewMemoryFailedMessageStore() *MemoryFailedMessageStore {
	return &MemoryFailedMessageStore{
		messages: make(map[string]*Message),
	}
}

// Store 存储失败的消息
// 参数:
//   - msg: *Message 失败的消息
// 返回:
//   - error 错误信息
func (s *MemoryFailedMessageStore) Store(msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.messages[msg.ID] = msg
	return nil
}

// Retrieve 检索所有失败的消息
// 返回:
//   - []*Message 失败消息列表
//   - error 错误信息
func (s *MemoryFailedMessageStore) Retrieve() ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	messages := make([]*Message, 0, len(s.messages))
	for _, msg := range s.messages {
		messages = append(messages, msg)
	}
	
	return messages, nil
}

// Remove 移除已成功重发的消息
// 参数:
//   - id: string 消息ID
// 返回:
//   - error 错误信息
func (s *MemoryFailedMessageStore) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	delete(s.messages, id)
	return nil
}

// Clear 清空所有失败消息
// 返回:
//   - error 错误信息
func (s *MemoryFailedMessageStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.messages = make(map[string]*Message)
	return nil
}