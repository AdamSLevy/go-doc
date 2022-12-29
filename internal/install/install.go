package install

import (
	"log"
	"os"

	"aslevy.com/go-doc/internal/completion"
)

func Completion() {
	if completion.Enabled {
		return
	}

	log.SetFlags(0)
	log.SetPrefix("")

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
