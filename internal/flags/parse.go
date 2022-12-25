package flags

import (
	"flag"
	"strings"

	"aslevy.com/go-doc/internal/completion"
	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/install"
)

// AddParse adds all flags defined in this package to fs and then calls
// fs.Parse with the provided args. It returns any non-flag arguments.
//
// In contrast to flag.Parse, flags appearing after the first non-flag argument
// and between non-flag arguments are also parsed.
//
// Special handling is provided for the -complete flag, which must be the first
// argument for it to take effect, and expose other completion flags.
//
// Special handling is also provided for when there are exactly 3 non-flag
// arguments. In this case, the last two arguments are joined with a dot. This
// is a hack to allow for a three argument syntax of:
//
//   go doc <pkg> <type> <method|field>
//
// to be equivalent to:
//
//   go doc <pkg> <type>.<method|field>
//
//syntax:text
func AddParse(fs *flag.FlagSet, args ...string) []string {
	addAllFlags(fs)
	args = addCompletionFlags(fs, args...)
	args = parse(fs, args...)

	if len(args) == 3 {
		return []string{args[0], args[1] + "." + args[2]}
	}

	if debug {
		dlog.Enable()
	}

	if completion.Enabled {
		godoc.NoImports = true
	} else {
		install.IfRequested()
	}

	return args
}
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
