package completion

import (
	"fmt"
	"go/token"
	"io"
	"log"
	"strings"
	"unicode"
	"unicode/utf8"

	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/godoc"
)

var (
	Requested bool

	Current int

	PkgsOnly  bool
	ShortPath bool
)

type Completer struct {
	out  io.Writer
	dirs godoc.PackageDirs

	args    []string
	current int

	unexported        bool
	matchCase         bool
	matchPartialTypes bool
}

func NewCompleter(out io.Writer, dirs godoc.PackageDirs, unexported, matchCase bool, args []string) *Completer {
	// Normally we allow partial types on either side of a dot when
	// specifying a <type>.<method|field>
	// i.e. go doc http cli.d<tab> -> go doc http Client.Do
	matchPartialTypes := true
	// Since flags.Parse joins the last 2 of 3 args with a dot, we treat
	// the 3 argument case as if it were the 2 argument case. The Zsh
	// completion matching handles ignoring the leading "<type>." from the
	// completions for the third argument.
	//
	// The key difference is that when completing the third argument, we
	// need a fully matched type that go doc will resolve. This is because
	// we cannot alter previous arguments on the command line.
	current := Current
	if current == 3 {
		current = 2
		matchPartialTypes = false
	}
	// Assume we are completing the final argument if Current is not set.
	if current == 0 {
		current = len(args)
	} else {
		// If the user is completing an argument they haven't begun
		// typing, then len(args) will be one less than current. Pad
		// args with an empty string just in case. Any extra args
		// beyond current are ignored.
		args = append(args, "")
		// If we still don't have enough args, then we can't proceed.
		if len(args) < current {
			log.Fatal("cannot complete argument -current=%d with only %d args", current, len(args)-1)
		}
	}

	return &Completer{
		out:               out,
		dirs:              dirs,
		args:              args,
		current:           current,
		unexported:        unexported,
		matchCase:         matchCase,
		matchPartialTypes: matchPartialTypes,
	}
}

func (c *Completer) Complete(pkg godoc.PackageInfo, userPath, symbol, method string) bool {
	dlog.Printf("completing arg %d: pkg:%v userPath:%s symbol:%s", c.current, pkg != nil, userPath, symbol)
	switch c.current {
	case 1:
		return c.completeFirstArg(c.args[0], pkg, userPath, symbol, method)
	case 2:
		return c.completeSecondArg(c.args[1], pkg, symbol, method)
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
	// Determine what we are completing. main.parseArgs has already done
	// the hard work of determining the package, if any can be parsed.

	// If parseArgs fails to parse a package, it puts the entire arg into
	// the symbol assuming its a symbol in the local package.
	//
	// It could also be an incomplete package path that couldn't be parsed.
	// If it contains path separators, then it cannot be a valid symbol.
	const pathSeparators = `/\` // File paths on windows may use backslashes.
	invalidSymbol := strings.ContainsAny(symbol, pathSeparators)

	// If the symbol is NOT INVALID, and is either not empty or the arg
	// ends with a dot, then the user is likely trying to complete
	// a symbol.
	const dot = "."
	typingSymbol := !invalidSymbol && // The symbol cannot be invalid.
		(symbol != "" || strings.HasSuffix(arg, dot)) // We have a parsed symbol or the user just typed a dot.

	symbolRequested := !PkgsOnly && // We've been asked not to complete symbols.
		pkg != nil && // We can't complete symbols without a parsed package.
		typingSymbol // The user must be typing a symbol to offer symbol completions.

	if symbolRequested {
		iPrefix := ignoredPrefix(arg, symbol, method)
		matched = c.completeSecondArg(arg[len(iPrefix):], pkg, symbol, method)
		if userPath != "" { // The user has specified a package.
			// If we have matches, then we need to inform the Zsh
			// completion script to ignore the "<pkg>." prefix so
			// it can match the second argument completions against
			// the first.
			if matched {
				// The Zsh completion script parses this line
				// and uses it to set the IPREFIX if found.
				c.Println("IPREFIX=" + iPrefix)
			}
			// Since the user has specified a package and we're
			// into completing the symbol then we will not complete
			// packages.
			//
			// The pkg may not be the one intended by the partial
			// package path. If we don't have matches, then we'll
			// get called again with the next matching package, if
			// any.
			//
			// TODO: Provide completions for multiple matching
			// packages, rather than stopping at the first.
			return
		}
		// We completed a symbol from the local package. But it is
		// possible the user is actually trying to complete a package,
		// and the symbol matches are just coincidental. So go on to
		// suggest packages as well.
	}

	c.completePackages(arg)
	// Always return true to force main to exit. Otherwise it will loop
	// infinitely if parseArgs returned more=true.
	return true
}

// ignoredPrefix returns the "<pkg>." portion of arg. We can't rely on userPath
// since parseArgs may alter it from what was typed.
//
// This is done by removing the "<sym>[.<method>]" suffix from arg.
func ignoredPrefix(arg, symbol, method string) string {
	// If we have a method, the we have a dot between it and the symbol.
	symLen := len(method)
	if symLen > 0 {
		symLen++
	}
	symLen += len(symbol)

	// What remains must be the length of the package path.
	pkgLen := len(arg) - symLen
	return arg[:pkgLen]

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
