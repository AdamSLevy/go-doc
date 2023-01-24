package flagvar

// DoErr returns a BoolValue which calls f when Set is called.
func DoErr(f func() error) BoolValue {
	return Bool(do(f))
}

// Do is like DoErr but for functions which return no error.
func Do(f func()) BoolValue {
	return DoErr(func() error { f(); return nil })
}

type do func() error

func (_ do) String() string { return "false" }
func (f do) Set(val string) error {
	var run bool
	if err := Value(&run).Set(val); err != nil {
		return err
	}
	if run {
		return f()
	}
	return nil
}
