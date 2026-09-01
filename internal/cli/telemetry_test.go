package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muthuishere/aos/internal/obs"
)

// recordingCtx builds a context whose telemetry lands in a temp directory, and
// returns a reader for the spans it wrote.
func recordingCtx(t *testing.T, r *Registry, goos string) (*Ctx, func() []obs.Span, func() string) {
	t.Helper()
	dir := t.TempDir()
	env := envOf(map[string]string{"AOS_TELEMETRY_DIR": dir})

	var out, errOut bytes.Buffer
	c := &Ctx{
		Registry: r,
		Stdout:   &out,
		Stderr:   &errOut,
		Env:      env,
		GOOS:     goos,
		Version:  "test",
		Source:   "cli",
		Recorder: obs.NewRecorder(env, "test"),
	}

	raw := func() string {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return ""
		}
		var b strings.Builder
		for _, entry := range entries {
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			b.Write(data)
		}
		return b.String()
	}
	spans := func() []obs.Span {
		var out []obs.Span
		for _, line := range strings.Split(strings.TrimSpace(raw()), "\n") {
			if line == "" {
				continue
			}
			var span obs.Span
			if err := json.Unmarshal([]byte(line), &span); err != nil {
				t.Fatalf("span line is not JSON: %v (%s)", err, line)
			}
			out = append(out, span)
		}
		return out
	}
	return c, spans, raw
}

func attrOf(t *testing.T, span obs.Span, key string) obs.AttributeVal {
	t.Helper()
	for _, attr := range span.Attributes {
		if attr.Key == key {
			return attr.Value
		}
	}
	t.Fatalf("span has no attribute %q: %+v", key, span.Attributes)
	return obs.AttributeVal{}
}

func telemetryRegistry() *Registry {
	r := NewRegistry()
	r.Describe("demo", "Demo")
	r.Add(
		&Command{Group: "demo", Name: "ok", Summary: "Succeeds", Run: func(*Ctx, []string) error { return nil }},
		&Command{Group: "demo", Name: "fail", Summary: "Fails", Run: func(*Ctx, []string) error {
			return fmt.Errorf("boom")
		}},
	)
	return r
}

