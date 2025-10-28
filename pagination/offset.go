package pagination

// OffsetRequest 表示基于偏移量的分页请求参数。
// - Offset: 跳过的记录数（从 0 开始）
// - Limit: 本页希望返回的最大记录数
type OffsetRequest struct {
	Offset int `json:"offset,omitempty" form:"offset"`
	Limit  int `json:"limit,omitempty" form:"limit"`
}

// OffsetResponse 为基于偏移量的分页响应元信息。
type OffsetResponse struct {
	Offset int   `json:"offset"`           // 当前偏移量
	Limit  int   `json:"limit"`            // 每页条数
	Total  int64 `json:"total"`            // 总记录数
	HasMore bool `json:"has_more"`         // 是否还有更多数据
}

// Normalize 对请求的 Offset 和 Limit 进行归一化（默认值与上限钳制）。
// Offset 必须 >= 0，Limit 应用默认值与上限。
func (r *OffsetRequest) Normalize() {
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

// GetNextOffset 计算下一页的偏移量。
// 如果当前页已是最后一页，返回当前偏移量。
func (r *OffsetRequest) GetNextOffset() int {
	return r.Offset + r.Limit
}

// GetPrevOffset 计算上一页的偏移量。
// 如果已是第一页，返回 0。
func (r *OffsetRequest) GetPrevOffset() int {
	prev := r.Offset - r.Limit
	if prev < 0 {
		return 0
	}
	return prev
}

// Calculate 根据总记录数计算分页元信息。
// total: 符合条件的总记录数
// actualCount: 本次查询实际返回的记录数
func (r *OffsetResponse) Calculate(req OffsetRequest, total int64, actualCount int) {
	r.Offset = req.Offset
	r.Limit = req.Limit
	r.Total = total
	// 判断是否还有下一页
	r.HasMore = int64(req.Offset+actualCount) < total
}

// IsFirstPage 判断是否为第一页。
func (r *OffsetRequest) IsFirstPage() bool {
	return r.Offset == 0
}

// IsLastPage 判断是否为最后一页（需要知道总记录数）。
func (r *OffsetRequest) IsLastPage(total int64) bool {
	return int64(r.Offset+r.Limit) >= total
}
