package errors

import (
	"testing"
	"unicode/utf8"
)

func TestValidator_InNotInDeepEqual(t *testing.T) {
	v := NewValidator()
	v.In("tags", []string{"a"}, []interface{}{[]string{"a"}})
	if v.HasErrors() {
		t.Fatalf("slice value should match via DeepEqual")
	}
	v.In("tags", []string{"a"}, []interface{}{[]string{"b"}})
	if !v.HasErrors() {
		t.Fatalf("different slice should fail In")
	}
	v.Clear()
	v.NotIn("obj", map[string]int{"k": 1}, []interface{}{map[string]int{"k": 1}})
	if !v.HasErrors() {
		t.Fatalf("matching map should fail NotIn")
	}
}

func TestValidator_RuneLength(t *testing.T) {
	v := NewValidator()
	value := "中文AB"
	if utf8.RuneCountInString(value) != 4 {
		t.Fatalf("fixture")
	}
	v.Length("name", value, 4)
	if v.HasErrors() {
		t.Fatalf("rune length 4 should pass: %+v", v.GetErrors())
	}
	v.MinLength("name", value, 5)
	if !v.HasErrors() {
		t.Fatalf("min length 5 should fail")
	}
}

func TestIsEmptyKinds(t *testing.T) {
	var p *int
	var s []int
	if !isEmpty(nil) || !isEmpty(p) || !isEmpty(s) || !isEmpty("") || !isEmpty("  ") {
		t.Fatalf("expected empty values")
	}
	if !isEmpty(0) || !isEmpty(uint(0)) || !isEmpty(0.0) || !isEmpty(false) {
		t.Fatalf("zero values should be empty")
	}
	n := 1
	if isEmpty(&n) || isEmpty([]int{1}) || isEmpty("x") {
		t.Fatalf("non-empty values")
	}
}

func TestValidator_GetErrorAndClear(t *testing.T) {
	v := NewValidator().Required("name", "").Email("email", "bad")
	if v.GetError() == nil {
		t.Fatalf("expected combined error")
	}
	v.Clear()
	if v.HasErrors() || v.GetError() != nil {
		t.Fatalf("clear failed")
	}
	if err := ValidateLength("name", "中文", 2, 2); err != nil {
		t.Fatalf("ValidateLength runes: %v", err)
	}
}
