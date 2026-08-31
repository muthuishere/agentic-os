package groups

import (
	"fmt"

	"github.com/muthuishere/agentic-os/internal/sys"
)

// showSubtitle puts up a borderless, topmost WinForms window.
//
// Three window styles do the work that matters. WS_EX_NOACTIVATE means showing
// the caption never moves the keyboard away from what the user is typing into;
// WS_EX_TRANSPARENT (with WS_EX_LAYERED, which it needs) makes clicks fall
// through to whatever is underneath; WS_EX_TOOLWINDOW keeps it out of Alt-Tab.
// They are applied in Load, before the window is ever shown, because an
// extended style set afterwards would not stop the first activation.
func showSubtitle(req subtitleRequest) (string, error) {
	// Windows PowerShell first: WinForms wants a single-threaded apartment, and
	// 5.1 is STA by default where pwsh is not.
	exe := sys.FirstAvailable("powershell", "pwsh")
	if exe == "" {
		return "", errUnsupported
	}
	args := []string{"-NoProfile", "-NonInteractive"}
	if exe == "powershell" {
		args = append(args, "-STA")
	}
	args = append(args, "-WindowStyle", "Hidden", "-Command", subtitleScript(req))

	// Spawn, never capture: this process owns a window for the whole duration,
	// and reading its output would block the CLI until the caption expired.
	if err := sys.Spawn(exe, args...); err != nil {
		return "", err
	}
	return "overlay", nil
}

func subtitleScript(req subtitleRequest) string {
	var originY string
	switch req.Position {
	case "top":
		originY = "$area.Y + 60"
	case "center":
		originY = "$area.Y + [int](($area.Height - $form.Height) / 2)"
	default:
		originY = "$area.Bottom - $form.Height - 60"
	}

	// --size is in pixels; GDI+ font sizes are points, which are 96/72 of a
	// pixel on a standard-DPI screen.
	points := req.Size * 3 / 4
	if points < 6 {
		points = 6
	}

	return fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms, System.Drawing
$api = Add-Type -PassThru -Name Subtitle -Namespace AgenticOs -MemberDefinition @'
[DllImport("user32.dll")] public static extern int GetWindowLong(IntPtr hWnd, int index);
[DllImport("user32.dll")] public static extern int SetWindowLong(IntPtr hWnd, int index, int value);
'@

$form = New-Object System.Windows.Forms.Form
$form.FormBorderStyle = 'None'
$form.TopMost = $true
$form.ShowInTaskbar = $false
$form.StartPosition = 'Manual'
$form.BackColor = [System.Drawing.Color]::Black
$form.Opacity = 0.78

$label = New-Object System.Windows.Forms.Label
$label.AutoSize = $true
$label.Font = New-Object System.Drawing.Font('Segoe UI', %d, [System.Drawing.FontStyle]::Bold)
$label.ForeColor = [System.Drawing.Color]::White
$label.BackColor = [System.Drawing.Color]::Transparent
$label.Text = '%s'
$label.Location = New-Object System.Drawing.Point(30, 18)
$form.Controls.Add($label)
$form.ClientSize = New-Object System.Drawing.Size(($label.PreferredWidth + 60), ($label.PreferredHeight + 36))

$area = [System.Windows.Forms.Screen]::PrimaryScreen.WorkingArea
$form.Location = New-Object System.Drawing.Point(
  ($area.X + [int](($area.Width - $form.Width) / 2)),
  (%s))

$form.Add_Load({
  $handle = $form.Handle
  $exStyle = $api::GetWindowLong($handle, -20)
  # LAYERED | TRANSPARENT | NOACTIVATE | TOOLWINDOW
  [void]$api::SetWindowLong($handle, -20, $exStyle -bor 0x80000 -bor 0x20 -bor 0x8000000 -bor 0x80)
})

$form.Show()
# A hand-pumped loop instead of Application::Run, which activates the form.
$deadline = (Get-Date).AddSeconds(%d)
while ((Get-Date) -lt $deadline -and -not $form.IsDisposed) {
  [System.Windows.Forms.Application]::DoEvents()
  Start-Sleep -Milliseconds 30
}
if (-not $form.IsDisposed) { $form.Close() }
`, points, escapePS(req.Text), originY, req.Seconds)
}
