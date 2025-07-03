package mqs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileFailedMessageStore 基于本地文件的失败消息存储实现
// 实现了 FailedMessageStore 接口
type FileFailedMessageStore struct {
	filePath string      // 存储文件路径
	mu       sync.RWMutex // 读写锁，保护文件操作
}

// failedMessagesFile 文件存储格式
type failedMessagesFile struct {
	Messages map[string]*Message `json:"messages"` // 使用 ID 作为键存储消息
}

// NewFileFailedMessageStore 创建基于文件的失败消息存储
// 参数:
//   - filePath: string 存储文件路径
// 返回:
//   - *FileFailedMessageStore 存储实例
//   - error 错误信息
func NewFileFailedMessageStore(filePath string) (*FileFailedMessageStore, error) {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}
	
	store := &FileFailedMessageStore{
		filePath: filePath,
	}
	
	// 如果文件不存在，创建空文件
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := store.saveToFile(map[string]*Message{}); err != nil {
			return nil, fmt.Errorf("初始化存储文件失败: %w", err)
		}
	}
	
	return store, nil
}

// Store 存储失败的消息
// 参数:
//   - msg: *Message 失败的消息
// 返回:
//   - error 错误信息
func (s *FileFailedMessageStore) Store(msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 读取现有消息
	messages, err := s.loadFromFile()
	if err != nil {
		return fmt.Errorf("读取存储文件失败: %w", err)
	}
	
	// 添加新消息
	messages[msg.ID] = msg
	
	// 保存到文件
	if err := s.saveToFile(messages); err != nil {
		return fmt.Errorf("保存到文件失败: %w", err)
	}
	
	return nil
}

// Retrieve 检索所有失败的消息
// 返回:
//   - []*Message 失败消息列表
//   - error 错误信息
func (s *FileFailedMessageStore) Retrieve() ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	messages, err := s.loadFromFile()
	if err != nil {
		return nil, fmt.Errorf("读取存储文件失败: %w", err)
	}
	
	// 转换为切片
	result := make([]*Message, 0, len(messages))
	for _, msg := range messages {
		result = append(result, msg)
	}
	
	return result, nil
}

// Remove 移除已成功重发的消息
// 参数:
//   - id: string 消息ID
// 返回:
//   - error 错误信息
func (s *FileFailedMessageStore) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 读取现有消息
	messages, err := s.loadFromFile()
	if err != nil {
		return fmt.Errorf("读取存储文件失败: %w", err)
	}
	
	// 删除指定消息
	delete(messages, id)
	
	// 保存到文件
	if err := s.saveToFile(messages); err != nil {
		return fmt.Errorf("保存到文件失败: %w", err)
	}
	
	return nil
}

// Clear 清空所有失败消息
// 返回:
//   - error 错误信息
func (s *FileFailedMessageStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 保存空的消息映射
	if err := s.saveToFile(map[string]*Message{}); err != nil {
		return fmt.Errorf("清空文件失败: %w", err)
	}
	
	return nil
}

// Count 获取失败消息数量
// 返回:
//   - int 消息数量
//   - error 错误信息
func (s *FileFailedMessageStore) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	messages, err := s.loadFromFile()
	if err != nil {
		return 0, fmt.Errorf("读取存储文件失败: %w", err)
	}
	
	return len(messages), nil
}

// GetFilePath 获取存储文件路径
// 返回:
//   - string 文件路径
func (s *FileFailedMessageStore) GetFilePath() string {
	return s.filePath
}

// loadFromFile 从文件加载消息
// 返回:
//   - map[string]*Message 消息映射
//   - error 错误信息
func (s *FileFailedMessageStore) loadFromFile() (map[string]*Message, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，返回空映射
			return make(map[string]*Message), nil
		}
		return nil, err
	}
	
	// 如果文件为空，返回空映射
	if len(data) == 0 {
		return make(map[string]*Message), nil
	}
	
	var fileData failedMessagesFile
	if err := json.Unmarshal(data, &fileData); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}
	
	if fileData.Messages == nil {
		fileData.Messages = make(map[string]*Message)
	}
	
	return fileData.Messages, nil
}

// saveToFile 保存消息到文件
// 参数:
//   - messages: map[string]*Message 消息映射
// 返回:
//   - error 错误信息
func (s *FileFailedMessageStore) saveToFile(messages map[string]*Message) error {
	fileData := failedMessagesFile{
		Messages: messages,
	}
	
	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化JSON失败: %w", err)
	}
	
	// 写入临时文件，然后原子性重命名，避免写入过程中的数据损坏
	tempPath := s.filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	
	if err := os.Rename(tempPath, s.filePath); err != nil {
		// 清理临时文件
		os.Remove(tempPath)
		return fmt.Errorf("重命名文件失败: %w", err)
	}
	
	return nil
}

// Backup 备份存储文件
// 参数:
//   - backupPath: string 备份文件路径
// 返回:
//   - error 错误信息
func (s *FileFailedMessageStore) Backup(backupPath string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// 确保备份目录存在
	dir := filepath.Dir(backupPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建备份目录失败: %w", err)
	}
	
	// 读取原文件
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("读取原文件失败: %w", err)
	}
	
	// 写入备份文件
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("写入备份文件失败: %w", err)
	}
	
	return nil
}

// Restore 从备份文件恢复
// 参数:
//   - backupPath: string 备份文件路径
// 返回:
//   - error 错误信息
func (s *FileFailedMessageStore) Restore(backupPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 读取备份文件
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("读取备份文件失败: %w", err)
	}
	
	// 验证JSON格式
	var fileData failedMessagesFile
	if err := json.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("备份文件格式错误: %w", err)
	}
	
	// 写入当前存储文件
	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("恢复文件失败: %w", err)
	}
	
	return nil
}