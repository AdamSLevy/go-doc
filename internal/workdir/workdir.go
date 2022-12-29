package workdir

import (
	"os"
	"path/filepath"
	"strings"
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
		return path
	}
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
