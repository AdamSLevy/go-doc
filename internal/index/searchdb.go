package index

import (
	"context"
	"database/sql"
	"strings"

	"aslevy.com/go-doc/internal/godoc"
)

type handleFunc func(godoc.PackageDir) error

func (idx *Index) search(ctx context.Context, path string, partial bool) (*sql.Rows, error) {
	const query = `
SELECT packageImportPath, dir, min(partialNumParts) FROM partial 
  WHERE parts LIKE ? AND partialNumParts >= ?
  GROUP BY packageImportPath
  ORDER BY 
    partial.numParts ASC,
    class ASC, 
    packageDir.importPath ASC, 
    packageDir.numParts ASC, 
    relativePath ASC;
`
	partsLike, numParts := toPartsLike(path, partial)
	return idx.db.QueryContext(ctx, query, partsLike, numParts)
}
func toPartsLike(path string, partial bool) (like string, numParts int) {
	parts := strings.Split(path, "/")
	numParts = len(parts)
	for _, part := range parts {
		if len(like) == 0 {
			if len(part) == 0 {
				continue
			}
		} else {
			like += "/"
		}
		like += part
		if partial {
			like += "%"
		}
	}
	return
}
func scanPackageDir(row sqlRow) (godoc.PackageDir, error) {
	var pkg godoc.PackageDir
	return pkg, row.Scan(&pkg.ImportPath, &pkg.Dir)
}
