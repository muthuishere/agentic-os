package cli

import (
	"bytes"
	"strings"
	"testing"
)

func testRegistry() *Registry {
	r := NewRegistry()
	r.Describe("audio", "Audio controls")
	r.Describe("system", "System controls")
	noop := func(*Ctx, []string) error { return nil }
	r.Add(
		&Command{Group: "audio", Name: "volume", Summary: "Volume", Run: noop},
		&Command{Group: "audio", Name: "output set default", Summary: "Default sink", Run: noop},
		&Command{Group: "system", Name: "lock", Summary: "Lock", Aliases: []string{"lock"}, Run: noop},
		&Command{Group: "system", Name: "hyprctl", Summary: "Wayland only", Platforms: []string{"linux"}, Run: noop},
	)
	return r
}

func TestResolvePrefersLongestRoute(t *testing.T) {
	r := testRegistry()
	cmd, rest, err := r.Resolve([]string{"audio", "output", "set", "default", "--dry-run"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cmd.Route() != "audio output set default" {
		t.Fatalf("got route %q", cmd.Route())
	}
	if len(rest) != 1 || rest[0] != "--dry-run" {
		t.Fatalf("got rest %v", rest)
	}
}

func TestResolveShorterRouteStillWins(t *testing.T) {
	r := testRegistry()
	cmd, rest, err := r.Resolve([]string{"audio", "volume", "40"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cmd.Route() != "audio volume" || len(rest) != 1 || rest[0] != "40" {
		t.Fatalf("got %q with rest %v", cmd.Route(), rest)
	}
}

func TestResolveAlias(t *testing.T) {
	r := testRegistry()
	cmd, _, err := r.Resolve([]string{"lock"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if cmd.Route() != "system lock" {
		t.Fatalf("got route %q", cmd.Route())
	}
}

func TestResolveBareGroupAsksForHelp(t *testing.T) {
	r := testRegistry()
	_, _, err := r.Resolve([]string{"audio"})
	if _, ok := err.(*GroupHelpError); !ok {
		t.Fatalf("want GroupHelpError, got %v", err)
	}
}

func TestResolveUnknownCommandNamesTheGroup(t *testing.T) {
	r := testRegistry()
	_, _, err := r.Resolve([]string{"audio", "nope"})
	if err == nil || !strings.Contains(err.Error(), "audio") {
		t.Fatalf("want an error mentioning the group, got %v", err)
	}
}

func TestDuplicateRouteWarns(t *testing.T) {
	r := testRegistry()
	r.Add(&Command{Group: "audio", Name: "volume", Summary: "Dup"})
	if len(r.Warnings()) == 0 {
		t.Fatal("want a duplicate-route warning")
	}
}

func TestSupportsRespectsPlatforms(t *testing.T) {
	r := testRegistry()
	cmd := r.Lookup("system", "hyprctl")
	if cmd.Supports("darwin") {
		t.Fatal("linux-only command should not support darwin")
	}
	if !cmd.Supports("linux") {
		t.Fatal("linux-only command should support linux")
	}
	if !r.Lookup("audio", "volume").Supports("windows") {
		t.Fatal("a command with no platforms should support every platform")
	}
}

func TestRunUnsupportedExitsTwo(t *testing.T) {
	r := testRegistry()
	var out, errOut bytes.Buffer
	c := &Ctx{Registry: r, Stdout: &out, Stderr: &errOut, Env: func(string) string { return "" }, GOOS: "darwin"}
	if code := Run(c, []string{"system", "hyprctl"}); code != 2 {
		t.Fatalf("want exit 2, got %d", code)
	}
	if !strings.Contains(errOut.String(), "not supported on darwin") {
		t.Fatalf("want a platform message, got %q", errOut.String())
	}
}

func TestRunHelpFlags(t *testing.T) {
	r := testRegistry()
	var out bytes.Buffer
	c := &Ctx{Registry: r, Stdout: &out, Stderr: &out, Env: func(string) string { return "" }, GOOS: "darwin"}
	if code := Run(c, []string{"audio", "volume", "--help"}); code != 0 {
		t.Fatalf("want exit 0, got %d", code)
	}
	if !strings.Contains(out.String(), "agentic-os audio volume") {
		t.Fatalf("help did not name the command: %q", out.String())
	}
}
