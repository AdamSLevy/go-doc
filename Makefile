.PHONY: all install go-doc clean

all: go-doc

install:
	go install

go-doc:
	go build -o bin/go-doc

clean:
	rm -f bin/*
