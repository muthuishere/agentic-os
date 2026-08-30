package groups

import "github.com/muthuishere/agentic-os/internal/sys"

func takeScreenshot(req captureRequest) error {
	// -x silences the shutter sound; the interactive modes still show the UI.
	args := []string{"-x"}
	switch req.Mode {
	case captureRegion:
		args = append(args, "-i")
	case captureWindow:
		args = append(args, "-i", "-W")
	}
	if req.Copy {
		args = append(args, "-c")
	} else {
		args = append(args, req.Path)
	}
	_, err := sys.Output("screencapture", args...)
	return err
}
