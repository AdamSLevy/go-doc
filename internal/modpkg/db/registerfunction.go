package db

import (
	"database/sql/driver"
	"strings"

	"modernc.org/sqlite"
)

func init() {
	sqlite.MustRegisterDeterministicScalarFunction("concat_ws", -1, func(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		sep := args[0].(string)
		elems := make([]string, len(args)-1)
		for i, arg := range args[1:] {
			elems[i] = arg.(string)
		}
		return strings.Join(elems, sep), nil
	})
}
