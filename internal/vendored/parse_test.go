package vendored

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"aslevy.com/go-doc/internal/benchmark"
	"aslevy.com/go-doc/internal/godoc"
	"github.com/stretchr/testify/require"
)

var testVendorDir = filepath.FromSlash("../index/testdata/module/vendor")

func TestParseVendoredModules(t *testing.T) {
	goModVendor(t, testVendorDir)
	ctx := context.Background()
	modPkgs, err := ParseModulePackages(ctx, testVendorDir)
	require.NoError(t, err)
	require.Equal(t, vendoredModules, modPkgs)
}

type T interface {
	require.TestingT
	Helper()
	Cleanup(func())
}

func goModVendor(t T, vendorDir string) {
	t.Helper()
	modDir := filepath.Dir(vendorDir)
	cmd := exec.Command("go", "mod", "vendor")
	cmd.Dir = modDir
	require.NoError(t, cmd.Run())
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(filepath.Join(modDir, "vendor"))) })
}

var vendoredModules = ModulePackages{
	testModule("aslevy.com/go-doc"): testPackages(
		"aslevy.com/go-doc/testdata/codeblocks",
	),
	testModule("github.com/alecthomas/chroma"): testPackages(
		"github.com/alecthomas/chroma",
		"github.com/alecthomas/chroma/formatters",
		"github.com/alecthomas/chroma/formatters/html",
		"github.com/alecthomas/chroma/formatters/svg",
		"github.com/alecthomas/chroma/lexers",
		"github.com/alecthomas/chroma/lexers/a",
		"github.com/alecthomas/chroma/lexers/b",
		"github.com/alecthomas/chroma/lexers/c",
		"github.com/alecthomas/chroma/lexers/circular",
		"github.com/alecthomas/chroma/lexers/d",
		"github.com/alecthomas/chroma/lexers/e",
		"github.com/alecthomas/chroma/lexers/f",
		"github.com/alecthomas/chroma/lexers/g",
		"github.com/alecthomas/chroma/lexers/h",
		"github.com/alecthomas/chroma/lexers/i",
		"github.com/alecthomas/chroma/lexers/internal",
		"github.com/alecthomas/chroma/lexers/j",
		"github.com/alecthomas/chroma/lexers/k",
		"github.com/alecthomas/chroma/lexers/l",
		"github.com/alecthomas/chroma/lexers/m",
		"github.com/alecthomas/chroma/lexers/n",
		"github.com/alecthomas/chroma/lexers/o",
		"github.com/alecthomas/chroma/lexers/p",
		"github.com/alecthomas/chroma/lexers/q",
		"github.com/alecthomas/chroma/lexers/r",
		"github.com/alecthomas/chroma/lexers/s",
		"github.com/alecthomas/chroma/lexers/t",
		"github.com/alecthomas/chroma/lexers/v",
		"github.com/alecthomas/chroma/lexers/w",
		"github.com/alecthomas/chroma/lexers/x",
		"github.com/alecthomas/chroma/lexers/y",
		"github.com/alecthomas/chroma/lexers/z",
		"github.com/alecthomas/chroma/quick",
		"github.com/alecthomas/chroma/styles",
	),
	testModule("github.com/aymanbagabas/go-osc52"): testPackages(
		"github.com/aymanbagabas/go-osc52",
	),
	testModule("github.com/aymerick/douceur"): testPackages(
		"github.com/aymerick/douceur/css",
		"github.com/aymerick/douceur/parser",
	),
	testModule("github.com/charmbracelet/glamour"): testPackages(
		"github.com/charmbracelet/glamour",
		"github.com/charmbracelet/glamour/ansi",
	),
	testModule("github.com/davecgh/go-spew"): testPackages(
		"github.com/davecgh/go-spew/spew",
	),
	testModule("github.com/dlclark/regexp2"): testPackages(
		"github.com/dlclark/regexp2",
		"github.com/dlclark/regexp2/syntax",
	),
	testModule("github.com/gorilla/css"): testPackages(
		"github.com/gorilla/css/scanner",
	),
	testModule("github.com/lucasb-eyer/go-colorful"): testPackages(
		"github.com/lucasb-eyer/go-colorful",
	),
	testModule("github.com/mattn/go-isatty"): testPackages(
		"github.com/mattn/go-isatty",
	),
	testModule("github.com/mattn/go-runewidth"): testPackages(
		"github.com/mattn/go-runewidth",
	),
	testModule("github.com/microcosm-cc/bluemonday"): testPackages(
		"github.com/microcosm-cc/bluemonday",
		"github.com/microcosm-cc/bluemonday/css",
	),
	testModule("github.com/muesli/reflow"): testPackages(
		"github.com/muesli/reflow/ansi",
		"github.com/muesli/reflow/indent",
		"github.com/muesli/reflow/padding",
		"github.com/muesli/reflow/wordwrap",
	),
	testModule("github.com/muesli/termenv"): testPackages(
		"github.com/muesli/termenv",
	),
	testModule("github.com/olekukonko/tablewriter"): testPackages(
		"github.com/olekukonko/tablewriter",
	),
	testModule("github.com/rivo/uniseg"): testPackages(
		"github.com/rivo/uniseg",
	),
	testModule("github.com/yuin/goldmark"): testPackages(
		"github.com/yuin/goldmark",
		"github.com/yuin/goldmark/ast",
		"github.com/yuin/goldmark/extension",
		"github.com/yuin/goldmark/extension/ast",
		"github.com/yuin/goldmark/parser",
		"github.com/yuin/goldmark/renderer",
		"github.com/yuin/goldmark/renderer/html",
		"github.com/yuin/goldmark/text",
		"github.com/yuin/goldmark/util",
	),
	testModule("github.com/yuin/goldmark-emoji"): testPackages(
		"github.com/yuin/goldmark-emoji",
		"github.com/yuin/goldmark-emoji/ast",
		"github.com/yuin/goldmark-emoji/definition",
	),
	testModule("golang.org/x/net"): testPackages(
		"golang.org/x/net/html",
		"golang.org/x/net/html/atom",
	),
	testModule("golang.org/x/sys"): testPackages(
		"golang.org/x/sys/internal/unsafeheader",
		"golang.org/x/sys/unix",
		"golang.org/x/sys/windows",
	),
}

func testModule(importPath string) godoc.PackageDir {
	return godoc.NewPackageDir(importPath, filepath.Join(testVendorDir, filepath.FromSlash(importPath)))
}
func testPackages(importPaths ...string) []godoc.PackageDir {
	pkgs := make([]godoc.PackageDir, len(importPaths))
	for i, importPath := range importPaths {
		pkgs[i] = godoc.NewPackageDir(importPath, "")
	}
	return pkgs
}

func BenchmarkParseVendoredModules(b *testing.B) {
	ctx := context.Background()
	var modPkgs ModulePackages
	benchmark.Run(b, func() {
		goModVendor(b, "testdata/module/")
	}, func() {
		modPkgs, _ = ParseModulePackages(ctx, testVendorDir)
	})
	b.Log("num modules: ", len(modPkgs))
	// b.Log("modules:", mods)
}
