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

	where, params := searchWhereParams(path, partial, maxParts)
	query := fmt.Sprintf(`
SELECT 
  packageImportPath, 
  packageDir, 
  min(partialNumParts) 
FROM 
  partialPackage
WHERE %s
GROUP BY packageImportPath
ORDER BY 
  partialNumParts  ASC,
  class            ASC, 
  moduleImportPath ASC,
  relativeNumParts ASC,
  relativePath     ASC;
`, where)
	dlogSearch.Printf("query: \n%s", query)
	dlogSearch.Printf("params: \n%+v", params)
	return idx.db.QueryContext(ctx, query, params...)
}
func searchWhereParams(path string, partial bool, maxParts int) (where string, params []any) {
	if path == "" {
		if !partial {
			return "false", nil
		}
		return "true", nil
	}
	const whereQuery = `(
    ? AND
    partialNumParts = ? AND
    parts LIKE ?
)`
	numParts, like := searchLike(path, partial)
	if !partial {
		return whereQuery, []any{true, numParts, like.String()}
	}

	var queryBldr strings.Builder
	for i := 1; i <= maxParts; i++ {
		if queryBldr.Len() > 0 {
			queryBldr.WriteString(` OR `)
		}
		queryBldr.WriteString(whereQuery)

		validWhere := i >= numParts
		params = append(params, validWhere, i, like.String())

		if !validWhere {
			continue
		}

		if !partial {
			break
		}

		like.WriteString("/%")
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
