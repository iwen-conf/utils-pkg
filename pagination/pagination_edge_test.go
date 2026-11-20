package pagination

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestPaginationNilAndBoundaryGuards(t *testing.T) {
	// Nil pointer safety on CursorRequest & OffsetRequest & OffsetResponse
	var nilCursor *CursorRequest
	nilCursor.Normalize()

	var nilOffset *OffsetRequest
	nilOffset.Normalize()
	if nilOffset.GetNextOffset() != 0 {
		t.Fatalf("nil GetNextOffset should be 0")
	}
	if nilOffset.GetPrevOffset() != 0 {
		t.Fatalf("nil GetPrevOffset should be 0")
	}
	if !nilOffset.IsFirstPage() {
		t.Fatalf("nil IsFirstPage should be true")
	}
	if !nilOffset.IsLastPage(10) {
		t.Fatalf("nil IsLastPage should be true")
	}

	var nilResp *OffsetResponse
	nilResp.Calculate(OffsetRequest{Offset: 0, Limit: 10}, 100, 10)

	// Unnormalized negative / zero edge cases on OffsetRequest & OffsetResponse
	negReq := OffsetRequest{Offset: -10, Limit: 0}
	if got := negReq.GetNextOffset(); got != 0 {
		t.Fatalf("negative offset + zero limit GetNextOffset = %d, want 0", got)
	}
	posNoLimit := OffsetRequest{Offset: 30, Limit: 0}
	if got := posNoLimit.GetPrevOffset(); got != 30 {
		t.Fatalf("zero limit GetPrevOffset = %d, want 30", got)
	}
	negLimit := OffsetRequest{Offset: -5, Limit: -3}
	if !negLimit.IsLastPage(0) {
		t.Fatalf("total=0 should be last page")
	}
	if negLimit.IsLastPage(10) {
		t.Fatalf("clamped 0+0 < 10 should not be last page")
	}

	var resp OffsetResponse
	resp.Calculate(OffsetRequest{Offset: -5, Limit: 10}, -100, -3)
	if resp.Offset != 0 || resp.Total != 0 || resp.HasMore {
		t.Fatalf("negative Calculate inputs were not clamped: %+v", resp)
	}

	// Int64 saturation guard
	hugeReq := OffsetRequest{Offset: math.MaxInt, Limit: math.MaxInt}
	if !hugeReq.IsLastPage(math.MaxInt64) {
		t.Fatalf("saturated int64 end should reach MaxInt64")
	}

	// MaxCursorLength & empty cursor guards on Base64JSONCodec and HMACCodec
	key := []byte("0123456789abcdef0123456789abcdef")
	hmacCodec, err := NewHMACCodec(key)
	if err != nil {
		t.Fatalf("NewHMACCodec: %v", err)
	}
	var out testPayload
	if err := hmacCodec.Decode("", &out); !errors.Is(err, ErrEmptyCursor) {
		t.Fatalf("HMACCodec empty cursor: got %v, want ErrEmptyCursor", err)
	}
	oversized := strings.Repeat("a", MaxCursorLength+1)
	if err := hmacCodec.Decode(oversized, &out); !errors.Is(err, ErrInvalidCursorFormat) {
		t.Fatalf("HMACCodec oversized cursor: got %v, want ErrInvalidCursorFormat", err)
	}
	if err := (Base64JSONCodec{}).Decode(oversized, &out); !errors.Is(err, ErrInvalidCursorFormat) {
		t.Fatalf("Base64JSONCodec oversized cursor: got %v, want ErrInvalidCursorFormat", err)
	}
}
