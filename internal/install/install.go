package install

import (
	"flag"
	"log"
	"os"

	"aslevy.com/go-doc/internal/completion"
	"aslevy.com/go-doc/internal/flagvar"
)

func AddFlags(fs *flag.FlagSet) {
	fs.Var(flagvar.Do(Completion), "install-completion", "install files for Zsh completion")
}

func Completion() {
	if completion.Requested {
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