func TestEveryInvocationRecordsASpan(t *testing.T) {
	c, spans, _ := recordingCtx(t, telemetryRegistry(), "darwin")

	if code := Run(c, []string{"demo", "ok"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	got := spans()
	if len(got) != 1 {
		t.Fatalf("want 1 span, got %d", len(got))
	}
	span := got[0]
	if span.Name != "demo ok" {
		t.Fatalf("span name = %q, want the route", span.Name)
	}
	if span.Status.Code != obs.StatusOK {
		t.Fatalf("status = %+v, want ok", span.Status)
	}
	if got := attrOf(t, span, "agentic_os.route"); got.String == nil || *got.String != "demo ok" {
		t.Fatalf("route attribute = %+v", got)
	}
	if got := attrOf(t, span, "agentic_os.source"); got.String == nil || *got.String != "cli" {
		t.Fatalf("source attribute = %+v", got)
	}
	if got := attrOf(t, span, "agentic_os.ok"); got.Bool == nil || !*got.Bool {
		t.Fatalf("ok attribute = %+v", got)
	}
	if span.StartNanos == 0 || span.EndNanos < span.StartNanos {
		t.Fatalf("timestamps are wrong: start=%d end=%d", span.StartNanos, span.EndNanos)
	}
}

// A failure is the invocation most worth having a record of, and the one an
// early implementation dropped by returning before the span was written.
func TestAFailedInvocationIsStillRecorded(t *testing.T) {
	c, spans, _ := recordingCtx(t, telemetryRegistry(), "darwin")

	if code := Run(c, []string{"demo", "fail"}); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	got := spans()
	if len(got) != 1 {
		t.Fatalf("want 1 span for the failure, got %d", len(got))
	}
	if got[0].Status.Code != obs.StatusError {
		t.Fatalf("status = %+v, want error", got[0].Status)
	}
	if exit := attrOf(t, got[0], "agentic_os.exit_code"); exit.Int == nil || *exit.Int != 1 {
		t.Fatalf("exit_code attribute = %+v", exit)
	}
	if ok := attrOf(t, got[0], "agentic_os.ok"); ok.Bool == nil || *ok.Bool {
		t.Fatalf("ok attribute = %+v, want false", ok)
	}
}

// An unresolved route is recorded under the word that was typed, so a typo an
// agent keeps repeating shows up in `obs stats` instead of vanishing.
func TestAnUnknownCommandIsRecordedUnderTheWordTyped(t *testing.T) {
	c, spans, _ := recordingCtx(t, telemetryRegistry(), "darwin")

	if code := Run(c, []string{"demoo", "ok", "extra"}); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	got := spans()
	if len(got) != 1 || got[0].Name != "demoo" {
		t.Fatalf("spans = %+v", got)
	}
	if count := attrOf(t, got[0], "agentic_os.arg_count"); count.Int == nil || *count.Int != 2 {
		t.Fatalf("arg_count = %+v, want 2", count)
	}
}

func TestAnEmptyInvocationIsRecordedAsHelp(t *testing.T) {
	c, spans, _ := recordingCtx(t, telemetryRegistry(), "darwin")

	if code := Run(c, nil); code != 0 {
		t.Fatalf("exit %d", code)
	}
	got := spans()
	if len(got) != 1 || got[0].Name != "help" {
		t.Fatalf("spans = %+v", got)
	}
}

// PRIVACY. Command arguments routinely carry file paths, message bodies, and
// tokens a person typed. Telemetry counts them and never copies them; an
// earlier version put the joined argv in the span name, which meant
// `msg send --channel=ops "<anything>"` was written to disk verbatim.
func TestTelemetryRecordsTheArgumentCountAndNeverTheArguments(t *testing.T) {
	c, spans, raw := recordingCtx(t, telemetryRegistry(), "darwin")

	const secret = "hunter2-do-not-log-me"
	if code := Run(c, []string{"demo", "ok", secret, "--token=" + secret}); code != 0 {
		t.Fatalf("exit %d", code)
	}

	got := spans()
	if len(got) != 1 {
		t.Fatalf("want 1 span, got %d", len(got))
	}
	if count := attrOf(t, got[0], "agentic_os.arg_count"); count.Int == nil || *count.Int != 2 {
		t.Fatalf("arg_count = %+v, want 2", count)
	}
	if written := raw(); strings.Contains(written, secret) {
		t.Fatalf("an argument's contents reached the telemetry file:\n%s", written)
	}
}

// The MCP server reuses the same dispatch, and telling a person at a terminal
// apart from an agent calling a tool is the reason the attribute exists.
func TestAnMCPInvocationIsRecordedAsMCP(t *testing.T) {
	c, spans, _ := recordingCtx(t, telemetryRegistry(), "darwin")

	if got := Invoke(c, []string{"demo", "ok"}, ""); got.Exit != 0 {
		t.Fatalf("%+v", got)
	}
	got := spans()
	if len(got) != 1 {
		t.Fatalf("want 1 span, got %d", len(got))
	}
	if source := attrOf(t, got[0], "agentic_os.source"); source.String == nil || *source.String != "mcp" {
		t.Fatalf("source = %+v, want mcp", source)
	}
}

// A command refused for want of a display never runs, but the refusal is still
// work this machine was asked to do.
func TestARefusedGUICommandIsRecorded(t *testing.T) {
	r := NewRegistry()
	r.Describe("window", "Windows")
	r.Add(&Command{Group: "window", Name: "list", Summary: "List", NeedsDisplay: true,
		Run: func(*Ctx, []string) error { return nil }})

	c, spans, _ := recordingCtx(t, r, "linux") // no DISPLAY in the test env
	if code := Run(c, []string{"window", "list"}); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	got := spans()
	if len(got) != 1 || got[0].Status.Code != obs.StatusError {
		t.Fatalf("spans = %+v", got)
	}
	if exit := attrOf(t, got[0], "agentic_os.exit_code"); exit.Int == nil || *exit.Int != 2 {
		t.Fatalf("exit_code = %+v, want 2", exit)
	}
}

// Telemetry is opt-out, and opting out has to mean nothing is written at all.
func TestTelemetryOffWritesNothing(t *testing.T) {
	dir := t.TempDir()
	env := envOf(map[string]string{
		"AOS_TELEMETRY_DIR": dir,
		"AOS_TELEMETRY":     "off",
	})
	var out bytes.Buffer
	c := &Ctx{
		Registry: telemetryRegistry(), Stdout: &out, Stderr: &out,
		Env: env, GOOS: "darwin", Version: "test",
		Recorder: obs.NewRecorder(env, "test"),
	}
	if code := Run(c, []string{"demo", "ok"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("telemetry is off but %d files were written", len(entries))
	}
}
