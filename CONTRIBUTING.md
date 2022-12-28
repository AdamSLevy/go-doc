# Contributing

Thanks for checking out my project! All contributions are welcome from simple
bug reports, questions, and feature requests.

PRs are also welcome. Consider opening an issue to discuss it first. 

Please review the [Design Goals](DESIGN.md#design-goals) before you start
writing code.


## Dev tips

### Original Go Doc Source diff

The make targets `diff-stat`, `diff-main.go`, `diff-pkg.go`, `diff-dirs.go`
allow you to examine the diff of the original source files to the current work
tree.

```
$ make diff-stat
git diff --stat official-go-doc -- main.go dirs.go pkg.go
 dirs.go |  14 ++++++++++++++
 main.go |  46 ++++++++++++++++++++++++++++++++++++++++------
 pkg.go  | 282 +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++---------------------------------------------------------------------------
 3 files changed, 261 insertions(+), 81 deletions(-)
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
