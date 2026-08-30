package groups

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/muthuishere/agentic-os/internal/cli"
	"github.com/muthuishere/windowctl"
)

func init() {
	register(func(r *cli.Registry) {
		r.Describe("window", "Desktop windows: list, focus, move, resize, arrange")
		r.Add(
			&cli.Command{
				Group: "window", Name: "list",
				Summary: "List open windows",
				Args:    "[--app=<name>] [--title=<substring>] [--json]",
				Examples: []string{
					"agentic-os window list",
					"agentic-os window list --app=Ghostty",
				},
				Run: runWindowList,
			},
			&cli.Command{
				Group: "window", Name: "focus",
				Summary:  "Bring a window to the front",
				Args:     "<app> | --app=<name> | --title=<substring>",
				Examples: []string{"agentic-os window focus Ghostty"},
				Run:      runWindowFocus,
			},
			&cli.Command{
				Group: "window", Name: "move",
				Summary: "Move a window to a zone, a split, or absolute bounds",
				Args:    "<app> (--zone=<1A|2:1> | --at=<x,y,w,h>) [--monitor=<n>]",
				Examples: []string{
					"agentic-os window move Ghostty --zone=1B",
					"agentic-os window move Chrome --monitor=2 --zone=2:1",
					"agentic-os window move Slack --at=0,25,1920,1055",
				},
				Run: runWindowMove,
			},
			&cli.Command{
				Group: "window", Name: "resize",
				Summary:  "Resize a window in place",
				Args:     "<app> --w=<width> --h=<height>",
				Examples: []string{"agentic-os window resize Chrome --w=900 --h=700"},
				Run:      runWindowResize,
			},
			&cli.Command{
				Group: "window", Name: "wait",
				Summary:  "Wait for a window to appear, then print it",
				Args:     "<app> [--timeout=<ms>]",
				Examples: []string{"agentic-os window wait Ghostty --timeout=5000"},
				Run:      runWindowWait,
			},
			&cli.Command{
				Group: "window", Name: "arrange",
				Summary: "Apply a saved layout of many windows in one pass",
				Args:    "<layout.json>",
				Examples: []string{
					"agentic-os window arrange ~/.config/agentic-os/layouts/work.json",
				},
				Run: runWindowArrange,
			},
		)
	})
}

// matchFrom reads the --app / --title pair every window command accepts. A bare
// positional is treated as an application name, which is the common intent.
func matchFrom(set *argSet) (windowctl.Match, error) {
	match := windowctl.Match{
		App:   set.String("app", ""),
		Title: set.String("title", ""),
	}
	if match.App == "" && match.Title == "" && len(set.Rest) > 0 {
		match.App = strings.Join(set.Rest, " ")
	}
	if match.App == "" && match.Title == "" {
		return match, fmt.Errorf("name a window: a bare app name, --app=<name>, or --title=<substring>")
	}
	return match, nil
}

func runWindowList(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args, "app", "title")
	if err != nil {
		return err
	}
	if err := set.Reject("app", "title", "json"); err != nil {
		return err
	}

	windows, err := windowctl.ListWindows(windowctl.Filter{
		App:   set.String("app", ""),
		Title: set.String("title", ""),
	})
	if err != nil {
		return err
	}

	if set.Has("json") {
		enc := json.NewEncoder(c.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(windows)
	}
	if len(windows) == 0 {
		c.Println("no windows matched")
		return &cli.ExitError{Code: 1}
	}

	width := 0
	for _, w := range windows {
		if len(w.App) > width {
			width = len(w.App)
		}
	}
	for _, w := range windows {
		focus := " "
		if w.Focused {
			focus = "*"
		}
		c.Printf("%s %-*s  mon %d  %dx%d%s%s  %s\n",
			focus, width, w.App, w.Monitor,
			w.Bounds.W, w.Bounds.H, signed(w.Bounds.X), signed(w.Bounds.Y),
			w.Title)
	}
	return nil
}

// findWindow returns the first window a match selects.
func findWindow(match windowctl.Match) (windowctl.Window, bool) {
	windows, err := windowctl.ListWindows(windowctl.Filter(match))
	if err != nil || len(windows) == 0 {
		return windowctl.Window{}, false
	}
	return windows[0], true
}

// resolveMatch turns a user's app-or-title guess into one the window backend
// can act on.
//
// Not every platform reports an application name: on X11 the window list comes
// from `wmctrl -lpG`, which has no WM_CLASS column, so the App field carries the
// client's hostname instead. A bare `agentic-os window focus xterm` therefore
// has to fall back to matching the title, or it would work on macOS and never
// on Linux.
func resolveMatch(match windowctl.Match) (windowctl.Match, error) {
	if _, ok := findWindow(match); ok {
		return match, nil
	}
	if match.App != "" && match.Title == "" {
		byTitle := windowctl.Match{Title: match.App}
		if _, ok := findWindow(byTitle); ok {
			return byTitle, nil
		}
	}
	return match, fmt.Errorf("no window matched %s", describeMatch(match))
}

func describeMatch(match windowctl.Match) string {
	var parts []string
	if match.App != "" {
		parts = append(parts, "app "+match.App)
	}
	if match.Title != "" {
		parts = append(parts, "title "+match.Title)
	}
	return strings.Join(parts, " and ")
}

