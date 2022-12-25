package install

import (
	"os"

	"aslevy.com/go-doc/internal/completion"
	"aslevy.com/go-doc/internal/dlog"
)

var Requested bool

var log = dlog.New(os.Stderr, "", 0)

func IfRequested() {
	if !Requested || completion.Enabled {
		return
	}

	log.Enable()
	log.Println("Installing files for zsh completion...")

	var ret int
	for _, spec := range files {
		if err := spec.Install(); err != nil {
			log.Println(err)
			ret = 1 // install failed or was skipped
		}
	}

	os.Exit(ret)
}
