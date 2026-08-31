package groups

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/muthuishere/agentic-os/internal/cli"
	toolnexus "github.com/muthuishere/toolnexus/golang"
)

// defaultServeAddr binds loopback only. Serving this registry is remote control
// of the machine, so reaching it from elsewhere should be a deliberate act
// (an SSH tunnel, or an explicit --addr).
const defaultServeAddr = "127.0.0.1:14320"

func init() {
	register(func(r *cli.Registry) {
		r.Describe("serve", "Expose this machine's commands to other agents")
		r.Add(
			&cli.Command{
				Group: "serve", Name: "mcp",
				Summary: "Serve every command as an MCP tool over streamable HTTP",
				Args:    "[--addr=<host:port>] [--groups=<a,b>] [--gui=on|off|auto] [--quiet]",
				Examples: []string{
					"aos serve mcp",
					"aos serve mcp --addr=127.0.0.1:9000 --groups=window,capture,exec",
					"aos serve mcp --gui=off   # screenless tools only",
				},
				Run: runServeMCP,
			},
			&cli.Command{
				Group: "serve", Name: "tools",
				Summary:  "Print the MCP tool catalogue this machine would expose",
				Args:     "[--groups=<a,b>]",
				Examples: []string{"aos serve tools --groups=window"},
				Run:      runServeTools,
			},
		)
	})
}

// toolName converts a route into an MCP tool name. MCP names allow only
// [A-Za-z0-9_-], so the spaces that separate route words become underscores:
// `audio output volume` is exposed as `audio_output_volume`.
func toolName(route string) string {
	return strings.ReplaceAll(route, " ", "_")
}

// buildTools turns the command registry into MCP tools. Every command becomes
// one tool taking the same arguments a person would type, so an agent and a
// terminal drive the identical code path.
func buildTools(c *cli.Ctx, only map[string]bool, gui guiMode) []toolnexus.Tool {
	includeGUI := gui == guiOn || (gui == guiAuto && cli.HasDisplay(c.Env, c.GOOS))

	var tools []toolnexus.Tool
	for _, cmd := range c.Registry.Commands() {
		if cmd.Hidden || cmd.Blocking || !cmd.Supports(c.GOOS) {
			continue
		}
		// Offering an agent a tool that cannot run is worse than not offering
		// it: it will try, fail, and try again.
		if cmd.NeedsDisplay && !includeGUI {
			continue
		}
		if len(only) > 0 && !only[cmd.Group] {
			continue
		}
		// `serve` itself is deliberately not exposed: an agent starting more
		// servers through the server is a loop nobody asked for.
		if cmd.Group == "serve" {
			continue
		}

		route := cmd.Route()
		tools = append(tools, toolnexus.NativeTool(
			toolName(route),
			describeTool(cmd),
			toolnexus.JSONSchema{
				"type": "object",
				"properties": map[string]any{
					"args": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Arguments, exactly as typed after `aos " + route + "`.",
					},
					"stdin": map[string]any{
						"type":        "string",
						"description": "Text piped to the command on stdin.",
					},
				},
			},
			func(ctx context.Context, args map[string]any) (string, error) {
				return invokeTool(c, route, args)
			},
		))
	}
	return tools
}

// describeTool writes the tool description an agent reads when choosing: the
// summary, the argument shape, and the examples the CLI already documents.
func describeTool(cmd *cli.Command) string {
	var b strings.Builder
	b.WriteString(cmd.Summary)
	if cmd.Args != "" {
		fmt.Fprintf(&b, "\n\nUsage: aos %s %s", cmd.Route(), cmd.Args)
	}
	if len(cmd.Examples) > 0 {
		b.WriteString("\n\nExamples:\n  " + strings.Join(cmd.Examples, "\n  "))
	}
	if cmd.Sudo {
		b.WriteString("\n\nRequires elevated privileges.")
	}
	return b.String()
}

func invokeTool(c *cli.Ctx, route string, args map[string]any) (string, error) {
	argv := strings.Fields(route)
	if raw, ok := args["args"]; ok {
		extra, err := toStringSlice(raw)
		if err != nil {
			return "", err
		}
		argv = append(argv, extra...)
	}
	stdin, _ := args["stdin"].(string)

	result := cli.Invoke(c, argv, stdin)
	if result.Exit != 0 {
		message := strings.TrimSpace(result.Stderr)
		if message == "" {
			message = strings.TrimSpace(result.Stdout)
		}
		if message == "" {
			message = fmt.Sprintf("exit status %d", result.Exit)
		}
		return "", fmt.Errorf("%s", message)
	}
	// Some commands say everything on stderr (progress, cursors); pass it along
	// rather than returning a confusingly empty success.
	output := result.Stdout
	if strings.TrimSpace(output) == "" {
		output = result.Stderr
	}
	if strings.TrimSpace(output) == "" {
		return "ok", nil
	}
	return output, nil
}

