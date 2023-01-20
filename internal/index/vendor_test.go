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
	testModule("github.com/davecgh/go-spew",
		"github.com/davecgh/go-spew/spew",
	),
}

func testModule(importPath string, pkgs ...string) module {
	mod := newModule(importPath, filepath.Join(testVendorDir, filepath.FromSlash(importPath)))
	mod.Class = classRequired
	mod.addPackages(pkgs...)
	return mod
}
