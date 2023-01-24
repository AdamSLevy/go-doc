package flagvar

import (
	"flag"
)

// Latch returns a flag.Value that will only be set once. In other words only
// the first appearance of the flag will be used.
//
// Any subsequent appearances will be silently ignored.
//
// For example, a latching bool flag could be defined as follows:
//
//	fs := flag.NewFlagSet("example", flag.ExitOnError)
//	myBool := false
//	val := flagvar.Latch(flagvar.Value(&myBool)
//	fs.Var(val, "mybool", "my bool flag")
//
// The value of myBool will be true after parsing the following command line:
//
//	example -mybool -mybool=false
func Latch(val flag.Value) flag.Value { return &latch{val: val} }

type latch struct {
	val flag.Value
	set bool
}

func (l *latch) IsBoolFlag() bool {
	boolVal, ok := l.val.(interface{ IsBoolFlag() bool })
	return ok && boolVal.IsBoolFlag()
}
func (l latch) String() string { return l.val.String() }
func (l *latch) Set(s string) error {
	if l.set {
		return nil
	}
	l.set = true
	return l.val.Set(s)
}
