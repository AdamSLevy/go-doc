package godoc

import "flag"

var (
	NoImports  bool
	ShowStdlib bool
	NoLocation bool
)

func AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&NoImports, "imports-off", false, "do not show the imports for referenced packages")
	fs.BoolVar(&ShowStdlib, "imports-stdlib", false, "show imports for referenced stdlib packages")
	fs.BoolVar(&NoLocation, "location-off", false, "do not show symbol file location i.e. // /path/to/circle.go +314")
}
