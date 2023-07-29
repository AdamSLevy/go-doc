// This file along with schema.sql define the schema for the database.
//
// For each SQL table there is a corresponding Go type and Index methods for
// selecting, inserting, or updating rows.

package index

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"hash/crc32"
	"runtime/debug"
	"time"

	"aslevy.com/go-doc/internal/godoc"
)

// _schema is the SQL schema for the index database.
//
//go:embed schema.sql
var _schema []byte

var schemaCRC = func() uint32 {
	crc := crc32.NewIEEE()
	crc.Write(_schema)
	return crc.Sum32()
}()

type metadata struct {
	CreatedAt time.Time
	UpdatedAt time.Time

	BuildRevision string
	GoVersion     string
}

func (idx *Index) selectMetadata(ctx context.Context) (metadata, error) {
	const query = `
SELECT createdAt, updatedAt, buildRevision, goVersion FROM metadata WHERE rowid=1;
`
	return scanMetadata(idx.db.QueryRowContext(ctx, query))
}
func scanMetadata(row sqlRow) (metadata, error) {
	var meta metadata
	return meta, row.Scan(
		&meta.CreatedAt,
		&meta.UpdatedAt,
		&meta.BuildRevision,
		&meta.GoVersion,
	)
}

func (idx *Index) upsertMetadata(ctx context.Context) error {
	const query = `
INSERT INTO metadata(rowid, buildRevision, goVersion) VALUES (1, ?, ?)
  ON CONFLICT(rowid) DO 
    UPDATE SET 
      updatedAt=CURRENT_TIMESTAMP, 
      buildRevision=excluded.buildRevision,
      goVersion=excluded.goVersion;
`
	if _, err := idx.tx.ExecContext(ctx, query, buildRevision, goVersion); err != nil {
		return fmt.Errorf("failed to upsert metadata: %w", err)
	}
	return nil
}

var buildRevision, goVersion string = func() (string, string) {
	var buildRevision string
	info, ok := debug.ReadBuildInfo()
	if !ok {
		panic("debug.ReadBuildInfo() failed")
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			buildRevision = s.Value
			break
		}
	}
	return buildRevision, info.GoVersion
}()

type class = int

const (
	classStdlib class = iota
	classLocal
	classRequired
	classNotRequired
)

func classString(c class) string {
	switch c {
	case classStdlib:
		return "stdlib"
	case classLocal:
		return "local"
	case classRequired:
		return "required"
	case classNotRequired:
		return "not required"
	default:
		return "unknown class"
	}
}

type module struct {
	ID         int64
	ImportPath string
	Dir        string
	Class      class
	Vendor     bool
}

func (idx *Index) selectModule(ctx context.Context, importPath string) (module, error) {
	stmt, err := idx.tx.PrepareContext(ctx, `
SELECT rowid, importPath, dir, class, vendor FROM module WHERE importPath=?;
`)
	if err != nil {
		return module{}, err
	}
	return scanModule(stmt.QueryRowContext(ctx, importPath))
}
func scanModule(row sqlRow) (module, error) {
	var mod module
	return mod, row.Scan(&mod.ID, &mod.ImportPath, &mod.Dir, &mod.Class, &mod.Vendor)
}

type sqlRow interface {
	Scan(dest ...any) error
}

func (idx *Index) insertModule(ctx context.Context, pkgDir godoc.PackageDir, class class, vendor bool) (int64, error) {
	stmt, err := idx.tx.PrepareContext(ctx, `
INSERT INTO module (importPath, dir, class, vendor) VALUES (?, ?, ?, ?);
`)
	if err != nil {
		return -1, err
	}
	res, err := stmt.ExecContext(ctx, pkgDir.ImportPath, pkgDir.Dir, int(class), vendor)
	if err != nil {
		return -1, nil
	}
	return res.LastInsertId()
}

