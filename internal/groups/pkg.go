package groups

import (
	"fmt"

	"github.com/muthuishere/agentic-os/internal/cli"
	"github.com/muthuishere/agentic-os/internal/sys"
)

// packageManager adapts one native package manager to a shared verb set.
// Each field is the argv prefix for that verb; the user's package names are
// appended. A nil prefix means the manager has no such verb.
type packageManager struct {
	Name    string
	Bin     string
	Install []string
	Remove  []string
	Search  []string
	List    []string
	Refresh []string
	Upgrade []string
}

func init() {
	register(func(r *cli.Registry) {
		r.Describe("pkg", "Native package manager, one verb set on every OS")
		verbs := []struct {
			name, summary, args string
			needsArgs           bool
			argv                func(*packageManager) []string
		}{
			{"install", "Install packages", "<package...>", true,
				func(m *packageManager) []string { return m.Install }},
			{"remove", "Remove packages", "<package...>", true,
				func(m *packageManager) []string { return m.Remove }},
			{"search", "Search for packages", "<term>", true,
				func(m *packageManager) []string { return m.Search }},
			{"list", "List installed packages", "", false,
				func(m *packageManager) []string { return m.List }},
			{"refresh", "Refresh the package index", "", false,
				func(m *packageManager) []string { return m.Refresh }},
			{"upgrade", "Upgrade installed packages", "", false,
				func(m *packageManager) []string { return m.Upgrade }},
		}

		for _, verb := range verbs {
			argv, needsArgs := verb.argv, verb.needsArgs
			name := verb.name
			r.Add(&cli.Command{
				Group: "pkg", Name: name,
				Summary:  verb.summary,
				Args:     verb.args,
				Examples: []string{"aos pkg " + name + " " + verb.args},
				Run: func(c *cli.Ctx, args []string) error {
					return runPkgVerb(c, name, argv, needsArgs, args)
				},
			})
		}

		r.Add(&cli.Command{
			Group: "pkg", Name: "manager",
			Summary:  "Print which package manager this machine uses",
			Examples: []string{"aos pkg manager        # homebrew"},
			Run: func(c *cli.Ctx, _ []string) error {
				manager := detectPackageManager()
				if manager == nil {
					return &cli.ExitError{Code: 1, Message: "no supported package manager found"}
				}
				c.Println(manager.Name)
				return nil
			},
		})
	})
}

func runPkgVerb(c *cli.Ctx, verb string, argv func(*packageManager) []string, needsArgs bool, args []string) error {
	manager := detectPackageManager()
	if manager == nil {
		return &cli.ExitError{Code: 1, Message: "no supported package manager found on this machine"}
	}
	prefix := argv(manager)
	if prefix == nil {
		return fmt.Errorf("%s has no %s command", manager.Name, verb)
	}
	if needsArgs && len(args) == 0 {
		return fmt.Errorf("`pkg %s` needs at least one package name", verb)
	}
	// Package managers are interactive (prompts, progress bars), so hand them
	// the real terminal rather than capturing their output.
	return passthroughExit(sys.Passthrough(manager.Bin, append(prefix, args...)...))
}

// passthroughExit turns a child process failure into our own exit code without
// printing a second, redundant error line.
func passthroughExit(err error) error {
	if err == nil {
		return nil
	}
	if code := exitCodeOf(err); code > 0 {
		return &cli.ExitError{Code: code}
	}
	return err
}
