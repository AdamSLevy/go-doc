# `go doc` Improved, with Zsh Completion

This is a drop in replacement for `go doc` with a number of key improvements.
Additionally it provides advanced package and symbol Zsh completion for go doc.

## Key Features
- Advanced Zsh completion of go doc arguments, with package and symbol
  descriptions. This also improves on existing completion for all other go
  subcommands. Note: Bash is not supported, and is not a current goal.
- Colorized output with syntax highlighting for modern terminal emulators with
  `-fmt=term` or `GODOC_FORMAT=term`. A few other output modes can also be used
  including markdown and html. See the -fmt flag.
- More flexible argument parsing. 
  - Flags can be placed anywhere, including after and between non-flag
    arguments. 
  - Three arguments are interpretted as `go doc <pkg> <type>.<method|field>`.
- Imports for external symbols which are referenced in the displayed
  documentation are shown at the top. Stdlib imports are omitted by default but
  can be included with `-stdlib`. Imports can be omitted with the `-no-imports`
  flag. This is very useful when dealing with overloaded package names.
- The path to the file and line of a requested symbol is shown as a comment
  below the rendered symbol. This can be omitted with `-no-location`.A
- If the `-open` flag is set, instead of showing the docs, the file containing
  a requested symbol is opened using EDITOR.


### Key Completion Features

- Complete packages and symbols for go doc, and packages for other go
  subcommands.
- Use of completion tags provide fine grained control over completion
  suggestions. `^Xn` cycles through the next tag.
- Package suggestions include relative and absolute path completion limited to
  directories containing go files.
- Package suggestions are module aware and reflect what go doc will accept as
  arguments.
- Intuitive package suggestion matching allows for easily finding packages
  without knowing their full or exact import path.
- Package suggestions interpret the special path segment `...` to match 0 or
  more package path segments. This is useful when you only know the start of
  a few parts of the import path.
- Packages with an `internal` segment are not suggested unless what the user
  has typed specifically matches it.
- Package and symbol suggestions are displayed with short comments.
- Symbol suggestions are displayed and grouped as their go definition appears
  in go doc.
- Symbol suggestions follow go doc case insensitive matching.
- Symbol suggestions respect go doc flags for unexported (-u) and exact case
  symbol matching (-c).
- Ability to parse and complete the full single argument go doc syntax: `go doc
  path/to/pkg.<sym>.<method|field>`

## Install

1. Clone the repo. Currently I don't have the vanity import path set up, so you
   need to clone it manually.
2. cd path/to/go-doc && make install

This will run `go install` and then `go-doc -install-completion` which will
prompt you about the three files it can install. Type `?[enter]` for more info
about each file. They are summarized below as well.

### Go Drop In Replacement
If you want to use `go-doc` as a drop in replacement for `go doc` then you need
to install the go shim script to a directory in your PATH occuring before the
directory where the official go binary is.

By default it installs it to `$HOME/bin`. It's up to you to put this at the
front of your PATH. 
```
export PATH="$HOME/bin:$PATH"
```

The go shim script will run the official go binary normally with whatever
arguments it is passed except when the first argument is `doc`, in which case
it calls `go-doc` with the remaining arguments.

If you ever want to call the official `go doc` instead, you can add a dash as
the first argument before `doc`: `go - doc ...`


### Zsh Completion

If you use Oh My Zsh, then you can put `zsh/plugin/go` into
`~/.oh-my-zsh/custom/plugins`.

Add `go` to your list of enabled plugins.

```zsh
plugins=(
  # ... your other plugins ...
  go
)
```

The only other requirement is that `go-doc` is in your PATH, which if you have
your GOPATH and GOBIN set up correctly, it will be after `go install`.

If you don't use Oh My Zsh then I assume you can figure out the best way to put
`zsh/plugins/go/_golang` somewhere in your `FPATH` and know how to enable Zsh
completion generally.

#### Recommended Zstyles

To get the most out of the completion I recommend the following zstyle options.
You can just run these in your terminal directly to try them out for the
current session. Put them in your .zshrc or wherever they can get loaded when
you open a new shell.

```zsh
# group the different type of matches under their descriptions
zstyle ':completion:*' group-name ''

zstyle ':completion:*:*:-command-:*:*' group-order alias builtins functions commands

zstyle ':completion:*' matcher-list '' 'm:{a-zA-Z}={A-Za-z}' 'r:|[._-/]=* r:|=*' 'l:|=* r:|=*'

zstyle ':completion:*' accept-exact false

# format descriptions, messages, and warnings
zstyle ':completion:*:*:*:*:descriptions' format '%F{green}-- %d --%f'
zstyle ':completion:*:messages' format ' %F{purple} -- %d --%f'
zstyle ':completion:*:warnings' format ' %F{red}-- no matches found --%f'
```
