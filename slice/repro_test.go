package slice

import (
	"math"
	"reflect"
	"testing"
)

func TestRepro_UnionDeduplicatesAndClonesWhenOneEmpty(t *testing.T) {
	src := []int{1, 1, 2}
	got := Union([]int{}, src)
	if !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("BUG: Union([]int{}, [1,1,2]) did not deduplicate: got %v, want [1 2]", got)
	}
	got[0] = 99
	if src[0] != 1 {
		t.Fatalf("BUG: Union aliased input slice backing array")
	}
}

func TestRepro_DifferenceClonesWhenSecondEmpty(t *testing.T) {
	src := []int{1, 2, 2, 3}
	got := Difference(src, []int{})
	got[0] = 99
	if src[0] != 1 {
		t.Fatalf("BUG: Difference aliased input slice backing array")
	}
}

func TestRepro_SortHandlesNaNWithoutViolatingOrdering(t *testing.T) {
	input := []float64{3.0, math.NaN(), 1.0, 2.0}
	got := Sort(input)
	if !math.IsNaN(got[0]) || got[1] != 1.0 || got[2] != 2.0 || got[3] != 3.0 {
		t.Fatalf("BUG: Sort failed total ordering with NaN: got %v", got)
	}
}
