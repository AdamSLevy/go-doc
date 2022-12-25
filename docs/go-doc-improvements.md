# Go Doc Improvements

This is a list of improvements to `go doc`.

## Implemented
- Caching layer for faster package path resolution in modules with large
  numbers of imports.
- Uses `GODOC_PAGER` or `PAGER` pager for long output if set.
- Colorized output with `-format term` or `GODOC_FORMAT=term`.
- Show import statements for any packages referenced by symbols in displayed
  documentation.
- More forgiving argument parsing: 
  - Flags may appear anywhere after `go doc`, including after or between
    arguments.
  - Three non-flag arguments are interpretted as `<pkg> <type>.<method|field>`.
- Show location of a symbol within its package. i.e. `// file.go +line` where
  the symbol is defined within the package.
- The -open flag causes go doc to open the file to the line containing the
  first matching symbol it finds.

## Road map
- Hyperlinks for packages and symbols that lead to [https://pkg.go.dev/](). See
  [this](https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda).
- Improved caching with a package search index.
- Alternative package resolution priority. i.e. match against packages imported
  from the local package first, then from the local module, then stdlib...
  e.g. for `func Handle(w http.ResponseWriter, r *http.Request) error` show
  `import "net/http"` with the docs.
