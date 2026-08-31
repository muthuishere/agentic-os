package groups

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/muthuishere/agentic-os/internal/cli"
	"github.com/muthuishere/windowctl"
)

func init() {
	register(func(r *cli.Registry) {
		r.Describe("watch", "Follow what changes on this machine, one JSON line per event")
		r.Add(
			&cli.Command{
				Group: "watch", Name: "clipboard",
				Summary:  "Print a JSON line whenever the clipboard text changes",
				Blocking: true,
				Args:     "[--interval=<500ms>] [--max=<n>] [--content]",
				Examples: []string{
					"agentic-os watch clipboard",
					"agentic-os watch clipboard --max=1 --content",
				},
				Run: runWatchClipboard,
			},
			&cli.Command{
				Group: "watch", Name: "window",
				Summary:  "Print a JSON line whenever the focused window changes",
				Blocking: true,
				Args:     "[--interval=<500ms>] [--max=<n>]",
				Examples: []string{
					"agentic-os watch window",
					"agentic-os watch window --max=1",
				},
				Run: runWatchWindow,
			},
		)
	})
}

// defaultWatchInterval is a compromise: fast enough that a change feels caught
// as it happens, slow enough that polling `pbpaste` or the window list forever
// costs nothing anyone will notice.
const defaultWatchInterval = 500 * time.Millisecond

// minWatchInterval keeps a typo like `--interval=1` from turning a watcher into
// a busy loop that shells out thousands of times a second.
const minWatchInterval = 50 * time.Millisecond

// watchEvent is the one shape every watcher emits. A consumer should be able to
// read any `watch` stream with the same parser, so the envelope — event, at,
// seq — is fixed and each watcher fills only the fields it has. Pointers mark
// the numeric fields that have a meaningful zero (an empty clipboard, monitor 0
// meaning off-screen) so they are omitted rather than reported as 0.
type watchEvent struct {
	Event string `json:"event"`
	At    string `json:"at"`
	Seq   int    `json:"seq"`

	// clipboard
	Length *int   `json:"length,omitempty"`
	Digest string `json:"digest,omitempty"`
	Text   string `json:"text,omitempty"`

	// window
	App      string `json:"app,omitempty"`
	Title    string `json:"title,omitempty"`
	Monitor  *int   `json:"monitor,omitempty"`
	PID      int    `json:"pid,omitempty"`
	WindowID string `json:"window_id,omitempty"`
}

// tracker reports transitions in a sequence of observations. The first sample
// is the state the caller was already in, not news, so it only establishes the
// baseline — that is what makes `--max=1` mean "wait for the next change".
type tracker struct {
	started bool
	last    string
}

func (t *tracker) changed(key string) bool {
	if !t.started {
		t.started, t.last = true, key
		return false
	}
	if key == t.last {
		return false
	}
	t.last = key
	return true
}

// parseInterval accepts a Go duration (`500ms`, `2s`) and a bare number, which
// is read as milliseconds to match `msg listen --interval`.
func parseInterval(value string, fallback time.Duration) (time.Duration, error) {
	if value == "" {
		return fallback, nil
	}
	if ms, err := strconv.Atoi(value); err == nil {
		return time.Duration(ms) * time.Millisecond, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("--interval must be a duration like 500ms, got %q", value)
	}
	return d, nil
}

// clipboardEvent describes a clipboard change. The text is only included when
// the caller asks for it: clipboards routinely hold passwords, tokens and
// whatever the last `pbcopy` swept up, and a watcher left running in a log or
// piped to an agent would capture all of it. Length plus a short digest is
// enough to see *that* it changed and to correlate two events, which is what a
// watcher is for.
func clipboardEvent(at time.Time, text string, withContent bool) watchEvent {
	length := len(text)
	event := watchEvent{
		Event:  "clipboard",
		At:     at.Format(time.RFC3339),
		Length: &length,
		Digest: digest(text),
	}
	if withContent {
		event.Text = text
	}
	return event
}

