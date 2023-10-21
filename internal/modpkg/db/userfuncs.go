package db

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"modernc.org/sqlite"
)

const (
	FuncNumPathParts      = "num_path_parts"
	FuncFirstPathPart     = "first_path_part"
	FuncTrimFirstPathPart = "trim_first_path_part"
)

func RegisterUserFuncs() {
	argTypeError := func(pos int, got, want interface{}) error {
		return fmt.Errorf("expected %T for arg %d, got %T", got, 1, want)
	}
	sqlite.MustRegisterDeterministicScalarFunction(FuncNumPathParts, 1,
		func(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			str, ok := args[0].(string)
			if !ok {
				return nil, argTypeError(0, args[0], str)
			}
			str = strings.Trim(str, "/")

			if str == "" {
				return int64(0), nil
			}

			return int64(strings.Count(str, "/")) + 1, nil
		})
	sqlite.MustRegisterDeterministicScalarFunction(FuncFirstPathPart, 1,
		func(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			str, ok := args[0].(string)
			if !ok {
				return nil, argTypeError(0, args[0], str)
			}
			str = strings.Trim(str, "/")

			left, _, _ := strings.Cut(str, "/")
			return left, nil
		})
	sqlite.MustRegisterDeterministicScalarFunction(FuncTrimFirstPathPart, 1,
		func(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			str, ok := args[0].(string)
			if !ok {
				return nil, argTypeError(0, args[0], str)
			}
			str = strings.Trim(str, "/")

			_, right, _ := strings.Cut(str, "/")
			return right, nil
		})
}
