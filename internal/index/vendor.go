package index

import (
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/godoc"
	islices "aslevy.com/go-doc/internal/slices"
)

func (pkgIdx *Packages) syncVendored(vendor module) (vendored moduleList) {
	if pkgIdx.mode != ForceSync && !vendor.needsSyncVendor() {
		return pkgIdx.modules.vendored()
	}

	vendored = vendor.syncVendoredModules()
	// progressBar.ChangeMax(progressBar.GetMax() + len(vendored))
	for _, mod := range vendored {
		var pkgs packageList
		if pos, found := pkgIdx.modules.Search(mod); found {
			pkgs = pkgIdx.modules[pos].Packages
		}
		added, removed := islices.DiffSorted(pkgs, mod.Packages, comparePackages)
		pkgIdx.syncPartials(mod, added, removed)
		// progressBar.Add(1)
	}
	vendored.Insert(vendor)
	return
}
func (modList moduleList) vendored() (vendored moduleList) {
	for _, mod := range modList {
		if mod.Vendor {
			vendored = append(vendored, mod)
		}
	}
	return
}

func (vendor module) needsSyncVendor() bool {
	if !vendor.Vendor || filepath.Base(vendor.Dir) != "vendor" {
		return false
	}
	info, err := os.Stat(vendor.Dir)
	if err != nil {
		log.Printf("failed to stat %s: %v", vendor.Dir, err)
		return true
	}
	return info.ModTime().After(vendor.UpdatedAt)
}
func (vendor *module) syncVendoredModules() moduleList {
	mods := parseVendoredModules(vendor.Dir)
	if len(mods) > 0 {
		vendor.UpdatedAt = time.Now()
	}
	return mods
}

const modulesTxtFileName = "modules.txt"

func parseVendoredModules(vendorDir string) (mods moduleList) {
	debug.Printf("syncing vendor dir %s", vendorDir)

	modTxtPath := filepath.Join(vendorDir, modulesTxtFileName)
	f, err := os.Open(modTxtPath)
	if err != nil {
		log.Printf("failed to open %s: %v", modTxtPath, err)
		return nil
	}
	defer f.Close()

	return parseModulesTxtData(vendorDir, f)
}
func parseModulesTxtData(vendorDir string, data io.Reader) (mods moduleList) {
	updatedAt := time.Now()
	var mod module
	lines := bufio.NewScanner(data)
	for lines.Scan() {
		modImportPath, _, pkgImportPath := parseModuleTxtLine(lines.Text())
		if modImportPath != "" {
			if len(mod.Packages) > 0 {
				mods.Insert(mod)
			}
			mod = module{
				PackageDir: godoc.PackageDir{modImportPath, filepath.Join(vendorDir, modImportPath)},
				Class:      classRequired,
				Vendor:     true,
				UpdatedAt:  updatedAt,
			}
			continue
		}
		if pkgImportPath != "" {
			mod.addPackages(pkgImportPath)
		}
	}
	if len(mod.Packages) > 0 {
		mods = append(mods, mod)
	}

	if err := lines.Err(); err != nil {
		log.Printf("failed to parse lines from %s: %v", filepath.Join(vendorDir, modulesTxtFileName), err)
	}
	return mods
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
