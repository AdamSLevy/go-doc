package main

import (
	"bytes"
	"log"
	"os/exec"

	"aslevy.com/go-doc/internal/godoc"
)

var GOMODCACHE, GOMOD string

func init() {
	stdout, err := exec.Command("go", "env", "GOROOT", "GOMODCACHE", "GOMOD").Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			log.Fatalf("failed to determine GOROOT: 'go env GOROOT' failed:\n%s", ee.Stderr)
		}
		log.Fatalf("failed to determine GOROOT: $GOROOT is not set and could not run 'go env GOROOT':\n\t%s", err)
	}

	lines := bytes.Split(stdout, []byte("\n"))
	if len(lines) < 3 {
		panic("failed to parse stdout from `go env GOROOT GOMODCACHE GOMOD`\n" + string(stdout))
	}
	buildCtx.GOROOT = string(bytes.TrimSpace(lines[0]))
	GOMODCACHE = string(bytes.TrimSpace(lines[1]))
	GOMOD = string(bytes.TrimSpace(lines[2]))
}

var xdirs godoc.Dirs = dirs.PackageDirs()

func (dirs *Dirs) PackageDirs() *PackageDirs { return (*PackageDirs)(dirs) }

type PackageDirs Dirs

func (d *PackageDirs) dirs() *Dirs { return (*Dirs)(d) }

func (d *PackageDirs) Next() (godoc.PackageDir, bool) {
	dir, ok := d.dirs().Next()
	return godoc.NewPackageDir(dir.importPath, dir.dir), ok
}

func (d *PackageDirs) Reset() { d.dirs().Reset() }

func (d *PackageDirs) FilterExact(string) error   { return godoc.ErrFilterNotSupported }
func (d *PackageDirs) FilterPartial(string) error { return godoc.ErrFilterNotSupported }

func (dirs *Dirs) registerPackage(importPath, dir string) { dirs.scan <- Dir{importPath, dir, true} }
