package index

import (
	"go/build"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aslevy.com/go-doc/internal/godoc"
	"github.com/stretchr/testify/require"
)

// func init() { Debug.Enable() }

type moduleSyncTest struct {
	name           string
	files          []fileSpec
	allPkgs        []string
	added, removed []string
}

type fileSpec struct {
	files  []string
	remove bool
}

func (spec fileSpec) exec(t *testing.T, modDir string) {
	t.Helper()
	require := require.New(t)
	for _, file := range spec.files {
		path := filepath.Join(modDir, file)
		if spec.remove {
			debug.Printf("removing %s", path)
			require.NoError(os.Remove(path))
		} else {
			debug.Printf("touching %s", path)
			touchFile(t, filepath.Join(modDir, file))
		}
	}
}
func touchFile(t *testing.T, path string) {
	t.Helper()
	require := require.New(t)
	require.NoError(os.MkdirAll(filepath.Dir(path), 0755))
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		f, err := os.Create(path)
		require.NoError(err)
		require.NoError(f.Close())
		return
	}
	require.NoError(err)
	currentTime := time.Now().Local()
	require.NoError(os.Chtimes(path, currentTime, currentTime))
}

var syncTests = []moduleSyncTest{{
	name: "initial sync",
	files: []fileSpec{{
		files: []string{
			"internal/formerpkg/formerpkg.go",
			"internal/formerpkg/formerpkg.yaml",
			"internal/pkg/pkg.go",
			"internal/removed/removed.go",
			"src/unchanged/unchanged.go",
		},
	}},
	added: []string{
		"internal/formerpkg",
		"internal/pkg",
		"internal/removed",
		"src/unchanged",
	},
}, {
	name: "add and remove packages",
	files: []fileSpec{{
		files: []string{
			"internal/added/added.go",
		},
	}, {
		files: []string{
			"internal/formerpkg/formerpkg.go",
			"internal/removed/removed.go",
			"internal/removed/.",
		},
		remove: true,
	}},
	added: []string{
		"internal/added",
	},
	removed: []string{
		"internal/formerpkg",
		"internal/removed",
	},
}}

func TestModuleSync(t *testing.T) {
	mod := toModule(godoc.PackageDir{"example.com/module", t.TempDir()})
	var allPkgs packageList
	for _, test := range syncTests {
		t.Run(test.name, func(t *testing.T) {
			for _, spec := range test.files {
				spec.exec(t, mod.Dir)
			}

			added, removed := mod.sync()

			require := require.New(t)
			require.WithinDuration(time.Now(), mod.UpdatedAt, time.Millisecond, "Module.updatedAt")

			expRemoved := toPackageList(mod, test.removed...)
			require.Equal(expRemoved, removed, "removed")

			expAdded := toPackageList(mod, test.added...)
			require.Equal(expAdded, added, "added")

			allPkgs.Remove(expRemoved...)
			allPkgs.Insert(expAdded...)
			require.Equal(allPkgs, mod.Packages, "all packages")
		})
	}
}

func toPackageList(mod module, importPaths ...string) (pkgs packageList) {
	if len(importPaths) == 0 {
		return
	}
	pkgs = make(packageList, 0, len(importPaths))
	for _, path := range importPaths {
		pkgs.Insert(mod.newPackage(path))
	}
	return
}

func BenchmarkModuleSync(b *testing.B) {
	var m module
	runBenchmark(b, func() {
		goModVendor(b, "testdata/module/")
		m = newModule("example.com/module/vendor", "testdata/module/vendor")
	}, func() {
		m.Packages = nil
		m.sync()
	})
	b.Log("num packages: ", len(m.Packages))
	// b.Log("packages:", m.Packages)
}
func BenchmarkModuleSync_stdlib(b *testing.B) {
	var m module
	runBenchmark(b, func() {
		m = newModule("", filepath.Join(build.Default.GOROOT, "src"))
	}, func() {
		m.Packages = nil
		m.sync()
	})
	b.Log("num packages: ", len(m.Packages))
	// b.Log("packages:", m.Packages)
}

func BenchmarkNewSync_stdlib(b *testing.B) {
	var pkgIdx *Packages
	codeRoots := []godoc.PackageDir{
		{"", filepath.Join(build.Default.GOROOT, "src")},
		{"cmd", filepath.Join(build.Default.GOROOT, "src", "cmd")},
	}
	runBenchmark(b, nil, func() {
		pkgIdx = New(codeRoots, WithNoProgressBar())
	})

	b.Log("longest import path: ", len(pkgIdx.partials))
	b.Log("num modules: ", len(pkgIdx.modules))
}
func BenchmarkSync_unchanged_stdlib(b *testing.B) {
	var changed bool
	var pkgIdx *Packages
	codeRoots := []godoc.PackageDir{
		{"", filepath.Join(build.Default.GOROOT, "src")},
		{"cmd", filepath.Join(build.Default.GOROOT, "src", "cmd")},
	}
	runBenchmark(b, func() {
		pkgIdx = New(codeRoots, WithNoProgressBar())
	}, func() {
		pkgIdx.updatedAt = time.Time{}
		changed = pkgIdx.sync(codeRoots)
	})

	b.Log("longest import path: ", len(pkgIdx.partials))
	b.Log("num modules: ", len(pkgIdx.modules))
	b.Log("changed: ", changed)
}
