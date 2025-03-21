package slice

import (
	"reflect"
	"testing"
)

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		element  string
		expected bool
	}{
		{"存在的元素", []string{"a", "b", "c"}, "b", true},
		{"不存在的元素", []string{"a", "b", "c"}, "d", false},
		{"空切片", []string{}, "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Contains(tt.slice, tt.element); got != tt.expected {
				t.Errorf("Contains() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int
		expected []int
	}{
		{"有重复元素", []int{1, 2, 2, 3, 3, 4}, []int{1, 2, 3, 4}},
		{"无重复元素", []int{1, 2, 3, 4}, []int{1, 2, 3, 4}},
		{"空切片", []int{}, []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Unique(tt.slice); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Unique() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIntersection(t *testing.T) {
	tests := []struct {
		name     string
		slice1   []int
		slice2   []int
		expected []int
	}{
		{"有交集", []int{1, 2, 3}, []int{2, 3, 4}, []int{2, 3}},
		{"无交集", []int{1, 2}, []int{3, 4}, []int{}},
		{"空切片", []int{}, []int{1, 2}, []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Intersection(tt.slice1, tt.slice2); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Intersection() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUnion(t *testing.T) {
	tests := []struct {
		name     string
		slice1   []int
		slice2   []int
		expected []int
	}{
		{"有重复元素", []int{1, 2}, []int{2, 3}, []int{1, 2, 3}},
		{"无重复元素", []int{1, 2}, []int{3, 4}, []int{1, 2, 3, 4}},
		{"空切片", []int{}, []int{1, 2}, []int{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Union(tt.slice1, tt.slice2); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Union() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDifference(t *testing.T) {
	tests := []struct {
		name     string
		slice1   []int
		slice2   []int
		expected []int
	}{
		{"有差集", []int{1, 2, 3}, []int{2, 3}, []int{1}},
		{"无差集", []int{1, 2}, []int{1, 2}, []int{}},
		{"空切片", []int{}, []int{1, 2}, []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Difference(tt.slice1, tt.slice2); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Difference() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFilter(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int
		pred     func(int) bool
		expected []int
	}{
		{"过滤偶数", []int{1, 2, 3, 4}, func(x int) bool { return x%2 == 0 }, []int{2, 4}},
		{"过滤全部", []int{1, 2, 3}, func(x int) bool { return false }, []int{}},
		{"空切片", []int{}, func(x int) bool { return true }, []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Filter(tt.slice, tt.pred); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Filter() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMap(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int
		trans    func(int) string
		expected []string
	}{
		{"转换为字符串", []int{1, 2}, func(x int) string { return string(rune('A' - 1 + x)) }, []string{"A", "B"}},
		{"空切片", []int{}, func(x int) string { return "" }, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Map(tt.slice, tt.trans); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Map() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestReduce(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int
		initial  int
		reducer  func(int, int) int
		expected int
	}{
		{"求和", []int{1, 2, 3}, 0, func(acc, x int) int { return acc + x }, 6},
		{"空切片", []int{}, 0, func(acc, x int) int { return acc + x }, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Reduce(tt.slice, tt.initial, tt.reducer); got != tt.expected {
				t.Errorf("Reduce() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSort(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int
		expected []int
	}{
		{"乱序数组", []int{3, 1, 4, 1, 5, 9}, []int{1, 1, 3, 4, 5, 9}},
		{"已排序数组", []int{1, 2, 3, 4}, []int{1, 2, 3, 4}},
		{"空切片", []int{}, []int{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sort(tt.slice); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Sort() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func BenchmarkSliceOperations(b *testing.B) {
	slice1 := []int{1, 2, 3, 4, 5}
	slice2 := []int{4, 5, 6, 7, 8}

	b.Run("Contains", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Contains(slice1, 3)
		}
	})

	b.Run("Unique", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Unique(slice1)
		}
	})

	b.Run("Intersection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Intersection(slice1, slice2)
		}
	})

	b.Run("Union", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Union(slice1, slice2)
		}
	})

	b.Run("Difference", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Difference(slice1, slice2)
		}
	})
}

func TestSliceOperations_Parallel(t *testing.T) {
	tests := []struct {
		name     string
		slice1   []int
		slice2   []int
		expected []int
		opType   string
	}{
		{
			name:     "交集-有重叠元素",
			slice1:   []int{1, 2, 3},
			slice2:   []int{2, 3, 4},
			expected: []int{2, 3},
			opType:   "intersection",
		},
		{
			name:     "并集-有重复元素",
			slice1:   []int{1, 2},
			slice2:   []int{2, 3},
			expected: []int{1, 2, 3},
			opType:   "union",
		},
		{
			name:     "差集-有差异元素",
			slice1:   []int{1, 2, 3},
			slice2:   []int{2, 3},
			expected: []int{1},
			opType:   "difference",
		},
	}

	for _, tt := range tests {
		tt := tt // 捕获循环变量
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var result []int
			switch tt.opType {
			case "intersection":
				result = Intersection(tt.slice1, tt.slice2)
			case "union":
				result = Union(tt.slice1, tt.slice2)
			case "difference":
				result = Difference(tt.slice1, tt.slice2)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("%s = %v, want %v", tt.opType, result, tt.expected)
			}
		})
	}
}