package groups

import (
	"fmt"
	"io"
	"strings"

	"github.com/muthuishere/agentic-os/internal/cli"
)

func init() {
	register(func(r *cli.Registry) {
		r.Describe("clipboard", "Read and write the system clipboard")
		r.Add(
			&cli.Command{
				Group: "clipboard", Name: "copy",
				Summary: "Copy stdin, or the given words, to the clipboard",
				Args:    "[text...]",
				Examples: []string{
					"echo hello | agentic-os clipboard copy",
					`agentic-os clipboard copy "some text"`,
				},
				Run: runClipboardCopy,
			},
			&cli.Command{
				Group: "clipboard", Name: "paste",
				Summary:  "Print the clipboard to stdout",
				Examples: []string{"agentic-os clipboard paste > note.txt"},
				Run:      runClipboardPaste,
			},
		)
	})
}

func runClipboardCopy(c *cli.Ctx, args []string) error {
	text := strings.Join(args, " ")
	if len(args) == 0 {
		piped, err := io.ReadAll(c.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		text = string(piped)
	}
	return clipboardWrite(text)
}

func runClipboardPaste(c *cli.Ctx, _ []string) error {
	text, err := clipboardRead()
	if err != nil {
		return err
	}
	c.Println(text)
	return nil
}
