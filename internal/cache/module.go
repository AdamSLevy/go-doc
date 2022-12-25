package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	Dir      string
	Rebuild  bool
	Disabled bool
)

type Module struct {
	ImportPath string
	Dir        string

	Packages []Package

	CreatedAt time.Time
}

type Package struct {
	ImportPath string
	Dir        string
}

func NewModule(importPath, dir string) Module {
	return Module{
		ImportPath: importPath,
		Dir:        dir,
		CreatedAt:  time.Now(),
	}
}

type Modules struct {
	GoModCache string
	GoVersion  string
	CacheDir   string
}

func (c *Modules) Save(mod *Module) error {
	dir, file := c.moduleCachePath(mod.ImportPath, mod.Dir)
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

func (c *Modules) Load(importPath, moduleDir string) (*Module, error) {
	dir, file := c.moduleCachePath(importPath, moduleDir)
	if file == "" {
		return nil, fmt.Errorf("module is outside of go mod cache")
	}

	f, err := os.Open(filepath.Join(dir, file))
	if err != nil {
		return nil, fmt.Errorf("failed to open module cache file: %w", err)
	}
	defer f.Close()

	var module Module
	d := json.NewDecoder(f)
	if err := d.Decode(&module); err != nil {
		return nil, fmt.Errorf("failed to decode module to cache file: %w", err)
	}

	return &module, nil
}

const moduleCacheName = "-module.json"

func (c *Modules) moduleCachePath(importPath, moduleDir string) (string, string) {
	switch importPath {
	case "":
		moduleDir = filepath.Join("go", "stdlib@"+c.GoVersion)
	case "cmd":
		moduleDir = filepath.Join("go", "cmd@"+c.GoVersion)
	default:
		if strings.HasPrefix(moduleDir, c.GoModCache) {
			moduleDir = moduleDir[len(c.GoModCache):]
		} else {
			return "", ""
		}
	}
	dir, file := filepath.Split(moduleDir)
	return filepath.Join(c.CacheDir, dir), file + moduleCacheName
}
