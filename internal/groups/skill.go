package groups

import (
	"encoding/json"

	"github.com/muthuishere/agentic-os/internal/cli"
	"github.com/muthuishere/agentic-os/internal/skill"
)

func init() {
	register(func(r *cli.Registry) {
		r.Describe("skill", "Teach a coding agent to use this machine")
		r.Add(
			&cli.Command{
				Group: "skill", Name: "install",
				Summary: "Install the bundled agent skill for Claude Code and other agents",
				Args:    "[--json]",
				Examples: []string{
					"agentic-os skill install",
				},
				Run: runSkillInstall,
			},
			&cli.Command{
				Group: "skill", Name: "uninstall",
				Summary:  "Remove the installed agent skill",
				Args:     "[--json]",
				Examples: []string{"agentic-os skill uninstall"},
				Run:      runSkillUninstall,
			},
			&cli.Command{
				Group: "skill", Name: "show",
				Summary:  "Print the bundled skill without installing it",
				Examples: []string{"agentic-os skill show"},
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
				Summary: "Print where the skill installs to",
				Run: func(c *cli.Ctx, _ []string) error {
					hosts, err := skill.Hosts(c.Env)
					if err != nil {
						return err
					}
					for _, host := range hosts {
						c.Printf("%-8s %s\n", host.Name, host.Root+"/"+skill.Name)
					}
					return nil
				},
			},
		)
	})
}

func runSkillInstall(c *cli.Ctx, args []string) error {
	return reportSkill(c, args, skill.Install)
}

func runSkillUninstall(c *cli.Ctx, args []string) error {
	return reportSkill(c, args, skill.Uninstall)
}

// reportSkill runs an install or uninstall and prints what happened per host.
func reportSkill(c *cli.Ctx, args []string, action func(func(string) string) ([]skill.Result, error)) error {
	set, err := parseArgs(args)
	if err != nil {
		return err
	}
	if err := set.Reject("json"); err != nil {
		return err
	}

	results, err := action(c.Env)
	if err != nil {
		return err
	}
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
