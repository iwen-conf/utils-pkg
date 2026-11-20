package pagination

import "math"

// OffsetRequest 表示基于偏移量的分页请求参数。
// - Offset: 跳过的记录数（从 0 开始）
// - Limit: 本页希望返回的最大记录数
type OffsetRequest struct {
	Offset int `json:"offset,omitempty" form:"offset"`
	Limit  int `json:"limit,omitempty" form:"limit"`
}

// OffsetResponse 为基于偏移量的分页响应元信息。
type OffsetResponse struct {
	Offset  int   `json:"offset"`   // 当前偏移量
	Limit   int   `json:"limit"`    // 每页条数
	Total   int64 `json:"total"`    // 总记录数
	HasMore bool  `json:"has_more"` // 是否还有更多数据
}

// Normalize 对请求的 Offset 和 Limit 进行归一化（默认值与上限钳制）。
// Offset 必须 >= 0，Limit 应用默认值与上限。
func (r *OffsetRequest) Normalize() {
	if r == nil {
		return
	}
	if r.Offset < 0 {
		r.Offset = 0
	}
	if r.Limit < MinLimit {
		r.Limit = DefaultLimit
	}
	if r.Limit > MaxLimit {
		r.Limit = MaxLimit
	}
}

// GetNextOffset 计算下一页的偏移量（做饱和加法保护，防止超大 Offset 发生 int 溢出）。
func (r *OffsetRequest) GetNextOffset() int {
	if r == nil {
		return 0
	}
	offset := r.Offset
	if offset < 0 {
		offset = 0
	}
	limit := r.Limit
	if limit <= 0 {
		return offset
	}
	if offset > math.MaxInt-limit {
		return math.MaxInt
	}
	return offset + limit
}

// GetPrevOffset 计算上一页的偏移量。
// 如果已是第一页，返回 0。
func (r *OffsetRequest) GetPrevOffset() int {
	if r == nil || r.Offset <= 0 {
		return 0
	}
	if r.Limit <= 0 {
		return r.Offset
	}
	if r.Offset <= r.Limit {
		return 0
	}
	return r.Offset - r.Limit
}

func saturatingAddInt64(a, b int64) int64 {
	if b > 0 && a > math.MaxInt64-b {
		return math.MaxInt64
	}
	return a + b
}

// Calculate 根据总记录数计算分页元信息。
// total: 符合条件的总记录数
// actualCount: 本次查询实际返回的记录数
func (r *OffsetResponse) Calculate(req OffsetRequest, total int64, actualCount int) {
	if r == nil {
		return
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}
	if total < 0 {
		total = 0
	}
	if actualCount < 0 {
		actualCount = 0
	}
	r.Offset = offset
	r.Limit = req.Limit
	r.Total = total
	end := saturatingAddInt64(int64(offset), int64(actualCount))
	r.HasMore = end < total
}

// IsFirstPage 判断是否为第一页。
func (r *OffsetRequest) IsFirstPage() bool {
	if r == nil {
		return true
	}
	return r.Offset <= 0
}

// IsLastPage 判断是否为最后一页（需要知道总记录数；防止超大 Offset 溢出）。
func (r *OffsetRequest) IsLastPage(total int64) bool {
	if r == nil || total <= 0 {
		return true
	}
	offset := r.Offset
	if offset < 0 {
		offset = 0
	}
	limit := r.Limit
	if limit < 0 {
		limit = 0
	}
	end := saturatingAddInt64(int64(offset), int64(limit))
	return end >= total
}
