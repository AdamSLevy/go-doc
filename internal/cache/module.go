package cache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"aslevy.com/go-doc/internal/godoc"
)

var (
	Dir      string = getCacheDir()
	Rebuild  bool
	Disabled bool
)

func getCacheDir() string {
	if dir := os.Getenv("GODOC_CACHE_DIR"); dir != "" {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".cache", "go-doc")
	}
	return ""
}

type Cache struct {
	GoVersion string
	Dir       string
	Rebuild   bool
	Disabled  bool
}

func New(goCmd string) *Cache {
	goVer := goVersion(goCmd)
	return &Cache{
		GoVersion: goVer,
		Dir:       Dir,
		Rebuild:   Rebuild,
		Disabled:  Disabled || goVer == "" || Dir == "-" || Dir == "",
	}
}
func goVersion(goCmd string) string {
	stdout, _ := exec.Command(goCmd, "env", "GOVERSION").Output()
	return string(bytes.TrimSpace(stdout))
}

type Module struct {
	Root      godoc.Dir
	Packages  []godoc.Dir
	CreatedAt time.Time
}

func (c *Cache) NewModule(importPath, dir string) *Module {
	if c.Disabled ||
		!strings.Contains(filepath.Base(dir), "@") {
		return nil
	}
	return &Module{
		Root: godoc.Dir{
			ImportPath: importPath,
			Dir:        dir,
		},
		CreatedAt: time.Now(),
	}
}

func (m *Module) AddPackage(importPath, dir string) {
	if m == nil {
		return
	}
	m.Packages = append(m.Packages, godoc.Dir{
		ImportPath: importPath,
		Dir:        dir,
	})
}

func (m *Module) registerPackages(registerPackage func(importPath, dir string)) {
	for _, pkg := range m.Packages {
		registerPackage(pkg.ImportPath, pkg.Dir)
	}
}

func (c *Cache) Save(mod *Module) error {
	if c.Disabled || mod == nil || len(mod.Packages) == 0 {
		return nil
	}
	dir, file := c.moduleCachePath(mod.Root.ImportPath, mod.Root.Dir)
	if file == "" {
		return fmt.Errorf("module is outside of go mod cache")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache dir: %w", err)
	}

	f, err := os.Create(filepath.Join(dir, file))
	if err != nil {
		return fmt.Errorf("failed to create module cache file: %w", err)
	}
	defer f.Close()

	e := json.NewEncoder(f)
	if err := e.Encode(mod); err != nil {
		return fmt.Errorf("failed to encode module to cache file: %w", err)
	}

	return nil
}

type RegisterPackage = func(importPath, dir string)

func (c *Cache) Load(importPath, moduleDir string, register RegisterPackage) error {
	if c.Disabled || c.Rebuild {
		return nil
	}

	dir, file := c.moduleCachePath(importPath, moduleDir)
	if file == "" {
		return fmt.Errorf("module is outside of go mod cache")
	}

	f, err := os.Open(filepath.Join(dir, file))
	if err != nil {
		return fmt.Errorf("failed to open module cache file: %w", err)
	}
	defer f.Close()

	var module Module
	d := json.NewDecoder(f)
	if err := d.Decode(&module); err != nil {
		return fmt.Errorf("failed to decode module to cache file: %w", err)
	}

	if len(module.Packages) == 0 {
		return fmt.Errorf("no packages found in module cache file")
	}

	module.registerPackages(register)

	return nil
}

const (
	moduleCacheName   = "-module.json"
	moduleCacheStdlib = "stdlib@"
)

func (c *Cache) moduleCachePath(importPath, modDir string) (dir, filename string) {
	switch importPath {
	case "", "cmd":
		return c.Dir, moduleCacheStdlib + c.GoVersion + moduleCacheName
	}
	numImportSegments := strings.Count(importPath, "/")
	numDirSegments := strings.Count(modDir, string(os.PathSeparator))
	if numImportSegments > numDirSegments {
		return "", ""
	}
	slash := len(modDir)
	var lastSlash int
	for i := 0; i < numImportSegments; i++ {
		slash = strings.LastIndex(modDir[:slash], string(os.PathSeparator))
		if lastSlash == 0 {
			lastSlash = slash
		}
	}
	rel := modDir[slash+1 : lastSlash+1]
	dir = filepath.Join(c.Dir, rel)
	filename = modDir[lastSlash+1:] + moduleCacheName
	return dir, filename
}
