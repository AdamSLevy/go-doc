package workdir

import (
	"os"
	"path/filepath"
	"strings"

	"aslevy.com/go-doc/internal/dlog"
)

var (
	workingDirectory      string
	workingDirectoryDepth int
)

func Get() (string, error) {
	if workingDirectory != "" {
		return workingDirectory, nil
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	workingDirectoryDepth = strings.Count(workingDirectory, string(filepath.Separator))
	return workingDirectory, nil
}

type Sub struct {
	Env  string
	Path string
}

func Rel(path string, subs ...Sub) string {
	for _, sub := range subs {
		if sub.Env == "" || sub.Path == "" {
			continue
		}
		if strings.HasPrefix(path, sub.Path) {
			return "$" + filepath.Join(sub.Env, path[len(sub.Path):])
		}
	}
	wd, err := Get()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(wd, path)
	if err != nil {
		dlog.Printf("failed to relativize %q: %v", path, err)
		return path
	}
	dlog.Printf("asdf relativized %q to %q", path, rel)
	switch rel[0] {
	case '.', os.PathSeparator:
	default:
		return "." + string(filepath.Separator) + rel
	}
	if strings.Count(rel, ".."+string(filepath.Separator)) >= workingDirectoryDepth {
		return path
	}
	return rel
}
