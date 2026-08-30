package groups

import (
	"fmt"

	"github.com/muthuishere/agentic-os/internal/sys"
)

func takeScreenshot(req captureRequest) error {
	if req.Mode != captureFullscreen {
		// Snip & Sketch owns interactive capture on Windows and always hands
		// the result to the clipboard, so honour --copy and reject a --out that
		// it cannot satisfy.
		if !req.Copy {
			return fmt.Errorf("%s capture on windows goes to the clipboard; add --copy", req.Mode)
		}
		_, err := sys.Output("explorer.exe", "ms-screenclip:")
		return err
	}

	target := "$bmp.Save('" + escapePS(req.Path) + "')"
	if req.Copy {
		target = "[System.Windows.Forms.Clipboard]::SetImage($bmp)"
	}
	_, err := sys.PowerShell(`
Add-Type -AssemblyName System.Windows.Forms, System.Drawing
$b = [System.Windows.Forms.SystemInformation]::VirtualScreen
$bmp = New-Object System.Drawing.Bitmap $b.Width, $b.Height
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.CopyFromScreen($b.Left, $b.Top, 0, 0, $bmp.Size)
` + target + `
$g.Dispose(); $bmp.Dispose()
`)
	return err
}

// escapePS makes a string safe inside a PowerShell single-quoted literal.
func escapePS(value string) string {
	out := make([]rune, 0, len(value))
	for _, r := range value {
		if r == '\'' {
			out = append(out, '\'')
		}
		out = append(out, r)
	}
	return string(out)
}
