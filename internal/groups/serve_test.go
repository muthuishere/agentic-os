package groups

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/muthuishere/agentic-os/internal/cli"
	toolnexus "github.com/muthuishere/toolnexus/golang"
)

// serveRegistry has one command of each kind the tool surface has to decide
// about: ordinary, blocking, display-dependent, platform-specific, hidden, and
// `serve` itself.
func serveRegistry() *cli.Registry {
	r := cli.NewRegistry()
	r.Describe("file", "Files")
	r.Describe("msg", "Messages")
	r.Describe("window", "Windows")
	r.Describe("system", "System")
	r.Describe("serve", "Serve")
	noop := func(*cli.Ctx, []string) error { return nil }
	r.Add(
		&cli.Command{Group: "file", Name: "read", Summary: "Read a file", Args: "<path>",
			Examples: []string{"aos file read go.mod"}, Run: noop},
		&cli.Command{Group: "msg", Name: "send", Summary: "Send a message", Run: noop},
		&cli.Command{Group: "msg", Name: "listen", Summary: "Follow the hub", Blocking: true, Run: noop},
		&cli.Command{Group: "window", Name: "list", Summary: "List windows", NeedsDisplay: true, Run: noop},
		&cli.Command{Group: "system", Name: "hyprctl", Summary: "Wayland only", Platforms: []string{"linux"}, Run: noop},
		&cli.Command{Group: "system", Name: "secret", Summary: "Hidden", Hidden: true, Run: noop},
		&cli.Command{Group: "serve", Name: "mcp", Summary: "Serve MCP", Run: noop},
	)
	return r
}

func serveCtx(goos string, env map[string]string) (*cli.Ctx, *bytes.Buffer) {
	var out bytes.Buffer
	return &cli.Ctx{
		Registry: serveRegistry(),
		Stdin:    io.NopCloser(strings.NewReader("")),
		Stdout:   &out,
		Stderr:   &out,
		Env:      func(key string) string { return env[key] },
		GOOS:     goos,
		Version:  "test",
	}, &out
}

func toolNames(tools []toolnexus.Tool) map[string]bool {
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	return names
}

// A tool that never returns is worse than a missing one: an agent calls
// `msg listen`, the request never completes, and the session is stuck until it
// times out. Blocking commands are therefore left off the MCP surface even
// though they are fine at a terminal.
func TestBuildToolsExcludesBlockingCommands(t *testing.T) {
	c, _ := serveCtx("darwin", nil)
	names := toolNames(buildTools(c, nil, guiOn))

	if names["msg_listen"] {
		t.Fatal("a blocking command must not be exposed as a tool")
	}
	if !names["msg_send"] {
		t.Fatal("the non-blocking command in the same group should still be a tool")
	}
}

// Offering an agent a GUI tool on a headless machine makes it try, fail, and
// try again. Whether a display exists is what decides, unless --gui overrides.
func TestBuildToolsIncludesGUIToolsOnlyWhenThereIsADisplay(t *testing.T) {
	cases := []struct {
		name    string
		goos    string
		env     map[string]string
		gui     guiMode
		wantGUI bool
	}{
		{"auto on a headless linux box", "linux", nil, guiAuto, false},
		{"auto with X11", "linux", map[string]string{"DISPLAY": ":0"}, guiAuto, true},
		{"auto with wayland", "linux", map[string]string{"WAYLAND_DISPLAY": "wayland-1"}, guiAuto, true},
		{"auto on macOS assumes a session", "darwin", nil, guiAuto, true},
		{"--gui=on forces them in", "linux", nil, guiOn, true},
		{"--gui=off forces them out", "darwin", nil, guiOff, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := serveCtx(tc.goos, tc.env)
			names := toolNames(buildTools(c, nil, tc.gui))
			if got := names["window_list"]; got != tc.wantGUI {
				t.Errorf("window_list exposed = %v, want %v", got, tc.wantGUI)
			}
			// The screenless tool is exposed either way; only the GUI ones move.
			if !names["file_read"] {
				t.Error("a screenless command must be exposed in every mode")
			}
		})
	}
}

func TestBuildToolsExcludesUnsupportedHiddenAndServeCommands(t *testing.T) {
	c, _ := serveCtx("darwin", nil)
	names := toolNames(buildTools(c, nil, guiOn))

	if names["system_hyprctl"] {
		t.Error("a linux-only command must not be a tool on darwin")
	}
	if names["system_secret"] {
		t.Error("a hidden command must not be a tool")
	}
	// An agent starting more servers through the server is a loop nobody asked for.
	if names["serve_mcp"] {
		t.Error("`serve` must not expose itself")
	}
}

func TestBuildToolsHonoursTheGroupFilter(t *testing.T) {
	c, _ := serveCtx("darwin", nil)
	names := toolNames(buildTools(c, map[string]bool{"file": true}, guiOn))

	if !names["file_read"] || len(names) != 1 {
		t.Fatalf("--groups=file yielded %v", names)
	}
}

// MCP tool names allow only [A-Za-z0-9_-], so every route word separator has to
// become an underscore — a name with a space is rejected by the client, not by
// us, which makes it a confusing failure to debug.
func TestToolNameIsMCPSafe(t *testing.T) {
	cases := map[string]string{
		"audio output set default": "audio_output_set_default",
		"doctor":                   "doctor",
		"file read":                "file_read",
	}
	for route, want := range cases {
		if got := toolName(route); got != want {
			t.Errorf("toolName(%q) = %q, want %q", route, got, want)
		}
	}

	c, _ := serveCtx("darwin", nil)
	for _, tool := range buildTools(c, nil, guiOn) {
		if strings.ContainsAny(tool.Name, " \t") {
			t.Errorf("tool name %q contains whitespace", tool.Name)
		}
	}
}

