// Package dlog provides a simple debug logging facility that mirrors the
// stdlib log package API, but writes exclusively to os.Stderr.
//
// By default the debug logger is disabled until Enable is called. There is no
// way to disable the debug logger once it has been enabled.
//
// NOTICE: All functions in this package are safe to call concurrently with
// each other EXCEPT for Enable.
package dlog

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"aslevy.com/go-doc/internal/flagvar"
	"github.com/davecgh/go-spew/spew"
)

const (
	// This ensures that the corrent file and line number is used when
	// logging.
	loggerCallDepth = 1

	// The defaultLogger methods are accessed through an additional
	// function call.
	defaultLoggerCallDepth = loggerCallDepth + 1
)

var defaultLogger = newLogger(os.Stderr, "debug", log.Lshortfile)

func Default() Logger { return defaultLogger }

// SetOutput overrides the output for the top level print functions to w. By
// default the output is os.Stderr.
//
// This function must be called prior to calling Enable for it to take effect.
func SetOutput(w io.Writer) { defaultLogger.SetOutput(w) }

// Enable debug logging.
//
// Prior to first calling Enable, all other functions in this package produce
// no output.
//
// After calling Enable, all other functions in this package write to
// os.Stderr.
//
// There is no way to disable the debug logger once it has been enabled.
//
// Calling Enable multiple times has no effect.
//
// NOTICE: This function is NOT safe to call concurrently with any other
// functions in this package.
func Enable()                        { defaultLogger.Enable() }
func EnableFlag() flag.Value         { return defaultLogger.EnableFlag() }
func Print(v ...any)                 { defaultLogger.Output(2, fmt.Sprint(v...)) }
func Printf(format string, v ...any) { defaultLogger.Output(2, fmt.Sprintf(format, v...)) }
func Println(v ...any)               { defaultLogger.Output(2, fmt.Sprintln(v...)) }
func Dump(v ...any)                  { defaultLogger.Dump(v...) }
func Child(prefix string) Logger     { return defaultLogger.Child(prefix) }

// Logger is a simple debug logger API. It will not produce output until Enable
// is first called.
type Logger interface {
	// Enable the logger so that it produces output instead of discarding
	// it. All calls to Print, Printf, Println, and Dump after this is
	// called will produce output.
	//
	// There is no way to disable the logger once enabled.
	//
	// It is not safe to call concurrently with other methods.
	Enable()
	EnableFlag() flag.Value

	// Child returns a new Logger with the same settings as the parent with
	// the specified prefix appended to the parent logger's prefix.
	//
	// Child and parent loggers are enabled independently. Enabling
	// a parent does not enable the child and vice versa.
	Child(prefix string) Logger

	// Print, Printf, Println, and SetOutput are the same as the log.Logger
	// methods by the same name.
	Print(...any)
	Printf(string, ...any)
	Println(...any)
	SetOutput(io.Writer)

	// Dump prints the spew representation of the arguments.
	Dump(...any)
}

type logger struct {
	*log.Logger
	mu      sync.Mutex
	output  io.Writer
	enable  sync.Once
	enabled bool
}

// New returns a new Logger that does not write to output until after
// Logger.Enable is first called.
func New(output io.Writer, prefix string, flag int) Logger { return newLogger(output, prefix, flag) }
func newLogger(output io.Writer, prefix string, flag int) *logger {
	return &logger{
		output: output,
		Logger: log.New(io.Discard, prefix+": ", flag),
	}
}

func (l *logger) Child(child string) Logger {
	prefix := l.Prefix()
	if child != "" {
		prefix = strings.TrimSpace(prefix) + child
	}
	output := l.output
	if l.enabled {
		output = l.Writer()
	}
	return newLogger(output, prefix, l.Flags())
}

func (l *logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.enabled {
		panic("cannot set output after logger has been enabled")
	}
	l.output = w
}
func (l *logger) Enable() {
	l.enable.Do(func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.Logger.SetOutput(l.output)
		l.enabled = true
	})
}
func (l *logger) Dump(v ...any) { spew.Fdump(l.Writer(), v...) }

func (l *logger) EnableFlag() flag.Value {
	return flagvar.Do(l.Enable)
}
