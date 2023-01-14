package index

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

type Packages struct {
	modules  moduleList
	partials rightPartialListsByNumSlash

	createdAt    time.Time
	updatedAt    time.Time
	syncProgress *progressbar.ProgressBar
}

func New(required ...Module) *Packages {
	var pkgIdx Packages
	pkgIdx.createdAt = time.Now()
	pkgIdx.sync(required...)
	return &pkgIdx
}

func Load(path string, required ...Module) (*Packages, error) {
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
func Decode(r io.Reader, required ...Module) (*Packages, error) {
	var idx Packages
	if err := json.NewDecoder(r).Decode(&idx); err != nil {
		return nil, err
	}
	idx.sync(required...)
	return &idx, nil
}

func (pkgIdx Packages) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return pkgIdx.Encode(f)
}
func (pkgIdx Packages) Encode(w io.Writer) error { return json.NewEncoder(w).Encode(pkgIdx) }

func (pkgIdx Packages) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Modules   moduleList
		Partials  rightPartialListsByNumSlash
		CreatedAt time.Time
		UpdatedAt time.Time
	}{
		Modules:   pkgIdx.modules,
		Partials:  pkgIdx.partials,
		CreatedAt: pkgIdx.createdAt,
		UpdatedAt: pkgIdx.updatedAt,
	})
}
