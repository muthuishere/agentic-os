package groups

import (
	"sort"

	"github.com/muthuishere/agentic-os/internal/cli"
)

func init() {
	register(func(r *cli.Registry) {
		r.Describe("system", "Session power state: lock, sleep, restart, shutdown, logout")
		r.Add(
			&cli.Command{
				Group: "system", Name: "lock",
				Summary:  "Lock the screen",
				Aliases:  []string{"lock"},
				Examples: []string{"aos system lock"},
				Run:      func(c *cli.Ctx, _ []string) error { return systemLock() },
			},
			&cli.Command{
				Group: "system", Name: "sleep",
				Summary: "Put the machine to sleep",
				Run:     func(c *cli.Ctx, _ []string) error { return systemSleep() },
			},
			&cli.Command{
				Group: "system", Name: "restart",
				Summary: "Restart the machine",
				Run:     func(c *cli.Ctx, _ []string) error { return systemRestart() },
			},
			&cli.Command{
				Group: "system", Name: "shutdown",
				Summary: "Power the machine off",
				Run:     func(c *cli.Ctx, _ []string) error { return systemShutdown() },
			},
			&cli.Command{
				Group: "system", Name: "logout",
				Summary: "Log the current user out",
				Run:     func(c *cli.Ctx, _ []string) error { return systemLogout() },
			},
			&cli.Command{
				Group: "system", Name: "info",
				Summary:  "Print OS, host, and hardware facts",
				Examples: []string{"aos system info"},
				Run:      runSystemInfo,
			},
		)
	})
}

func runSystemInfo(c *cli.Ctx, _ []string) error {
	facts, err := systemInfo()
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(facts))
	for key := range facts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	width := 0
	for _, key := range keys {
		if len(key) > width {
			width = len(key)
		}
	}
	for _, key := range keys {
		c.Printf("%-*s  %s\n", width, key, facts[key])
	}
	return nil
}
