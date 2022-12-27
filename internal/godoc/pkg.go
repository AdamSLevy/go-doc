// Package godoc provides interfaces for APIs defined in the cmd/go-doc main
// package that are also used by the completion package.
//
// This allows us to share code with the completion package without major
// refactors of the cmd/go-doc package. Minimizing the diff makes it easier to
// merge upstream changes to the official go doc.
package godoc

import (
	"go/ast"
	"go/doc"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Package exposes the information about a package that is needed by the
// completion package.
type PackageInfo interface {
	Doc() *doc.Package
	IsTypedValue(value *doc.Value) bool
	IsConstructor(value *doc.Func) bool

	// OneLineNode returns a one-line summary of the given input node.
	//
	// If no non-empty valName is given, the summary will be of the first
	// exported value in the node, if any exist, and otherwise the empty
	// string.
	//
	// If a non-empty valName is given and the node is an *ast.GenDecl, the
	// summary will be of the value (const or var) with that name. This
	// allows completion to render one line summaries for values that don't
	// come first in a value declaration.
	//
	// Only the first valName is considered.
	OneLineNode(node ast.Node, name ...string) string

	// FindTypeSpec returns the ast.TypeSpec within the declaration that
	// defines the symbol. The name must match exactly.
	FindTypeSpec(decl *ast.GenDecl, symbol string) *ast.TypeSpec
}

// IsExported reports whether the name is an exported identifier.
// If the unexported flag (-u) is true, IsExported returns true because
// it means that we treat the name as if it is exported.
var IsExported func(name string) bool

// MatchPartial is like Match but also returns true if the user's symbol is
// a prefix of the program's. An empty user string matches any program string.
func MatchPartial(user, program string) bool {
	return match(user, program, true)
}

// Match reports whether the user's symbol matches the program's.
// A lower-case character in the user's string matches either case in the program's.
// The program string must be exported.
func Match(user, program string) bool {
	return match(user, program, false)
}

// match reports whether the user's symbol matches the program's.
// A lower-case character in the user's string matches either case in the program's.
// The program string must be exported.
//
// If partial is true, the user's symbol may be a prefix of the program's. In
// this case an empty user string matches any program string.
func match(user, program string, partial bool) bool {
	if !IsExported(program) {
		return false
	}
	if MatchCase {
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