// windowEvent describes a focus change. It carries what identifies the window
// to a person and to a follow-up command — app, title, monitor — not its
// geometry, which changes for reasons that are not a focus change.
func windowEvent(at time.Time, w windowctl.Window) watchEvent {
	monitor := w.Monitor
	return watchEvent{
		Event:    "window",
		At:       at.Format(time.RFC3339),
		App:      w.App,
		Title:    w.Title,
		Monitor:  &monitor,
		PID:      w.PID,
		WindowID: w.ID,
	}
}

// digest fingerprints clipboard content so two events can be compared without
// the content itself travelling with them. Truncated because it identifies a
// change, and is not meant to be checked against anything.
func digest(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])[:12]
}

// focusKey is the identity a focus change is measured against. The window id is
// not enough on its own: a browser tab switch keeps the id and changes only the
// title, and that is a change worth reporting.
func focusKey(w windowctl.Window) string {
	return strings.Join([]string{w.ID, w.App, w.Title, strconv.Itoa(w.Monitor)}, "\x00")
}

func runWatchClipboard(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args, "interval", "max")
	if err != nil {
		return err
	}
	// --content is a bare flag, so it stays out of parseArgs' value list.
	if err := set.Reject("interval", "max", "content"); err != nil {
		return err
	}
	interval, max, err := watchOptions(set)
	if err != nil {
		return err
	}
	withContent := set.Has("content")

	return watchLoop(c, interval, max, func(at time.Time) (string, watchEvent, error) {
		text, err := clipboardRead()
		if err != nil {
			return "", watchEvent{}, err
		}
		return text, clipboardEvent(at, text, withContent), nil
	})
}

func runWatchWindow(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args, "interval", "max")
	if err != nil {
		return err
	}
	if err := set.Reject("interval", "max"); err != nil {
		return err
	}
	interval, max, err := watchOptions(set)
	if err != nil {
		return err
	}

	return watchLoop(c, interval, max, func(at time.Time) (string, watchEvent, error) {
		window, err := focusedWindow()
		if err != nil {
			return "", watchEvent{}, err
		}
		return focusKey(window), windowEvent(at, window), nil
	})
}

// focusedWindow returns the window the desktop currently considers frontmost.
func focusedWindow() (windowctl.Window, error) {
	windows, err := windowctl.ListWindows(windowctl.Filter{})
	if err != nil {
		return windowctl.Window{}, err
	}
	for _, w := range windows {
		if w.Focused {
			return w, nil
		}
	}
	return windowctl.Window{}, fmt.Errorf("no focused window")
}

// watchOptions reads the two flags every watcher shares.
func watchOptions(set *argSet) (time.Duration, int, error) {
	interval, err := parseInterval(set.String("interval", ""), defaultWatchInterval)
	if err != nil {
		return 0, 0, err
	}
	if interval < minWatchInterval {
		return 0, 0, fmt.Errorf("--interval must be at least %s", minWatchInterval)
	}
	max, err := set.Int("max", 0)
	if err != nil {
		return 0, 0, err
	}
	if max < 0 {
		return 0, 0, fmt.Errorf("--max must not be negative")
	}
	return interval, max, nil
}

// watchLoop polls, prints an event whenever poll's key changes, and stops after
// max events (0 meaning never) or on Ctrl-C. Everything a watcher differs in
// lives in poll, so the two commands cannot drift apart in how they end, how
// they tick, or how they handle a failing sample.
func watchLoop(c *cli.Ctx, interval time.Duration, max int, poll func(time.Time) (string, watchEvent, error)) error {
	// Ctrl-C should end the loop cleanly rather than killing it mid-write.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var seen tracker
	seq := 0
	for {
		key, event, err := poll(time.Now())
		if err != nil {
			// A window that is briefly unfocused, or a clipboard holding
			// something the platform tool cannot read as text, is not a reason
			// to end a long-running watcher; report and try the next tick.
			c.Warnf("agentic-os: %v\n", err)
		} else if seen.changed(key) {
			seq++
			event.Seq = seq
			line, err := json.Marshal(event)
			if err != nil {
				return err
			}
			c.Printf("%s\n", line)
			if max > 0 && seq >= max {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
