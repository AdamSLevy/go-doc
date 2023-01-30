package index

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"aslevy.com/go-doc/internal/godoc"
)

func (idx *Index) syncVendoredModules(ctx context.Context, vendorRoot godoc.PackageDir) error {
	if idx.vendorUnchanged(vendorRoot) {
		return nil
	}
	const modulesTxt = "modules.txt"
	modTxtPath := filepath.Join(vendorRoot.Dir, modulesTxt)
	f, err := os.Open(modTxtPath)
	if err != nil {
		log.Printf("failed to open %s: %v", modTxtPath, err)
		return nil
	}
	defer f.Close()

	return idx.syncVendoredModulesTxtFile(ctx, vendorRoot, f)
}
func (idx *Index) vendorUnchanged(vendor godoc.PackageDir) bool {
	info, err := os.Stat(vendor.Dir)
	if err != nil {
		log.Printf("failed to stat %s: %v", vendor.Dir, err)
		return true
	}
	return idx.UpdatedAt.After(info.ModTime())
}
func (idx *Index) syncVendoredModulesTxtFile(ctx context.Context, vendorRoot godoc.PackageDir, data io.Reader) error {
	const vendor = true
	var (
		err              error
		modID            int64
		modRoot          godoc.PackageDir
		modKeep, pkgKeep []int64
	)
	lines := bufio.NewScanner(data)
	for lines.Scan() && ctx.Err() == nil {
		modImportPath, _, pkgImportPath := parseModuleTxtLine(lines.Text())
		if modImportPath != "" {
			if modID > 0 {
				if err := idx.prunePackages(ctx, modID, pkgKeep); err != nil {
					return err
				}
				pkgKeep = pkgKeep[:0]
			}
			modRoot = godoc.NewPackageDir(
				modImportPath,
				filepath.Join(vendorRoot.Dir, modImportPath),
			)
			modID, _, err = idx.upsertModule(ctx, modRoot, classRequired, vendor)
			if err != nil {
				return err
			}
			modKeep = append(modKeep, modID)
			continue
		}
		if pkgImportPath != "" && modID > 0 {
			pkgID, err := idx.syncPackage(ctx, modID, modRoot, godoc.NewPackageDir(pkgImportPath, ""))
			if err != nil {
				return err
			}
			pkgKeep = append(pkgKeep, pkgID)
		}
	}
	if modID > 0 && len(pkgKeep) > 0 {
		if err := idx.prunePackages(ctx, modID, pkgKeep); err != nil {
			return err
		}
	}

	return idx.pruneModules(ctx, vendor, modKeep)
}
func parseModuleTxtLine(line string) (modImportPath, modVersion, pkgImportPath string) {
	defer func() {
		dlog.Printf("parseModuleTxtLine(%q) (%q, %q, %q)",
			line, modImportPath, modVersion, pkgImportPath)
	}()
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
