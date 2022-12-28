# Contributing

Thanks for checking out my project! All contributions are welcome from simple
bug reports, questions, and feature requests.

PRs are also welcome. Consider opening an issue to discuss it first. 

Please review the [Design Goals](DESIGN.md#design-goals) before you start
writing code.


## Dev tips

### Original Go Doc Source diff

The make targets `diff`, `diff-all`, `diff-main`, `diff-pkg`, `diff-dirs` allow
you to examine the diff of the original source files to the current work tree.

`make diff` shows the `git diff --stat` of the original source files, and then
of the `*_extra.go` source files.
```
$ make diff
git diff --stat official-go-doc -- main.go dirs.go pkg.go
 dirs.go |  14 ++++++++++++++
 main.go |  46 +++++++++++++++++++++++++++++++++++++++++-----
 pkg.go  | 282 +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++---------------------------------------------------------------------------
 3 files changed, 262 insertions(+), 80 deletions(-)

git diff --stat official-go-doc -- *_extra.go
 dirs_extra.go |  21 +++++++++++++++++++++
 pkg_extra.go  | 104 ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
 2 files changed, 125 insertions(+)

```

`make diff-dirs` along with `diff-main` and `diff-pkg` show the actual diff of
that file.
```
$ make diff-dirs
git diff -p official-go-doc -- dirs.go
diff --git a/dirs.go b/dirs.go
index 60ad6d3..f624f09 100644
--- a/dirs.go
+++ b/dirs.go
@@ -16,6 +16,8 @@ import (
        "sync"

        "golang.org/x/mod/semver"
+
+       "aslevy.com/go-doc/internal/cache"
 )
...
```

### Use `go run .` instead of `go-doc`

The go shim script allows you to set `GOSHIM_GODOC` to whatever you want used
as the thing to execute instead of `go-doc`. It defaults to `go-doc` if not
set. When developing this is useful to be able to just invoke `go doc` and have
it run it from the current directory.

The following will actually `exec go run . http`
```
export GOSHIM_GODOC="go run ."
go doc http 
```
