package sequences

import (
	"fmt"
	"math"
	"sync"
	"strconv"
)

// 基于本地内存的id生成器
type LocalSequence struct {
	current int64
	mu sync.Mutex
	strLen int // 字符串长度
	maxValue int64 // 最大值
}

func NewLocalSequence(strLen int) *LocalSequence {
	maxValue := int64(math.Pow10(strLen) - 1)
	return &LocalSequence{
		current: 0,
		mu: sync.Mutex{},
		strLen: strLen,
		maxValue: maxValue,
	}
}

// 生成下一个id
// 返回:
// - 下一个id
// - 错误
func (l *LocalSequence) Next() (string,error){
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.current >= l.maxValue {
		l.current = 0
	}
	l.current ++

	fmtEx:="%0"+strconv.Itoa(l.strLen)+"d"
	fmtVal:=fmt.Sprintf(fmtEx,l.current)

	return fmtVal, nil

}
