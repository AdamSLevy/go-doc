// Package flagvar provides composable flag.Value implementations.
package flagvar

import (
	"encoding"
	"flag"
	"time"
)

// Value converts val to a flag.Value if it is one of the types supported by
// flag.FlagSet. Otherwise it returns nil.
//
// The concrete type of the returned flag.Value is from the stdlib flag
// package. So given,
//
//	fs := flag.NewFlagSet("example", flag.ExitOnError)
//	myBool := false
//
// the following two lines are functionally equivalent,
//
//	fs.BoolVar(&myBool, "mybool", myBool, "my bool flag")
//	fs.Var(flagvar.Value(&myBool), "mybool", "my bool flag")
func Value(val any) flag.Value {
	type setFunc func(string) error
	type encodingText interface {
		encoding.TextUnmarshaler
		encoding.TextMarshaler
	}

	// The stdlib flag package does not directly expose its flag.Value
	// implementations. So add the flag to a FlagSet and then look it up.
	var fs flag.FlagSet
	const name = "val"
	switch val := val.(type) {
	case flag.Value:
		return val
	case encodingText:
		fs.TextVar(val, name, val, "")
	case setFunc:
		fs.Func(name, "", val)
	case *time.Duration:
		fs.DurationVar(val, name, *val, "")
	case *bool:
		fs.BoolVar(val, name, *val, "")
	case *int:
		fs.IntVar(val, name, *val, "")
	case *int64:
		fs.Int64Var(val, name, *val, "")
	case *uint:
		fs.UintVar(val, name, *val, "")
	case *uint64:
		fs.Uint64Var(val, name, *val, "")
	case *float64:
		fs.Float64Var(val, name, *val, "")
	case *string:
		fs.StringVar(val, name, *val, "")
	}
	return fs.Lookup(name).Value
}
