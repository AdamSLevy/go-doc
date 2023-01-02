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

licenses.csv: go.mod go.sum
	go-licenses csv . | tee licenses.csv

.PHONY: diff diff-all diff-main diff-dirs diff-pkg

diff:
	git diff --stat official-go-doc -- main.go dirs.go pkg.go doc_test.go
	@echo
	git diff --stat official-go-doc -- *extra.go

diff-all:
	git diff -p official-go-doc -- main.go dirs.go pkg.go doc_test.go

diff-main:
	git diff -p official-go-doc -- main.go

diff-dirs:
	git diff -p official-go-doc -- dirs.go

diff-pkg:
	git diff -p official-go-doc -- pkg.go

diff-test:
	git diff -p official-go-doc -- doc_test.go
