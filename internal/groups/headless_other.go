//go:build !linux

package groups

import (
	"os"

	"github.com/muthuishere/aos/internal/cli"
)

func isLinux() bool { return false }

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	_, err := os.FindProcess(pid)
	return err == nil
}

// A virtual display has no equivalent on macOS or Windows; these commands are
// registered with Platforms: linux, so dispatch rejects them before this runs.
func runHeadlessStart(c *cli.Ctx, _ []string) error { return errUnsupported }
func runHeadlessStop(c *cli.Ctx, _ []string) error  { return errUnsupported }
