package index

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"runtime/debug"
	"time"

	"aslevy.com/go-doc/internal/godoc"
)

type sqlRow interface {
	Scan(dest ...any) error
}

type sync struct {
	CreatedAt time.Time
	UpdatedAt time.Time

	BuildRevision string
}

func (idx *Index) loadSync(ctx context.Context) error {
	const _syncSelect = `
SELECT createdAt, updatedAt, buildRevision FROM sync WHERE rowid=1;
`
	return idx.db.QueryRowContext(ctx, _syncSelect).Scan(&idx.CreatedAt, &idx.UpdatedAt, &idx.BuildRevision)
}

func (idx *Index) upsertSync(ctx context.Context) error {
	const _syncUpsert = `
INSERT INTO sync(rowid, buildRevision) VALUES (1, ?)
  ON CONFLICT(rowid) DO 
    UPDATE SET 
      updatedAt=CURRENT_TIMESTAMP, 
      buildRevision=excluded.buildRevision
    WHERE rowid=1;
`
	_, err := idx.tx.ExecContext(ctx, _syncUpsert, getBuildRevision())
	if err != nil {
		return fmt.Errorf("failed to upsert sync: %w", err)
	}
	return nil
}

var buildRevision string

func getBuildRevision() string {
	if buildRevision != "" {
		return buildRevision
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			buildRevision = s.Value
			break
		}
	}
	return buildRevision
}

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

type prepared struct {
	loadModule   *sql.Stmt
	insertModule *sql.Stmt
	updateModule *sql.Stmt

	getPackageID  *sql.Stmt
	insertPackage *sql.Stmt

	insertPartial *sql.Stmt
}

type _module struct {
	ID         int64
	ImportPath string
	Dir        string
	Class      class
	Vendor     bool
}

func scanModule(row sqlRow) (_module, error) {
	var mod _module
	return mod, row.Scan(&mod.ID, &mod.ImportPath, &mod.Dir, &mod.Class, &mod.Vendor)
}

func (idx *Index) loadModule(ctx context.Context, importPath string) (_module, error) {
	stmt, err := idx.loadModuleStmt(ctx)
	if err != nil {
		return _module{}, err
	}
	return scanModule(stmt.QueryRowContext(ctx, importPath))
}
func (idx *Index) loadModuleStmt(ctx context.Context) (*sql.Stmt, error) {
	if idx.prepared.loadModule != nil {
		return idx.prepared.loadModule, nil
	}
	const query = `
SELECT rowid, importPath, dir, class, vendor FROM module WHERE importPath=?;
`
	var err error
	idx.prepared.loadModule, err = idx.tx.PrepareContext(ctx, query)
	return idx.prepared.loadModule, err
}

func (idx *Index) insertModule(ctx context.Context, pkgDir godoc.PackageDir, class class, vendor bool) (int64, error) {
	stmt, err := idx.insertModuleStmt(ctx)
	if err != nil {
		return -1, err
	}
	res, err := stmt.ExecContext(ctx, pkgDir.ImportPath, pkgDir.Dir, int(class), vendor)
	if err != nil {
		return -1, nil
	}
	return res.LastInsertId()
}
func (idx *Index) insertModuleStmt(ctx context.Context) (*sql.Stmt, error) {
	if idx.prepared.insertModule != nil {
		return idx.prepared.insertModule, nil
	}
	const query = `
INSERT INTO module (importPath, dir, class, vendor) VALUES (?, ?, ?, ?);
`
	var err error
	idx.prepared.insertModule, err = idx.tx.PrepareContext(ctx, query)
	return idx.prepared.insertModule, err
}

