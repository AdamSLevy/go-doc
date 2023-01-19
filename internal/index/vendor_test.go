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

func goModVendor(t *testing.T, modDir string) {
	t.Helper()
	cmd := exec.Command("go", "mod", "vendor")
	cmd.Dir = filepath.FromSlash(modDir)
	require.NoError(t, cmd.Run())
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(filepath.Join(modDir, "vendor"))) })
}

func testEqualModule(t *testing.T, exp, got Module) {
	t.Helper()
	assert := assert.New(t)
	assert.Equal(exp.ImportPath, got.ImportPath)
	assert.Equal(exp.Dir, got.Dir, exp.ImportPath)
	assert.Equal(exp.packages, got.packages, exp.ImportPath)
	assert.True(got.vendor, exp.ImportPath)
	updatedAt := exp.updatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	assert.WithinDuration(updatedAt, got.updatedAt, time.Millisecond, exp.ImportPath)
}

var testVendorDir = filepath.FromSlash("./testdata/module/vendor")
var vendoredModules = moduleList{
	newModule("aslevy.com/go-doc",
		"aslevy.com/go-doc/testdata/codeblocks",
	),
	newModule("github.com/davecgh/go-spew",
		"github.com/davecgh/go-spew/spew",
	),
}

func newModule(importPath string, pkgs ...string) Module {
	mod := NewModule(importPath, filepath.Join(testVendorDir, filepath.FromSlash(importPath)))
	mod.class = classRequired
	mod.addPackages(pkgs...)
	return mod
}
