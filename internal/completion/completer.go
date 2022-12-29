package completion

import (
	"fmt"
	"go/token"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

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

	unexported bool
	matchCase  bool
}

func NewCompleter(out io.Writer, dirs godoc.PackageDirs, unexported, matchCase bool) Completer {
	return Completer{out: out, dirs: dirs}
}

func (c Completer) Complete(args []string, pkg godoc.PackageInfo, userPath, symbol, method string) bool {
	dlog.Printf("completing arg %d: pkg:%v userPath:%s symbol:%s", Arg, pkg != nil, userPath, symbol)
	if Arg == 0 {
		Arg = len(args)
	}
	args = append(args, "")
	if len(args) < Arg {
		// We don't have enough arguments to complete.
		return false
	}
	switch Arg {
	case 1:
		return c.completeFirstArg(args[0], pkg, userPath, symbol, method)
	case 2, 3:
		return c.completeSecondArg(args[1], pkg, symbol, method)
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
func (c Completer) completeFirstArg(arg string, pkg godoc.PackageInfo, userPath, symbol, method string) (matched bool) {
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
	validSymbol := strings.HasPrefix(symbol, dot) &&
		!strings.ContainsAny(symbol, `/\`)
	if validSymbol {
		symbol = symbol[1:]

	}
	if !PkgsOnly &&
		pkg != nil &&
		(validSymbol || arg == "") {
		matched = c.completeSecondArg(arg, pkg, symbol, method)
		if validSymbol {
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
func (c Completer) completeSecondArg(arg string, pkg godoc.PackageInfo, symbol, method string) bool {
	// We cannot proceed without a package.
	if pkg == nil {
		return false
	}

	if method != "" || strings.HasSuffix(arg, ".") {
		// We are completing a <symbol>.<method|field> in the package.
		return c.completeMethodOrField(pkg, symbol, method)
	}

	// We are completing a symbol in the package.
	return c.completeSymbol(pkg, symbol)
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

// IsExported reports whether the name is an exported identifier.
// If the unexported flag (-u) is true, IsExported returns true because
// it means that we treat the name as if it is exported.
func (c Completer) IsExported(name string) bool {
	return c.unexported || token.IsExported(name)
}

// MatchPartial is like Match but also returns true if the user's symbol is
// a prefix of the program's. An empty user string matches any program string.
func (c Completer) MatchPartial(user, program string) bool {
	return c.match(user, program, true)
}

// Match reports whether the user's symbol matches the program's.
// A lower-case character in the user's string matches either case in the program's.
// The program string must be exported.
func (c Completer) Match(user, program string) bool {
	return c.match(user, program, false)
}

// match reports whether the user's symbol matches the program's.
// A lower-case character in the user's string matches either case in the program's.
// The program string must be exported.
//
// If partial is true, the user's symbol may be a prefix of the program's. In
// this case an empty user string matches any program string.
func (c Completer) match(user, program string, partial bool) bool {
	if !c.IsExported(program) {
		return false
	}
	if c.matchCase {
		return program == user ||
			(partial && strings.HasPrefix(program, user))
	}
	for _, u := range user {
		// p is the first rune in program, or utf8.RuneError if empty or invalid.
		// w is the index of the next rune in program, or 0 if empty or invalid.
		p, w := utf8.DecodeRuneInString(program)
		// remove the first rune from program
		program = program[w:]
		if u == p {
			continue
		}
		if unicode.IsLower(u) && simpleFold(u) == simpleFold(p) {
			continue
		}
		return false
	}
	// program will be empty if we have an exact match
	return partial || program == ""
}

// simpleFold returns the minimum rune equivalent to r
// under Unicode-defined simple case folding.
func simpleFold(r rune) rune {
	for {
		r1 := unicode.SimpleFold(r)
		if r1 <= r {
			return r1 // wrapped around, found min
		}
		r = r1
	}
}
