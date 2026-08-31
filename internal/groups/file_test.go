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

	"github.com/muthuishere/agentic-os/internal/cli"
)

// testCtx is a context wired to buffers, so a command's output can be asserted
// on without touching the process's stdio.
func testCtx(stdin string) (*cli.Ctx, *bytes.Buffer, *bytes.Buffer) {
	var out, errOut bytes.Buffer
	return &cli.Ctx{
		Registry: cli.NewRegistry(),
		Stdin:    io.NopCloser(strings.NewReader(stdin)),
		Stdout:   &out,
		Stderr:   &errOut,
		Env:      func(string) string { return "" },
		GOOS:     runtime.GOOS,
		Version:  "test",
	}, &out, &errOut
}

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// `file delete` is the one command here that cannot be undone. Refusing a
// filesystem root and the home directory is checked directly, because the
// guard is the whole safety story: an agent that resolves a variable to the
// empty string and types `file delete $HOME` must be stopped by this function,
// not by luck.
func TestIsProtectedPathRefusesRootsAndHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory on this machine: %v", err)
	}

	protected := []string{home, filepath.Clean(home) + string(filepath.Separator)}
	if runtime.GOOS == "windows" {
		protected = append(protected, filepath.VolumeName(filepath.Clean(home))+`\`)
	} else {
		protected = append(protected, "/")
	}
	for _, path := range protected {
		abs, err := filepath.Abs(path)
		if err != nil {
			t.Fatal(err)
		}
		if !isProtectedPath(abs) {
			t.Errorf("isProtectedPath(%q) = false; deleting it is unrecoverable", abs)
		}
	}

	// A path inside the home directory is ordinary and must stay deletable,
	// or the guard would make the command useless.
	allowed := []string{
		filepath.Join(home, "Downloads", "scratch"),
		filepath.Join(t.TempDir(), "scratch"),
	}
	for _, path := range allowed {
		if isProtectedPath(path) {
			t.Errorf("isProtectedPath(%q) = true; ordinary paths must be deletable", path)
		}
	}
}

// The guard runs on the absolute, cleaned path, so a relative or ~-shaped route
// back to the home directory is refused too.
func TestFileDeleteRefusesTheHomeDirectoryHoweverItIsSpelled(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory on this machine: %v", err)
	}

	for _, arg := range []string{"~", home, filepath.Join(home, "..", filepath.Base(home))} {
		c, _, _ := testCtx("")
		err := runFileDelete(c, []string{arg})
		if err == nil {
			t.Fatalf("`file delete %s` was allowed", arg)
		}
		if !strings.Contains(err.Error(), "refusing to delete") {
			t.Fatalf("`file delete %s` failed for the wrong reason: %v", arg, err)
		}
	}
}

func TestFileDeleteRemovesAFileAndNeedsRecursiveForADirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(dir, "nested")
	if err := os.MkdirAll(filepath.Join(nested, "deeper"), 0o755); err != nil {
		t.Fatal(err)
	}

	c, _, _ := testCtx("")
	if err := runFileDelete(c, []string{file}); err != nil {
		t.Fatalf("delete a file: %v", err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("the file is still there")
	}

	// A directory without --recursive must fail rather than delete a tree the
	// caller did not ask about.
	if err := runFileDelete(c, []string{nested}); err == nil {
		t.Fatal("deleting a directory without --recursive must fail")
	}
	if _, err := os.Stat(nested); err != nil {
		t.Fatal("the refused directory was removed anyway")
	}
	if err := runFileDelete(c, []string{nested, "--recursive"}); err != nil {
		t.Fatalf("delete with --recursive: %v", err)
	}
	if _, err := os.Stat(nested); !os.IsNotExist(err) {
		t.Fatal("--recursive did not remove the directory")
	}
}

// A line range is 1-indexed and inclusive. A backwards range and a non-numeric
// bound both used to slip through and panic on the slice; they must be errors.
func TestParseLineRangeRejectsBackwardsAndNonNumericBounds(t *testing.T) {
	cases := []struct {
		spec string
		ok   bool
		from int
		to   int
	}{
		{"10:40", true, 10, 40},
		{"1:1", true, 1, 1},
		{"40:10", false, 0, 0},  // backwards
		{"a:b", false, 0, 0},    // non-numeric on both sides
		{"1:b", false, 0, 0},    // non-numeric end
		{"a:10", false, 0, 0},   // non-numeric start
		{"0:10", false, 0, 0},   // lines are 1-indexed
		{"-3:10", false, 0, 0},  // no negative start
		{"10", false, 0, 0},     // missing the separator
		{"", false, 0, 0},       // empty
		{"10:", false, 0, 0},    // missing end
		{":10", false, 0, 0},    // missing start
		{"1:2:3", false, 0, 0},  // "2:3" is not a number
		{" 1: 2 ", false, 0, 0}, // no whitespace tolerance, and none is claimed
	}
	for _, tc := range cases {
		from, to, err := parseLineRange(tc.spec)
		if tc.ok {
			if err != nil {
				t.Errorf("parseLineRange(%q) failed: %v", tc.spec, err)
			} else if from != tc.from || to != tc.to {
				t.Errorf("parseLineRange(%q) = %d:%d, want %d:%d", tc.spec, from, to, tc.from, tc.to)
			}
			continue
		}
		if err == nil {
			t.Errorf("parseLineRange(%q) = %d:%d, want an error", tc.spec, from, to)
		}
	}
}

func TestFileReadWholeFileAndLineRange(t *testing.T) {
	path := writeTemp(t, "lines.txt", "one\ntwo\nthree\nfour\n")

	c, out, _ := testCtx("")
	if err := runFileRead(c, []string{path}); err != nil {
		t.Fatalf("read: %v", err)
	}
	if out.String() != "one\ntwo\nthree\nfour\n" {
		t.Fatalf("whole-file read = %q", out.String())
	}

	c, out, _ = testCtx("")
	if err := runFileRead(c, []string{path, "--lines=2:3"}); err != nil {
		t.Fatalf("read range: %v", err)
	}
	if out.String() != "two\nthree\n" {
		t.Fatalf("range read = %q", out.String())
	}
}

// A range that runs off the end of the file is clamped rather than panicking,
// which is what an agent asking for `--lines=1:1000` will do routinely.
func TestFileReadClampsARangePastTheEndOfTheFile(t *testing.T) {
	path := writeTemp(t, "short.txt", "one\ntwo\n")

	c, out, _ := testCtx("")
	if err := runFileRead(c, []string{path, "--lines=1:1000"}); err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.HasPrefix(out.String(), "one\ntwo") {
		t.Fatalf("clamped read = %q", out.String())
	}

	// A start past the end is nothing to print, not an error.
	c, out, _ = testCtx("")
	if err := runFileRead(c, []string{path, "--lines=500:600"}); err != nil {
		t.Fatalf("read past the end: %v", err)
	}
	if out.String() != "" {
		t.Fatalf("want no output, got %q", out.String())
	}
}

func TestFileReadRejectsBadInvocations(t *testing.T) {
	path := writeTemp(t, "lines.txt", "one\n")
	cases := map[string][]string{
		"no path":       {},
		"two paths":     {path, path},
		"typo'd flag":   {path, "--lnes=1:2"},
		"backwards":     {path, "--lines=4:1"},
		"non-numeric":   {path, "--lines=a:b"},
		"missing file":  {filepath.Join(t.TempDir(), "absent.txt")},
		"no line value": {path, "--lines"},
	}
	for name, args := range cases {
		c, _, _ := testCtx("")
		if err := runFileRead(c, args); err == nil {
			t.Errorf("%s: want an error for %v", name, args)
		}
	}
}

func TestFileWriteAndAppendFromArgumentsAndStdin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "notes.txt")

	// Words become one line; the parent directory is created.
	c, _, _ := testCtx("")
	if err := writeFile(c, []string{path, "first", "line"}, false); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := readFile(t, path); got != "first line\n" {
		t.Fatalf("wrote %q", got)
	}

	// Appending keeps what is there.
	c, _, _ = testCtx("")
	if err := writeFile(c, []string{path, "second"}, true); err != nil {
		t.Fatalf("append: %v", err)
	}
	if got := readFile(t, path); got != "first line\nsecond\n" {
		t.Fatalf("appended %q", got)
	}

	// With no words, stdin is the content, byte for byte.
	c, _, _ = testCtx("piped, no trailing newline")
	if err := writeFile(c, []string{path}, false); err != nil {
		t.Fatalf("write from stdin: %v", err)
	}
	if got := readFile(t, path); got != "piped, no trailing newline" {
		t.Fatalf("stdin write = %q", got)
	}

	c, _, _ = testCtx("")
	if err := writeFile(c, nil, false); err == nil {
		t.Fatal("`file write` with no path must fail")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestFileCopyAndMove(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(source, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	c, _, _ := testCtx("")
	destination := filepath.Join(dir, "sub", "b.txt")
	if err := runFileCopy(c, []string{source, destination}); err != nil {
		t.Fatalf("copy: %v", err)
	}
	if readFile(t, destination) != "payload" {
		t.Fatal("copy did not carry the content")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal("copy removed the source")
	}

	moved := filepath.Join(dir, "moved.txt")
	if err := runFileMove(c, []string{source, moved}); err != nil {
		t.Fatalf("move: %v", err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatal("move left the source behind")
	}
	if readFile(t, moved) != "payload" {
		t.Fatal("move did not carry the content")
	}

	// A directory source is refused rather than silently copying nothing.
	if err := runFileCopy(c, []string{dir, filepath.Join(dir, "clone")}); err == nil {
		t.Fatal("copying a directory must fail")
	}
	for _, args := range [][]string{{source}, {}, {source, moved, "extra"}} {
		if err := runFileCopy(c, args); err == nil {
			t.Errorf("`file copy` with %v must fail", args)
		}
	}
}

func TestFileStatJSONShape(t *testing.T) {
	path := writeTemp(t, "stat.txt", "1234567890")

	c, out, _ := testCtx("")
	if err := runFileStat(c, []string{path, "--json"}); err != nil {
		t.Fatalf("stat: %v", err)
	}
	var info fileInfoJSON
	if err := json.Unmarshal(out.Bytes(), &info); err != nil {
		t.Fatalf("stat --json is not JSON: %v (%s)", err, out.String())
	}
	if info.Name != "stat.txt" || info.Size != 10 || info.Dir {
		t.Fatalf("stat = %+v", info)
	}
	if info.Modified == "" || info.Mode == "" {
		t.Fatalf("stat is missing mode/modified: %+v", info)
	}
}

// Dotfiles are hidden unless --all, because a listing an agent reads should be
// the files a person means by "what is in here".
func TestFileListHidesDotfilesUntilAll(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"visible.txt", ".hidden"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	c, out, _ := testCtx("")
	if err := runFileList(c, []string{dir, "--json"}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if strings.Contains(out.String(), ".hidden") {
		t.Fatalf("dotfile listed without --all:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "visible.txt") {
		t.Fatalf("ordinary file missing:\n%s", out.String())
	}

	c, out, _ = testCtx("")
	if err := runFileList(c, []string{dir, "--all", "--json"}); err != nil {
		t.Fatalf("list --all: %v", err)
	}
	if !strings.Contains(out.String(), ".hidden") {
		t.Fatalf("--all did not include the dotfile:\n%s", out.String())
	}

	c, _, _ = testCtx("")
	if err := runFileList(c, []string{dir, dir}); err == nil {
		t.Fatal("`file list` with two paths must fail")
	}
}

func TestExpandHomeOnlyTouchesALeadingTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory on this machine: %v", err)
	}
	if got := expandHome("~"); got != home {
		t.Errorf("expandHome(~) = %q, want %q", got, home)
	}
	if got, want := expandHome("~/notes.txt"), filepath.Join(home, "notes.txt"); got != want {
		t.Errorf("expandHome = %q, want %q", got, want)
	}
	// A tilde anywhere else is part of the name, not a home reference.
	for _, path := range []string{"notes~.txt", "./~/notes", "~notes"} {
		if got := expandHome(path); got != path {
			t.Errorf("expandHome(%q) = %q, want it untouched", path, got)
		}
	}
}

// TestIsProtectedPathRefusesSystemDirectories pins a real gap. The guard's
// final check was `filepath.Dir(cleaned) == cleaned`, true only for the root,
// which the preceding check already caught — so every system directory fell
// through. `file delete /Users --recursive` reached os.RemoveAll and was
// stopped only by the macOS kernel refusing to unlink a mount point; on Linux
// `/etc` would have proceeded. `file delete` is reachable by an agent over
// MCP, so this must hold without a human reading the path first.
func TestIsProtectedPathRefusesSystemDirectories(t *testing.T) {
	protected := []string{
		filepath.FromSlash("/Users"),
		filepath.FromSlash("/etc"),
		filepath.FromSlash("/usr"),
		filepath.FromSlash("/var"),
		filepath.FromSlash("/tmp"),
	}
	for _, path := range protected {
		absolute, err := filepath.Abs(path)
		if err != nil {
			t.Fatalf("abs %q: %v", path, err)
		}
		if !isProtectedPath(absolute) {
			t.Errorf("isProtectedPath(%q) = false; a direct child of the root must be refused", absolute)
		}
	}
}

// TestIsProtectedPathAllowsOrdinaryPaths is the other half: the guard must not
// become so broad that it refuses the deletes people actually want.
func TestIsProtectedPathAllowsOrdinaryPaths(t *testing.T) {
	allowed := []string{
		filepath.Join(t.TempDir(), "scratch"),
		filepath.FromSlash("/tmp/build-output"),
		filepath.FromSlash("/var/log/agentic-os"),
	}
	for _, path := range allowed {
		absolute, err := filepath.Abs(path)
		if err != nil {
			t.Fatalf("abs %q: %v", path, err)
		}
		if isProtectedPath(absolute) {
			t.Errorf("isProtectedPath(%q) = true; an ordinary path must stay deletable", absolute)
		}
	}
}
