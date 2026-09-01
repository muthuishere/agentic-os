package groups

import (
	"fmt"
	"strings"
	"time"

	"github.com/muthuishere/aos/internal/cli"
)

// subtitleRequest is one caption: what to say, for how long, and where on
// screen. It carries no "wait" — blocking is the caller's business, and every
// backend behaves the same way there.
type subtitleRequest struct {
	Text     string
	Seconds  int
	Position string // bottom | top | center
	Size     int    // text height in pixels
}

const (
	subtitleSeconds = 10
	subtitleSize    = 48
)

func init() {
	register(func(r *cli.Registry) {
		r.Describe("subtitle", "Narrate what an agent is doing, on screen, without taking focus")
		r.Add(
			&cli.Command{
				Group: "subtitle", Name: "show",
				Summary: "Show large text over everything for a few seconds",
				Args:    "<text...> [--seconds=<n>] [--position=bottom|top|center] [--size=<px>] [--wait]",
				Examples: []string{
					`aos subtitle show "installing dependencies"`,
					`aos subtitle show "running the migration" --seconds=30 --position=top`,
					`aos subtitle show "step 1 of 3" --seconds=4 --wait`,
					"# prints which backend was used: an overlay, or a notification it fell back to",
				},
				Run: runSubtitleShow,
			},
			&cli.Command{
				Group: "subtitle", Name: "test",
				Summary:  "Show a short sample, to check subtitles work on this machine",
				Examples: []string{"aos subtitle test"},
				Run:      runSubtitleTest,
			},
		)
	})
}

func runSubtitleShow(c *cli.Ctx, args []string) error {
	// --wait takes no value, so it stays out of valueFlags: `--wait "text"`
	// must not swallow the caption.
	set, err := parseArgs(args, "seconds", "position", "size")
	if err != nil {
		return err
	}
	if err := set.Reject("seconds", "position", "size", "wait"); err != nil {
		return err
	}

	req := subtitleRequest{
		// A caption is one line. Folding newlines here means no backend has to
		// decide what a line break means inside its own script literal.
		Text:     strings.NewReplacer("\n", " ", "\r", " ").Replace(strings.Join(set.Rest, " ")),
		Position: set.String("position", "bottom"),
	}
	if strings.TrimSpace(req.Text) == "" {
		return fmt.Errorf("`subtitle show` needs some text to show")
	}
	switch req.Position {
	case "bottom", "top", "center":
	default:
		return fmt.Errorf("--position must be bottom, top, or center, got %q", req.Position)
	}
	if req.Seconds, err = set.Int("seconds", subtitleSeconds); err != nil {
		return err
	}
	if req.Seconds < 1 {
		return fmt.Errorf("--seconds must be at least 1")
	}
	if req.Size, err = set.Int("size", subtitleSize); err != nil {
		return err
	}
	if req.Size < 8 || req.Size > 400 {
		return fmt.Errorf("--size must be between 8 and 400 pixels")
	}

	// The backend names what it actually did. A caption that quietly became a
	// desktop notification looks like a success from here, so it has to say so.
	mode, err := showSubtitle(req)
	if err != nil {
		return err
	}
	c.Printf("%s  %ds  %s\n", mode, req.Seconds, req.Position)

	// Showing a subtitle returns immediately so it can sit over other work.
	// --wait is for the script that wants the next step to follow this one.
	if set.Has("wait") {
		time.Sleep(time.Duration(req.Seconds) * time.Second)
	}
	return nil
}

// runSubtitleTest goes through the same parsing as `show`, so what it proves is
// the path a real caption takes rather than a shortcut around it.
func runSubtitleTest(c *cli.Ctx, _ []string) error {
	return runSubtitleShow(c, []string{"aos subtitle works", "--seconds=3"})
}
