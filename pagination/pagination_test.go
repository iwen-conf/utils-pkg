package pagination

import (
	"testing"
	
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func TestParseLimitOffset(t *testing.T) {
	tests := []struct {
		name      string
		limitStr  string
		offsetStr string
		wantLimit int
		wantOffset int
		wantErr   bool
	}{
		{
			name:      "有效参数",
			limitStr:  "20",
			offsetStr: "40",
			wantLimit: 20,
			wantOffset: 40,
			wantErr:   false,
		},
		{
			name:      "默认最大值",
			limitStr:  "100",
			offsetStr: "0",
			wantLimit: 100,
			wantOffset: 0,
			wantErr:   false,
		},
		{
			name:      "默认最小值",
			limitStr:  "1",
			offsetStr: "0",
			wantLimit: 1,
			wantOffset: 0,
			wantErr:   false,
		},
		{
			name:      "无效的limit",
			limitStr:  "abc",
			offsetStr: "0",
			wantLimit: 0,
			wantOffset: 0,
			wantErr:   true,
		},
		{
			name:      "无效的offset",
			limitStr:  "10",
			offsetStr: "xyz",
			wantLimit: 0,
			wantOffset: 0,
			wantErr:   true,
		},
		{
			name:      "limit超出范围(过大)",
			limitStr:  "101",
			offsetStr: "0",
			wantLimit: 0,
			wantOffset: 0,
			wantErr:   true,
		},
		{
			name:      "limit超出范围(过小)",
			limitStr:  "0",
			offsetStr: "0",
			wantLimit: 0,
			wantOffset: 0,
			wantErr:   true,
		},
		{
			name:      "offset为负数",
			limitStr:  "10",
			offsetStr: "-1",
			wantLimit: 0,
			wantOffset: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLimit, gotOffset, err := ParseLimitOffset(tt.limitStr, tt.offsetStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLimitOffset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotLimit != tt.wantLimit {
				t.Errorf("ParseLimitOffset() gotLimit = %v, want %v", gotLimit, tt.wantLimit)
			}
			if gotOffset != tt.wantOffset {
				t.Errorf("ParseLimitOffset() gotOffset = %v, want %v", gotOffset, tt.wantOffset)
			}
		})
	}
}

func TestParseLimitOffsetWithRange(t *testing.T) {
	tests := []struct {
		name      string
		limitStr  string
		offsetStr string
		minLimit  int
		maxLimit  int
		wantLimit int
		wantOffset int
		wantErr   bool
	}{
		{
			name:      "自定义范围内的有效参数",
			limitStr:  "15",
			offsetStr: "30",
			minLimit:  5,
			maxLimit:  50,
			wantLimit: 15,
			wantOffset: 30,
			wantErr:   false,
		},
		{
			name:      "自定义范围外的参数(过小)",
			limitStr:  "3",
			offsetStr: "0",
			minLimit:  5,
			maxLimit:  50,
			wantLimit: 0,
			wantOffset: 0,
			wantErr:   true,
		},
		{
			name:      "自定义范围外的参数(过大)",
			limitStr:  "60",
			offsetStr: "0",
			minLimit:  5,
			maxLimit:  50,
			wantLimit: 0,
			wantOffset: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLimit, gotOffset, err := ParseLimitOffsetWithRange(tt.limitStr, tt.offsetStr, tt.minLimit, tt.maxLimit)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLimitOffsetWithRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotLimit != tt.wantLimit {
				t.Errorf("ParseLimitOffsetWithRange() gotLimit = %v, want %v", gotLimit, tt.wantLimit)
			}
			if gotOffset != tt.wantOffset {
				t.Errorf("ParseLimitOffsetWithRange() gotOffset = %v, want %v", gotOffset, tt.wantOffset)
			}
		})
	}
}