func (idx *Index) updateModule(ctx context.Context, modID int64, pkgDir godoc.PackageDir, class class, vendor bool) error {
	stmt, err := idx.tx.PrepareContext(ctx, `
UPDATE module SET (dir, class, vendor) = (?, ?, ?) WHERE rowid=?;
`)
	if err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx, modID, pkgDir.Dir, int(class), vendor)
	return err
}

func (idx *Index) pruneModules(ctx context.Context, vendor bool, keep []int64) error {
	query := fmt.Sprintf(`
DELETE FROM module WHERE vendor=? AND rowid NOT IN (%s);
`, placeholders(len(keep)))
	_, err := idx.tx.ExecContext(ctx, query, pruneModulesArgs(vendor, keep)...)
	return err
}
func placeholders(n int) string {
	var buf bytes.Buffer
	for i := 0; i < n; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('?')
	}
	return buf.String()
}
func pruneModulesArgs(vendor bool, keep []int64) []any {
	args := make([]any, 0, len(keep)+1)
	args = append(args, vendor)
	for _, id := range keep {
		args = append(args, id)
	}
	return args
}

type package_ struct {
	ID           int64
	ModuleID     int64
	RelativePath string
	NumParts     int
}

func (idx *Index) selectPackageID(ctx context.Context, modID int64, relativePath string) (int64, error) {
	stmt, err := idx.tx.PrepareContext(ctx, `
SELECT rowid FROM package WHERE moduleId=? AND relativePath=?;
`)
	if err != nil {
		return -1, err
	}
	var id int64
	return id, stmt.QueryRowContext(ctx, modID, relativePath).Scan(&id)
}

func (idx *Index) insertPackage(ctx context.Context, modID int64, relativePath string) (int64, error) {
	stmt, err := idx.tx.PrepareContext(ctx, `
INSERT INTO package(moduleId, relativePath) VALUES (?, ?);
`)
	if err != nil {
		return -1, err
	}
	res, err := stmt.ExecContext(ctx, modID, relativePath)
	if err != nil {
		return -1, fmt.Errorf("failed to insert package: %w", err)
	}
	return res.LastInsertId()
}

func (idx *Index) prunePackages(ctx context.Context, modID int64, keep []int64) error {
	dlog.Printf("pruning unused packages for module %d", modID)
	query := fmt.Sprintf(`
DELETE FROM package WHERE moduleId=? AND rowid NOT IN (%s);
`, placeholders(len(keep)))
	_, err := idx.tx.ExecContext(ctx, query, prunePackagesArgs(modID, keep)...)
	if err != nil {
		return fmt.Errorf("failed to prune packages: %w", err)
	}
	return nil
}
func prunePackagesArgs(modID int64, keep []int64) []any {
	args := make([]any, 0, len(keep)+1)
	args = append(args, modID)
	for _, id := range keep {
		args = append(args, id)
	}
	return args
}

type partial struct {
	ID        int64
	PackageID int64
	Parts     string
	NumParts  int
}

func (idx *Index) insertPartial(ctx context.Context, pkgID int64, parts string) (int64, error) {
	stmt, err := idx.tx.PrepareContext(ctx, `
INSERT INTO partial(packageId, parts) VALUES (?, ?);
`)
	if err != nil {
		return -1, err
	}

	res, err := stmt.ExecContext(ctx, pkgID, parts)
	if err != nil {
		return -1, fmt.Errorf("failed to insert partial: %w", err)
	}
	return res.LastInsertId()
}

type sqlTx struct {
	*sql.Tx
	stmts map[string]*sql.Stmt
}

func newSqlTx(tx *sql.Tx) *sqlTx {
	return &sqlTx{
		Tx:    tx,
		stmts: make(map[string]*sql.Stmt),
	}
}

func (tx *sqlTx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	stmt, ok := tx.stmts[query]
	if ok {
		return stmt, nil
	}
	stmt, err := tx.Tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	tx.stmts[query] = stmt
	return stmt, nil
}
func (tx *sqlTx) Prepare(query string) (*sql.Stmt, error) {
	return tx.PrepareContext(context.Background(), query)
}
