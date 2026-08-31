package cli

import (
	"bytes"
	"strings"
	"testing"
)

// TestGroupHelpFlag pins a bug that affected every group: `aos <group>
// --help` is documented and was broken, reporting the help flag as an unknown
// command and then suggesting the command that had just failed. Only the bare
// group name worked.
func TestGroupHelpFlag(t *testing.T) {
	r := NewRegistry()
	r.Describe("window", "Desktop windows")
	noop := func(*Ctx, []string) error { return nil }
	r.Add(
		&Command{Group: "window", Name: "list", Summary: "List open windows", Run: noop},
		&Command{Group: "window", Name: "focus", Summary: "Focus a window", Run: noop},
	)

	for _, flag := range []string{"--help", "-h", "help"} {
		var out bytes.Buffer
		c := &Ctx{Registry: r, Stdout: &out, Stderr: &out,
			Env: func(string) string { return "" }, GOOS: "darwin"}

		if code := Run(c, []string{"window", flag}); code != 0 {
			t.Errorf("window %s: exit %d, want 0\n%s", flag, code, out.String())
			continue
		}
		text := out.String()
		if !strings.Contains(text, "Desktop windows") || !strings.Contains(text, "window list") {
			t.Errorf("window %s did not render the group's help:\n%s", flag, text)
		}
		if strings.Contains(text, "unknown command") {
			t.Errorf("window %s reported an unknown command:\n%s", flag, text)
		}
	}
}

// TestBareGroupStillShowsHelp guards the path that already worked, so the fix
// above cannot regress it.
func TestBareGroupStillShowsHelp(t *testing.T) {
	r := NewRegistry()
	r.Describe("audio", "Output volume and mute")
	r.Add(&Command{Group: "audio", Name: "volume", Summary: "Volume",
		Run: func(*Ctx, []string) error { return nil }})

	var out bytes.Buffer
	c := &Ctx{Registry: r, Stdout: &out, Stderr: &out,
		Env: func(string) string { return "" }, GOOS: "darwin"}
	if code := Run(c, []string{"audio"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "audio volume") {
		t.Fatalf("bare group did not list its commands:\n%s", out.String())
	}
}

// TestUnknownSubcommandStillErrors makes sure the fix did not turn a genuine
// typo into a help screen.
func TestUnknownSubcommandStillErrors(t *testing.T) {
	r := NewRegistry()
	r.Describe("audio", "Output volume and mute")
	r.Add(&Command{Group: "audio", Name: "volume", Summary: "Volume",
		Run: func(*Ctx, []string) error { return nil }})

	var out bytes.Buffer
	c := &Ctx{Registry: r, Stdout: &out, Stderr: &out,
		Env: func(string) string { return "" }, GOOS: "darwin"}
	if code := Run(c, []string{"audio", "volme"}); code == 0 {
		t.Fatalf("a typo should not succeed:\n%s", out.String())
	}
}
