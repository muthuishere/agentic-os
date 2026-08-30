package groups

import (
	"os"
	"os/exec"

	"github.com/muthuishere/agentic-os/internal/sys"
)

func takeScreenshot(req captureRequest) error {
	if sys.Has("grim") {
		return grimScreenshot(req)
	}
	if tool := sys.FirstAvailable("maim", "scrot"); tool != "" {
		return x11Screenshot(tool, req)
	}
	return errUnsupported
}

func grimScreenshot(req captureRequest) error {
	var args []string
	if req.Mode == captureRegion || req.Mode == captureWindow {
		if !sys.Has("slurp") {
			return errUnsupported
		}
		region, err := sys.Output("slurp")
		if err != nil {
			return err
		}
		args = append(args, "-g", region)
	}
	if req.Copy {
		return pipeToClipboard(append([]string{"grim"}, append(args, "-")...))
	}
	_, err := sys.Output("grim", append(args, req.Path)...)
	return err
}

func x11Screenshot(tool string, req captureRequest) error {
	var args []string
	if req.Mode == captureRegion {
		args = append(args, "-s")
	}
	if req.Copy {
		return pipeToClipboard(append([]string{tool}, append(args, "/dev/stdout")...))
	}
	_, err := sys.Output(tool, append(args, req.Path)...)
	return err
}

// pipeToClipboard streams an image-producing command into the clipboard tool,
// so a capture never has to touch disk on its way there.
func pipeToClipboard(argv []string) error {
	copyTool := sys.FirstAvailable("wl-copy", "xclip")
	switch copyTool {
	case "wl-copy":
		return pipe(argv, []string{"wl-copy", "--type", "image/png"})
	case "xclip":
		return pipe(argv, []string{"xclip", "-selection", "clipboard", "-t", "image/png"})
	}
	return errUnsupported
}

func pipe(producer, consumer []string) error {
	src := exec.Command(producer[0], producer[1:]...)
	dst := exec.Command(consumer[0], consumer[1:]...)
	src.Stderr, dst.Stderr = os.Stderr, os.Stderr

	stdout, err := src.StdoutPipe()
	if err != nil {
		return err
	}
	dst.Stdin = stdout

	if err := src.Start(); err != nil {
		return err
	}
	if err := dst.Start(); err != nil {
		return err
	}
	if err := src.Wait(); err != nil {
		return err
	}
	return dst.Wait()
}
