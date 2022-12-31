## completion
compctl -g "*.go" gofmt # standard go tools
compctl -g "*.go" gccgo # gccgo

# gc
for p in 5 6 8; do
  compctl -g "*.${p}" ${p}l
  compctl -g "*.go" ${p}g
done
unset p

# ## aliases
# alias gob='go build'
# alias goc='go clean'
# alias god='go doc'
# alias gof='go fmt'
# alias gofa='go fmt ./...'
# alias gofx='go fix'
# alias gog='go get'
# alias goga='go get ./...'
# alias goi='go install'
# alias gol='go list'
# alias gom='go mod'
# alias gopa='cd $GOPATH'
# alias gopb='cd $GOPATH/bin'
# alias gops='cd $GOPATH/src'
# alias gor='go run'
# alias got='go test'
# alias gota='go test ./...'
# alias goto='go tool'
# alias gotoc='go tool compile'
# alias gotod='go tool dist'
# alias gotofx='go tool fix'
# alias gov='go vet'
# alias gow='go work'

__bin_go() {
  local GO=${GOSHIM_GO:-$(which -p go)}
  "$GO" "$@"
}
__bin_go-doc() {
  local GODOC=${GOSHIM_GODOC:-$(which -p go-doc)}
  "$GODOC" "$@"
}

# go is a shim for the real go which uses go-doc in place of `go doc`
go() {
  case $1 in
    doc)
      shift
      __bin_go-doc "$@"
      return
      ;;
    - | --)
      # If we're being invoked as `go -` or `go --`, then we need to pass the
      # arguments through to the real `go` binary.
      shift
      ;;
  esac

  __bin_go "$@"
}
