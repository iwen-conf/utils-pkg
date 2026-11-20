package ptr_test

import (
	"errors"
	"fmt"

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
