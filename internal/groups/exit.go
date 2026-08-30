package groups

import (
	"errors"
	"os/exec"
)

// exitCodeOf recovers a child process's exit status, or 0 when the failure was
// something else (the binary was missing, the context expired).
func exitCodeOf(err error) int {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	return 0
}
