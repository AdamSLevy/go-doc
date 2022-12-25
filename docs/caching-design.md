# Caching

## Current Process

### Package Discovery
1. Get code roots. 
  a. GOROOT
  b. GOPATH(s) if not using modules.
  c. vendor directory if it exists.
  d. Paths for all modules imported by go.mod.
2. Walk code roots (BFS) and add all directories containing go files to the
   package list.

### Package Resolution
1. Iterate through all packages in the list.
2. Return the first one which matches, or has as its suffix, the user provided,
   potentially partial, package path.

### Package Suggestion
1. Iterate through all packages in the list.
2. Return any which match the user provided partial package path. In this case,
   matching means all partial path segments are the prefix of sequential
   package path segments.


## Cache Invalidation

### Package Discovery
1. GOROOT: Cached by go version.
2. GOPATH: ??
3. vendor: Contains a modules.txt indicating module versions. See below.
4. Imported Go Modules: Since go has deterministic builds, package lists for
   imported modules can be cached by module version.

# Indexing

Currently all packages are searched in discovery order to obtain a match.

A tree of path segments from right to left could be created to allow indexing
by path segment.

For example, if our package list is
```
gif/foo/bar
wow/foo/bar
hat/bat/bar
car/bat/bar
car/bar
car/bat
butter/beer
```

We'd have the following graph(s)
```
- bar
  - foo
    - gif
    - wow
  - bat
    - hat
    - car
  - car
- bat
  - car
- beer
  - butter
```

The downside of an index is that generally speaking the entire index needs to
be loaded for package resolution to start. The current implementation allows
package resultion to start immediately, so that the entire program can exit
once the first result is found.

If the index can be cached though you can immediately start resolving packages.

### Suggesting 

The tree index works well and simply if you know you have the right-most
segment of the package path. But we may have the beginning or middle of a path
when suggesting packages.

Maybe I need a more connected graph.


```
    car/bar
car/bat/bar
car/bat/bar
hat/bat/bar
hat/car/bar
gif/foo/bar
wow/foo/bar
bug/bat
car/bat
butter/beer
```

```
- bar
  - foo/bar
    - gif/foo/bar
    - wow/foo/bar
  - bat/bar
    - hat
    - car
  - car/bar
    - ""
    - hat
- bat
  - car
  - bug
- beer
  - butter
```

b/b
c/b
```
- bar
  - car: car/bar
  - bat: car/bat/bar hat/bat/bar 
  - foo: gif/foo/bar wow/foo/bar 
- bat
  - bug: bug/bat
  - car: car/bat car/bat/bar
  - hat: hat/bat/bar
- beer
  - butter: butter/beer
- foo
  - gif: gif/foo/bar
  - wow: wow/foo/bar
- car: car/bar car/bat car/bat/bar
- bug: bug/bat
- butter: butter/beer
- gif: gif/foo/bar
- wow: wow/foo/bar
- hat: hat/bat/bar
```

```
- bar:
