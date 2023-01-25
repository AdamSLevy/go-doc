package flagvar

import "flag"

// DoErr returns a BoolValue which calls f when Set is called.
func DoErr(f func() error) flag.Value {
	return &do{f: f}
}

// Do is like DoErr but for functions which return no error.
func Do(f func()) flag.Value {
	return DoErr(func() error { f(); return nil })
}

type do struct {
	f    func() error
	done bool
}

func (do) IsBoolFlag() bool { return true }
func (do) String() string   { return "false" }
func (d *do) Set(val string) error {
	var run bool
	if err := Value(&run).Set(val); err != nil {
		return err
	}
	defer func() { d.done = true }()
	if !run || d.done {
		return nil
	}
	return d.f()
}
