package pagination

import (
	"math"
	"testing"
)

func TestRepro_OffsetOverflowDoesNotProduceNegativeOrWrongBounds(t *testing.T) {
	req := OffsetRequest{
		Offset: math.MaxInt - 5,
		Limit:  20,
	}
	if next := req.GetNextOffset(); next < req.Offset {
		t.Fatalf("BUG: GetNextOffset overflowed int: %d", next)
	}
	if !req.IsLastPage(100) {
		t.Fatalf("BUG: IsLastPage overflowed int and returned false for MaxInt-5 vs total=100")
	}
	var resp OffsetResponse
	resp.Calculate(req, 100, 20)
	if resp.HasMore {
		t.Fatalf("BUG: Calculate overflowed int and reported HasMore=true beyond total")
	}
}
