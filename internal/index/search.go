package index

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"aslevy.com/go-doc/internal/godoc"
)

var dlogSearch = dlog.Child("search")

type SearchOption func(*searchOptions)

type searchOptions struct {
	matchPartials bool
}

func newSearchOptions(opts ...SearchOption) searchOptions {
	var o searchOptions
	WithSearchOptions(opts...)(&o)
	return o
}
func WithSearchOptions(opts ...SearchOption) SearchOption {
	return func(o *searchOptions) {
		for _, opt := range opts {
			opt(o)
		}
	}
}
func WithMatchPartials() SearchOption {
	return func(o *searchOptions) {
		o.matchPartials = true
	}
}

func (idx *Index) Search(ctx context.Context, path string, opts ...SearchOption) ([]godoc.PackageDir, error) {
	rows, err := idx.searchRows(ctx, path, opts...)
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

func (idx *Index) searchRows(ctx context.Context, path string, opts ...SearchOption) (*sql.Rows, error) {
	if err := idx.waitSync(); err != nil {
		return nil, err
	}

	query, params, err := idx.searchQueryParams(ctx, path, opts...)
	if err != nil {
		return nil, err
	}
	dlogSearch.Printf("query: \n%s", query)
	dlogSearch.Printf("params: \n%+v", params)
	return idx.db.QueryContext(ctx, query, params...)
}

func (idx *Index) searchQueryParams(ctx context.Context, path string, opts ...SearchOption) (query string, params []any, _ error) {
	where, params, err := idx.searchWhereParams(ctx, path, opts...)
	if err != nil {
		return "", nil, err
	}

	const selectQuery = `
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
`
	return fmt.Sprintf(selectQuery, where), params, err
}
func (idx *Index) searchWhereParams(ctx context.Context, path string, opts ...SearchOption) (where string, params []any, _ error) {
	o := newSearchOptions(opts...)
	if !o.matchPartials {
		return idx.searchWhereParamsExact(path)
	}
	return idx.searchWhereParamsPartial(ctx, path)
}
func (idx *Index) searchWhereParamsExact(path string) (where string, params []any, _ error) {
	if path == "" {
		return "FALSE", nil, nil
	}

	where = `(
    partialNumParts = ? AND
    parts = ?
)`
	params = []any{
		strings.Count(path, "/") + 1,
		path,
	}
	return
}
func (idx *Index) searchWhereParamsPartial(ctx context.Context, path string) (where string, params []any, _ error) {
	if path == "" {
		return "TRUE", nil, nil
	}

	maxParts, err := idx.maxPartialNumParts(ctx)
	if err != nil {
		return "", nil, err
	}

	const whereQuery = `(
    ? AND
    partialNumParts = ? AND
    parts LIKE ?
)`
	numParts, like := searchLike(path)

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

		like.WriteString("/%")
	}
	return queryBldr.String(), params, nil
}
func searchLike(path string) (numParts int, like *strings.Builder) {
	like = new(strings.Builder)
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
		like.WriteByte('%')
	}
	return numParts, like
}
func (idx *Index) maxPartialNumParts(ctx context.Context) (int, error) {
	const query = `SELECT MAX(numParts) FROM partial;`
	var max int
	return max, idx.db.QueryRowContext(ctx, query).Scan(&max)
}
