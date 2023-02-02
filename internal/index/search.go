package index

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"aslevy.com/go-doc/internal/godoc"
)

var dlogSearch = dlog.Child("search")

func (idx *Index) Search(ctx context.Context, path string, partial bool) ([]godoc.PackageDir, error) {
	rows, err := idx.searchRows(ctx, path, partial)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil
	}

	var pkgs []godoc.PackageDir
	return pkgs, scanPackageDirs(rows, func(pkg godoc.PackageDir) error {
		pkgs = append(pkgs, pkg)
		return nil
	})
}
func scanPackageDirs(rows *sql.Rows, handler func(godoc.PackageDir) error) error {
	defer rows.Close()
	for rows.Next() {
		pkg, err := scanPackageDir(rows)
		if err != nil {
			return err
		}
		if err := handler(pkg); err != nil {
			return err
		}
	}
	return rows.Err()
}
func scanPackageDir(row sqlRow) (godoc.PackageDir, error) {
	var pkg godoc.PackageDir
	var min int
	return pkg, row.Scan(&pkg.ImportPath, &pkg.Dir, &min)
}

func (idx *Index) searchRows(ctx context.Context, path string, partial bool) (*sql.Rows, error) {
	if err := idx.waitSync(); err != nil {
		return nil, err
	}

	maxParts, err := idx.maxPartialNumParts(ctx)
	if err != nil {
		return nil, err
	}

	subQuery, params := searchSelectParams(path, partial, maxParts)
	query := fmt.Sprintf(`
SELECT 
  packageImportPath, 
  packageDir, 
  min(partialNumParts) 
FROM (`+"%s"+`
)
GROUP BY packageImportPath
ORDER BY 
  partialNumParts  ASC,
  class            ASC, 
  moduleImportPath ASC,
  relativeNumParts ASC,
  relativePath     ASC;
`, subQuery)
	dlogSearch.Printf("query: \n%s", query)
	return idx.db.QueryContext(ctx, query, params...)
}
func searchSelectParams(path string, partial bool, maxParts int) (subQuery string, params []any) {
	query := `
  SELECT
    *
  FROM partialPackage 
  WHERE`
	if path == "" {
		// The empty string is a prefix of all packages, but will never
		// match any single package exactly.
		if partial {
			// If we're doing a partial search, then return all
			// packages. All packages have at least one part.
			query += `
    partialNumParts = 1`
		} else {
			// If we're not doing a partial search, then we match
			// nothing.
			query += `
    FALSE`
		}
		return query, nil
	}
	query += `
    partialNumParts = ? AND
    parts LIKE ?`
	var queryBldr strings.Builder
	numParts, like := searchLike(path, partial)
	for numParts <= maxParts {
		if queryBldr.Len() > 0 {
			queryBldr.WriteString(`
  UNION`)
		}
		queryBldr.WriteString(query)

		params = append(params, numParts)
		params = append(params, like.String())
		like.WriteString("/%")
		numParts++
		if !partial {
			break
		}
	}
	return queryBldr.String(), params
}
func searchLike(path string, partial bool) (numParts int, like *strings.Builder) {
	like = &strings.Builder{}
	parts := strings.Split(path, "/")
	numParts = len(parts)
	like.Grow(len(path) + numParts)
	for _, part := range parts {
		// like must not start with "%/" to avoid causing sqlite to
		// perform a full table scan. So if like is empty and the part
		// is empty, move on.
		// https://www.sqlite.org/optoverview.html#the_like_optimization
		if like.Len() > 0 {
			like.WriteByte('/')
		} else if len(part) == 0 {
			numParts--
			continue
		}
		like.WriteString(part)
		if partial {
			like.WriteByte('%')
		}
	}
	return numParts, like
}
func (idx *Index) maxPartialNumParts(ctx context.Context) (int, error) {
	const query = `SELECT MAX(numParts) FROM partial;`
	var max int
	return max, idx.db.QueryRowContext(ctx, query).Scan(&max)
}
