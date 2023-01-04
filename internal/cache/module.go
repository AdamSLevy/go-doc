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

	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/godoc"
)

var (
	Dir      string = getCacheDir()
	Rebuild  bool
	Disabled bool
)

const (
	PathSeparator       = string(os.PathSeparator)
	ImportPathSeparator = "/"
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
	Dir string

	GoVersion  string
	GoModCache string

	Rebuild  bool
	Disabled bool
}

func New(goCmd string) *Cache {
	goVersion, goModCache := goInfo(goCmd)
	disabled := Disabled ||
		goVersion == "" || goModCache == "" ||
		Dir == "-" || Dir == ""
	return &Cache{
		Dir:        Dir,
		GoVersion:  goVersion,
		GoModCache: goModCache,
		Rebuild:    Rebuild,
		Disabled:   disabled,
	}
}
func goInfo(goCmd string) (goVersion, goModCache string) {
	defer func() {
	}()

	args := []string{"env", "GOVERSION", "GOMODCACHE"}
	stdout, err := exec.Command(goCmd, args...).Output()
	if err != nil {
		dlog.Printf("failed to run `%s %s`: %v", goCmd, strings.Join(args, " "), err)
		return "", ""
	}

	stdout = bytes.TrimSpace(stdout)
	values := bytes.SplitN(stdout, []byte{'\n'}, 3)
	if len(values) != 2 {
		dlog.Printf("failed to parse the output of `%s %s`: %q", goCmd, strings.Join(args, " "), string(stdout))
		return "", ""
	}
	goVersion = string(bytes.TrimSpace(values[0]))
	goModCache = string(bytes.TrimSpace(values[1]))
	goModCache = filepath.Clean(goModCache)
	dlog.Printf("cache: go info: GOVERSION=%q", goVersion)
	dlog.Printf("cache: go info: GOMODCACHE=%q", goModCache)
	return
}

type Module struct {
	Root      godoc.Dir
	Packages  []godoc.Dir
	CachePath string
	CreatedAt time.Time
}

func (c *Cache) NewModule(importPath, dir string) *Module {
	cachePath := c.moduleCachePath(importPath, dir)
	if cachePath == "" {
		return nil
	}
	return &Module{
		Root: godoc.Dir{
			ImportPath: importPath,
			Dir:        dir,
		},
		CachePath: cachePath,
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
	if c.Disabled ||
		mod == nil ||
		mod.CachePath == "" ||
		len(mod.Packages) == 0 {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(mod.CachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache dir: %w", err)
	}

	f, err := os.Create(mod.CachePath)
	if err != nil {
		return fmt.Errorf("failed to create module cache file: %w", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(mod); err != nil {
		return fmt.Errorf("failed to encode module to cache file: %w", err)
	}

	return nil
}

type RegisterPackage = func(importPath, dir string)

func (c *Cache) Load(importPath, moduleDir string, register RegisterPackage) error {
	if c.Disabled || c.Rebuild {
		return fmt.Errorf("cache disabled")
	}

	cachePath := c.moduleCachePath(importPath, moduleDir)
	if cachePath == "" {
		return fmt.Errorf("module is outside of go mod cache")
	}

	f, err := os.Open(cachePath)
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

func (c *Cache) moduleCachePath(importPath, modDir string) string {
	if c.Disabled {
		return ""
	}

	if isStdlib(importPath) {
		return filepath.Join(c.Dir, "go-stdlib",
			// importPath is empty or "cmd", so this will either be
			// "src" or "src/cmd"
			"src", importPath) +
			"@" + c.GoVersion + moduleCacheName
	}

	if !c.isInGoModCache(modDir) || !hasVersionSuffix(modDir) {
		return ""
	}

	// relModDir is the module directory relative to the go mod cache. We
	// want to use the same directory structure as the go mod cache because
	// the file names it uses are robust to case-insensitive file systems.
	relModDir := strings.TrimPrefix(modDir, c.GoModCache)
	return filepath.Join(c.Dir, relModDir) + moduleCacheName
}
func isStdlib(importPath string) bool {
	switch importPath {
	case "", "cmd":
		return true
	}
	return false
}
func (c *Cache) isInGoModCache(dir string) bool {
	return strings.HasPrefix(dir, c.GoModCache+PathSeparator)
}
func hasVersionSuffix(dir string) bool {
	return strings.Contains(filepath.Base(dir), "@")
}
