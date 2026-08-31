package groups

import (
	"path/filepath"
	"strings"
	"testing"
)

// These tests cover the argument handling of `capture screenshot` only. Every
// case here fails before any capture is attempted, so nothing on this machine's
// screen — or absence of one — changes the result.

// A named target (monitor, region, window) is a deterministic capture windowctl
// does identically on every OS; only the interactive modes go to the platform's
// own screenshot tool. Getting that split wrong silently changes which backend
// runs, so it is asserted directly.
func TestTargetedRecognisesEveryNamedTarget(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{[]string{"--monitor=2"}, true},
		{[]string{"--region=0,0,800,600"}, true},
		{[]string{"--app=Chrome"}, true},
		{[]string{"--title=Inbox"}, true},
		{[]string{"region"}, false}, // the interactive mode, not a target
		{[]string{"--copy"}, false}, // output choice, not a target
		{nil, false},                // plain fullscreen
	}
	for _, tc := range cases {
		set, err := parseArgs(tc.args, "out", "monitor", "region", "app", "title")
		if err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		if got := targeted(set); got != tc.want {
			t.Errorf("targeted(%v) = %v, want %v", tc.args, got, tc.want)
		}
	}
}

// A targeted capture writes a file; there is no interactive step to put on the
// clipboard. Saying so is better than producing an empty clipboard.
func TestScreenshotRefusesCopyWithATargetedCapture(t *testing.T) {
	for _, args := range [][]string{
		{"--copy", "--monitor=1"},
		{"--copy", "--region=0,0,10,10"},
		{"--copy", "--app=Chrome"},
	} {
		c, _, _ := testCtx("")
		err := runScreenshot(c, args)
		if err == nil {
			t.Fatalf("%v was allowed", args)
		}
		if !strings.Contains(err.Error(), "--copy") {
			t.Fatalf("%v failed for the wrong reason: %v", args, err)
		}
	}
}

func TestScreenshotRejectsUnknownModesAndFlags(t *testing.T) {
	cases := map[string][]string{
		"unknown positional": {"--copy", "everything"},
		"typo'd flag":        {"--monitr=1"},
		"bad monitor value":  {"--monitor=two", "--out=" + filepath.Join(t.TempDir(), "x.png")},
		"bad region shape":   {"--region=0,0,800", "--out=" + filepath.Join(t.TempDir(), "x.png")},
	}
	for name, args := range cases {
		c, _, _ := testCtx("")
		if err := runScreenshot(c, args); err == nil {
			t.Errorf("%s: `capture screenshot %v` must fail", name, args)
		}
	}
}

// The three spellings of each mode exist because an agent will type any of
// them; they must all reach the same capture.
func TestScreenshotModeAliasesAreAccepted(t *testing.T) {
	for _, arg := range []string{"fullscreen", "full", "screen", "region", "area", "select", "window", "win"} {
		set, err := parseArgs([]string{arg, "--copy"}, "out", "monitor", "region", "app", "title")
		if err != nil {
			t.Fatalf("%q: %v", arg, err)
		}
		if len(set.Rest) != 1 || set.Rest[0] != arg {
			t.Fatalf("%q was not kept as a positional: %v", arg, set.Rest)
		}
	}
}
