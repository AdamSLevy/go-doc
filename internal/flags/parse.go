package flags

import (
	"flag"
	"strings"

	"aslevy.com/go-doc/internal/completion"
	"aslevy.com/go-doc/internal/godoc"
)

// Parse is like [flag.FlagSet.Parse], but it adds all flags defined in this
// package to fs, and also processes all flags appearing after or between
// non-flag arguments.
//
// Special handling is provided for when there are exactly 3 non-flag
// arguments. In such case, the last two arguments are joined with a dot. This
// is a hack to allow for a three argument syntax of:
//
//   go doc <pkg> <type> <method|field>
//
// to be equivalent to:
//
//   go doc <pkg> <type>.<method|field>
//
// Special handling is provided for the following flags.
//
// # -debug
//
// If the -debug flag is present, then debug logging is enabled via
// [internal/dlog.Enable].
//
// # -install-completion
//
// If the -install-completion flag is present, then the completion script
// assets are installed and the program exits. All other arguments are ignored.
//
// # -complete
//
// If args[0] == "-complete", then completion is enabled and other completion
// specific flags are also added to fs. The -complete flag is not recognized in
// any other position in args.
//
//syntax:text
func Parse(fs *flag.FlagSet, args ...string) {
	if len(args) > 0 && args[0] == "-complete" {
		args = args[1:]
		completion.Enabled = true
		addCompletionFlags(fs)
	}
	addAllFlags(fs)
	args = parse(fs, args...)

	// Skip imports if we are completing since they are not needed and
	// require additional parsing of the AST.
	godoc.NoImports = godoc.NoImports || completion.Enabled

	// When there are exactly three arguments, join the last two with
	// a dot, unless we are in completion mode.
	// <pkg> <type> <method|field> -> <pkg> <type>.<method|field>
	if len(args) == 3 && !completion.Enabled {
		method := args[2]
		args = args[:2]
		args[1] += "." + method
	}

	// Final call to parse with only the non-flag arguments so that
	// fs.Args() returns the correct values.
	fs.Parse(args)
}

// parse calls fs.Parse(args) recursively until all -flag arguments have been
// parsed, including those appearing after non-flag arguments. The non-flag
// arguments are returned.
func parse(fs *flag.FlagSet, args ...string) []string {
	// Parse everything up to the first non-flag argument.
	fs.Parse(args)

	// Collect the non-flag arguments up to the next flag argument, then
	// recurse to parse the next set of arguments as flags again.
	args = make([]string, 0, fs.NArg())
	for i, arg := range fs.Args() {
		if strings.HasPrefix(arg, "-") {
			return append(args, parse(fs, fs.Args()[i:]...)...)
		}
		args = append(args, arg)
	}

	// We have parsed all flags and collected all non-flag arguments.
	return args
}
