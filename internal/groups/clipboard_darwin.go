package groups

import (
	"os/exec"
	"strings"

	"github.com/muthuishere/aos/internal/sys"
)

func clipboardWrite(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func clipboardRead() (string, error) {
	return sys.Output("pbpaste")
}
