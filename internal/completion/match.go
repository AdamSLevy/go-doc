package completion

import "strings"

const indent = "    "

// Match represents a single matching completion.
//
// It is serialized to a bespoke string format with ":" delimited values. The
// Zsh completion script parses this to determine various properties of the
// completion, like its tag, display, and description.
//
// This odd bespoke format is used because it is relatively simple to parse
// using Zsh parameter expansion and avoids external dependencies like jq for
// parsing JSON.
type Match struct {
	Pkg   string
	Type  string
	Match string

	Display       string
	DisplayIndent bool

	Describe string

	Tag string
}

// String returns the string representation of the match which is the following
// format. Empty fields are omitted if shown in [brackets].
//
//   [<tag>:][[<pkg>.]<type>.]<match>:<display>:<describe>
//
// Note: If m.Tag is TagStructFields or TagInterfaceMethods, `<type>.` is also
// prepended to `<display>`.
func (m Match) String() string {
	var match string
	if m.Pkg != "" {
		match += m.Pkg + "."
	}
	var typeDot string
	if m.Type != "" {
		typeDot = m.Type + "."
		match += typeDot
	}
	match += m.Match

	var display string
	if m.DisplayIndent {
		display = indent
	}
	switch m.Tag {
	case TagStructFields, TagInterfaceMethods:
		// Because of the way struct fields and interface methods are
		// rendered, there is nothing which identifies their associated
		// type, so we need to add the type prefix so the user can
		// actually know what type is being referred to.
		display += typeDot
	}
	if m.Display != "" {
		display += m.Display
	} else {
		display += m.Match
	}
	fields := make([]string, 0, 4)
	if m.Tag != "" {
		fields = append(fields, m.Tag)
	}
	fields = append(fields, match, strings.ReplaceAll(display, ":", "\\:"), m.Describe)
	return strings.Join(fields, ":")
}

func NewMatch(match string, opts ...MatchOption) (m Match) {
	if match == "" {
		panic("empty completion")
	}
	m.Match = match
	WithOpts(opts...)(&m)
	return
}

type MatchOption func(*Match)

func WithOpts(opts ...MatchOption) MatchOption {
	return func(m *Match) {
		for _, opt := range opts {
			opt(m)
		}
	}
}

func WithDisplay(display string) MatchOption {
	return func(m *Match) { m.Display = display }
}

func WithDisplayIndent(indented bool) MatchOption {
	return func(m *Match) { m.DisplayIndent = indented }
}

func WithDescription(describe string) MatchOption {
	return func(m *Match) { m.Describe = describe }
}

func WithNoPrefix() MatchOption {
	return func(m *Match) { m.DisplayIndent = true }
}
func WithType(typeName string) MatchOption {
	return func(m *Match) { m.Type = typeName }
}
func WithPackage(pkgPath string) MatchOption {
	return func(m *Match) { m.Pkg = pkgPath }
}

func WithTag(tag Tag) MatchOption {
	return func(m *Match) { m.Tag += tag }
}

// Tag is a string which is used to categorize completions.
//
// Tag is a type alias purely for documentation purposes.
type Tag = string

const (
	// TagPackages contains matches for packages.
	TagPackages Tag = "packages"

	// TagConsts contains the first const in each non-typed const group
	// declaration, just as go doc displays consts in the package summary.
	//
	// Typed const groups are shown under the types tag with the given
	// type, just as go doc organizes them.
	TagConsts Tag = "consts"

	// TagAllConsts contains all consts, including subsequent names in
	// grouped const declarations, and typed consts.
	//
	// Since any const name in a const group will return the same output
	// from go doc, this tag should only be checked last as a fallback.
	TagAllConsts Tag = "all-consts"

	// TagVars contains the first var in each non-typed var group
	// declaration, just as go doc displays vars in the package summary.
	//
	// Typed var groups are shown under the types tag with the given type,
	// just as go doc organizes them.
	TagVars Tag = "vars"

	// TagAllVars contains all vars, including subsequent names in grouped
	// var declarations, and typed vars.
	//
	// Since any var name in a var group will return the same output from
	// go doc, this tag should only be checked last as a fallback.
	TagAllVars Tag = "all-vars"

	// TagFuncs contains all functions in the package, except for factory
	// functions for exported types, which are listed under the types tag
	// with the type they provide.
	TagFuncs Tag = "funcs"

	// TagTypes contains all types with their associated var and const
	// declarations and factory functions.
	TagTypes Tag = "types"

	// TagTypeMethods contains all methods in the form "<type>.<method>".
	TagTypeMethods Tag = "type-methods"

	// TagInterfaceMethods contains all interface methods in the form
	// "<type>.<method>"
	TagInterfaceMethods Tag = "interface-methods"

	// TagStructFields contains all struct fields in the form
	// "<type>.<field>"
	TagStructFields Tag = "struct-fields"

	// TagMethods contains all methods without the preceding "<type>."
	//
	// Usually these should only be shown after no other matches have been
	// found.
	TagMethods Tag = "methods"
)
