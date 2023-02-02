package main

import (
	"bytes"
	"log"
	"os/exec"
	"testing"

	"aslevy.com/go-doc/internal/benchmark"
)

func BenchmarkDirsNext(b *testing.B) {
	var (
		pkg Dir
		ok  bool
	)
	benchmark.Run(b, nil, func() {
		dirs := newDirs()
		for {
			pkg, ok = dirs.Next()
			if !ok {
				return
			}
		}
	})
	b.Log("pkg: ", pkg, "ok: ", ok)
}
func newDirs(extra ...Dir) Dirs {
	if buildCtx.GOROOT == "" {
		stdout, err := exec.Command("go", "env", "GOROOT").Output()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
				log.Fatalf("failed to determine GOROOT: $GOROOT is not set and 'go env GOROOT' failed:\n%s", ee.Stderr)
			}
			log.Fatalf("failed to determine GOROOT: $GOROOT is not set and could not run 'go env GOROOT':\n\t%s", err)
		}
		buildCtx.GOROOT = string(bytes.TrimSpace(stdout))
	}

	dirs.hist = make([]Dir, 0, 1000)
	dirs.hist = append(dirs.hist, extra...)
	dirs.scan = make(chan Dir)
	// log.Println("code roots:", codeRoots())
	go dirs.walk(codeRoots())
	return dirs
}
