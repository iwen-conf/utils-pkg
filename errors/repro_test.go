package errors

import (
	"strings"
	"testing"
)

func TestRepro_RichErrorNilChainMethodsDoNotPanic(t *testing.T) {
	var e *RichError
	gotCode := e.WithCode(400001)
	if gotCode == nil || gotCode.Code != 400001 {
		t.Fatalf("expected non-nil RichError from nil.WithCode, got %+v", gotCode)
	}
	gotMsg := e.WithMsg("bad")
	if gotMsg == nil || gotMsg.Msg != "bad" {
		t.Fatalf("expected non-nil RichError from nil.WithMsg, got %+v", gotMsg)
	}
}

func TestRepro_CloneHandlesNilInsideMapAndSliceContext(t *testing.T) {
	err := New("CODE", "msg").
		WithContext("sliceWithNil", []interface{}{"a", nil, "b"}).
		WithContext("mapWithNil", map[string]interface{}{"k": nil, "v": 1})

	cloned := Clone(err)
	m := cloned.Context["mapWithNil"].(map[string]interface{})
	if _, exists := m["k"]; !exists {
		t.Fatalf("BUG: Clone deleted nil-valued key from context map")
	}
	s := cloned.Context["sliceWithNil"].([]interface{})
	if len(s) != 3 || s[1] != nil {
		t.Fatalf("BUG: Clone corrupted slice containing nil: %v", s)
	}
}

func TestRepro_StackSkipsShortcutHelperFrame(t *testing.T) {
	e := RichBadRequest("bad input")
	st := e.Stack()
	lines := strings.Split(strings.TrimSpace(st), "\n")
	if len(lines) == 0 {
		t.Fatalf("expected stack frames")
	}
	if strings.Contains(lines[0], "rich_api.go") {
		t.Fatalf("BUG: top stack frame points inside errors/rich_api.go instead of caller: %s", lines[0])
	}
}

func TestRepro_LegacyTypedNilErrorDoesNotPanic(t *testing.T) {
	var legacy *Error
	var err error = legacy
	if err.Error() != "" {
		t.Fatalf("expected empty string for typed-nil *Error, got %q", err.Error())
	}
}
