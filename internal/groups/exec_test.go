package groups

import (
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/muthuishere/aos/internal/cli"
)

// captureJSON runs `exec capture` and decodes what it printed, failing the test
// if the output is not parseable — which is the property the command exists for.
func captureJSON(t *testing.T, args []string) (execResult, error) {
	t.Helper()
	c, out, _ := testCtx("")
	err := runExecCapture(c, args)

	var result execResult
	if decodeErr := json.Unmarshal(out.Bytes(), &result); decodeErr != nil {
		t.Fatalf("`exec capture %v` did not print parseable JSON: %v\n%s", args, decodeErr, out.String())
	}
	return result, err
}

// goHelper returns an argv that runs the Go toolchain, which is the one program
// guaranteed to be present wherever these tests run — no POSIX shell assumed.
func goHelper(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("go")
	if err != nil {
		t.Skip("needs the go toolchain on PATH to run a portable child process")
	}
	return path
}

// The point of `exec capture` is that an agent gets stdout, stderr and the exit
// code as one object instead of interleaving streams itself.
func TestExecCaptureReturnsStdoutStderrAndExitCode(t *testing.T) {
	goBin := goHelper(t)

	result, err := captureJSON(t, []string{goBin, "env", "GOOS"})
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if result.Exit != 0 {
		t.Fatalf("exit = %d, want 0", result.Exit)
	}
	if strings.TrimSpace(result.Stdout) != runtime.GOOS {
		t.Fatalf("stdout = %q, want %q", result.Stdout, runtime.GOOS)
	}
	if len(result.Command) != 3 || result.Command[1] != "env" {
		t.Fatalf("command = %v; the echoed argv is what makes the record self-describing", result.Command)
	}
	if result.Duration < 0 {
		t.Fatalf("duration_ms = %d", result.Duration)
	}
}

// REGRESSION. A failing child is the case an agent most needs to read, and it
// is the case that is easiest to get wrong: the command exits non-zero itself,
// so anything that short-circuits before printing leaves the caller with an
// empty body and no way to see why. Non-zero MUST still produce valid JSON
// carrying the child's exit code and its stderr.
func TestExecCaptureStillPrintsJSONWhenTheChildFails(t *testing.T) {
	goBin := goHelper(t)

	// `go env` with an unknown flag: non-zero exit, output on stderr.
	result, err := captureJSON(t, []string{goBin, "env", "--no-such-flag"})
	if err == nil {
		t.Fatal("a failing child must mirror its status in our exit code")
	}
	var exitErr *cli.ExitError
	if !asExitError(err, &exitErr) || exitErr.Code == 0 {
		t.Fatalf("err = %v, want a non-zero ExitError", err)
	}
	// The JSON is the payload, so the error must not add a second message line.
	if exitErr.Message != "" {
		t.Fatalf("ExitError carries a message %q; the JSON is the payload", exitErr.Message)
	}
	if result.Exit == 0 {
		t.Fatalf("exit = 0 for a failing command: %+v", result)
	}
	if strings.TrimSpace(result.Stderr) == "" {
		t.Fatalf("stderr was dropped: %+v", result)
	}
}

// A command that never started has no exit code at all. Reporting exit 0 there
// would tell an agent a typo'd program name succeeded; `-1` plus the reason is
// the only honest answer, and it must not panic on the nil ProcessState.
func TestExecCaptureReportsACommandThatNeverStarted(t *testing.T) {
	result, err := captureJSON(t, []string{"aos-no-such-program"})
	if err == nil {
		t.Fatal("want a non-zero exit for a program that does not exist")
	}
	if result.Exit != -1 {
		t.Fatalf("exit = %d, want -1 for a command that never ran", result.Exit)
	}
	if result.Error == "" {
		t.Fatalf("no reason given for a command that never ran: %+v", result)
	}
}

// `--` is how a caller stops aos from claiming flags meant for the
// child: `exec capture -- ls -la` must run `ls -la`, not complain about `-la`.
func TestExecStripsTheLeadingSeparator(t *testing.T) {
	if got := stripSeparator([]string{"--", "ls", "-la"}); len(got) != 2 || got[0] != "ls" {
		t.Fatalf("stripSeparator = %v", got)
	}
	// Only a LEADING separator is ours; one further along belongs to the child.
	if got := stripSeparator([]string{"git", "--", "path"}); len(got) != 3 {
		t.Fatalf("stripSeparator ate the child's separator: %v", got)
	}
	if got := stripSeparator(nil); got != nil {
		t.Fatalf("stripSeparator(nil) = %v", got)
	}
}

func TestExecCommandsNeedACommand(t *testing.T) {
	for name, run := range map[string]func(*cli.Ctx, []string) error{
		"exec run":     runExecRun,
		"exec capture": runExecCapture,
		"exec shell":   runExecShell,
	} {
		c, _, _ := testCtx("")
		if err := run(c, []string{"--"}); err == nil {
			t.Errorf("`%s` with nothing after -- must fail", name)
		}
	}
}

// The child reads the caller's stdin, so a piped payload reaches it.
func TestExecCaptureFeedsStdinToTheChild(t *testing.T) {
	goBin := goHelper(t)

	dir := t.TempDir()
	program := dir + string(os.PathSeparator) + "main.go"
	source := "package main\n\nimport (\n\t\"io\"\n\t\"os\"\n)\n\nfunc main() { io.Copy(os.Stdout, os.Stdin) }\n"
	if err := os.WriteFile(program, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	c, out, _ := testCtx("hello from stdin")
	if err := runExecCapture(c, []string{goBin, "run", program}); err != nil {
		t.Fatalf("capture: %v\n%s", err, out.String())
	}
	var result execResult
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out.String())
	}
	if result.Stdout != "hello from stdin" {
		t.Fatalf("stdout = %q; stdin did not reach the child", result.Stdout)
	}
}

// asExitError is errors.As, spelled out to keep the assertion above readable.
func asExitError(err error, target **cli.ExitError) bool {
	exit, ok := err.(*cli.ExitError)
	if ok {
		*target = exit
	}
	return ok
}
