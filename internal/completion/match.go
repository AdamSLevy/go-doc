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
	Tag Tag

	Pkg   string
	Type  string
	Match string

	Display       string
	DisplayIndent bool

	Describe string
}

// String returns the string representation of the match which is the following
// format. Empty fields are omitted if shown in [brackets].
//
//	[<tag>:][[<pkg>.]<type>.]<match>:<display>:<describe>
//
// Note: If m.Tag is TagStructField or TagInterfaceMethod, `<type>.` is also
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
	case TagStructField, TagInterfaceMethod:
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

// NewMatch returns a new Match with the given match and options.
//
// It panics if match is empty.
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
	// TagPackage contains matches for packages.
	TagPackage Tag = "package"

	// TagPackageDir contains matches for package directories.
	TagPackageDir Tag = "package-dir"

	// TagConst contains the first const in each non-typed const group
	// declaration, just as go doc displays consts in the package summary.
	//
	// Typed const groups are shown under the types tag with the given
	// type, just as go doc organizes them.
	TagConst Tag = "const"

	// TagConstAll contains all consts, including subsequent names in
	// grouped const declarations, and typed consts.
	//
	// Since any const name in a const group will return the same output
	// from go doc, this tag should only be checked last as a fallback.
	TagConstAll Tag = "const-all"

	// TagVar contains the first var in each non-typed var group
	// declaration, just as go doc displays vars in the package summary.
	//
	// Typed var groups are shown under the types tag with the given type,
	// just as go doc organizes them.
	TagVar Tag = "var"

	// TagVarAll contains all vars, including subsequent names in grouped
	// var declarations, and typed vars.
	//
	// Since any var name in a var group will return the same output from
	// go doc, this tag should only be checked last as a fallback.
	TagVarAll Tag = "var-all"

	// TagFunc contains all functions in the package, except for factory
	// functions for exported types, which are listed under the types tag
	// with the type they provide.
	TagFunc Tag = "func"

	// TagType contains all types with their associated var and const
	// declarations and factory functions.
	TagType Tag = "type"

	// TagTypeMethod contains all methods in the form "<type>.<method>".
	TagTypeMethod Tag = "type-method"

	// TagInterfaceMethod contains all interface methods in the form
	// "<type>.<method>"
	TagInterfaceMethod Tag = "interface-method"

	// TagStructField contains all struct fields in the form
	// "<type>.<field>"
	TagStructField Tag = "struct-field"

	// TagMethod contains all methods without the preceding "<type>."
	//
	// Usually these should only be shown after no other matches have been
	// found.
	TagMethod Tag = "method"
)
