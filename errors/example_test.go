package errors_test

import (
	"encoding/json"
	stderrors "errors"
	"fmt"

	"github.com/iwen-conf/utils-pkg/errors"
)

func ExampleWrapRich() {
	dbErr := stderrors.New("dial tcp 10.0.0.12:5432: connection refused")
	appErr := errors.WrapRich(dbErr, errors.RichCodeDBError, "查询订单失败")

	// Safe external HTTP JSON response: never leaks internal dbErr or IP address.
	respJSON, _ := json.Marshal(appErr)
	fmt.Printf("status=%d body=%s\n", appErr.HTTPStatus(), respJSON)

	// Standard library errors.Is penetrates RichError cause chain.
	fmt.Printf("matches cause: %v\n", errors.Is(appErr, dbErr))

	// Output:
	// status=500 body={"code":500001,"msg":"查询订单失败"}
	// matches cause: true
}
