package completion

import (
	"fmt"
	"io"
	"strings"

	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/godoc"
)

var (
	Enabled bool

	Arg       int
	PkgsOnly  bool
	ShortPath bool
)

type Completer struct {
	out  io.Writer
	dirs godoc.PackageDirs
	opts []MatchOption
}

func NewCompleter(out io.Writer, dirs godoc.PackageDirs) Completer {
	return Completer{out: out, dirs: dirs}
}

func (c Completer) Complete(pkg godoc.PackageInfo, userPath, symbol string) bool {
	dlog.Printf("completing arg %d: pkg:%v userPath:%s symbol:%s", Arg, pkg != nil, userPath, symbol)
	switch Arg {
	case 1:
		return c.completeFirstArg(pkg, userPath, symbol)
	case 2:
		return c.completeSecondArg(pkg, symbol)
	default:
		dlog.Println("invalid number of arguments")
	}
	return false
}

// go doc <pkg>
// go doc <sym>[.<methodOrField>]
// go doc [<pkg>.]<sym>[.<methodOrField>]
// go doc [<pkg>.][<sym>.]<methodOrField>
//
// We could be completing:
// - a package,
// - a symbol or method in the local package,
// - a method on a given symbol in the local package,
// - a symbol or method in a given external package
// - a method on a symbol in an external package.
//
// Package groups
// - stdlib
// - imported by local package
// - within current module
// - imported by current module
// - everything remaining in GOPATH
func (c Completer) completeFirstArg(pkg godoc.PackageInfo, userPath, symbol string) (matched bool) {
	// The user may be trying to complete a package path, or
	// a symbol in the local package.

	// If there is no local package, then the user can't be trying
	// to complete a symbol.
	//
	// If symbol contains a slash, it can't be a symbol in the
	// local package.
	//
	// So we only complete symbols if there is a local package, and
	// symbol does not have a slash.
	fullArg := userPath + symbol
	const dot = "."
	hasDot := strings.HasPrefix(symbol, dot)
	if hasDot {
		symbol = symbol[1:]
	}
	if !PkgsOnly &&
		pkg != nil &&
		(hasDot || fullArg == "") {
		matched = c.completeSecondArg(pkg, symbol)
		if hasDot {
			c.Println("IPREFIX=" + userPath + dot)
			return matched
		}
	}
	// Otherwise we are completing a package, and we'll treat the
	// entire symbol as the userPath.
	return c.completePackages(fullArg)
}

// go doc <pkg> <sym>[.<methodOrField>]
//
// We could be completing:
// - a symbol or method in the given package
// - a method on a given symbol in the given package
func (c Completer) completeSecondArg(pkg godoc.PackageInfo, partial string) bool {
	dlog.Printf("completing second argument for package %q", pkg.Doc().ImportPath)

	symbolMethod := strings.SplitN(partial, ".", 3)
	switch len(symbolMethod) {
	case 1:
		partialSymbol := symbolMethod[0]
		return c.completeSymbol(pkg, partialSymbol)
	case 2:
		symbol := symbolMethod[0]
		partialMethodOrField := symbolMethod[1]
		return c.completeMethodOrField(pkg, symbol, partialMethodOrField)
	}
	// go doc does not accept more than two dot separated fields so don't
	// offer more suggestions if there are more than two fields.
	return false
}

func (c Completer) suggest(m Match) { c.Println(m) }

func (c Completer) Print(a ...any) (int, error) {
	return fmt.Fprint(c.out, a...)
}
func (c Completer) Println(a ...any) (int, error) {
	return fmt.Fprintln(c.out, a...)
}
func (c Completer) Printf(format string, a ...any) (int, error) {
	return fmt.Fprintf(c.out, format, a...)
}
