package pagination

import "testing"

func TestOffsetRequest_Normalize(t *testing.T) {
	tests := []struct {
		name         string
		offset       int
		limit        int
		wantOffset   int
		wantLimit    int
	}{
		{"zero values", 0, 0, 0, DefaultLimit},
		{"normal values", 10, 20, 10, 20},
		{"negative offset", -5, 20, 0, 20},
		{"zero limit", 10, 0, 10, DefaultLimit},
		{"negative limit", 10, -1, 10, DefaultLimit},
		{"limit beyond max", 10, MaxLimit + 50, 10, MaxLimit},
		{"limit at min", 10, MinLimit, 10, MinLimit},
		{"limit at max", 10, MaxLimit, 10, MaxLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := OffsetRequest{Offset: tt.offset, Limit: tt.limit}
			r.Normalize()
			if r.Offset != tt.wantOffset {
				t.Errorf("Offset = %d, want %d", r.Offset, tt.wantOffset)
			}
			if r.Limit != tt.wantLimit {
				t.Errorf("Limit = %d, want %d", r.Limit, tt.wantLimit)
			}
		})
	}
}

func TestOffsetRequest_GetNextOffset(t *testing.T) {
	tests := []struct {
		name   string
		offset int
		limit  int
		want   int
	}{
		{"first page", 0, 20, 20},
		{"second page", 20, 20, 40},
		{"custom offset", 15, 10, 25},
		{"large offset", 1000, 50, 1050},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := OffsetRequest{Offset: tt.offset, Limit: tt.limit}
			got := r.GetNextOffset()
			if got != tt.want {
				t.Errorf("GetNextOffset() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestOffsetRequest_GetPrevOffset(t *testing.T) {
	tests := []struct {
		name   string
		offset int
		limit  int
		want   int
	}{
		{"first page", 0, 20, 0},
		{"second page", 20, 20, 0},
		{"third page", 40, 20, 20},
		{"custom offset", 25, 10, 15},
		{"offset less than limit", 15, 20, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := OffsetRequest{Offset: tt.offset, Limit: tt.limit}
			got := r.GetPrevOffset()
			if got != tt.want {
				t.Errorf("GetPrevOffset() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestOffsetRequest_IsFirstPage(t *testing.T) {
	tests := []struct {
		name   string
		offset int
		want   bool
	}{
		{"offset 0", 0, true},
		{"offset 1", 1, false},
		{"offset 20", 20, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := OffsetRequest{Offset: tt.offset}
			got := r.IsFirstPage()
			if got != tt.want {
				t.Errorf("IsFirstPage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOffsetRequest_IsLastPage(t *testing.T) {
	tests := []struct {
		name   string
		offset int
		limit  int
		total  int64
		want   bool
	}{
		{"first page of one", 0, 20, 10, true},
		{"first page of many", 0, 20, 100, false},
		{"last page exact", 80, 20, 100, true},
		{"last page partial", 90, 20, 100, true},
		{"middle page", 40, 20, 100, false},
		{"beyond total", 100, 20, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := OffsetRequest{Offset: tt.offset, Limit: tt.limit}
			got := r.IsLastPage(tt.total)
			if got != tt.want {
				t.Errorf("IsLastPage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOffsetResponse_Calculate(t *testing.T) {
	tests := []struct {
		name        string
		req         OffsetRequest
		total       int64
		actualCount int
		wantHasMore bool
	}{
		{
			name:        "first page full",
			req:         OffsetRequest{Offset: 0, Limit: 20},
			total:       100,
			actualCount: 20,
			wantHasMore: true,
		},
		{
			name:        "first page partial",
			req:         OffsetRequest{Offset: 0, Limit: 20},
			total:       15,
			actualCount: 15,
			wantHasMore: false,
		},
		{
			name:        "last page exact",
			req:         OffsetRequest{Offset: 80, Limit: 20},
			total:       100,
			actualCount: 20,
			wantHasMore: false,
		},
		{
			name:        "last page partial",
			req:         OffsetRequest{Offset: 90, Limit: 20},
			total:       100,
			actualCount: 10,
			wantHasMore: false,
		},
		{
			name:        "middle page",
			req:         OffsetRequest{Offset: 40, Limit: 20},
			total:       100,
			actualCount: 20,
			wantHasMore: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := OffsetResponse{}
			resp.Calculate(tt.req, tt.total, tt.actualCount)
			
			if resp.Offset != tt.req.Offset {
				t.Errorf("Offset = %d, want %d", resp.Offset, tt.req.Offset)
			}
			if resp.Limit != tt.req.Limit {
				t.Errorf("Limit = %d, want %d", resp.Limit, tt.req.Limit)
			}
			if resp.Total != tt.total {
				t.Errorf("Total = %d, want %d", resp.Total, tt.total)
			}
			if resp.HasMore != tt.wantHasMore {
				t.Errorf("HasMore = %v, want %v", resp.HasMore, tt.wantHasMore)
			}
		})
	}
}
