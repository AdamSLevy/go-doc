package completion

import "strings"

const indent = "    "

type Match struct {
	Pkg   string
	Type  string
	Match string

	Display       string
	DisplayIndent bool

	Describe string

	Tag string
}

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

func WithType(typeName string) MatchOption {
	return func(m *Match) { m.Type = typeName }
}
func WithPackage(pkgPath string) MatchOption {
	return func(m *Match) { m.Pkg = pkgPath }
}

func WithTag(tag Tag) MatchOption {
	return func(m *Match) { m.Tag += tag }
}

type Tag = string

const (
	// TagPackages contains matches for packages.
	TagPackages Tag = "packages"

	// TagConsts contains the first const in each non-typed const group
	// declaration, just as go doc displays consts in the package summary.
	//
	// Typed const groups are shown under the types tag with the given
	// type, just as go doc organizes them.
	TagConsts = "consts"

	// TagAllConsts contains all consts, including subsequent names in
	// grouped const declarations, and typed consts.
	//
	// Since any const name in a const group will return the same output
	// from go doc, this tag should only be checked last as a fallback.
	TagAllConsts = "all-consts"

	// TagVars contains the first var in each non-typed var group
	// declaration, just as go doc displays vars in the package summary.
	//
	// Typed var groups are shown under the types tag with the given type,
	// just as go doc organizes them.
	TagVars = "vars"

	// TagAllVars contains all vars, including subsequent names in grouped
	// var declarations, and typed vars.
	//
	// Since any var name in a var group will return the same output from
	// go doc, this tag should only be checked last as a fallback.
	TagAllVars = "all-vars"

	// TagFuncs contains all functions in the package, except for factory
	// functions for exported types, which are listed under the types tag
	// with the type they provide.
	TagFuncs = "funcs"

	// TagTypes contains all types with their associated var and const
	// declarations and factory functions.
	TagTypes = "types"

	// TagTypeMethods contains all methods in the form "<type>.<method>".
	TagTypeMethods = "type-methods"

	// TagInterfaceMethods contains all interface methods in the form
	// "<type>.<method>"
	TagInterfaceMethods = "interface-methods"

	// TagStructFields contains all struct fields in the form
	// "<type>.<field>"
	TagStructFields = "struct-fields"

	// TagMethods contains all methods without the preceding "<type>."
	//
	// Usually these should only be shown after no other matches have been
	// found.
	TagMethods = "methods"
)