// toStringSlice accepts the array shapes a JSON tool call can produce.
func toStringSlice(raw any) ([]string, error) {
	switch value := raw.(type) {
	case nil:
		return nil, nil
	case []string:
		return value, nil
	case string:
		if value == "" {
			return nil, nil
		}
		return []string{value}, nil
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("args must be strings, got %T", item)
			}
			out = append(out, text)
		}
		return out, nil
	}
	return nil, fmt.Errorf("args must be an array of strings, got %T", raw)
}

// guiMode decides whether the GUI commands are exposed as tools.
type guiMode int

const (
	// guiAuto exposes them only when this machine actually has a display.
	guiAuto guiMode = iota
	guiOn
	guiOff
)

func parseGUIMode(value string) (guiMode, error) {
	switch value {
	case "", "auto":
		return guiAuto, nil
	case "on", "yes", "true":
		return guiOn, nil
	case "off", "no", "false":
		return guiOff, nil
	}
	return guiAuto, fmt.Errorf("--gui must be on, off, or auto, got %q", value)
}

// groupFilter reads --groups=a,b into a set; an empty set means every group.
func groupFilter(set *argSet) map[string]bool {
	only := map[string]bool{}
	for _, name := range strings.Split(set.String("groups", ""), ",") {
		if name = strings.TrimSpace(name); name != "" {
			only[name] = true
		}
	}
	return only
}

func runServeTools(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args, "groups", "gui")
	if err != nil {
		return err
	}
	if err := set.Reject("groups", "gui", "json"); err != nil {
		return err
	}
	gui, err := parseGUIMode(set.String("gui", ""))
	if err != nil {
		return err
	}

	tools := buildTools(c, groupFilter(set), gui)
	if set.Has("json") {
		type toolJSON struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		payload := make([]toolJSON, 0, len(tools))
		for _, tool := range tools {
			payload = append(payload, toolJSON{tool.Name, tool.Description})
		}
		enc := json.NewEncoder(c.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}
	for _, tool := range tools {
		summary, _, _ := strings.Cut(tool.Description, "\n")
		c.Printf("%-28s %s\n", tool.Name, summary)
	}
	c.Warnf("%d tools\n", len(tools))
	return nil
}

func runServeMCP(c *cli.Ctx, args []string) error {
	set, err := parseArgs(args, "addr", "groups", "gui")
	if err != nil {
		return err
	}
	if err := set.Reject("addr", "groups", "gui", "quiet"); err != nil {
		return err
	}
	gui, err := parseGUIMode(set.String("gui", ""))
	if err != nil {
		return err
	}

	addr := set.String("addr", defaultServeAddr)
	tools := buildTools(c, groupFilter(set), gui)
	if len(tools) == 0 {
		return fmt.Errorf("no commands matched --groups")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Builtins are toolnexus's own shell/file tools; aos already exposes
	// its own, so leaving them on would offer an agent two ways to do one thing.
	toolkit, err := toolnexus.CreateToolkit(ctx, toolnexus.Options{
		ExtraTools: tools,
		Builtins:   false,
	})
	if err != nil {
		return err
	}

	quiet := set.Has("quiet")
	handle, err := toolkit.Serve(addr, toolnexus.ServeOptions{
		MCP: &toolnexus.MCPServeConfig{Name: "aos", Version: c.Version},
		OnCall: func(event toolnexus.OnCallEvent) {
			if quiet {
				return
			}
			status := "ok"
			if event.IsError {
				status = "error"
			}
			c.Warnf("%-28s %s %dms\n", event.Name, status, event.Ms)
		},
	})
	if err != nil {
		return err
	}
	defer handle.Stop()

	c.Printf("aos mcp  %s/mcp\n", handle.URL)
	c.Printf("tools           %d\n", len(tools))
	if !cli.HasDisplay(c.Env, c.GOOS) && gui != guiOn {
		c.Printf("gui             excluded — no display on this machine\n")
	}
	c.Printf("connect         claude mcp add --transport http aos %s/mcp\n", handle.URL)

	<-ctx.Done()
	c.Warnf("\nstopping\n")
	return nil
}
