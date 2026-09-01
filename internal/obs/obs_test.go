package obs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func envOf(pairs map[string]string) func(string) string {
	return func(key string) string { return pairs[key] }
}

func TestDisabledRecognisesEveryOffSpelling(t *testing.T) {
	cases := map[string]bool{
		"off": true, "0": true, "false": true, "no": true,
		"OFF": true, " Off ": true,
		"": false, "on": false, "1": false, "true": false, "yes": false,
	}
	for value, want := range cases {
		if got := Disabled(envOf(map[string]string{"AOS_TELEMETRY": value})); got != want {
			t.Errorf("Disabled(%q) = %v, want %v", value, got, want)
		}
	}
}

func TestDirHonoursTheExplicitDirectory(t *testing.T) {
	dir := t.TempDir()
	if got := Dir(envOf(map[string]string{"AOS_TELEMETRY_DIR": dir})); got != dir {
		t.Fatalf("Dir = %q, want %q", got, dir)
	}
	// A blank value is not a directory; it must fall through to the default.
	if got := Dir(envOf(map[string]string{"AOS_TELEMETRY_DIR": "   "})); got == "" || strings.TrimSpace(got) != got {
		t.Fatalf("Dir with a blank override = %q", got)
	}
}

func TestNewRecorderIsNilWhenTelemetryIsOff(t *testing.T) {
	r := NewRecorder(envOf(map[string]string{"AOS_TELEMETRY": "off"}), "test")
	if r != nil {
		t.Fatal("telemetry off must yield a nil recorder")
	}
	// A nil recorder has to stay usable: callers deliberately do not branch on it.
	r.Record(Span{Name: "anything"})
	if r.Directory() != "" {
		t.Fatal("a nil recorder has no directory")
	}
}

func recorderIn(t *testing.T, dir string) *Recorder {
	t.Helper()
	r := NewRecorder(envOf(map[string]string{"AOS_TELEMETRY_DIR": dir}), "test-version")
	if r == nil {
		t.Fatal("recorder should be enabled")
	}
	return r
}

func readSpans(t *testing.T, dir string) []Span {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read telemetry dir: %v", err)
	}
	var spans []Span
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "spans-") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if line == "" {
				continue
			}
			var span Span
			if err := json.Unmarshal([]byte(line), &span); err != nil {
				t.Fatalf("span line is not JSON: %v (%s)", err, line)
			}
			spans = append(spans, span)
		}
	}
	return spans
}

func TestRecordWritesOneJSONLinePerSpanAndCreatesTheDirectory(t *testing.T) {
	// Point at a directory that does not exist yet: recording must create it,
	// because the first command on a fresh machine is the common case.
	dir := filepath.Join(t.TempDir(), "nested", "telemetry")
	r := recorderIn(t, dir)

	now := time.Now()
	r.Record(Span{Name: "file read", StartNanos: now.UnixNano(), EndNanos: now.UnixNano()})
	r.Record(Span{Name: "file write", StartNanos: now.UnixNano(), EndNanos: now.UnixNano()})

	spans := readSpans(t, dir)
	if len(spans) != 2 {
		t.Fatalf("want 2 spans, got %d", len(spans))
	}
	if spans[0].Name != "file read" || spans[1].Name != "file write" {
		t.Fatalf("spans out of order: %+v", spans)
	}
}

// Trace and span ids identify the record; a span written without them is
// unjoinable to anything and would be dropped by a collector.
func TestRecordFillsInTraceAndSpanIDs(t *testing.T) {
	dir := t.TempDir()
	r := recorderIn(t, dir)
	r.Record(Span{Name: "demo", StartNanos: time.Now().UnixNano()})

	spans := readSpans(t, dir)
	if len(spans) != 1 {
		t.Fatalf("want 1 span, got %d", len(spans))
	}
	if len(spans[0].TraceID) != 32 {
		t.Fatalf("traceId = %q, want 16 bytes of hex", spans[0].TraceID)
	}
	if len(spans[0].SpanID) != 16 {
		t.Fatalf("spanId = %q, want 8 bytes of hex", spans[0].SpanID)
	}
}

