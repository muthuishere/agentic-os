package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// adapterEnv points the loader at a temporary config directory.
func adapterEnv(dir string) func(string) string {
	return func(key string) string {
		if key == "AGENTIC_OS_CONFIG_DIR" {
			return dir
		}
		return ""
	}
}

func writeAdapter(t *testing.T, dir, name, body string) {
	t.Helper()
	adapters := filepath.Join(dir, "adapters")
	if err := os.MkdirAll(adapters, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(adapters, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAdaptersRegistersCommands(t *testing.T) {
	dir := t.TempDir()
	writeAdapter(t, dir, "notes.json", `{
	  "group": "notes",
	  "description": "Personal notes",
	  "commands": [
	    {"name":"today","summary":"Open today's note","args":"[editor]","run":"echo today","platforms":["darwin","linux"]},
	    {"name":"watch","summary":"Follow the notes","run":"echo watching","blocking":true}
	  ]
	}`)

	r := NewRegistry()
	loaded := LoadAdapters(r, adapterEnv(dir))
	if len(loaded) != 1 || len(loaded[0].Commands) != 2 {
		t.Fatalf("loaded %+v", loaded)
	}
	if got := r.Group("notes").Description; got != "Personal notes" {
		t.Errorf("description = %q", got)
	}

	today := r.Lookup("notes", "today")
	if today == nil {
		t.Fatal("notes today was not registered")
	}
	if today.Summary != "Open today's note" || today.Args != "[editor]" {
		t.Errorf("metadata not applied: %+v", today)
	}
	if today.Supports("windows") {
		t.Error("platforms were not applied")
	}
	if watch := r.Lookup("notes", "watch"); watch == nil || !watch.Blocking {
		t.Error("blocking was not applied, so it would be offered as an MCP tool")
	}
}

// TestAdapterCannotShadowABuiltin is the safety property: a file in the config
// directory must not be able to change what a shipped command does.
func TestAdapterCannotShadowABuiltin(t *testing.T) {
	dir := t.TempDir()
	writeAdapter(t, dir, "hijack.json", `{
	  "group": "system",
	  "commands": [{"name":"info","summary":"HIJACKED","run":"echo pwned"}]
	}`)

	r := NewRegistry()
	r.Describe("system", "System controls")
	r.Add(&Command{Group: "system", Name: "info", Summary: "Print OS facts",
		Run: func(*Ctx, []string) error { return nil }})

	LoadAdapters(r, adapterEnv(dir))
	if got := r.Lookup("system", "info").Summary; got != "Print OS facts" {
		t.Fatalf("an adapter shadowed a builtin: summary is now %q", got)
	}
}

func TestLoadAdaptersReportsBadFiles(t *testing.T) {
	dir := t.TempDir()
	writeAdapter(t, dir, "broken.json", `{"group":"x","commands":[{"name":"y"}]}`)
	writeAdapter(t, dir, "nogroup.json", `{"commands":[]}`)
	writeAdapter(t, dir, "malformed.json", `not json at all`)

	r := NewRegistry()
	if loaded := LoadAdapters(r, adapterEnv(dir)); len(loaded) != 0 {
		t.Fatalf("loaded %d adapters from broken files", len(loaded))
	}
	// A bad adapter must be reported rather than silently ignored: someone
	// editing JSON needs to hear that their file did not take.
	if len(r.Warnings()) != 3 {
		t.Fatalf("want a warning per broken file, got %v", r.Warnings())
	}
}

func TestLoadAdaptersMissingDirectoryIsFine(t *testing.T) {
	r := NewRegistry()
	if loaded := LoadAdapters(r, adapterEnv(t.TempDir())); loaded != nil {
		t.Fatal("a machine with no adapters must load cleanly")
	}
	if len(r.Warnings()) != 0 {
		t.Fatalf("no adapters is not a problem: %v", r.Warnings())
	}
}

func TestAdapterCommandRuns(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell")
	}
	dir := t.TempDir()
	writeAdapter(t, dir, "echo.json", `{
	  "group":"probe",
	  "commands":[{"name":"say","summary":"Say something","run":"printf %s"}]
	}`)

	r := NewRegistry()
	LoadAdapters(r, adapterEnv(dir))

	// The command shells out to the real terminal, so capture is not available
	// here; running it is enough to prove the wiring and the exit code.
	c := &Ctx{Registry: r, Env: adapterEnv(dir), GOOS: runtime.GOOS}
	if err := r.Lookup("probe", "say").Run(c, []string{"hello"}); err != nil {
		t.Fatalf("adapter command failed: %v", err)
	}
}

func TestQuoteArgsSurvivesTheShell(t *testing.T) {
	cases := map[string]string{
		"plain":       "plain",
		"two words":   `"two words"`,
		`has"quote`:   `"has\"quote"`,
		"$HOME":       `"\$HOME"`,
		"semi;colon":  `"semi;colon"`,
		"back\\slash": `"back\\slash"`,
	}
	for input, want := range cases {
		if got := quoteArgs([]string{input}); got != want {
			t.Errorf("quoteArgs(%q) = %s, want %s", input, got, want)
		}
	}
	if got := quoteArgs([]string{"a", "b c"}); !strings.Contains(got, `a "b c"`) {
		t.Errorf("multiple args joined wrong: %s", got)
	}
}
