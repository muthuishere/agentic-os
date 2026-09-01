package groups

import (
	"os/exec"
	"strings"

	"github.com/muthuishere/aos/internal/sys"
)

// clipboardTools lists the copy/paste pairs to try, Wayland first.
var clipboardTools = []struct {
	copyCmd  []string
	pasteCmd []string
}{
	{[]string{"wl-copy"}, []string{"wl-paste", "--no-newline"}},
	{[]string{"xclip", "-selection", "clipboard", "-in"}, []string{"xclip", "-selection", "clipboard", "-out"}},
	{[]string{"xsel", "--clipboard", "--input"}, []string{"xsel", "--clipboard", "--output"}},
}

func clipboardWrite(text string) error {
	for _, tool := range clipboardTools {
		if !sys.Has(tool.copyCmd[0]) {
			continue
		}
		cmd := exec.Command(tool.copyCmd[0], tool.copyCmd[1:]...)
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	return errUnsupported
}

func clipboardRead() (string, error) {
	for _, tool := range clipboardTools {
		if !sys.Has(tool.pasteCmd[0]) {
			continue
		}
		return sys.Output(tool.pasteCmd[0], tool.pasteCmd[1:]...)
	}
	return "", errUnsupported
}
