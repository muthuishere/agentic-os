// Package obs records what this machine was asked to do.
//
// Every command invocation becomes one span, written as a JSON line. The shape
// is OpenTelemetry's: trace and span ids, nanosecond timestamps, typed
// attributes, a status. Nothing here talks to a collector — the point is that
// the record already exists, in the right shape, on the day someone wants to
// ship it somewhere. `obs export` assembles the lines into real OTLP without
// re-deriving anything.
package obs

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Span is one unit of work, shaped like an OTLP span.
type Span struct {
	TraceID    string      `json:"traceId"`
	SpanID     string      `json:"spanId"`
	ParentID   string      `json:"parentSpanId,omitempty"`
	Name       string      `json:"name"`
	Kind       string      `json:"kind"`
	StartNanos int64       `json:"startTimeUnixNano,string"`
	EndNanos   int64       `json:"endTimeUnixNano,string"`
	Attributes []Attribute `json:"attributes"`
	Status     Status      `json:"status"`
	Resource   []Attribute `json:"resource"`
}

// Attribute is an OTLP key/value pair.
type Attribute struct {
	Key   string       `json:"key"`
	Value AttributeVal `json:"value"`
}

// AttributeVal is OTLP's tagged value union; exactly one field is set.
type AttributeVal struct {
	String *string `json:"stringValue,omitempty"`
	Int    *int64  `json:"intValue,string,omitempty"`
	Bool   *bool   `json:"boolValue,omitempty"`
}

// Status codes follow OTLP: 0 unset, 1 ok, 2 error.
type Status struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

const (
	StatusOK    = 1
	StatusError = 2
)

func StringAttr(key, value string) Attribute {
	return Attribute{Key: key, Value: AttributeVal{String: &value}}
}

func IntAttr(key string, value int64) Attribute {
	return Attribute{Key: key, Value: AttributeVal{Int: &value}}
}

func BoolAttr(key string, value bool) Attribute {
	return Attribute{Key: key, Value: AttributeVal{Bool: &value}}
}

// Recorder appends spans to a daily file.
//
// Writes are best-effort by design: telemetry must never be the reason a
// command fails, so every error here is swallowed rather than returned.
type Recorder struct {
	dir      string
	resource []Attribute
	mu       sync.Mutex
}

// Disabled reports whether telemetry is switched off for this process.
func Disabled(env func(string) string) bool {
	switch strings.ToLower(strings.TrimSpace(env("AGENTIC_OS_TELEMETRY"))) {
	case "off", "0", "false", "no":
		return true
	}
	return false
}

// Dir is where spans are written: $AGENTIC_OS_TELEMETRY_DIR, else the
// platform's state directory. Runtime data, never the repo.
func Dir(env func(string) string) string {
	if dir := strings.TrimSpace(env("AGENTIC_OS_TELEMETRY_DIR")); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "agentic-os", "telemetry")
	}
	if runtime.GOOS == "windows" {
		if local := env("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, "agentic-os", "telemetry")
		}
	}
	return filepath.Join(home, ".local", "state", "agentic-os", "telemetry")
}

// NewRecorder builds a recorder, or nil when telemetry is disabled.
func NewRecorder(env func(string) string, version string) *Recorder {
	if Disabled(env) {
		return nil
	}
	host, _ := os.Hostname()
	return &Recorder{
		dir: Dir(env),
		resource: []Attribute{
			StringAttr("service.name", "agentic-os"),
			StringAttr("service.version", version),
			StringAttr("host.name", host),
			StringAttr("os.type", runtime.GOOS),
			StringAttr("host.arch", runtime.GOARCH),
		},
	}
}

// Record writes one span. A nil Recorder is a no-op, so callers never branch.
func (r *Recorder) Record(span Span) {
	if r == nil {
		return
	}
	span.Resource = r.resource
	if span.TraceID == "" {
		span.TraceID = newID(16)
	}
	if span.SpanID == "" {
		span.SpanID = newID(8)
	}

	line, err := json.Marshal(span)
	if err != nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(r.Path(time.Unix(0, span.StartNanos)),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	file.Write(append(line, '\n'))
}

// Path is the file spans for a given day are appended to. One file per day
// keeps a long-lived machine's telemetry readable and trivially prunable.
func (r *Recorder) Path(when time.Time) string {
	return filepath.Join(r.dir, "spans-"+when.Format("2006-01-02")+".ndjson")
}

// Dir exposes where this recorder writes.
func (r *Recorder) Directory() string {
	if r == nil {
		return ""
	}
	return r.dir
}

func newID(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return strings.Repeat("0", size*2)
	}
	return hex.EncodeToString(buf)
}
