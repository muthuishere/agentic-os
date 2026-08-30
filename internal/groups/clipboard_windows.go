package groups

import (
	"os/exec"
	"strings"

	"github.com/muthuishere/agentic-os/internal/sys"
)

func clipboardWrite(text string) error {
	// clip.exe takes the text on stdin verbatim, which avoids the quoting and
	// newline mangling that piping through PowerShell's Set-Clipboard invites.
	cmd := exec.Command("clip.exe")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func clipboardRead() (string, error) {
	return sys.PowerShell("Get-Clipboard -Raw")
}
