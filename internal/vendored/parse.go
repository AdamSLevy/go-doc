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
)

type ModulePackages map[godoc.PackageDir][]godoc.PackageDir

func ParseModulePackages(ctx context.Context, vendorDir string) (ModulePackages, error) {
	modPkgs := make(ModulePackages)
	return modPkgs, modPkgs.parse(ctx, vendorDir)
}

func (modPkgs ModulePackages) parse(ctx context.Context, vendorDir string) error {
	return Handler(func(_ context.Context, mod godoc.PackageDir, pkgs ...godoc.PackageDir) error {
		modPkgs[mod] = append(modPkgs[mod], pkgs...)
		return nil
	}).parse(ctx, vendorDir)
}

type Handler func(ctx context.Context, mod godoc.PackageDir, pkgs ...godoc.PackageDir) error

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
func (handle Handler) parseData(ctx context.Context, vendorDir string, data io.Reader) error {
	var mod godoc.PackageDir
	var pkgs []godoc.PackageDir
	lines := bufio.NewScanner(data)
	for lines.Scan() && ctx.Err() == nil {
		modImportPath, _, pkgImportPath := parseLine(lines.Text())
		if modImportPath != "" {
			if mod.Dir != "" {
				if err := handle(ctx, mod, pkgs...); err != nil {
					return err
				}
			}
			mod = godoc.NewPackageDir(
				modImportPath,
				filepath.Join(vendorDir, modImportPath),
			)
			pkgs = pkgs[:0]
			continue
		}
		if pkgImportPath != "" {
			if mod.Dir == "" {
				return fmt.Errorf("found package %q before a module", pkgImportPath)
			}
			if !strings.HasPrefix(pkgImportPath, mod.ImportPath) {
				return fmt.Errorf("package %q is not in module %q", pkgImportPath, mod.ImportPath)
			}
			pkgs = append(pkgs, godoc.NewPackageDir(pkgImportPath, ""))
		}
	}
	return nil
}
func parseLine(line string) (modImportPath, modVersion, pkgImportPath string) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return
	}
	switch fields[0] {
	case "#":
		// module
		if len(fields) < 3 {
			return
		}
		modImportPath, modVersion = fields[1], fields[2]
		if !strings.HasPrefix(modVersion, "v") {
			modVersion = ""
		}
	case "##":
		// ignore
	default:
		pkgImportPath = fields[0]
	}
	return
}
