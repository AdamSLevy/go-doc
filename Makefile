.PHONY: all install go-doc clean

all: go-doc

install:
	@echo "Installing go-doc..."
	go install
	@echo "Installing Zsh completion..."
	go-doc -install-completion
	@echo "done."

go-doc:
	go build -o bin/go-doc

clean:
	rm -f bin/*
