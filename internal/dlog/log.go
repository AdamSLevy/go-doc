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
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/davecgh/go-spew/spew"
)

const defaultCallDepth = 1

var defaultLogger = newLogger(os.Stderr, "debug: ", log.Lshortfile, defaultCallDepth+1)

// SetOutput overrides the output for the top level print functions to w. By
// default the output is os.Stderr.
//
// This function must be called prior to calling Enable for it to take effect.
func SetOutput(w io.Writer) { defaultLogger.output = w }

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
func Print(v ...any)                 { defaultLogger.Print(v...) }
func Printf(format string, v ...any) { defaultLogger.Printf(format, v...) }
func Println(v ...any)               { defaultLogger.Println(v...) }
func Dump(v ...any)                  { defaultLogger.Dump(v...) }

// Logger is a simple debug logger API. It will not produce output until Enable
// is first called.
type Logger interface {
	Print(...any)
	Printf(string, ...any)
	Println(...any)
	Dump(...any)
	Enable()
}

type logger struct {
	print   func(...any)
	printf  func(string, ...any)
	println func(...any)
	dump    func(...any)

	once      sync.Once
	output    io.Writer
	prefix    string
	flag      int
	calldepth int
}

// New returns a new Logger that does not write to output until after
// Logger.Enable is first called.
func New(output io.Writer, prefix string, flag int) Logger {
	return newLogger(output, prefix, flag, defaultCallDepth)
}
func newLogger(output io.Writer, prefix string, flag, calldepth int) *logger {
	return &logger{
		print:   nop,
		printf:  nopf,
		println: nop,
		dump:    nop,

		output:    output,
		prefix:    prefix,
		flag:      flag,
		calldepth: calldepth + 2,
	}
}
func nop(...any)          {}
func nopf(string, ...any) {}

func (l *logger) Enable() {
	l.once.Do(func() {
		lgr := log.New(l.output, l.prefix, l.flag)
		l.print = func(v ...any) { lgr.Output(l.calldepth, fmt.Sprint(v...)) }
		l.printf = func(format string, v ...any) { lgr.Output(l.calldepth, fmt.Sprintf(format, v...)) }
		l.println = func(v ...any) { lgr.Output(l.calldepth, fmt.Sprintln(v...)) }
		lgr.Output(l.calldepth+2, "debug logging enabled")
		spew := spew.NewDefaultConfig()
		l.dump = spew.Dump
	})
}
func (l *logger) Print(v ...any)                 { l.print(v...) }
func (l *logger) Printf(format string, v ...any) { l.printf(format, v...) }
func (l *logger) Println(v ...any)               { l.println(v...) }
func (l *logger) Dump(v ...any)                  { l.dump(v...) }
