package flagvar

import "flag"

// Bool returns BoolValue, which is a flag.Value that implements IsBoolFlag
// returning true, causing the flag to be treated as a boolean flag.
func Bool(val flag.Value) BoolValue { return &boolValue{val} }

type BoolValue interface {
	flag.Value
	IsBoolFlag() bool
}
type boolValue struct{ flag.Value }

func (b boolValue) IsBoolFlag() bool { return true }
