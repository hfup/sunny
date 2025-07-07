package sequences

// 自增id 生成器
type SequenceInf interface {
	Next() (string, error)
}