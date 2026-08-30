package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writePlugin(t *testing.T, dir, name, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		name += ".cmd"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDiscoverPluginsReadsMetadata(t *testing.T) {
	dir := t.TempDir()
	writePlugin(t, dir, "agentic-os-demo-do-thing", `#!/bin/sh
# agentic-os:summary=Do the thing
# agentic-os:args=<target>
# agentic-os:examples=agentic-os demo do thing a | agentic-os demo do thing b
# agentic-os:platforms=darwin | linux
# agentic-os:sudo=true
echo hi
`)
	r := NewRegistry()
	DiscoverPlugins(r, func(key string) string {
		if key == "AGENTIC_OS_BIN_DIR" {
			return dir
		}
		return ""
	})

	cmd := r.Lookup("demo", "do thing")
	if cmd == nil {
		t.Fatal("plugin was not registered")
	}
	if cmd.Summary != "Do the thing" || cmd.Args != "<target>" {
		t.Fatalf("bad metadata: %+v", cmd)
	}
	if len(cmd.Examples) != 2 {
		t.Fatalf("want 2 examples, got %v", cmd.Examples)
	}
	if !cmd.Sudo || cmd.Supports("windows") {
		t.Fatalf("sudo/platform metadata not applied: %+v", cmd)
	}
}

func TestBuiltinShadowsPlugin(t *testing.T) {
	dir := t.TempDir()
	writePlugin(t, dir, "agentic-os-demo-thing", "#!/bin/sh\n# agentic-os:summary=From script\n")

	r := NewRegistry()
	r.Add(&Command{Group: "demo", Name: "thing", Summary: "Built in", Run: func(*Ctx, []string) error { return nil }})
	DiscoverPlugins(r, func(key string) string {
		if key == "AGENTIC_OS_BIN_DIR" {
			return dir
		}
		return ""
	})

	if got := r.Lookup("demo", "thing").Summary; got != "Built in" {
		t.Fatalf("plugin overrode the builtin: %q", got)
	}
}

func TestParsePluginExplicitRoute(t *testing.T) {
	dir := t.TempDir()
	path := writePlugin(t, dir, "agentic-os-theme-bg-switcher",
		"#!/bin/sh\n# agentic-os:summary=Switch\n# agentic-os:route=theme bg-switcher\n")

	cmd := parsePlugin(path, "theme-bg-switcher")
	if cmd.Route() != "theme bg-switcher" {
		t.Fatalf("got route %q", cmd.Route())
	}
}
