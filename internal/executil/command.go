// Package executil provides utilities for working with exec.Cmd, namely shared
// by the internal/open and internal/pager packages.
package executil

import (
	"fmt"
	"os/exec"
)

// Command returns an exec.Cmd for the given args after first calling
// exec.LookPath on the first argument.
func Command(args ...string) (*exec.Cmd, error) {
	cmdPath, err := exec.LookPath(args[0])
	if err != nil {
		return nil, fmt.Errorf("executil: exec.LookPath: %w", err)
	}
	return exec.Command(cmdPath, args[1:]...), nil
}
