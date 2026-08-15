package errors

import (
	stderrors "errors"
	"fmt"
	"testing"
	"time"
)

func TestLegacyConstructorsAndMethods(t *testing.T) {
	detailed := NewWithDetails(CodeInvalidInput, "bad", "field x")
	if detailed.Details == "" || detailed.Error() == "" {
		t.Fatalf("NewWithDetails")
	}
	wrapped := WrapWithDetails(stderrors.New("io"), CodeDatabaseError, "db", "timeout")
	if wrapped.Unwrap() == nil {
		t.Fatalf("WrapWithDetails")
	}
	from := FromType(TimeoutError)
	if from.Code != CodeTimeout {
		t.Fatalf("FromType")
	}
	fromD := FromTypeWithDetails(DatabaseError, "q")
	if fromD.Details != "q" {
		t.Fatalf("FromTypeWithDetails")
	}
	typed := WrapWithType(stderrors.New("n"), NetworkError)
	if typed.Code != CodeNetworkError {
		t.Fatalf("WrapWithType")
	}

	b := NewBuilder().Wrap(stderrors.New("c")).Context("k", "v").UserID("u").RequestID("r").Operation("op")
	built := b.Build()
	if built.Original == nil || built.Context["user_id"] != "u" {
		t.Fatalf("builder extras")
	}

	if Internal("x").Code != CodeInternal || NotFound("u").Code != CodeNotFound {
		t.Fatalf("convenience constructors")
	}
	if Unauthorized("no").Code != CodeUnauthorized || Forbidden("no").Code != CodeForbidden {
		t.Fatalf("auth constructors")
	}
	if InvalidInput("f", "m").Code != CodeInvalidInput || MissingField("f").Code != CodeMissingField {
		t.Fatalf("input constructors")
	}
	if Timeout("op", time.Second).Code != CodeTimeout {
		t.Fatalf("timeout")
	}
	if Database("q", stderrors.New("e")).Code != CodeDatabaseError {
		t.Fatalf("database")
	}
	if Network("h", stderrors.New("e")).Code != CodeNetworkError {
		t.Fatalf("network")
	}

	mut := New(CodeInternal, "m").WithMessage("n").WithCode(CodeTimeout).WithOriginal(stderrors.New("o"))
	if mut.Message != "n" || mut.Code != CodeTimeout || mut.Original == nil {
		t.Fatalf("with* methods")
	}
}

func TestFormatCompareChainContext(t *testing.T) {
	err := New(CodeTimeout, "slow").WithDetails("d").WithContext("k", "v")
	if Format(nil, "json") != "" {
		t.Fatalf("format nil")
	}
	if Format(stderrors.New("x"), "json") != "x" {
		t.Fatalf("format std")
	}
	if Format(err, "short") == "" || Format(err, "json") == "" || Format(err, "detailed") == "" {
		t.Fatalf("format variants")
	}
	if !Compare(nil, nil) || Compare(err, nil) || !Compare(err, err) {
		t.Fatalf("compare")
	}
	chain := Chain(fmt.Errorf("outer: %w", err))
	if len(chain) < 2 {
		t.Fatalf("chain: %d", len(chain))
	}
	if _, ok := GetContext(err, "k"); !ok {
		t.Fatalf("GetContext")
	}
	if GetAllContext(err)["k"] != "v" {
		t.Fatalf("GetAllContext")
	}
	if GetCode(nil) != "" || GetCode(stderrors.New("x")) != "UNKNOWN_ERROR" {
		t.Fatalf("GetCode defaults")
	}
}

func TestValidatorRemainingRules(t *testing.T) {
	v := NewValidator()
	v.MaxLength("n", "abcdef", 3)
	v.URL("u", "://bad")
	v.Regex("r", "abc", `^\d+$`)
	v.Numeric("num", "x")
	v.Integer("i", "1.2")
	v.Min("m", 1, 2)
	v.Max("m", 5, 2)
	v.Range("m", 0, 1, 2)
	v.Date("d", "x", time.DateOnly)
	v.Before("b", time.Now(), time.Now().Add(-time.Hour))
	v.After("a", time.Now(), time.Now().Add(time.Hour))
	v.Custom("c", 1, "odd", func(v interface{}) bool { return false }, "no")
	if !v.HasErrors() || len(v.GetErrors()) == 0 {
		t.Fatalf("expected validation errors")
	}
	if ValidateRequired("n", "") == nil || ValidateEmail("e", "bad") == nil || ValidateRange("r", 0, 1, 2) == nil {
		t.Fatalf("convenience validators")
	}
	if ValidateLength("n", "abc", 10, 12) == nil {
		t.Fatalf("ValidateLength min")
	}
}

func TestRichShortcuts(t *testing.T) {
	if RichBadRequest("x").HTTPStatus() != 400 {
		t.Fatalf("http 400")
	}
	if !IsClientError(RichUnauthorized()) || !IsClientError(RichForbidden()) {
		t.Fatalf("client")
	}
	if RichNotFound("user").Code != RichCodeNotFound {
		t.Fatalf("not found")
	}
	if !IsServerError(RichInternal(stderrors.New("x"))) || !IsServerError(RichDBError(stderrors.New("x"))) {
		t.Fatalf("server")
	}
	changed := NewRich(1, "a").WithCode(2).WithMsg("b")
	if changed.Code != 2 || changed.Msg != "b" {
		t.Fatalf("with code/msg")
	}
	if WrapRichNoStack(nil, 1, "x") != nil {
		t.Fatalf("nil wrap")
	}
	if HTTP := NewRich(0, "ok").HTTPStatus(); HTTP != 200 {
		t.Fatalf("success http")
	}
	if NewRich(12, "x").HTTPStatus() != 500 {
		t.Fatalf("invalid http mapping")
	}
}
