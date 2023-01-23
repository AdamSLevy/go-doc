package dlog

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	prefix := "prefix"
	line := "abcdef"
	lgr := New(nil, prefix, log.Lshortfile)
	childPrefix := "child"
	child := lgr.Child(childPrefix)
	defChild := Child(childPrefix)

	for _, test := range []struct {
		name string
		Logger
		prefix string
	}{
		{"New", lgr, prefix},
		{"Logger.Child", child, prefix + ":" + childPrefix},
		{"Child", defChild, "debug:" + childPrefix},
	} {
		t.Run(test.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			lgr := test.Logger
			lgr.SetOutput(buf)

			lgr.Println(line)
			require := require.New(t)
			require.Empty(buf.String(), "output should be empty prior to calling Enable()")

			lgr.Enable()
			lgr.Println(line)
			output := buf.String()
			require.Contains(output, "log_test.go", "file name")
			require.Contains(output, test.prefix+": ", "prefix")
			require.Contains(output, line, "content")
		})
	}
}
func TestDefaultLogger(t *testing.T) {
	require := require.New(t)
	buf := bytes.NewBuffer(nil)
	SetOutput(buf)
	prefix := "debug"
	line := "abcdef"
	Println(line)
	require.Empty(buf.String(), "output should be empty prior to calling Enable()")

	Enable()
	Println(line)
	output := buf.String()
	require.Contains(output, "log_test.go")
	require.Contains(output, prefix+": ")
	require.Contains(output, line)
}
