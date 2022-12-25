package dlog

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/require"
)

// The point of this test is to make sure that the calldepth is correct for
// both New Loggers and the defaultLogger.
func TestLoggerShortfile(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	lgr := New(buf, "", log.Lshortfile)
	lgr.Enable()
	lgr.Println()
	t.Log("new logger output:", buf.String())
	require.Contains(t, buf.String(), "log_test.go")
	buf.Reset()

	defaultLogger.output = buf
	Enable()
	Println()
	t.Log("default logger output:", buf.String())
	require.Contains(t, buf.String(), "log_test.go")
}
