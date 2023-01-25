package flagvar

import (
	"flag"
	"fmt"
)

func Parse[E any](val *E, parse func(string) (E, error)) flag.Value {
	return &parseValue[E]{val, parse}
}

type parseValue[E any] struct {
	val   *E
	parse func(string) (E, error)
}

func (p parseValue[E]) String() string {
	if p.val == nil {
		return ""
	}
	return fmt.Sprint(*p.val)
}
func (p *parseValue[E]) Set(s string) error {
	v, err := p.parse(s)
	if err != nil {
		return err
	}
	*p.val = v
	return nil
}
