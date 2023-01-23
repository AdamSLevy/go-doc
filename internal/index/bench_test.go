package index

import (
	"go/build"
	"path/filepath"
	"testing"

	"aslevy.com/go-doc/internal/godoc"
)

// Benchmarks
//
// - Load
// - Save
//  - various size index files
//  - mostly comes down to encoding/decoding efficiency
//
// - Sync
//  - Force sync
//  - No changes
// - Search

// func BenchmarkLoad(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		Load("testdata/index.json", WithNoSync())
// 	}
// }

func BenchmarkParseVendoredModules(b *testing.B) {
	goModVendor(b, "testdata/module/")
	var mods moduleList
	for i := 0; i < b.N; i++ {
		mods = parseVendoredModules("testdata/module/vendor")
		if mods == nil {
			b.Fatal("failed to parse vendored modules")
		}
	}
	b.Log("num modules: ", len(mods))
	// b.Log("modules:", mods)
}

func BenchmarkModuleSync(b *testing.B) {
	goModVendor(b, "testdata/module/")
	m := newModule("example.com/module/vendor", "testdata/module/vendor")
	for i := 0; i < b.N; i++ {
		m.Packages = nil
		m.sync()
	}
	b.Log("num packages: ", len(m.Packages))
	// b.Log("packages:", m.Packages)
}
func BenchmarkModuleSync_stdlib(b *testing.B) {
	m := newModule("", filepath.Join(build.Default.GOROOT, "src"))
	for i := 0; i < b.N; i++ {
		m.Packages = nil
		m.sync()
	}
	b.Log("num packages: ", len(m.Packages))
	// b.Log("packages:", m.Packages)
}

func BenchmarkNewSync_stdlib(b *testing.B) {
	codeRoots := []godoc.PackageDir{
		{"", filepath.Join(build.Default.GOROOT, "src")},
		{"cmd", filepath.Join(build.Default.GOROOT, "src", "cmd")},
	}
	var pkgIdx *Packages
	for i := 0; i < b.N; i++ {
		pkgIdx = New(codeRoots, WithNoProgressBar())
	}
	b.Log("longest import path: ", len(pkgIdx.partials))
	b.Log("num modules: ", len(pkgIdx.modules))
}
