package e2e

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMachines(t *testing.T) {
	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("load %s: %v", ConfigPath(), err)
	}
	if config == nil || len(config.Targets) == 0 {
		t.Skipf("no machines configured: copy config.example.json to %s", ConfigPath())
	}

	for _, target := range config.Targets {
		t.Run(target.Name, func(t *testing.T) {
			// Targets are independent machines; running them together turns a
			// three-machine suite from three minutes into one.
			t.Parallel()
			runner := NewRunner(target)

			t.Run("core", func(t *testing.T) { testCore(t, runner) })
			t.Run("filesystem", func(t *testing.T) { testFilesystem(t, runner, target) })
			t.Run("telemetry", func(t *testing.T) { testTelemetry(t, runner) })
			t.Run("display", func(t *testing.T) { testDisplay(t, runner, target) })
		})
	}
}

// mustRun fails the test when a command could not be executed at all — a
// missing binary, an unreachable host, or a step that timed out.
func mustRun(t *testing.T, runner *Runner, args ...string) Result {
	t.Helper()
	result, err := runner.Run(args...)
	if err != nil {
		t.Fatalf("%s: %v\nstdout: %s\nstderr: %s",
			strings.Join(args, " "), err, result.Stdout, result.Stderr)
	}
	return result
}

func wantExit(t *testing.T, result Result, want int, args ...string) {
	t.Helper()
	if result.Exit != want {
		t.Errorf("%s: exit %d, want %d\nstdout: %s\nstderr: %s",
			strings.Join(args, " "), result.Exit, want, result.Stdout, result.Stderr)
	}
}

func wantContains(t *testing.T, result Result, needle string, args ...string) {
	t.Helper()
	if !strings.Contains(result.Combined(), needle) {
		t.Errorf("%s: output does not contain %q\nstdout: %s\nstderr: %s",
			strings.Join(args, " "), needle, result.Stdout, result.Stderr)
	}
}

func testCore(t *testing.T, runner *Runner) {
	result := mustRun(t, runner, "version")
	wantExit(t, result, 0, "version")
	if strings.TrimSpace(result.Stdout) == "" {
		t.Error("version printed nothing")
	}

	result = mustRun(t, runner, "commands", "--check")
	wantExit(t, result, 0, "commands --check")
	wantContains(t, result, "ok:", "commands --check")

	result = mustRun(t, runner, "system", "info")
	wantExit(t, result, 0, "system info")
	wantContains(t, result, "platform", "system info")

	// The command index has to parse: everything an agent sees comes from it.
	result = mustRun(t, runner, "commands", "--json")
	wantExit(t, result, 0, "commands --json")
	var index struct {
		OK       bool `json:"ok"`
		Commands []struct {
			Route     string `json:"route"`
			Available bool   `json:"available"`
		} `json:"commands"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &index); err != nil {
		t.Fatalf("commands --json is not valid JSON: %v", err)
	}
	if !index.OK || len(index.Commands) == 0 {
		t.Fatalf("commands --json reported ok=%v with %d commands", index.OK, len(index.Commands))
	}

	// doctor exits 1 when something is genuinely broken, so accept either code
	// but insist the report itself parses.
	result = mustRun(t, runner, "doctor", "--json")
	if result.Exit != 0 && result.Exit != 1 {
		t.Errorf("doctor --json: exit %d, want 0 or 1\n%s", result.Exit, result.Combined())
	}
	var checks []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &checks); err != nil {
		t.Fatalf("doctor --json is not valid JSON: %v\n%s", err, result.Stdout)
	}
	for _, check := range checks {
		if check.Status == "fail" {
			t.Logf("doctor reports %s failing", check.Name)
		}
	}
}

func testFilesystem(t *testing.T, runner *Runner, target Target) {
	// exec capture is the agent-facing shape: stdout, stderr, and exit as one
	// object. If this drifts, every MCP caller is affected.
	result := mustRun(t, runner, append([]string{"exec", "capture", "--"}, target.EchoArgs("e2e-probe")...)...)
	wantExit(t, result, 0, "exec capture")

	var captured struct {
		Exit   int    `json:"exit"`
		Stdout string `json:"stdout"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &captured); err != nil {
		t.Fatalf("exec capture is not valid JSON: %v\n%s", err, result.Stdout)
	}
	if captured.Exit != 0 || !strings.Contains(captured.Stdout, "e2e-probe") {
		t.Errorf("exec capture returned exit=%d stdout=%q", captured.Exit, captured.Stdout)
	}

	// An absolute path, because a remote step may run in a fresh directory.
	path := target.Path("agentic-os-e2e-probe.txt")
	payload := "e2e-round-trip"
	wantExit(t, mustRun(t, runner, "file", "write", path, payload), 0, "file write")

	result = mustRun(t, runner, "file", "read", path)
	wantExit(t, result, 0, "file read")
	wantContains(t, result, payload, "file read")

	wantExit(t, mustRun(t, runner, "file", "delete", path), 0, "file delete")

	// Deleting it twice must fail, or "delete" is not doing anything.
	if again := mustRun(t, runner, "file", "delete", path); again.Exit == 0 {
		t.Error("file delete succeeded on an already-deleted path")
	}
}