// The description is what an agent reads when choosing a tool, so it has to
// carry the usage line and the examples the CLI already documents.
func TestDescribeToolCarriesUsageAndExamples(t *testing.T) {
	cmd := &cli.Command{
		Group: "file", Name: "read", Summary: "Print a file",
		Args:     "<path> [--lines=<from>:<to>]",
		Examples: []string{"aos file read go.mod"},
	}
	got := describeTool(cmd)
	for _, want := range []string{
		"Print a file",
		"Usage: aos file read <path> [--lines=<from>:<to>]",
		"aos file read go.mod",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("description is missing %q:\n%s", want, got)
		}
	}
}

func TestParseGUIMode(t *testing.T) {
	cases := map[string]guiMode{
		"": guiAuto, "auto": guiAuto,
		"on": guiOn, "yes": guiOn, "true": guiOn,
		"off": guiOff, "no": guiOff, "false": guiOff,
	}
	for value, want := range cases {
		got, err := parseGUIMode(value)
		if err != nil || got != want {
			t.Errorf("parseGUIMode(%q) = %v, %v", value, got, err)
		}
	}
	if _, err := parseGUIMode("maybe"); err == nil {
		t.Fatal("want an error for an unknown --gui value")
	}
}

func TestGroupFilterSplitsAndIgnoresBlanks(t *testing.T) {
	set, err := parseArgs([]string{"--groups=window, capture ,,exec"}, "groups")
	if err != nil {
		t.Fatal(err)
	}
	only := groupFilter(set)
	if len(only) != 3 || !only["window"] || !only["capture"] || !only["exec"] {
		t.Fatalf("groupFilter = %v", only)
	}

	// No --groups means every group, which buildTools reads as an empty set.
	set, _ = parseArgs(nil, "groups")
	if len(groupFilter(set)) != 0 {
		t.Fatal("an absent --groups must not filter anything")
	}
}

// A tool call runs the same code path as the terminal, so a command's failure
// has to reach the agent as an error carrying the command's own message rather
// than as a successful-looking empty result.
func TestInvokeToolReportsFailureWithTheCommandsMessage(t *testing.T) {
	c, _ := serveCtx("darwin", nil)
	c.Registry.Add(&cli.Command{Group: "file", Name: "boom", Summary: "Fails",
		Run: func(*cli.Ctx, []string) error { return &cli.ExitError{Code: 1, Message: "no such channel"} }})

	if _, err := invokeTool(c, "file boom", nil); err == nil || !strings.Contains(err.Error(), "no such channel") {
		t.Fatalf("err = %v, want the command's own message", err)
	}
}

func TestInvokeToolPassesArgumentsAndStdinThrough(t *testing.T) {
	c, _ := serveCtx("darwin", nil)
	c.Registry.Add(&cli.Command{Group: "file", Name: "echo", Summary: "Echo",
		Run: func(inner *cli.Ctx, args []string) error {
			piped, _ := io.ReadAll(inner.Stdin)
			inner.Printf("args=%s stdin=%s", strings.Join(args, ","), piped)
			return nil
		}})

	got, err := invokeTool(c, "file echo", map[string]any{
		"args":  []any{"a", "b"},
		"stdin": "piped",
	})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if got != "args=a,b stdin=piped" {
		t.Fatalf("output = %q", got)
	}
}

// A command that says everything on stderr (progress, cursors) still succeeded;
// returning an empty string would read as "it did nothing".
func TestInvokeToolFallsBackToStderrThenToOK(t *testing.T) {
	c, _ := serveCtx("darwin", nil)
	c.Registry.Add(
		&cli.Command{Group: "file", Name: "onlystderr", Summary: "Warns", Run: func(inner *cli.Ctx, _ []string) error {
			inner.Warnf("next=8 printed=0\n")
			return nil
		}},
		&cli.Command{Group: "file", Name: "silent", Summary: "Silent", Run: func(*cli.Ctx, []string) error { return nil }},
	)

	got, err := invokeTool(c, "file onlystderr", nil)
	if err != nil || !strings.Contains(got, "next=8") {
		t.Fatalf("got %q, %v", got, err)
	}
	got, err = invokeTool(c, "file silent", nil)
	if err != nil || got != "ok" {
		t.Fatalf("a silent success should read as ok, got %q, %v", got, err)
	}
}

func TestInvokeToolRejectsBadlyTypedArguments(t *testing.T) {
	c, _ := serveCtx("darwin", nil)
	if _, err := invokeTool(c, "file read", map[string]any{"args": []any{1, 2}}); err == nil {
		t.Fatal("want an error when args are not strings")
	}
}

// `serve tools` is how a person checks what this machine would expose before
// starting a server, so its JSON must list exactly the tools that would exist.
func TestServeToolsJSONListsTheSurface(t *testing.T) {
	c, out := serveCtx("darwin", nil)
	if err := runServeTools(c, []string{"--groups=file", "--json", "--gui=off"}); err != nil {
		t.Fatalf("serve tools: %v", err)
	}
	if !strings.Contains(out.String(), `"name": "file_read"`) {
		t.Fatalf("output does not list file_read:\n%s", out.String())
	}
	if strings.Contains(out.String(), "window_list") {
		t.Fatalf("--gui=off still listed a GUI tool:\n%s", out.String())
	}

	c, _ = serveCtx("darwin", nil)
	if err := runServeTools(c, []string{"--gui=maybe"}); err == nil {
		t.Fatal("want an error for a bad --gui value")
	}
	c, _ = serveCtx("darwin", nil)
	if err := runServeTools(c, []string{"--grops=file"}); err == nil {
		t.Fatal("want an error for a typo'd flag")
	}
}
