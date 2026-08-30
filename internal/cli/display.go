package cli

import "strings"

// HasDisplay reports whether GUI-dependent commands have a display server to
// talk to.
//
// On Linux the answer is knowable: X11 and Wayland both advertise themselves in
// the environment, and a machine with neither is headless. On macOS and Windows
// there is no cheap, reliable probe for "is there a logged-in session", so a
// session is assumed and a genuine failure surfaces from the OS call itself
// rather than from a guess made here.
func HasDisplay(env func(string) string, goos string) bool {
	if goos != "linux" {
		return true
	}
	if env == nil {
		return false
	}
	return strings.TrimSpace(env("DISPLAY")) != "" || strings.TrimSpace(env("WAYLAND_DISPLAY")) != ""
}
