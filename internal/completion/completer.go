// Package completion generates possible completions for a given set of args
// and results of main.parseArgs.
//
// A Completer is initialized with the original command line arguments and some
// relevant flags and APIs. The results of parseArgs are then passed to
// Completer.Complete to generate possible completions.
//
// The results of parseArgs are primarily used to determine the parsing state
// of the single argument syntax: go doc <pkg>[.<sym>[.<method>]], and the
// returned built package, if any, is used to generate symbol and method
// completions.
//
// Completions are assigned a tag that is used by the Zsh completion script to
// categorize completions. See TagPackages for more information about all tags.
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
	// Requested represents if completions have been requested. This is set
	// true by internal/flags.Parse if -complete is the first argument.
	//
	// Generally this is checked outside of the package. Calls to
	// Completer.Complete will generate completions regardless of this
	// value.
	Requested bool

	// Current is the specified argument to complete as a 1-based index of
	// the normal (non-flag) arguments after `go doc`.
	//
	// If 0, it is unset, and the completer uses len(args).
	//
	// If greater than 3, we exit status 1.
	Current int

	// PkgsOnly causes Completer to only suggest packages, and never
	// suggest symbols.
	PkgsOnly bool

	// ShortPath causes Completer to only suggest the shortest unique
	// right-most path-segments.
	ShortPath bool
)

type Completer struct {
	out  io.Writer
	dirs godoc.Dirs

	args    []string
	current int

	unexported        bool
	matchCase         bool
	matchPartialTypes bool
}

func NewCompleter(out io.Writer, dirs godoc.Dirs, unexported, matchCase bool, args []string) *Completer {
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
			log.Fatalf("cannot complete argument -current=%d with only %d args", current, len(args)-1)
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

// completeFirstArg generates completion for the first argument of go doc.
//
// The first argument can be one of the following:
// go doc <pkg>
// go doc <sym>[.<methodOrField>]
// go doc [<pkg>.]<sym>[.<methodOrField>]
// go doc [<pkg>.][<sym>.]<methodOrField>
//
// So depending on how far the user has typed, we could be completing:
// - a package,
// - a symbol or method in the local package,
// - a method on a given symbol in the local package,
// - a symbol or method in a given external package
// - a method on a symbol in an external package.
//
// We determine the state based on a combination of the return values of
// parseArgs and the raw argument.
//
// If the arg ends in "." we assume we are now completing the next identifier,
// either a symbol, or a method on a symbol.
func (c Completer) completeFirstArg(arg string, pkg godoc.PackageInfo, userPath, symbol, method string) (matched bool) {
	// Determine what we are completing. main.parseArgs has already done
	// the hard work of determining the package, if any can be parsed.

	// If parseArgs fails to parse a package, it puts the entire arg into
	// the symbol assuming its a symbol in the local package.
	//
	// But it could also be an incomplete package path that couldn't be
	// parsed. If it contains path separators, then it cannot be a valid
	// symbol.
	//
	// Also if the user types "go doc .<tab>" then they are completing
	// a relative file path, not a symbol in the local package. So if the
	// arg is exactly "." then mark the symbol as invalid.
	const pathSeparators = `/\` // File paths on windows may use backslashes.
	invalidSymbol := strings.ContainsAny(symbol, pathSeparators) ||
		arg == "." // Just dot is not a symbol.

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
		iPrefix := packagePrefix(arg, symbol, method)
		matched = c.completeSecondArg(arg[len(iPrefix):], pkg, symbol, method)
		if userPath != "" { // The user has specified a package.
			// If we have matches, then we need to inform the Zsh
			// completion script to ignore the "<pkg>." prefix so
			// it can match the second argument completions against
			// the first.
			if matched {
				// The Zsh completion script parses this line
				// and uses it to set the IPREFIX if found.
				c.println("IPREFIX=" + iPrefix)
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

// packagePrefix returns the "<pkg>." portion of arg, if any, or the empty string.
//
// This additional parsing is necessary because we can't rely on userPath from
// parseArgs to because parseArgs alters it from what the user typed in certain
// cases.
//
// The arg is assumed to be in one of the following forms. This should be the
// case when we are completing symbols for the first argument.
//
//   - <pkg>.
//   - <pkg>.<sym>
//   - <pkg>.<sym>.
//   - <pkg>.<sym>.<method>
//   - <sym>.
//   - <sym>.<method>
//
// Since <pkg> may also contain one or more "." characters, we must work from
// right to left based on the provided symbol and method.
func packagePrefix(arg, symbol, method string) string {

	// lastDot is the index of the last . found in s, or -1.
	lastDot := func(s string) int { return strings.LastIndex(s, ".") }
	dot := lastDot(arg)

	haveMethod := len(method) > 0
	haveSymbol := len(symbol) > 0
	endsWithDot := dot == len(arg)-1

	if haveSymbol && (haveMethod || endsWithDot) {
		// arg is one of the following:
		//  - <pkg>.<sym>.
		//  - <pkg>.<sym>.<method>
		//  - <sym>.
		//  - <sym>.<method>
		// So dot is the index of the . after <sym>
		dot = lastDot(arg[:dot])
	}
	// Now dot is the index of the . after the <pkg>, or -1. We want to return <pkg>.
	return arg[:dot+1]
}

// completeSecondArg generates completion for the second argument of go doc:
//
//	go doc <pkg> <sym>[.<methodOrField>]
//
// So we could be completing:
// - a symbol in the given package
// - a method on a given symbol in the given package
//
// If the are ends in "." then we assume we are completing methods.
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

func (c Completer) suggest(m Match) { c.println(m) }

func (c Completer) print(a ...any) (int, error) {
	return fmt.Fprint(c.out, a...)
}
func (c Completer) println(a ...any) (int, error) {
	return fmt.Fprintln(c.out, a...)
}
func (c Completer) printf(format string, a ...any) (int, error) {
	return fmt.Fprintf(c.out, format, a...)
}

// isExported reports whether the name is an exported identifier.
// If the unexported flag (-u) is true, isExported returns true because
// it means that we treat the name as if it is exported.
func (c Completer) isExported(name string) bool {
	return c.unexported || token.IsExported(name)
}

// matchPartial is like Match but also returns true if the user's symbol is
// a prefix of the program's. An empty user string matches any program string.
func (c Completer) matchPartial(user, program string) bool {
	return c.match(user, program, true)
}

// matchFull reports whether the user's symbol matches the program's.
// A lower-case character in the user's string matches either case in the program's.
// The program string must be exported.
func (c Completer) matchFull(user, program string) bool {
	return c.match(user, program, false)
}

// match reports whether the user's symbol matches the program's.
// A lower-case character in the user's string matches either case in the program's.
// The program string must be exported.
//
// If partial is true, the user's symbol may be a prefix of the program's. In
// this case an empty user string matches any program string.
func (c Completer) match(user, program string, partial bool) bool {
	if !c.isExported(program) {
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
