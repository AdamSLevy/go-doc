package vendored

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/slices"
)

type ModulePackages map[godoc.PackageDir][]godoc.PackageDir

func ParseModulePackages(ctx context.Context, vendorDir string) (ModulePackages, error) {
	modPkgs := make(ModulePackages)
	return modPkgs, modPkgs.parse(ctx, vendorDir)
}

func (modPkgs ModulePackages) parse(ctx context.Context, vendorDir string) error {
	return Handler(func(_ context.Context, mod godoc.PackageDir) (PackageHandler, error) {
		return func(_ context.Context, pkgImportPath string) error {
			modPkgs[mod] = append(modPkgs[mod], godoc.NewPackageDir(pkgImportPath, filepath.Join(vendorDir, pkgImportPath)))
			return nil
		}, nil
	}).parse(ctx, vendorDir)
}

type Handler func(ctx context.Context, mod godoc.PackageDir) (PackageHandler, error)

type PackageHandler func(ctx context.Context, pkgImportPath string) error

func Parse(ctx context.Context, vendorDir string, handle Handler) error {
	return handle.parse(ctx, vendorDir)
}

func (handle Handler) parse(ctx context.Context, vendorDir string) error {
	const modulesTxt = "modules.txt"
	modTxtPath := filepath.Join(vendorDir, modulesTxt)
	modTxtFile, err := os.Open(modTxtPath)
	if err != nil {
		return err
	}
	defer modTxtFile.Close()

	return handle.parseData(ctx, vendorDir, modTxtFile)
}
func (handleMod Handler) parseData(ctx context.Context, vendorDir string, data io.Reader) error {
	var handlePkg PackageHandler
	lines := bufio.NewScanner(data)
	for lines.Scan() && ctx.Err() == nil {
		line := parseLine(lines.Text())
		if line == nil {
			continue
		}
		if line.Kind == modulesTxtLineKindModule {
			var err error
			handlePkg, err = handleMod(ctx, godoc.NewPackageDir(
				line.ImportPath,
				filepath.Join(vendorDir, line.ImportPath),
				godoc.WithVersion(line.FullVersion()),
			))
			if err != nil {
				return err
			}
			continue
		}
		if handlePkg == nil {
			continue
		}
		if err := handlePkg(ctx, line.ImportPath); err != nil {
			return err
		}
	}
	return nil
}

type modulesTxtLine struct {
	Kind modulesTxtLineKind

	Explicit  bool
	GoVersion string

	ImportPath        string
	Version           string
	ReplaceImportPath string
	ReplaceVersion    string
}

func (line modulesTxtLine) FullVersion() string {
	if line.Version == "" {
		// An empty version signals we should always sync the module.
		return ""
	}

	if line.ReplaceImportPath == "" {
		// No replacement so just the version.
		return line.Version
	}

	// A replacement without a version is a relative path replacement and
	// should generally always be synced.
	if line.ReplaceVersion == "" {
		return ""
	}

	// Construct a version that depends on the module version, its
	// replacement path, and its replacement version. If any one of these
	// changes we will re-sync the module.
	return fmt.Sprintf("%s=>%s@%s", line.Version, line.ReplaceImportPath, line.ReplaceVersion)
}

type modulesTxtLineKind int

const (
	modulesTxtLineKindUnknown modulesTxtLineKind = iota
	modulesTxtLineKindModule
	modulesTxtLineKindPackage
)

func parseLine(line string) *modulesTxtLine {
	var l modulesTxtLine
	fields := strings.Fields(line)

	if len(fields) == 0 {
		return nil
	}

	next, remaining := slices.PopFirst(fields)

	const goVersionPrefix = "##"
	if next == goVersionPrefix {
		// This is a go version line which we ignore.
		return nil
	}

	const (
		modulePrefix  = "#"
		replaceArrow  = "=>"
		versionPrefix = "v"
	)
	if next == modulePrefix {
		if len(remaining) < 2 {
			// invalid module line
			return nil
		}
		l.Kind = modulesTxtLineKindModule

		l.ImportPath, remaining = slices.PopFirst(remaining)

		next, remaining = slices.PopFirst(remaining)

		if next == replaceArrow {
			// Redundant replace line without complete versions
			// which we skip.
			// # path/to/mod => path/to/replace/mod v1.2.3
			// or
			// # path/to/mod => ../path/to/replace/mod
			return nil
		}

		if !strings.HasPrefix(next, versionPrefix) {
			// invalid module line
			return nil
		}
		l.Version = next

		// simple module line
		// # path/to/mod v1.2.3
		if len(remaining) == 0 {
			return &l
		}

		// module replace line
		// # path/to/mod v1.2.3 => path/to/replace/mod v1.3.2
		// or
		// # path/to/mod v1.2.3 => ../path/to/replace/mod
		if len(remaining) < 2 {
			// invalid module line
			return nil
		}

		next, remaining = slices.PopFirst(remaining)
		if next != replaceArrow {
			// invalid module line
			return nil
		}

		l.ReplaceImportPath, remaining = slices.PopFirst(remaining)
		if len(remaining) == 0 {
			// this is a relative path replacement, which has no
			// version, so we clear the version. for example:
			// # path/to/mod v1.2.3 => ../path/to/replace/mod
			l.Version = ""
			return &l
		}
		if !strings.HasPrefix(remaining[0], versionPrefix) {
			// invalid module line
			return nil
		}
		// this is a versioned replacement, like
		// # path/to/mod v1.2.3 => path/to/replace/mod v1.3.2
		l.ReplaceVersion = remaining[0]
		return &l
	}

	// this is a package line
	l.Kind = modulesTxtLineKindPackage
	l.ImportPath = next
	return &l
}
