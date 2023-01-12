package index

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/exp/slices"

	"aslevy.com/go-doc/internal/outfmt"
	"aslevy.com/go-doc/internal/pager"
)

type Packages interface {
	Search(path string, opts ...SearchOption) []string
	Encode(w io.Writer) error
	Save(path string) error
}

type packageIndex struct {
	Modules  moduleList
	Partials rightPartialListsByNumSlash

	syncProgress *progressbar.ProgressBar

	CreatedAt time.Time
	UpdatedAt time.Time
}

var _ Packages = (*packageIndex)(nil)

func New(required ...Module) Packages {
	pkgIdx := packageIndex{CreatedAt: time.Now()}
	pkgIdx.sync(required...)
	return &pkgIdx
}

func Load(path string, required ...Module) (Packages, error) {
	f, err := os.Open(path)
	if err != nil {
		return New(required...), err
	}
	defer f.Close()
	return Decode(f, required...)
}

// Decode the data from r into a new index.
//
// This is the inverse of Packages.Encode.
func Decode(r io.Reader, required ...Module) (Packages, error) {
	var idx packageIndex
	if err := json.NewDecoder(r).Decode(&idx); err != nil {
		return nil, err
	}
	idx.sync(required...)
	return &idx, nil
}

func (pkgIdx packageIndex) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return pkgIdx.Encode(f)
}
func (pkgIdx packageIndex) Encode(w io.Writer) error { return json.NewEncoder(w).Encode(pkgIdx) }

func (pkgIdx packageIndex) MarshalJSON() ([]byte, error) {
	pkgIdx.UpdatedAt = time.Now()
	type _packageIndex packageIndex
	return json.Marshal(_packageIndex(pkgIdx))
}

func (pkgIdx *packageIndex) sync(required ...Module) {
	progressBar := newProgressBar(len(pkgIdx.Modules), "syncing modules...")
	knownModules := append(moduleList{}, pkgIdx.Modules...)
	for _, req := range required {
		var mod *Module
		pos, found := pkgIdx.Modules.Search(req)
		if found {
			knownModules.Remove(req)
		} else {
			pkgIdx.Modules = slices.Insert(pkgIdx.Modules, pos, req)
		}
		mod = &pkgIdx.Modules[pos]
		// If the Dir has changed then we need to force a rescan. This
		// could be due to a minor version change, so its possible the
		// packages haven't changed much.
		if mod.Dir != req.Dir {
			mod.updatedAt = time.Time{}
		}
		added, removed := mod.sync()
		pkgIdx.syncPartials(*mod, added, removed)
		progressBar.Add(1)
	}

	// any remaining known modules have been removed...
	pkgIdx.Modules.Remove(knownModules...)
	for _, mod := range knownModules {
		pkgIdx.syncPartials(mod, nil, mod.packages)
		progressBar.Add(1)
	}
	progressBar.Finish()
	progressBar.Clear()
}
func (pkgIdx *packageIndex) syncPartials(mod Module, add, remove packageList) {
	modParts := strings.Split(mod.ImportPath, "/")
	for _, pkg := range remove {
		pkgIdx.Partials.Remove(modParts, pkg)
	}
	for _, pkg := range add {
		pkgIdx.Partials.Insert(modParts, pkg)
	}
}
func newProgressBar(total int, description string) *progressbar.ProgressBar {
	termMode := outfmt.Format == outfmt.Term && pager.IsTTY(os.Stderr)
	return progressbar.NewOptions(total,
		progressbar.OptionSetDescription("package index: "+description),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),               // show current count e.g. 3/5
		progressbar.OptionSetRenderBlankState(true), // render at 0%
		progressbar.OptionClearOnFinish(),           // clear bar when done
		progressbar.OptionUseANSICodes(termMode),
		progressbar.OptionEnableColorCodes(termMode),
	)
}
