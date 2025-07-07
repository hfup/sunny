package sequences

// 自增id 生成器
type SequenceInterface interface {
	Next() (string, error)
}