func TestRecordKeepsACallersOwnIDs(t *testing.T) {
	dir := t.TempDir()
	r := recorderIn(t, dir)
	r.Record(Span{
		Name:       "demo",
		TraceID:    "0123456789abcdef0123456789abcdef",
		SpanID:     "0123456789abcdef",
		StartNanos: time.Now().UnixNano(),
	})
	spans := readSpans(t, dir)
	if spans[0].TraceID != "0123456789abcdef0123456789abcdef" || spans[0].SpanID != "0123456789abcdef" {
		t.Fatalf("supplied ids were overwritten: %+v", spans[0])
	}
}

// The resource is what tells a collector which machine and which build produced
// the span; it is stamped by the recorder so no caller can forget it.
func TestRecordStampsTheResource(t *testing.T) {
	dir := t.TempDir()
	r := recorderIn(t, dir)
	r.Record(Span{Name: "demo", StartNanos: time.Now().UnixNano()})

	attrs := map[string]string{}
	for _, attr := range readSpans(t, dir)[0].Resource {
		if attr.Value.String != nil {
			attrs[attr.Key] = *attr.Value.String
		}
	}
	if attrs["service.name"] != "aos" {
		t.Fatalf("service.name = %q", attrs["service.name"])
	}
	if attrs["service.version"] != "test-version" {
		t.Fatalf("service.version = %q", attrs["service.version"])
	}
	for _, key := range []string{"host.name", "os.type", "host.arch"} {
		if _, ok := attrs[key]; !ok {
			t.Errorf("resource is missing %q: %v", key, attrs)
		}
	}
}

// Spans are grouped one file per day so a long-lived machine's telemetry stays
// readable and prunable; the file is named for the span's own start time, not
// for "now", so a backdated span does not land in today's file.
func TestPathIsOneFilePerDayNamedForTheSpan(t *testing.T) {
	dir := t.TempDir()
	r := recorderIn(t, dir)

	day := time.Date(2026, 5, 17, 9, 30, 0, 0, time.UTC)
	if got, want := filepath.Base(r.Path(day)), "spans-2026-05-17.ndjson"; got != want {
		t.Fatalf("Path = %q, want %q", got, want)
	}

	r.Record(Span{Name: "backdated", StartNanos: day.UnixNano(), EndNanos: day.UnixNano()})
	if _, err := os.Stat(r.Path(day)); err != nil {
		t.Fatalf("span was not written to its own day's file: %v", err)
	}
}

// Telemetry must never be the reason a command fails: an unwritable directory
// is swallowed, not returned or panicked.
func TestRecordSwallowsAnUnwritableDirectory(t *testing.T) {
	// A regular file where the directory should be: MkdirAll fails on every OS.
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := recorderIn(t, filepath.Join(blocker, "telemetry"))
	r.Record(Span{Name: "demo", StartNanos: time.Now().UnixNano()})
}

// Two commands can record at once (the MCP server serves concurrently), and a
// torn line would make the whole day's file unparseable.
func TestConcurrentRecordsProduceWholeLines(t *testing.T) {
	dir := t.TempDir()
	r := recorderIn(t, dir)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Record(Span{
				Name:       "concurrent",
				StartNanos: time.Now().UnixNano(),
				Attributes: []Attribute{StringAttr("agentic_os.route", "concurrent")},
			})
		}()
	}
	wg.Wait()

	if got := len(readSpans(t, dir)); got != 20 {
		t.Fatalf("want 20 parseable spans, got %d", got)
	}
}

func TestAttributeValuesEncodeAsOTLPUnions(t *testing.T) {
	line, err := json.Marshal([]Attribute{
		StringAttr("s", "text"),
		IntAttr("i", 42),
		BoolAttr("b", true),
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(line)
	// OTLP requires exactly one typed field per value, and intValue is a string.
	for _, want := range []string{`"stringValue":"text"`, `"intValue":"42"`, `"boolValue":true`} {
		if !strings.Contains(got, want) {
			t.Errorf("encoded attributes %s are missing %s", got, want)
		}
	}
}

func TestSpanTimestampsEncodeAsOTLPStrings(t *testing.T) {
	line, err := json.Marshal(Span{Name: "demo", StartNanos: 1700000000000000000, EndNanos: 1700000000100000000})
	if err != nil {
		t.Fatal(err)
	}
	// Nanosecond timestamps overflow a JSON number in most consumers, so OTLP
	// carries them as strings.
	if !strings.Contains(string(line), `"startTimeUnixNano":"1700000000000000000"`) {
		t.Fatalf("timestamps are not OTLP strings: %s", line)
	}
}
