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
	writePlugin(t, dir, "aos-demo-do-thing", `#!/bin/sh
# aos:summary=Do the thing
# aos:args=<target>
# aos:examples=aos demo do thing a | aos demo do thing b
# aos:platforms=darwin | linux
echo hi
`)
	r := NewRegistry()
	DiscoverPlugins(r, func(key string) string {
		if key == "AOS_BIN_DIR" {
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
	if cmd.Supports("windows") {
		t.Fatalf("platform metadata not applied: %+v", cmd)
	}
}

func TestBuiltinShadowsPlugin(t *testing.T) {
	dir := t.TempDir()
	writePlugin(t, dir, "aos-demo-thing", "#!/bin/sh\n# aos:summary=From script\n")

	r := NewRegistry()
	r.Add(&Command{Group: "demo", Name: "thing", Summary: "Built in", Run: func(*Ctx, []string) error { return nil }})
	DiscoverPlugins(r, func(key string) string {
		if key == "AOS_BIN_DIR" {
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
	path := writePlugin(t, dir, "aos-theme-bg-switcher",
		"#!/bin/sh\n# aos:summary=Switch\n# aos:route=theme bg-switcher\n")

	cmd := parsePlugin(path, "theme-bg-switcher")
	if cmd.Route() != "theme bg-switcher" {
		t.Fatalf("got route %q", cmd.Route())
	}
}

// The docs and the bundled agent skill both show the short forms — a file named
// `aos-<group>-<name>` describing itself with `# aos:summary=`. Only the file
// name was forgiving, so a plugin written exactly the documented way was
// discovered and then listed with an empty summary.
func TestPluginMetadataAcceptsBothTags(t *testing.T) {
	for _, tag := range []string{"aos", "agentic-os"} {
		t.Run(tag, func(t *testing.T) {
			dir := t.TempDir()
			writePlugin(t, dir, "aos-demo-thing",
				"#!/bin/sh\n# "+tag+":summary=From script\n# "+tag+":args=<target>\n")

			r := NewRegistry()
			DiscoverPlugins(r, func(key string) string {
				if key == "AOS_BIN_DIR" {
					return dir
				}
				return ""
			})

			cmd := r.Lookup("demo", "thing")
			if cmd == nil {
				t.Fatal("plugin was not discovered")
			}
			if cmd.Summary != "From script" {
				t.Fatalf("summary = %q, want %q", cmd.Summary, "From script")
			}
			if cmd.Args != "<target>" {
				t.Fatalf("args = %q, want %q", cmd.Args, "<target>")
			}
		})
	}
}
