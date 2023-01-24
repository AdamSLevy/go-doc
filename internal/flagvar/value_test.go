package flagvar

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var valueTests = []struct {
	name string
	val  any
	Set  string
}{{
	name: "bool",
	val:  new(bool),
	Set:  "true",
}, {
	name: "int",
	val:  new(int),
	Set:  "42",
}, {
	name: "string",
	val:  new(string),
	Set:  "hello",
}}

func TestValue(t *testing.T) {
	for _, test := range valueTests {
		t.Run(test.name, func(t *testing.T) {
			val := Value(test.val)
			require.NoError(t, val.Set(test.Set))
			require.Equal(t, test.Set, val.String())
		})
	}
}
