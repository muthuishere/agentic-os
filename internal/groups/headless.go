package groups

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/muthuishere/agentic-os/internal/cli"
	"github.com/muthuishere/agentic-os/internal/sys"
)

// headlessState is what `headless start` records so `status`, `run`, and `stop`
// can find the display again from a different process. It lives in the runtime
// config dir, not the repo.
type headlessState struct {
	Display string `json:"display"`
	Size    string `json:"size"`
	PID     int    `json:"pid"`
	WM      string `json:"wm,omitempty"`
	WMPID   int    `json:"wm_pid,omitempty"`
}

func init() {
	register(func(r *cli.Registry) {
		r.Describe("headless", "Run the desktop commands on a machine with no screen")
		r.Add(
			&cli.Command{
				Group: "headless", Name: "status",
				Summary:  "Report whether a display is available, and where it came from",
				Examples: []string{"aos headless status"},
				Run:      runHeadlessStatus,
			},
			&cli.Command{
				Group: "headless", Name: "start",
				Summary:   "Start a virtual display, so window and input commands work",
				Args:      "[--display=:99] [--size=<WxH>] [--wm]",
				Platforms: []string{"linux"},
				Examples: []string{
					"aos headless start",
					"aos headless start --display=:101 --size=1920x1080 --wm",
				},
				Run: runHeadlessStart,
			},
			&cli.Command{
				Group: "headless", Name: "stop",
				Summary:   "Stop the virtual display this machine started",
				Platforms: []string{"linux"},
				Examples:  []string{"aos headless stop"},
				Run:       runHeadlessStop,
			},
			&cli.Command{
				Group: "headless", Name: "run",
				Summary:  "Run a command against the managed virtual display",
				Args:     "<command> [args...]",
				Examples: []string{"aos headless run aos window list"},
				Run:      runHeadlessRun,
			},
		)
	})
}

// stateFile is where the managed display is recorded.
func stateFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "agentic-os", "headless.json"), nil
}

func readState() (headlessState, bool) {
	path, err := stateFile()
	if err != nil {
		return headlessState{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return headlessState{}, false
	}
	var state headlessState
	if err := json.Unmarshal(data, &state); err != nil {
		return headlessState{}, false
	}
	// A recorded display whose process is gone is stale, not current.
	if !processAlive(state.PID) {
		return state, false
	}
	return state, true
}

func writeState(state headlessState) error {
	path, err := stateFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func clearState() error {
	path, err := stateFile()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// AdoptManagedDisplay points this process at the virtual display `headless
// start` created, when the environment names none of its own. Without it a
// caller would have to export DISPLAY by hand after starting one, and every
// GUI command would refuse for no good reason.
//
// An environment that already names a display always wins: adopting over a real
// session would silently redirect commands to an invisible screen.
func AdoptManagedDisplay(env func(string) string) {
	if cli.HasDisplay(env, runtime.GOOS) {
		return
	}
	if state, running := readState(); running && state.Display != "" {
		os.Setenv("DISPLAY", state.Display)
	}
}

func runHeadlessStatus(c *cli.Ctx, _ []string) error {
	c.Printf("platform     %s\n", c.GOOS)

	switch c.GOOS {
	case "linux":
		x11, wayland := c.Env("DISPLAY"), c.Env("WAYLAND_DISPLAY")
		c.Printf("DISPLAY      %s\n", orDash(x11))
		c.Printf("WAYLAND      %s\n", orDash(wayland))
		c.Printf("xvfb         %s\n", presence(sys.Has("Xvfb")))

		state, running := readState()
		if running {
			c.Printf("managed      %s  %s  pid %d  (adopted)\n", state.Display, state.Size, state.PID)
			if state.WM != "" {
				c.Printf("wm           %s  pid %d\n", state.WM, state.WMPID)
			}
		} else if state.Display != "" {
			c.Printf("managed      stale record for %s; run `headless stop` to clear it\n", state.Display)
		} else {
			c.Println("managed      none")
		}
	default:
		// Neither macOS nor Windows has an Xvfb equivalent: the desktop
		// commands need a real logged-in session on the machine itself.
		c.Println("virtual      not available on this platform")
		c.Println("note         desktop commands need a logged-in session here")
	}

	c.Println()
	if cli.HasDisplay(c.Env, c.GOOS) {
		c.Println("display      yes — window, mouse, key, capture, and display commands will work")
		return nil
	}
	c.Println("display      no — only the screenless commands will work")
	c.Println()
	c.Println("Screenless commands: exec, file, msg, pkg, network, system, power,")
	c.Println("battery, font, debug, serve. Everything an agent needs on a server.")
	if c.GOOS == "linux" {
		c.Println()
		c.Println("For the rest, start a virtual display: aos headless start")
	}
	return &cli.ExitError{Code: 1}
}

func runHeadlessRun(c *cli.Ctx, args []string) error {
	args = stripSeparator(args)
	if len(args) == 0 {
		return fmt.Errorf("`headless run` needs a command")
	}

	state, running := readState()
	if !running {
		return &cli.ExitError{Code: 1, Message: "no managed display; run `aos headless start` first"}
	}
	// Pass the display through the environment rather than rewriting the
	// command, so anything that reads DISPLAY — ours or not — picks it up.
	os.Setenv("DISPLAY", state.Display)
	return passthroughExit(sys.Passthrough(args[0], args[1:]...))
}

func presence(ok bool) string {
	if ok {
		return "installed"
	}
	return "missing"
}
