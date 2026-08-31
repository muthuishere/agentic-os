package groups

import (
	"encoding/json"
	"fmt"

	"github.com/muthuishere/agentic-os/internal/cli"
	"github.com/muthuishere/agentic-os/internal/skill"
)

func init() {
	register(func(r *cli.Registry) {
		// `install --skills` rather than `skill install`: the same shape the
		// other tools here use, so one habit works across all of them, and so
		// `install` has room for whatever else becomes installable.
		r.Describe("install", "Install what this binary carries")
		r.Add(&cli.Command{
			Group:    "install",
			Summary:  "Install the bundled agent skill",
			Args:     "--skills [--json]",
			Examples: []string{"aos install --skills"},
			Run:      runInstall,
		})

		r.Describe("uninstall", "Remove what this binary installed")
		r.Add(&cli.Command{
			Group:    "uninstall",
			Summary:  "Remove the installed agent skill",
			Args:     "--skills [--json]",
			Examples: []string{"aos uninstall --skills"},
			Run:      runUninstall,
		})

		r.Describe("skill", "The agent skill bundled in this binary")
		r.Add(
			&cli.Command{
				Group: "skill", Name: "show",
				Summary:  "Print the bundled skill without installing it",
				Examples: []string{"aos skill show"},
				Run: func(c *cli.Ctx, _ []string) error {
					content, err := skill.Content()
					if err != nil {
						return err
					}
					c.Printf("%s", content)
					return nil
				},
			},
			&cli.Command{
				Group: "skill", Name: "path",
				Summary: "Print where `install --skills` would write",
				Run: func(c *cli.Ctx, _ []string) error {
					hosts, err := skill.Hosts(c.Env)
					if err != nil {
						return err
					}
					for _, host := range hosts {
						c.Printf("%-8s %s/%s\n", host.Name, host.Root, skill.Name)
					}
					return nil
				},
			},
		)
	})
}

func runInstall(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args)
	if err != nil {
		return err
	}
	if err := set.Reject("skills", "json"); err != nil {
		return err
	}
	// Requiring --skills keeps the verb honest: `install` on its own should not
	// guess what to install, and there will be more than one thing eventually.
	if !set.Has("skills") {
		return fmt.Errorf("say what to install: `aos install --skills`")
	}

	results, err := skill.Install(c.Env)
	if err != nil {
		return err
	}
	return reportSkill(c, set, results)
}

func runUninstall(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args)
	if err != nil {
		return err
	}
	if err := set.Reject("skills", "json"); err != nil {
		return err
	}
	if !set.Has("skills") {
		return fmt.Errorf("say what to remove: `aos uninstall --skills`")
	}

	results, err := skill.Uninstall(c.Env)
	if err != nil {
		return err
	}
	return reportSkill(c, set, results)
}

func reportSkill(c *cli.Ctx, set *argSet, results []skill.Result) error {
	if set.Has("json") {
		enc := json.NewEncoder(c.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	for _, result := range results {
		c.Printf("%-10s %-8s %s\n", result.Action, result.Host, result.Path)
	}
	return nil
}
