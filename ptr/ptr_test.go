package ptr_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/iwen-conf/utils-pkg/ptr"
)

type Role string

type Profile struct {
	Nickname *string
	Role     *Role
}

func ExampleOf() {
	p := Profile{
		Nickname: ptr.Of("Alice"),
		Role:     ptr.Of(Role("admin")),
	}

	var missing *string
	fmt.Printf("nick=%s role=%s fallback=%s\n",
		ptr.Zero(p.Nickname),
		ptr.Deref(p.Role, Role("guest")),
		ptr.Deref(missing, "anonymous"),
	)

	_, err := ptr.Value(missing)
	fmt.Printf("nil error: %v\n", errors.Is(err, ptr.ErrNilPointer))

	// Output:
	// nick=Alice role=admin fallback=anonymous
	// nil error: true
}

func TestOfCopyIsolation(t *testing.T) {
	orig := 42
	p := ptr.Of(orig)
	if p == nil || *p != 42 {
		t.Fatalf("Of(42) = %v, want *42", p)
	}
	orig = 99
	if *p != 42 {
		t.Fatalf("Of did not copy value: got %d, want 42", *p)
	}

	type item struct {
		Name  string
		Count int
	}
	src := item{Name: "alpha", Count: 1}
	ip := ptr.Of(src)
	src.Name = "beta"
	if ip.Name != "alpha" || ip.Count != 1 {
		t.Fatalf("Of(struct) shared state: got %+v", *ip)
	}
}

func TestValue(t *testing.T) {
	if got, err := ptr.Value(ptr.Of("hello")); err != nil || got != "hello" {
		t.Fatalf("Value(non-nil) = (%q, %v), want (%q, nil)", got, err, "hello")
	}

	var nilStr *string
	if got, err := ptr.Value(nilStr); !errors.Is(err, ptr.ErrNilPointer) || got != "" {
		t.Fatalf("Value(nil string) = (%q, %v), want (%q, ErrNilPointer)", got, err, "")
	}

	var nilDur *time.Duration
	if got, err := ptr.Value(nilDur); !errors.Is(err, ptr.ErrNilPointer) || got != 0 {
		t.Fatalf("Value(nil duration) = (%v, %v), want (0, ErrNilPointer)", got, err)
	}
}

func TestDeref(t *testing.T) {
	if got := ptr.Deref(ptr.Of(7), 99); got != 7 {
		t.Fatalf("Deref(non-nil) = %d, want 7", got)
	}
	var nilInt *int
	if got := ptr.Deref(nilInt, 99); got != 99 {
		t.Fatalf("Deref(nil) = %d, want 99", got)
	}
	if got := ptr.Deref((*Role)(nil), Role("viewer")); got != Role("viewer") {
		t.Fatalf("Deref(nil Role) = %q, want viewer", got)
	}
}

func TestZero(t *testing.T) {
	if got := ptr.Zero(ptr.Of(true)); !got {
		t.Fatalf("Zero(non-nil bool) = %v, want true", got)
	}
	var nilBool *bool
	if got := ptr.Zero(nilBool); got {
		t.Fatalf("Zero(nil bool) = %v, want false", got)
	}

	now := time.Unix(1700000000, 0).UTC()
	if got := ptr.Zero(ptr.Of(now)); !got.Equal(now) {
		t.Fatalf("Zero(non-nil time) = %v, want %v", got, now)
	}
	var nilTime *time.Time
	if got := ptr.Zero(nilTime); !got.IsZero() {
		t.Fatalf("Zero(nil time) = %v, want zero time", got)
	}
}
