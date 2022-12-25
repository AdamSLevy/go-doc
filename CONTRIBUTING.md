# Contributing

Thanks for checking out my project! All contributions are welcome from simple
bug reports, questions, and feature requests.

PRs are also welcome. Consider opening an issue to discuss it first. 

## Design Goals

Please make sure you review the following design goals before you start writing
code.

- Make `go doc` easier to use and more helpful to developers.
- Preserve the core functionality of `go doc`. Must work as a drop-in
  replacement. Official flag and argument syntax must continue to work.
- Keep the
  [diff](https://github.com/AdamSLevy/go-doc/compare/go-doc-official...main)
  with the official `cmd/doc` source small. This makes it easier to merge
  upstream changes to maintain comparable functionality.
