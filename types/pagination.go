package types

var (
	defaultMaxPageSize = 100 // 默认最大分页大小
)

func SetDefaultMaxPageSize(size int) {
	defaultMaxPageSize = size
}

type Pagination[T any] struct {
	PageSize int `json:"page_size"`
	PageNo   int `json:"page_no"`
	SearchOption T `json:"search_option"` // 搜索条件
}

func (p *Pagination[T]) Offset() int {
	if p.PageNo <= 0 {
		return 0
	}
	return (p.PageNo - 1) * p.PageSize
}

func (p *Pagination[T]) Limit() int {
	if p.PageSize <= 0 || p.PageSize > defaultMaxPageSize {
		return 10
	}
	return p.PageSize
}