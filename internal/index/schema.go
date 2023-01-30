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

type sqlDB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqlRow interface {
	Scan(dest ...any) error
}

type _sync struct {
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
		return fmt.Errorf("failed to insert sync: %w", err)
	}
	return nil
}
func getBuildRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return s.Value
		}
	}
	return ""
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
	const query = `
SELECT rowid, importPath, dir, class, vendor FROM module WHERE importPath=?;
`
	return scanModule(idx.tx.QueryRowContext(ctx, query, importPath))
}

func (idx *Index) insertModule(ctx context.Context, pkgDir godoc.PackageDir, class class, vendor bool) (int64, error) {
	const query = `
INSERT INTO module (importPath, dir, class, vendor) VALUES (?, ?, ?, ?);
`
	res, err := idx.tx.ExecContext(ctx, query, pkgDir.ImportPath, pkgDir.Dir, int(class), vendor)
	if err != nil {
		return -1, nil
	}
	return res.LastInsertId()
}

func (idx *Index) updateModule(ctx context.Context, modID int64, pkgDir godoc.PackageDir, class class, vendor bool) error {
	const query = `
UPDATE module SET (dir, class, vendor) = (?, ?, ?) WHERE rowid=?;
`
	_, err := idx.tx.ExecContext(ctx, query, modID, pkgDir.Dir, int(class), vendor)
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

type _package struct {
	ID           int64
	ModuleID     int64
	RelativePath string
	NumParts     int
}

func (idx *Index) getPackageID(ctx context.Context, modID int64, relativePath string) (int64, error) {
	const query = `
SELECT rowid FROM package WHERE moduleId=? AND relativePath=?;
`
	var id int64
	err := idx.tx.QueryRowContext(ctx, query, modID, relativePath).Scan(&id)
	return id, err
}
func (idx *Index) insertPackage(ctx context.Context, modID int64, relativePath string) (int64, error) {
	const query = `
INSERT INTO package(moduleId, relativePath) VALUES (?, ?);
`
	res, err := idx.tx.ExecContext(ctx, query, modID, relativePath)
	if err != nil {
		return -1, fmt.Errorf("failed to insert package: %w", err)
	}
	return res.LastInsertId()
}
func (idx *Index) prunePackages(ctx context.Context, modID int64, keep []int64) error {
	const query = `
DELETE FROM package WHERE moduleId=? AND rowid NOT IN (?);
`
	_, err := idx.tx.ExecContext(ctx, query, keep)
	return err
}

type _partial struct {
	ID        int64
	PackageID int64
	Parts     string
	NumParts  int
}

func (idx *Index) insertPartial(ctx context.Context, pkgID int64, parts string) (int64, error) {
	const query = `
INSERT INTO partial(packageId, parts) VALUES (?, ?);
`
	res, err := idx.tx.ExecContext(ctx, query, pkgID, parts)
	if err != nil {
		return -1, fmt.Errorf("failed to insert partial: %w", err)
	}
	return res.LastInsertId()
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