func runWindowFocus(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args, "app", "title")
	if err != nil {
		return err
	}
	if err := set.Reject("app", "title"); err != nil {
		return err
	}
	match, err := matchFrom(set)
	if err != nil {
		return err
	}
	resolved, err := resolveMatch(match)
	if err != nil {
		return err
	}
	return windowctl.Focus(resolved)
}

func runWindowMove(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args, "app", "title", "zone", "at", "monitor")
	if err != nil {
		return err
	}
	if err := set.Reject("app", "title", "zone", "at", "monitor"); err != nil {
		return err
	}
	match, err := matchFrom(set)
	if err != nil {
		return err
	}
	match, err = resolveMatch(match)
	if err != nil {
		return err
	}
	monitor, err := set.IntPtr("monitor")
	if err != nil {
		return err
	}

	zone, at := set.String("zone", ""), set.String("at", "")
	switch {
	case zone != "" && at != "":
		return fmt.Errorf("give either --zone or --at, not both")
	case zone != "":
		return windowctl.MoveZone(match, monitor, zone)
	case at != "":
		bounds, err := parseRect(at)
		if err != nil {
			return err
		}
		return windowctl.MoveCoords(match, monitor, bounds)
	case monitor != nil:
		// Monitor alone means "send it there and fill the screen".
		return windowctl.MoveZone(match, monitor, "full")
	}
	return fmt.Errorf("say where: --zone=<1A|2:1>, --at=<x,y,w,h>, or --monitor=<n>")
}

// parseRect reads the `x,y,w,h` form used by --at and --region.
func parseRect(value string) (windowctl.Rect, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 4 {
		return windowctl.Rect{}, fmt.Errorf("expected x,y,w,h — got %q", value)
	}
	numbers := make([]int, 4)
	for i, part := range parts {
		n, err := parseInt(strings.TrimSpace(part))
		if err != nil {
			return windowctl.Rect{}, fmt.Errorf("bad number %q in %q", part, value)
		}
		numbers[i] = n
	}
	return windowctl.Rect{X: numbers[0], Y: numbers[1], W: numbers[2], H: numbers[3]}, nil
}

func runWindowResize(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args, "app", "title", "w", "h")
	if err != nil {
		return err
	}
	if err := set.Reject("app", "title", "w", "h"); err != nil {
		return err
	}
	match, err := matchFrom(set)
	if err != nil {
		return err
	}
	width, err := set.Int("w", 0)
	if err != nil {
		return err
	}
	height, err := set.Int("h", 0)
	if err != nil {
		return err
	}
	if width <= 0 || height <= 0 {
		return fmt.Errorf("--w and --h must both be positive")
	}
	resolved, err := resolveMatch(match)
	if err != nil {
		return err
	}
	return windowctl.Resize(resolved, width, height)
}

func runWindowWait(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args, "app", "title", "timeout")
	if err != nil {
		return err
	}
	if err := set.Reject("app", "title", "timeout"); err != nil {
		return err
	}
	match, err := matchFrom(set)
	if err != nil {
		return err
	}
	timeout, err := set.Int("timeout", 10000)
	if err != nil {
		return err
	}
	window, err := waitForMatch(match, timeout)
	if err != nil {
		return err
	}
	c.Println(describeWindow(window))
	return nil
}

// waitForMatch polls until a window matches, trying the title fallback on each
// pass so a window that only ever appears under its title is still caught.
func waitForMatch(match windowctl.Match, timeoutMS int) (windowctl.Window, error) {
	deadline := time.Now().Add(time.Duration(timeoutMS) * time.Millisecond)
	for {
		if resolved, err := resolveMatch(match); err == nil {
			if window, ok := findWindow(resolved); ok {
				return window, nil
			}
		}
		if time.Now().After(deadline) {
			return windowctl.Window{}, fmt.Errorf("no window matched %s within %dms",
				describeMatch(match), timeoutMS)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func describeWindow(w windowctl.Window) string {
	return fmt.Sprintf("%s  mon %d  %dx%d%s%s  %s",
		w.App, w.Monitor, w.Bounds.W, w.Bounds.H, signed(w.Bounds.X), signed(w.Bounds.Y), w.Title)
}

func runWindowArrange(c *cli.Ctx, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("`window arrange` takes one layout file")
	}
	data, err := os.ReadFile(expandHome(args[0]))
	if err != nil {
		return err
	}
	var entries []windowctl.BatchEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parse layout: %w", err)
	}

	// Batch never aborts on a failing entry, so report per-entry and fail at
	// the end only if something actually went wrong.
	failed := 0
	for _, result := range windowctl.Batch(entries) {
		name := result.Entry.App
		if name == "" {
			name = result.Entry.Title
		}
		if result.Err != nil {
			failed++
			c.Printf("fail  %s: %v\n", name, result.Err)
			continue
		}
		c.Printf("ok    %s\n", name)
	}
	if failed > 0 {
		return &cli.ExitError{Code: 1, Message: fmt.Sprintf("%d of %d entries failed", failed, len(entries))}
	}
	return nil
}