func TestGetPaginationParams(t *testing.T) {
	tests := []struct {
		name      string
		params    map[string]string
		limitKey  string
		offsetKey string
		wantLimit int
		wantOffset int
		wantErr   bool
	}{
		{
			name: "默认键名",
			params: map[string]string{
				"limit":  "20",
				"offset": "40",
			},
			limitKey:  "",
			offsetKey: "",
			wantLimit: 20,
			wantOffset: 40,
			wantErr:   false,
		},
		{
			name: "自定义键名",
			params: map[string]string{
				"page_size": "30",
				"start":     "60",
			},
			limitKey:  "page_size",
			offsetKey: "start",
			wantLimit: 30,
			wantOffset: 60,
			wantErr:   false,
		},
		{
			name: "缺少参数(使用默认值)",
			params: map[string]string{},
			limitKey:  "",
			offsetKey: "",
			wantLimit: 100, // 默认最大值
			wantOffset: 0,   // 默认从0开始
			wantErr:   false,
		},
		{
			name: "无效参数",
			params: map[string]string{
				"limit":  "abc",
				"offset": "0",
			},
			limitKey:  "",
			offsetKey: "",
			wantLimit: 0,
			wantOffset: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLimit, gotOffset, err := GetPaginationParams(tt.params, tt.limitKey, tt.offsetKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPaginationParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotLimit != tt.wantLimit {
				t.Errorf("GetPaginationParams() gotLimit = %v, want %v", gotLimit, tt.wantLimit)
			}
			if gotOffset != tt.wantOffset {
				t.Errorf("GetPaginationParams() gotOffset = %v, want %v", gotOffset, tt.wantOffset)
			}
		})
	}
}

func TestGetPaginationParamsFromContext(t *testing.T) {
	tests := []struct {
		name       string
		queryParams map[string]string
		wantLimit  int
		wantOffset int
		wantErr    bool
	}{
		{
			name: "有效参数",
			queryParams: map[string]string{
				"limit":  "20",
				"offset": "40",
			},
			wantLimit:  20,
			wantOffset: 40,
			wantErr:    false,
		},
		{
			name: "缺少参数(使用默认值)",
			queryParams: map[string]string{},
			wantLimit:  100, // 默认最大值
			wantOffset: 0,   // 默认从0开始
			wantErr:    false,
		},
		{
			name: "无效参数",
			queryParams: map[string]string{
				"limit":  "abc",
				"offset": "0",
			},
			wantLimit:  0,
			wantOffset: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的请求上下文
			req := protocol.NewRequest(consts.MethodGet, "/test", nil)
			queryString := ""
			for k, v := range tt.queryParams {
				if queryString != "" {
					queryString += "&"
				}
				queryString += k + "=" + v
			}
			req.SetQueryString(queryString)
			c := app.RequestContext{}
			c.Request = *req

			gotLimit, gotOffset, err := GetPaginationParamsFromContext(&c)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPaginationParamsFromContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotLimit != tt.wantLimit {
				t.Errorf("GetPaginationParamsFromContext() gotLimit = %v, want %v", gotLimit, tt.wantLimit)
			}
			if gotOffset != tt.wantOffset {
				t.Errorf("GetPaginationParamsFromContext() gotOffset = %v, want %v", gotOffset, tt.wantOffset)
			}
		})
	}
}

func TestCalculateTotalPages(t *testing.T) {
	tests := []struct {
		name       string
		totalItems int
		limit      int
		want       int
	}{
		{
			name:       "整除",
			totalItems: 100,
			limit:      10,
			want:       10,
		},
		{
			name:       "不整除",
			totalItems: 101,
			limit:      10,
			want:       11,
		},
		{
			name:       "零项目",
			totalItems: 0,
			limit:      10,
			want:       0,
		},
		{
			name:       "无效limit",
			totalItems: 100,
			limit:      0,
			want:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateTotalPages(tt.totalItems, tt.limit); got != tt.want {
				t.Errorf("CalculateTotalPages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateCurrentPage(t *testing.T) {
	tests := []struct {
		name   string
		offset int
		limit  int
		want   int
	}{
		{
			name:   "第一页",
			offset: 0,
			limit:  10,
			want:   1,
		},
		{
			name:   "第二页",
			offset: 10,
			limit:  10,
			want:   2,
		},
		{
			name:   "不在页边界",
			offset: 15,
			limit:  10,
			want:   2, // 15/10 + 1 = 2.5 -> 2
		},
		{
			name:   "无效limit",
			offset: 10,
			limit:  0,
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateCurrentPage(tt.offset, tt.limit); got != tt.want {
				t.Errorf("CalculateCurrentPage() = %v, want %v", got, tt.want)
			}
		})
	}
}