func (idx *Index) updateModule(ctx context.Context, modID int64, pkgDir godoc.PackageDir, class class, vendor bool) error {
	stmt, err := idx.updateModuleStmt(ctx)
	if err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx, modID, pkgDir.Dir, int(class), vendor)
	return err
}
func (idx *Index) updateModuleStmt(ctx context.Context) (*sql.Stmt, error) {
	if idx.prepared.updateModule != nil {
		return idx.prepared.updateModule, nil
	}
	const query = `
UPDATE module SET (dir, class, vendor) = (?, ?, ?) WHERE rowid=?;
`
	var err error
	idx.prepared.updateModule, err = idx.tx.PrepareContext(ctx, query)
	return idx.prepared.updateModule, err
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

type _package struct {
	ID           int64
	ModuleID     int64
	RelativePath string
	NumParts     int
}

func (idx *Index) getPackageID(ctx context.Context, modID int64, relativePath string) (int64, error) {
	stmt, err := idx.getPackageIDStmt(ctx)
	if err != nil {
		return -1, err
	}
	var id int64
	err = stmt.QueryRowContext(ctx, modID, relativePath).Scan(&id)
	return id, err
}
func (idx *Index) getPackageIDStmt(ctx context.Context) (*sql.Stmt, error) {
	if idx.prepared.getPackageID != nil {
		return idx.prepared.getPackageID, nil
	}
	const query = `
SELECT rowid FROM package WHERE moduleId=? AND relativePath=?;
`
	var err error
	idx.prepared.getPackageID, err = idx.tx.PrepareContext(ctx, query)
	return idx.prepared.getPackageID, err
}

func (idx *Index) insertPackage(ctx context.Context, modID int64, relativePath string) (int64, error) {
	stmt, err := idx.insertPackageStmt(ctx)
	if err != nil {
		return -1, err
	}
	res, err := stmt.ExecContext(ctx, modID, relativePath)
	if err != nil {
		return -1, fmt.Errorf("failed to insert package: %w", err)
	}
	return res.LastInsertId()
}
func (idx *Index) insertPackageStmt(ctx context.Context) (*sql.Stmt, error) {
	if idx.prepared.insertPackage != nil {
		return idx.prepared.insertPackage, nil
	}
	const query = `
INSERT INTO package(moduleId, relativePath) VALUES (?, ?);
`
	var err error
	idx.prepared.insertPackage, err = idx.tx.PrepareContext(ctx, query)
	return idx.prepared.insertPackage, err
}

func (idx *Index) prunePackages(ctx context.Context, modID int64, keep []int64) error {
	dlog.Printf("pruning unused packages for module %d", modID)
	query := fmt.Sprintf(`
DELETE FROM package WHERE moduleId=? AND rowid NOT IN (%s);
`, placeholders(len(keep)))
	_, err := idx.tx.ExecContext(ctx, query, prunePackagesArgs(modID, keep)...)
	return err
}
func prunePackagesArgs(modID int64, keep []int64) []any {
	args := make([]any, 0, len(keep)+1)
	args = append(args, modID)
	for _, id := range keep {
		args = append(args, id)
	}
	return args
}

type _partial struct {
	ID        int64
	PackageID int64
	Parts     string
	NumParts  int
}

func (idx *Index) insertPartial(ctx context.Context, pkgID int64, parts string) (int64, error) {
	stmt, err := idx.insertPartialStmt(ctx)
	if err != nil {
		return -1, err
	}

	res, err := stmt.ExecContext(ctx, pkgID, parts)
	if err != nil {
		return -1, fmt.Errorf("failed to insert partial: %w", err)
	}
	return res.LastInsertId()
}
func (idx *Index) insertPartialStmt(ctx context.Context) (*sql.Stmt, error) {
	if idx.prepared.insertPartial != nil {
		return idx.prepared.insertPartial, nil
	}
	const query = `
INSERT INTO partial(packageId, parts) VALUES (?, ?);
`
	var err error
	idx.prepared.insertPartial, err = idx.tx.PrepareContext(ctx, query)
	return idx.prepared.insertPartial, err
}

//go:embed schema.sql
var _schema []byte

func schema() (queries []string) {
	const numQueries = 7 // number of queries in schema.sql
	queries = make([]string, 0, numQueries)
	scanner := schemaScanner(_schema)
	for scanner.Scan() {
		queries = append(queries, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		panic(fmt.Errorf("failed to scan schema.sql: %w", err))
	}
	return
}
func schemaScanner(data []byte) *bufio.Scanner {
	s := bufio.NewScanner(bytes.NewReader(data))
	s.Split(sqlSplit)
	return s
}
func sqlSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	defer func() {
		// Trim the token of any leading or trailing whitespace.
		token = bytes.TrimSpace(token)
		if len(token) == 0 {
			// Ensure we don't return an empty token.
			token = nil
		}
	}()

	semiColon := bytes.Index(data, []byte(";"))
	if semiColon == -1 {
		// No semi-colon yet...
		if atEOF {
			// That's everything...
			return len(data), data, nil
		}
		// Ask for more data so we can find the EOL.
		return 0, nil, nil
	}
	// We found a semi-colon...
	return semiColon + 1, data[:semiColon+1], nil
}
