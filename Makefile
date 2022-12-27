.PHONY: all install go-doc clean diff diff-show

all: go-doc

install:
	@echo "Installing go-doc..."
	go install
	@echo "Installing Zsh completion..."
	go-doc -install-completion
	@echo "done."

diff-stat:
	git diff --stat official-go-doc -- main.go dirs.go pkg.go

diff-all:
	git diff -p official-go-doc -- main.go dirs.go pkg.go

diff-main.go:
	git diff -p official-go-doc -- main.go

diff-dirs.go:
	git diff -p official-go-doc -- dirs.go

diff-pkg.go:
	git diff -p official-go-doc -- pkg.go

go-doc:
	go build -o bin/go-doc

clean:
	rm -f bin/*
