package groups

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/muthuishere/agentic-os/internal/cli"
)

// starterAdapter is what `adapters example` prints: a working file someone can
// edit rather than a schema they have to translate.
const starterAdapter = `{
  "group": "my",
  "description": "My own commands",
  "commands": [
    {
      "name": "hello",
      "summary": "Say hello, to prove adapters work",
      "args": "[name]",
      "examples": ["aos my hello world"],
      "run": "echo hello"
    },
    {
      "name": "notes",
      "summary": "Open today's note",
      "run": "$EDITOR ~/notes/$(date +%F).md",
      "platforms": ["darwin", "linux"]
    }
  ]
}
`

func init() {
	register(func(r *cli.Registry) {
		r.Describe("adapters", "Commands you add yourself, without writing a program")
		r.Add(
			&cli.Command{
				Group: "adapters", Name: "list",
				Summary:  "List the adapters this machine has loaded",
				Args:     "[--json]",
				Examples: []string{"aos adapters list"},
				Run:      runAdaptersList,
			},
			&cli.Command{
				Group: "adapters", Name: "path",
				Summary:  "Print the directory adapters are read from",
				Examples: []string{"aos adapters path"},
				Run: func(c *cli.Ctx, _ []string) error {
					c.Println(cli.AdapterDir(c.Env))
					return nil
				},
			},
			&cli.Command{
				Group: "adapters", Name: "example",
				Summary: "Print a starter adapter, ready to save and edit",
				Args:    "[--write]",
				Examples: []string{
					"aos adapters example",
					"aos adapters example --write",
				},
				Run: runAdaptersExample,
			},
		)
	})
}

func runAdaptersList(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args)
	if err != nil {
		return err
	}
	if err := set.Reject("json"); err != nil {
		return err
	}

	// Re-read rather than remembering what was loaded at startup: this command
	// is most useful right after someone edits a file.
	adapters := cli.LoadAdapters(cli.NewRegistry(), c.Env)
	if set.Has("json") {
		enc := json.NewEncoder(c.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(adapters)
	}
	if len(adapters) == 0 {
		c.Printf("no adapters in %s\n", cli.AdapterDir(c.Env))
		c.Println("run `aos adapters example --write` to start one")
		return nil
	}
	for _, adapter := range adapters {
		c.Printf("%-12s %d commands  %s\n", adapter.Group, len(adapter.Commands), adapter.Path)
	}
	return nil
}

func runAdaptersExample(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args)
	if err != nil {
		return err
	}
	if err := set.Reject("write"); err != nil {
		return err
	}
	if !set.Has("write") {
		c.Printf("%s", starterAdapter)
		return nil
	}

	dir := cli.AdapterDir(c.Env)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "my.json")
	// Never overwrite someone's work on a command whose whole job is to be a
	// starting point.
	if _, err := os.Stat(path); err == nil {
		return &cli.ExitError{Code: 1, Message: path + " already exists"}
	}
	if err := os.WriteFile(path, []byte(starterAdapter), 0o644); err != nil {
		return err
	}
	c.Println(path)
	return nil
}
