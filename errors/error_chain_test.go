package errors

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strings"
	"testing"
)

func TestIsAsFollowStandardChain(t *testing.T) {
	root := stderrors.New("root")
	legacy := Wrap(root, CodeInternal, "wrapped")
	wrapped := fmt.Errorf("outer: %w", legacy)

	if !Is(wrapped, root) {
		t.Fatalf("Is should unwrap %%w to root")
	}
	var got *Error
	if !As(wrapped, &got) || got.Code != CodeInternal {
		t.Fatalf("As should find *Error in %%w chain")
	}
	if !IsCode(wrapped, CodeInternal) {
		t.Fatalf("IsCode should find legacy code")
	}
	if IsCode(wrapped, "") || IsCode(wrapped, "NOPE") {
		t.Fatalf("IsCode empty/mismatch")
	}
}

func TestGettersWalkWrappedError(t *testing.T) {
	inner := New(CodeTimeout, "timeout").WithContext("severity", SeverityHigh).WithContext("category", CategoryNetwork)
	wrapped := fmt.Errorf("svc: %w", inner)
	if GetCode(wrapped) != CodeTimeout {
		t.Fatalf("GetCode: %s", GetCode(wrapped))
	}
	if GetSeverity(wrapped) != SeverityHigh {
		t.Fatalf("GetSeverity: %v", GetSeverity(wrapped))
	}
	if GetCategory(wrapped) != CategoryNetwork {
		t.Fatalf("GetCategory: %v", GetCategory(wrapped))
	}
}

func TestRichHelpersWalkWrappedError(t *testing.T) {
	rich := WrapRich(stderrors.New("db down"), RichCodeDBError, "数据库错误")
	wrapped := fmt.Errorf("repo: %w", rich)
	if FromRichError(wrapped) != rich {
		t.Fatalf("FromRichError should recover wrapped *RichError")
	}
	if RichErrorCode(wrapped, 1) != RichCodeDBError {
		t.Fatalf("RichErrorCode: %d", RichErrorCode(wrapped, 1))
	}
	if !IsRichErrorCode(wrapped, RichCodeDBError) || !IsServerError(wrapped) {
		t.Fatalf("rich helpers failed")
	}
	if !Is(wrapped, rich.Cause()) {
		t.Fatalf("standard Is should reach cause")
	}
}

func TestRootKeepsJoinedError(t *testing.T) {
	joined := stderrors.Join(stderrors.New("a"), stderrors.New("b"))
	if Root(joined) != joined {
		t.Fatalf("joined error has no single unwrap; Root must keep it")
	}
	inner := stderrors.New("leaf")
	if Root(fmt.Errorf("mid: %w", inner)) != inner {
		t.Fatalf("single chain Root")
	}
}

func TestMarshalJSONDoesNotLeakSensitiveCause(t *testing.T) {
	cause := stderrors.New("postgres://user:secret@10.0.0.8:5432/db token=abc")
	err := WrapRich(cause, RichCodeDBError, "数据库错误")
	data, marshalErr := err.MarshalJSON()
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	s := string(data)
	for _, leak := range []string{"cause", "secret", "token=abc", "10.0.0.8", "postgres"} {
		if strings.Contains(s, leak) {
			t.Fatalf("JSON leaked %q: %s", leak, s)
		}
	}
	type resp struct {
		Error *RichError `json:"error"`
	}
	body, err2 := json.Marshal(resp{Error: err})
	if err2 != nil {
		t.Fatal(err2)
	}
	if strings.Contains(string(body), "secret") {
		t.Fatalf("embedded JSON leaked cause")
	}
}
