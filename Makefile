.PHONY: all go-doc install clean 

all: go-doc

go-doc:
	go build -o bin/go-doc

clean:
	rm bin/go-doc

install:
	@echo "Installing go-doc..."
	go install
	@echo "Installing Zsh completion..."
	go-doc -install-completion
	@echo "done."

.PHONY: diff diff-all diff-main diff-dirs diff-pkg

diff:
	git diff --stat official-go-doc -- main.go dirs.go pkg.go
	@echo
	git diff --stat official-go-doc -- *_extra.go

diff-all:
	git diff -p official-go-doc -- main.go dirs.go pkg.go

diff-main:
	git diff -p official-go-doc -- main.go

diff-dirs:
	git diff -p official-go-doc -- dirs.go

diff-pkg:
	git diff -p official-go-doc -- pkg.go
