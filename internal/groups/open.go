package groups

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/muthuishere/agentic-os/internal/cli"
	"github.com/muthuishere/windowctl"
)

func init() {
	register(func(r *cli.Registry) {
		r.Describe("open", "Hand a file, folder, or URL to its default application")
		r.Add(&cli.Command{
			Group:   "open",
			Summary: "Open a file, folder, or URL with the system default handler",
			Args:    "<path-or-url>",
			Examples: []string{
				"agentic-os open https://claude.ai",
				"agentic-os open .",
			},
			Run: runOpen,
		})

		r.Describe("launch", "Start applications")
		r.Add(&cli.Command{
			Group:   "launch",
			Summary: "Launch an application, optionally waiting for its window",
			Args:    "<app> [args...] [--wait[=<ms>]]",
			Examples: []string{
				`agentic-os launch "Visual Studio Code"`,
				"agentic-os launch Ghostty --wait",
			},
			Run: runLaunch,
		})

		r.Describe("webapp", "Open a site as a standalone browser window")
		r.Add(&cli.Command{
			Group: "webapp", Name: "open",
			Summary:  "Open a URL in a chromeless app window",
			Args:     "<url>",
			Examples: []string{"agentic-os webapp open https://app.slack.com"},
			Run:      runWebapp,
		})
	})
}

func runLaunch(c *cli.Ctx, args []string) error {
	// `wait` is deliberately not a value flag: bare `--wait` means "use the
	// default timeout", and an explicit one is given as `--wait=5000`.
	set, err := parseArgs(args)
	if err != nil {
		return err
	}
	if err := set.Reject("wait"); err != nil {
		return err
	}
	if len(set.Rest) == 0 {
		return fmt.Errorf("`launch` needs an application name")
	}

	app := set.Rest[0]
	if err := launchApp(app, set.Rest[1:]); err != nil {
		return err
	}
	if !set.Has("wait") {
		return nil
	}

	// Launching is asynchronous everywhere, so --wait is what makes `launch`
	// composable: the next command can assume the window is really there.
	timeout, err := set.Int("wait", 10000)
	if err != nil {
		return err
	}
	window, err := waitForMatch(windowctl.Match{App: app}, timeout)
	if err != nil {
		return err
	}
	c.Println(describeWindow(window))
	return nil
}

func runOpen(c *cli.Ctx, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("`open` needs a path or URL")
	}
	return openTarget(args[0])
}

func runWebapp(c *cli.Ctx, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("`webapp open` needs a URL")
	}
	target := args[0]
	if !strings.Contains(target, "://") {
		target = "https://" + target
	}
	if _, err := url.ParseRequestURI(target); err != nil {
		return fmt.Errorf("not a usable URL: %q", args[0])
	}
	return openWebapp(target)
}
