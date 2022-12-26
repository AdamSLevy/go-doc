# Design

This file contains an overview of the design and implementation of various
aspects of this project.

Aside from anyone wishing to understand this project in particular, it should
serve as a valuable resource for anyone interested in how official go doc
works, or in writing custom Zsh completions.

## Design Goals

- Make `go doc` easier to use and more helpful to developers.
- Preserve the core functionality of `go doc`. Must work as a drop-in
  replacement. Official flag and argument syntax must continue to work.
- Minimize the [diff] with the official `cmd/doc` source to keep it easy to
  merge upstream changes to maintain comparable functionality. See [Staying
  Current].
- Avoid external dependencies other than Go and Zsh.

[diff]:https://github.com/AdamSLevy/go-doc/compare/go-doc-official...main

## Go doc

The official go doc implementation is in package
[`cmd/doc`](https://pkg.go.dev/cmd/doc).

There are three source files with the following concerns.
- main.go -- Flag and [Argument Parsing], [Package Resolution]
- dirs.go -- [Package Discovery]
- pkg.go  -- Symbol resolution, [Doc Rendering]

Additionally there is a `doc_test.go` file and a `testdata` directory with
packages used in the test. The test runs the `do` function against various
arguments and checks the output against a set of regexes.


### Argument Parsing
[Argument Parsing]: #argument-parsing

The code for flag and argument parsing is mostly straightforward. After flag
parsing, non-flag arguments are parsed by `parseArgs`. It returns
a `*build.Package`, the user provided package path and symbol, and a bool
called more, which is true when the package path is a right-partial, and there
are still other packages the partial hasn't been checked against. 

After a matching package is found, if no symbol was specified, the package docs
are printed. Otherwise, the package is searched for that symbol. If no matching
symbol is found, parseArgs is called again to continue the search for the next
matching package. This continues in a loop until the first matching
package/symbol, or there are no more packages to search.

When there are two distinct arguments, parsing the package and symbol is
trivial.

It gets a little hairy in `parseArgs` in the case of a single argument when
trying to disambiguate between the package import path, which could have a dot
in the last segment, and the symbol, which may also have a dot as in:
`<pkg>.<type>.<method|field>`

After ruling out several edge cases, the basic strategy of `parseArgs` is to:
1. Find the last slash.
2. Find the first dot after the last slash.
3. Check if everything up to that dot resolves to a package, if so return it.
4. Otherwise go back to 2 and try again, but consider everything up to the next
   dot to be the package.

If this all fails it tries to resolve the entire arg as a symbol in the package
in the current directory, unless the argument has a slash in it, in which case
log.Fatal, because that couldn't possibly be just a symbol.

There's a bit more to step 3. See [Package Resolution] below.


#### Changes

Flag definitions and parsing have been moved to `internal/flags`. See the
package docs for more details.

Flexible flag and argument parsing and the `-complete` flag are handled by
`internal/flags.AddParse`.

When in completion mode, log.Fatal is not called if we fail to match a package,
so that the completion code can go on generate possible completions. Completion
mode is enabled when `-complete` is the very first argument. 

The completion code also relies on the official argument parsing provided by
parseArgs. This allows accurate completions for single arguments of the form
`<pkg>.<sym>`


### Package Resolution
[Package Resolution]: #package-resolution

A package package may be:
- A relative or absolute path: starts with `.` `..` `/` or `\`
- A full import path: `os` `encoding/json` `github.com/stretchr/testify/require`
- A right-partial import path: `http` `json` `testify/require` 

If that fails, `findNextPackage` is used to find partial matches. It calls
`dirs.Next` to iterate through all of the [discovered
packages](#package-discovery). For each known package, the user provided
package path is checked for an exact or partial match. The full import path of
the first encountered match is returned, along with a bool indicating whether
there are more packages to search. 

The next call to `findNextPackage` will continue the search where it last left
off. However `dirs.Reset` will reset the search from the beginning of the
complete package list.


### Symbol Resolution

Symbols are matched case insensitively such that a user's lower case characters
match upper case characters in the symbol name, but given upper case characters
must match exactly. For example, `getString` will match a symbol named
`GetString` but not a symbol named `Getstring`.

Matching is done by `match` in `pkg.go`. Special care is taken to properly
handle unicode runes properly. Unexported symbol names never match unless the
`-u` unexported flag was provided.


### Package Discovery
[Package Discovery]: #package-discovery

The complete list of packages known to `go doc` is generated by walking the
directories of the stdlib, the current module, and all imported modules. Any
directory encountered containing at least one go file is added to a list of
candidate packages. Note that the directory may not actually build to a single
package successfully. Building the package is only attempted by `parseArgs`
after it has determined that the import path is a match.

The Dirs type is a singleton for caching the package directories so they can be
searched again without re-walking the code roots. The search begins by calling
`dirsInit()`, which initializes a `go/build.Context`, determines the code
roots, and then launches `Dirs.walk()` in a goroutine to sequentially walk each
code root breadth-first, stopping at directories which start with . or \_, are
named `testdata`, or define a separate module.

The code roots are determined by `findCodeRoots()`. In module mode, the code
roots are parsed from exec'ing `go list -m -f="{{.Path}}\t{{.Dir}}" all`.
Otherwise code roots may be determined by the vendor directory or legacy GOPATH
if not a module.

As the walk finds candidate packages, they are sent to an unbuffered channel.
As the `Dirs.Next()` function is called in the search for a matching package,
the dirs are read from this channel, cached in a slice, and returned. This
effectively blocks the directory walk from going any further than the first
matching package.

If the list must be searched again, `Dirs.Reset()` is called and `Next` starts
iterating from its first cached package until it again must start reading from
the channel. When `Dirs.walk()` has completed, the channel is closed and Next
will also return false when the end of the list is reached.


### Caching


### Doc Rendering
[Doc Rendering]: #doc-rendering

After a package is found, `parsePackage` is called to perform further parsing
of the package files. A `*Package` is returned which bundles the
`build.Package`, `token.FileSet`, `docs.Package`, and `ast.Package`, along with other 


#### Typed Values


### Colorized Markdown with Glamour


### Syntax Highlighting


### Referenced Imports


### Staying Current
[Staying Current]:#staying-current

As Go releases new versions, the official `cmd/doc` source, and possibly its
behavior, will inevitably change over time. To prevent this project from
becoming stale and out of date, it will be prudent to merge upstream changes
back to this code.

The unmodified official go doc source is maintained on the branch
`official-go-doc`. 

The `main` branch is based on the `official-go-doc` branch, so it should be
relatively trivial to merge upstream changes back to main.

The only modification to the official source on this branch was removal of the
`internal/testenv` import from `doc_test.go` to allow the tests to build. Since
this package is internal to the stdlib it is impossible to import here.


## Zsh Completion


### Docs and Resources



### Handling Sub Commands



### Package Import Path Completion Matching




