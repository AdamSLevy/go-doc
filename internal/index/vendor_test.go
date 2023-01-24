package index

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVendoredModules(t *testing.T) {
	goModVendor(t, "./testdata/module")
	mods := parseVendoredModules(testVendorDir)
	require.Len(t, mods, len(vendoredModules))
	for i, mod := range mods {
		testEqualModule(t, vendoredModules[i], mod)
	}
}

type T interface {
	require.TestingT
	Helper()
	Cleanup(func())
}

func goModVendor(t T, modDir string) {
	t.Helper()
	cmd := exec.Command("go", "mod", "vendor")
	cmd.Dir = filepath.FromSlash(modDir)
	require.NoError(t, cmd.Run())
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(filepath.Join(modDir, "vendor"))) })
}

func testEqualModule(t *testing.T, exp, got module) {
	t.Helper()
	assert := assert.New(t)
	assert.Equal(exp.ImportPath, got.ImportPath)
	assert.Equal(exp.Dir, got.Dir, exp.ImportPath)
	assert.Equal(exp.Packages, got.Packages, exp.ImportPath)
	assert.True(got.Vendor, exp.ImportPath)
	updatedAt := exp.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	assert.WithinDuration(updatedAt, got.UpdatedAt, time.Millisecond, exp.ImportPath)
}

var testVendorDir = filepath.FromSlash("./testdata/module/vendor")
var vendoredModules = moduleList{
	testModule("aslevy.com/go-doc",
		"aslevy.com/go-doc/testdata/codeblocks",
	),
	testModule("github.com/alecthomas/chroma",
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
	testModule("github.com/aymanbagabas/go-osc52",
		"github.com/aymanbagabas/go-osc52",
	),
	testModule("github.com/aymerick/douceur",
		"github.com/aymerick/douceur/css",
		"github.com/aymerick/douceur/parser",
	),
	testModule("github.com/charmbracelet/glamour",
		"github.com/charmbracelet/glamour",
		"github.com/charmbracelet/glamour/ansi",
	),
	testModule("github.com/davecgh/go-spew",
		"github.com/davecgh/go-spew/spew",
	),
	testModule("github.com/dlclark/regexp2",
		"github.com/dlclark/regexp2",
		"github.com/dlclark/regexp2/syntax",
	),
	testModule("github.com/gorilla/css",
		"github.com/gorilla/css/scanner",
	),
	testModule("github.com/lucasb-eyer/go-colorful",
		"github.com/lucasb-eyer/go-colorful",
	),
	testModule("github.com/mattn/go-isatty",
		"github.com/mattn/go-isatty",
	),
	testModule("github.com/mattn/go-runewidth",
		"github.com/mattn/go-runewidth",
	),
	testModule("github.com/microcosm-cc/bluemonday",
		"github.com/microcosm-cc/bluemonday",
		"github.com/microcosm-cc/bluemonday/css",
	),
	testModule("github.com/muesli/reflow",
		"github.com/muesli/reflow/ansi",
		"github.com/muesli/reflow/indent",
		"github.com/muesli/reflow/padding",
		"github.com/muesli/reflow/wordwrap",
	),
	testModule("github.com/muesli/termenv",
		"github.com/muesli/termenv",
	),
	testModule("github.com/olekukonko/tablewriter",
		"github.com/olekukonko/tablewriter",
	),
	testModule("github.com/rivo/uniseg",
		"github.com/rivo/uniseg",
	),
	testModule("github.com/yuin/goldmark",
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
	testModule("github.com/yuin/goldmark-emoji",
		"github.com/yuin/goldmark-emoji",
		"github.com/yuin/goldmark-emoji/ast",
		"github.com/yuin/goldmark-emoji/definition",
	),
	testModule("golang.org/x/net",
		"golang.org/x/net/html",
		"golang.org/x/net/html/atom",
	),
	testModule("golang.org/x/sys",
		"golang.org/x/sys/internal/unsafeheader",
		"golang.org/x/sys/unix",
		"golang.org/x/sys/windows",
	),
}

func testModule(importPath string, pkgs ...string) module {
	mod := newModule(importPath, filepath.Join(testVendorDir, filepath.FromSlash(importPath)))
	mod.Class = classRequired
	mod.addPackages(pkgs...)
	return mod
}

func BenchmarkParseVendoredModules(b *testing.B) {
	var mods moduleList
	runBenchmark(b, func() {
		goModVendor(b, "testdata/module/")
	}, func() {
		mods = parseVendoredModules("testdata/module/vendor")
	})
	b.Log("num modules: ", len(mods))
	// b.Log("modules:", mods)
}
