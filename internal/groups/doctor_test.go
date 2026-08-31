package groups

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muthuishere/agentic-os/internal/cli"
)

// doctorCtx builds a context with a controllable environment. Only the checks
// that need nothing from the machine are exercised here: the screenshot,
// clipboard, window and messenger checks deliberately touch real hardware and a
// real hub, and asserting on them would make the suite depend on this desk.
func doctorCtx(goos string, env map[string]string) *cli.Ctx {
	r := cli.NewRegistry()
	r.Describe("file", "Files")
	r.Describe("system", "System")
	noop := func(*cli.Ctx, []string) error { return nil }
	r.Add(
		&cli.Command{Group: "file", Name: "read", Summary: "Read", Run: noop},
		&cli.Command{Group: "system", Name: "hyprctl", Summary: "Wayland only", Platforms: []string{"linux"}, Run: noop},
	)

	var out bytes.Buffer
	return &cli.Ctx{
		Registry: r,
		Stdin:    io.NopCloser(strings.NewReader("")),
		Stdout:   &out,
		Stderr:   &out,
		Env:      func(key string) string { return env[key] },
		GOOS:     goos,
		Version:  "test",
	}
}

// The registry check is the one that can fail before any hardware is involved:
// a duplicate route or a group-less command is a programming error that would
// otherwise only show up as a mysteriously missing command.
func TestDoctorRegistryCheckReportsRegistrationProblems(t *testing.T) {
	c := doctorCtx("darwin", nil)
	if got := checkRegistry(c); got.Status != statusOK {
		t.Fatalf("a clean registry should pass: %+v", got)
	}
	// Counting is per platform, so a linux-only command is not "available" here.
	if got := checkRegistry(c).Detail; !strings.Contains(got, "1 of 2") {
		t.Fatalf("detail = %q, want the available/total counts for darwin", got)
	}

	c.Registry.Add(&cli.Command{Group: "file", Name: "read", Summary: "Duplicate"})
	got := checkRegistry(c)
	if got.Status != statusFail {
		t.Fatalf("a duplicate route must fail the check: %+v", got)
	}
	if got.Remedy == "" {
		t.Fatal("a check that reports a problem without a remedy has only moved the work")
	}
}

// Telemetry that cannot be written is worse than none, because it looks
// enabled — so the check writes for real rather than testing for a directory.
func TestDoctorTelemetryCheckIsFunctional(t *testing.T) {
	dir := t.TempDir()
	c := doctorCtx("darwin", map[string]string{"AGENTIC_OS_TELEMETRY_DIR": dir})
	if got := checkTelemetry(c); got.Status != statusOK {
		t.Fatalf("a writable directory should pass: %+v", got)
	}
	// The probe file must not be left behind.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("the doctor probe was left in the telemetry directory: %v", entries)
	}

	// A regular file where the directory should be: unwritable on every OS.
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	c = doctorCtx("darwin", map[string]string{"AGENTIC_OS_TELEMETRY_DIR": filepath.Join(blocker, "telemetry")})
	got := checkTelemetry(c)
	if got.Status != statusFail {
		t.Fatalf("an unwritable telemetry directory must fail: %+v", got)
	}
	if !strings.Contains(got.Remedy, "AGENTIC_OS_TELEMETRY_DIR") {
		t.Fatalf("remedy = %q; it must name the variable to set", got.Remedy)
	}
}

// Telemetry switched off is a report, not a fault: `doctor` stays usable in a
// health-check loop on a machine that opted out.
func TestDoctorTelemetryOffIsAWarningNotAFailure(t *testing.T) {
	c := doctorCtx("darwin", map[string]string{"AGENTIC_OS_TELEMETRY": "off"})
	got := checkTelemetry(c)
	if got.Status != statusWarn {
		t.Fatalf("telemetry off should warn, not fail: %+v", got)
	}
	if err := doctorExit([]checkResult{got}); err != nil {
		t.Fatalf("a warning must not fail the process: %v", err)
	}
}

func TestDoctorDisplayCheckExplainsAHeadlessMachine(t *testing.T) {
	cases := []struct {
		name       string
		goos       string
		env        map[string]string
		want       checkStatus
		wantRemedy string
	}{
		{"linux with X11", "linux", map[string]string{"DISPLAY": ":0"}, statusOK, ""},
		{"headless linux offers a virtual display", "linux", nil, statusWarn, "headless start"},
		{"macOS assumes a session", "darwin", nil, statusOK, ""},
	}
	for _, tc := range cases {
		got := checkDisplay(doctorCtx(tc.goos, tc.env))
		if got.Status != tc.want {
			t.Errorf("%s: status = %q, want %q", tc.name, got.Status, tc.want)
		}
		if tc.wantRemedy != "" && !strings.Contains(got.Remedy, tc.wantRemedy) {
			t.Errorf("%s: remedy = %q, want it to mention %q", tc.name, got.Remedy, tc.wantRemedy)
		}
	}
}

// Permission gating is a macOS concept; on the other platforms the check must
// say so rather than reporting a failure nobody can act on.
func TestDoctorPermissionsCheckIsAPassOffMacOS(t *testing.T) {
	for _, goos := range []string{"linux", "windows"} {
		got := checkPermissions(doctorCtx(goos, nil))
		if got.Status != statusOK || !strings.Contains(got.Detail, goos) {
			t.Errorf("%s: %+v", goos, got)
		}
	}
}

// The exit code is the contract for scripts: only a hard failure fails the
// process, so a warning-only run stays green.
func TestDoctorExitFailsOnlyOnAFailure(t *testing.T) {
	warnings := []checkResult{
		{Name: "display", Status: statusWarn},
		{Name: "packages", Status: statusOK},
	}
	if err := doctorExit(warnings); err != nil {
		t.Fatalf("warnings must not fail the process: %v", err)
	}

	err := doctorExit(append(warnings, checkResult{Name: "registry", Status: statusFail}))
	exit, ok := err.(*cli.ExitError)
	if !ok || exit.Code != 1 {
		t.Fatalf("a failure must exit 1, got %v", err)
	}
}

func TestDoctorPluginsCheckCountsExternalCommands(t *testing.T) {
	c := doctorCtx("darwin", nil)
	c.Registry.Add(&cli.Command{Group: "demo", Name: "thing", Summary: "Plugin", Binary: "/usr/local/bin/agentic-os-demo-thing"})

	got := checkPlugins(c)
	if got.Status != statusOK || !strings.Contains(got.Detail, "1 external") {
		t.Fatalf("plugins check = %+v", got)
	}
}

func TestDoctorRejectsATypoedFlag(t *testing.T) {
	c := doctorCtx("darwin", nil)
	if err := runDoctor(c, []string{"--jsn"}); err == nil {
		t.Fatal("want an error for --jsn")
	}
}