func testTelemetry(t *testing.T, runner *Runner) {
	// The steps above were themselves recorded, so stats must now have work in
	// it. This is the check that telemetry is actually wired to dispatch.
	result := mustRun(t, runner, "obs", "stats")
	wantExit(t, result, 0, "obs stats")
	if strings.Contains(result.Combined(), "no recorded work") {
		t.Error("obs stats found nothing after running commands; telemetry is not recording")
	}

	result = mustRun(t, runner, "obs", "export", "--since=1h")
	wantExit(t, result, 0, "obs export")
	var otlp struct {
		ResourceSpans []struct {
			ScopeSpans []struct {
				Spans []map[string]any `json:"spans"`
			} `json:"scopeSpans"`
		} `json:"resourceSpans"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &otlp); err != nil {
		t.Fatalf("obs export is not valid OTLP JSON: %v", err)
	}
	if len(otlp.ResourceSpans) == 0 || len(otlp.ResourceSpans[0].ScopeSpans) == 0 {
		t.Fatal("obs export produced no resourceSpans")
	}
	if len(otlp.ResourceSpans[0].ScopeSpans[0].Spans) == 0 {
		t.Error("obs export produced no spans")
	}
}

func testDisplay(t *testing.T, runner *Runner, target Target) {
	if !target.HasDisplay {
		// The interesting assertion on a headless machine is the refusal: a
		// GUI command must fail clearly, not obscurely and not silently.
		result := mustRun(t, runner, "window", "list")
		wantExit(t, result, 2, "window list")
		wantContains(t, result, "needs a display", "window list")

		result = mustRun(t, runner, "headless", "status")
		wantContains(t, result, "display", "headless status")
		return
	}

	result := mustRun(t, runner, "display", "list")
	wantExit(t, result, 0, "display list")
	if strings.TrimSpace(result.Stdout) == "" {
		t.Error("display list printed no monitors on a machine with a display")
	}

	wantExit(t, mustRun(t, runner, "window", "list"), 0, "window list")

	// A capture that writes an empty file is the failure that matters, so
	// check the reported geometry rather than only the exit code.
	shot := target.Path("agentic-os-e2e-shot.png")
	result = mustRun(t, runner, "capture", "screenshot", "--monitor=1", "--out="+shot)
	wantExit(t, result, 0, "capture screenshot")
	wantContains(t, result, "x", "capture screenshot")

	result = mustRun(t, runner, "file", "stat", shot)
	wantExit(t, result, 0, "file stat")
	if strings.Contains(result.Stdout, "size      0") {
		t.Error("screenshot wrote an empty file")
	}
	mustRun(t, runner, "file", "delete", shot)
}
