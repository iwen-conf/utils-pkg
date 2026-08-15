package errors

import (
	stderrors "errors"
	"testing"
)

func TestLegacyErrorWrapAndBuilder(t *testing.T) {
	root := stderrors.New("io")
	err := Wrap(root, CodeInternal, "failed")
	if err.Unwrap() != root {
		t.Fatalf("unwrap")
	}
	if !Is(err, root) {
		t.Fatalf("Is through Wrap")
	}

	built := NewBuilder().Code(CodeTimeout).Message("slow").Details("x").Severity(SeverityHigh).Category(CategoryNetwork).Build()
	if built.Code != CodeTimeout || GetSeverity(built) != SeverityHigh {
		t.Fatalf("builder: %+v", built)
	}
}

func TestMergeAndCloneIsolation(t *testing.T) {
	if Merge() != nil {
		t.Fatalf("empty merge")
	}
	one := New(CodeNotFound, "missing")
	if Merge(nil, one) != one {
		t.Fatalf("single merge")
	}
	merged := Merge(one, stderrors.New("other"))
	if merged.Code != "MULTIPLE_ERRORS" {
		t.Fatalf("multi merge: %s", merged.Code)
	}

	src := New(CodeInternal, "x")
	src.WithContext("meta", map[string]interface{}{"k": "v"})
	src.WithContext("tags", []string{"a"})
	cloned := Clone(src)
	cloned.Context["meta"].(map[string]interface{})["k"] = "mut"
	if src.Context["meta"].(map[string]interface{})["k"] != "v" {
		t.Fatalf("clone leaked nested map")
	}
}

func TestIsRetryableAndCritical(t *testing.T) {
	if !IsRetryable(New(CodeTimeout, "t")) {
		t.Fatalf("timeout should be retryable")
	}
	if IsRetryable(New(CodeNotFound, "n")) {
		t.Fatalf("not found is not retryable")
	}
	crit := New(CodeInternal, "boom").WithContext("severity", SeverityCritical)
	if !IsCritical(crit) {
		t.Fatalf("critical")
	}
}
