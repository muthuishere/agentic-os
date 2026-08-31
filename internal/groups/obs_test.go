package groups

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/muthuishere/agentic-os/internal/cli"
	"github.com/muthuishere/agentic-os/internal/obs"
)

// obsCtx points the obs commands at a temp telemetry directory, so nothing here
// depends on what this machine has actually been asked to do.
func obsCtx(t *testing.T) (*cli.Ctx, *bytes.Buffer, string) {
	t.Helper()
	dir := t.TempDir()
	var out bytes.Buffer
	c := &cli.Ctx{
		Registry: cli.NewRegistry(),
		Stdin:    io.NopCloser(strings.NewReader("")),
		Stdout:   &out,
		Stderr:   &out,
		Env:      func(key string) string { return map[string]string{"AGENTIC_OS_TELEMETRY_DIR": dir}[key] },
		GOOS:     runtime.GOOS,
		Version:  "test-version",
	}
	return c, &out, dir
}

// seedSpans writes spans the way the recorder does, so the reader is tested
// against the real on-disk format rather than a convenient fixture.
func seedSpans(t *testing.T, dir string, spans ...obs.Span) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	byDay := map[string][]byte{}
	for _, span := range spans {
		line, err := json.Marshal(span)
		if err != nil {
			t.Fatal(err)
		}
		name := "spans-" + time.Unix(0, span.StartNanos).Format("2006-01-02") + ".ndjson"
		byDay[name] = append(byDay[name], append(line, '\n')...)
	}
	for name, data := range byDay {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func span(route, source string, at time.Time, durationMS int64, exit int) obs.Span {
	status := obs.Status{Code: obs.StatusOK}
	if exit != 0 {
		status = obs.Status{Code: obs.StatusError}
	}
	return obs.Span{
		Name:       route,
		Kind:       "SPAN_KIND_SERVER",
		StartNanos: at.UnixNano(),
		EndNanos:   at.Add(time.Duration(durationMS) * time.Millisecond).UnixNano(),
		Status:     status,
		Attributes: []obs.Attribute{
			obs.StringAttr("agentic_os.route", route),
			obs.StringAttr("agentic_os.source", source),
			obs.IntAttr("agentic_os.duration_ms", durationMS),
			obs.IntAttr("agentic_os.exit_code", int64(exit)),
		},
		Resource: []obs.Attribute{
			obs.StringAttr("service.name", "agentic-os"),
			obs.StringAttr("host.name", "test-host"),
		},
	}
}

// `obs export` is the whole reason spans are stored OTLP-shaped: a collector
// must accept the output as-is, which means the resourceSpans / scopeSpans /
// spans envelope has to be populated, not merely present.
func TestObsExportProducesOTLPShapedJSON(t *testing.T) {
	c, out, dir := obsCtx(t)
	now := time.Now()
	seedSpans(t, dir,
		span("file read", "cli", now.Add(-2*time.Minute), 12, 0),
		span("msg send", "mcp", now.Add(-time.Minute), 340, 1),
	)

	if err := runObsExport(c, nil); err != nil {
		t.Fatalf("export: %v", err)
	}

	var payload struct {
		ResourceSpans []struct {
			Resource struct {
				Attributes []obs.Attribute `json:"attributes"`
			} `json:"resource"`
			ScopeSpans []struct {
				Scope struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				} `json:"scope"`
				Spans []obs.Span `json:"spans"`
			} `json:"scopeSpans"`
		} `json:"resourceSpans"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("export is not OTLP-shaped JSON: %v\n%s", err, out.String())
	}
	if len(payload.ResourceSpans) != 1 {
		t.Fatalf("want one resourceSpans entry, got %d", len(payload.ResourceSpans))
	}
	resource := payload.ResourceSpans[0]
	if len(resource.Resource.Attributes) == 0 {
		t.Fatal("resource attributes are empty; a collector cannot attribute these spans")
	}
	if len(resource.ScopeSpans) != 1 {
		t.Fatalf("want one scopeSpans entry, got %d", len(resource.ScopeSpans))
	}
	scope := resource.ScopeSpans[0]
	if scope.Scope.Name != "agentic-os" || scope.Scope.Version != "test-version" {
		t.Fatalf("scope = %+v", scope.Scope)
	}
	if len(scope.Spans) != 2 {
		t.Fatalf("want 2 spans, got %d", len(scope.Spans))
	}
	if scope.Spans[0].Name != "file read" {
		t.Fatalf("spans lost their order: %+v", scope.Spans)
	}
	// The resource is hoisted into the envelope, so repeating it on every span
	// would double the payload for no gain.
	if scope.Spans[0].Resource != nil {
		t.Fatal("the per-span resource should be stripped once it is in the envelope")
	}
}

func TestObsExportWithNoSpansIsStillValidOTLP(t *testing.T) {
	c, out, _ := obsCtx(t)
	if err := runObsExport(c, nil); err != nil {
		t.Fatalf("export: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("empty export is not JSON: %v\n%s", err, out.String())
	}
	if _, ok := payload["resourceSpans"]; !ok {
		t.Fatalf("empty export has no envelope: %s", out.String())
	}
}

// --since is what makes these commands usable on a machine with months of
// history; a span outside the window must not be counted.
func TestObsCommandsHonourTheSinceWindow(t *testing.T) {
	c, out, dir := obsCtx(t)
	now := time.Now()
	seedSpans(t, dir,
		span("file read", "cli", now.Add(-48*time.Hour), 10, 0),
		span("msg send", "cli", now.Add(-5*time.Minute), 20, 0),
	)

	if err := runObsStats(c, []string{"--since=1h", "--json"}); err != nil {
		t.Fatalf("stats: %v", err)
	}
	var stats []routeStat
	if err := json.Unmarshal(out.Bytes(), &stats); err != nil {
		t.Fatalf("stats --json is not JSON: %v\n%s", err, out.String())
	}
	if len(stats) != 1 || stats[0].Route != "msg send" {
		t.Fatalf("--since=1h returned %+v", stats)
	}

	c, _, _ = obsCtx(t)
	if err := runObsStats(c, []string{"--since=yesterday"}); err == nil {
		t.Fatal("want an error for a --since that is not a duration")
	}
}

// The source attribute is what tells a person at a terminal apart from an agent
// calling a tool, so filtering on it has to actually filter.
func TestObsStatsFiltersBySource(t *testing.T) {
	c, out, dir := obsCtx(t)
	now := time.Now()
	seedSpans(t, dir,
		span("file read", "cli", now.Add(-time.Minute), 10, 0),
		span("file read", "mcp", now.Add(-time.Minute), 20, 0),
		span("msg send", "mcp", now.Add(-time.Minute), 30, 1),
	)

	if err := runObsStats(c, []string{"--source=mcp", "--json"}); err != nil {
		t.Fatalf("stats: %v", err)
	}
	var stats []routeStat
	if err := json.Unmarshal(out.Bytes(), &stats); err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, stat := range stats {
		total += stat.Calls
	}
	if total != 2 {
		t.Fatalf("--source=mcp counted %d calls: %+v", total, stats)
	}
}

func TestObsStatsRollsUpCallsErrorsAndLatency(t *testing.T) {
	c, out, dir := obsCtx(t)
	now := time.Now()
	seedSpans(t, dir,
		span("file read", "cli", now.Add(-4*time.Minute), 10, 0),
		span("file read", "cli", now.Add(-3*time.Minute), 30, 0),
		span("file read", "cli", now.Add(-2*time.Minute), 50, 1),
		span("msg send", "cli", now.Add(-time.Minute), 5, 0),
	)

	if err := runObsStats(c, []string{"--json"}); err != nil {
		t.Fatalf("stats: %v", err)
	}
	var stats []routeStat
	if err := json.Unmarshal(out.Bytes(), &stats); err != nil {
		t.Fatal(err)
	}
	// Busiest first is what someone scanning this wants.
	if len(stats) != 2 || stats[0].Route != "file read" {
		t.Fatalf("stats = %+v", stats)
	}
	read := stats[0]
	if read.Calls != 3 || read.Errors != 1 {
		t.Fatalf("calls/errors = %d/%d, want 3/1", read.Calls, read.Errors)
	}
	if read.MaxMillis != 50 {
		t.Fatalf("max = %dms, want 50", read.MaxMillis)
	}
	if read.ErrorRate < 0.33 || read.ErrorRate > 0.34 {
		t.Fatalf("error rate = %v, want ~1/3", read.ErrorRate)
	}
}

// Percentiles are nearest-rank, and the edges are where an off-by-one shows up:
// p95 of a single sample is that sample, never a zero.
func TestPercentileNearestRank(t *testing.T) {
	sorted := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	cases := []struct {
		p    int
		want int64
	}{{50, 50}, {95, 100}, {100, 100}, {0, 10}}
	for _, tc := range cases {
		if got := percentile(sorted, tc.p); got != tc.want {
			t.Errorf("percentile(p%d) = %d, want %d", tc.p, got, tc.want)
		}
	}
	if got := percentile([]int64{42}, 95); got != 42 {
		t.Errorf("p95 of one sample = %d, want 42", got)
	}
	if got := percentile(nil, 50); got != 0 {
		t.Errorf("percentile of nothing = %d, want 0", got)
	}
}

func TestObsTailPrintsTheMostRecentSpansNewestLast(t *testing.T) {
	c, out, dir := obsCtx(t)
	now := time.Now()
	seedSpans(t, dir,
		span("first", "cli", now.Add(-3*time.Minute), 10, 0),
		span("second", "cli", now.Add(-2*time.Minute), 10, 0),
		span("third", "mcp", now.Add(-time.Minute), 10, 1),
	)

	if err := runObsTail(c, []string{"--limit=2"}); err != nil {
		t.Fatalf("tail: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("--limit=2 printed %d lines:\n%s", len(lines), out.String())
	}
	if !strings.Contains(lines[0], "second") || !strings.Contains(lines[1], "third") {
		t.Fatalf("tail is not newest-last:\n%s", out.String())
	}
	// A failure has to be visible at a glance.
	if !strings.Contains(lines[1], "err") {
		t.Fatalf("a failed span is not marked:\n%s", out.String())
	}
}

// A telemetry directory that does not exist yet is the state of a fresh
// machine, and must read as "nothing recorded" rather than as an error.
func TestObsCommandsTreatAMissingDirectoryAsEmpty(t *testing.T) {
	c, out, dir := obsCtx(t)
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := runObsStats(c, nil); err != nil {
		t.Fatalf("stats on a fresh machine: %v", err)
	}
	if !strings.Contains(out.String(), "no recorded work") {
		t.Fatalf("output = %q", out.String())
	}
}

// A truncated or half-written line (a machine that lost power mid-append) must
// not take the rest of the day's telemetry with it.
func TestLoadSpansSkipsUnparseableLines(t *testing.T) {
	c, _, dir := obsCtx(t)
	now := time.Now()
	seedSpans(t, dir, span("file read", "cli", now, 10, 0))

	name := filepath.Join(dir, "spans-"+now.Format("2006-01-02")+".ndjson")
	file, err := os.OpenFile(name, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("{\"traceId\":\"half-written\n"); err != nil {
		t.Fatal(err)
	}
	file.Close()

	spans, err := loadSpans(c, 0, "")
	if err != nil {
		t.Fatalf("loadSpans: %v", err)
	}
	if len(spans) != 1 || spans[0].Name != "file read" {
		t.Fatalf("spans = %+v", spans)
	}
}

// Files are date-stamped, so lexical order is chronological — the property the
// reader relies on instead of sorting by timestamp.
func TestLoadSpansReadsDaysInChronologicalOrder(t *testing.T) {
	c, _, dir := obsCtx(t)
	// Deliberately seeded newest-first; the reader must still return them in
	// day order.
	seedSpans(t, dir,
		span("newer", "cli", time.Now().Add(-24*time.Hour), 10, 0),
		span("older", "cli", time.Now().Add(-48*time.Hour), 10, 0),
	)
	spans, err := loadSpans(c, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(spans) != 2 || spans[0].Name != "older" || spans[1].Name != "newer" {
		t.Fatalf("spans out of order: %+v", spans)
	}
}
