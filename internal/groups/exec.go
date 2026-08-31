package groups

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/muthuishere/agentic-os/internal/cli"
	"github.com/muthuishere/agentic-os/internal/sys"
)

// execResult is the machine-readable shape `exec capture` prints. It is the
// point of the command: an agent gets stdout, stderr, and the exit code as one
// parseable object instead of having to interleave streams itself.
type execResult struct {
	Command  []string `json:"command"`
	Exit     int      `json:"exit"`
	Stdout   string   `json:"stdout"`
	Stderr   string   `json:"stderr"`
	Duration int64    `json:"duration_ms"`
	Error    string   `json:"error,omitempty"`
}

func init() {
	register(func(r *cli.Registry) {
		r.Describe("exec", "Run commands, with output streamed or captured")
		r.Add(
			&cli.Command{
				Group: "exec", Name: "run",
				Summary: "Run a command with its output streamed to this terminal",
				Args:    "<command> [args...]",
				Examples: []string{
					"aos exec run git status",
				},
				Run: runExecRun,
			},
			&cli.Command{
				Group: "exec", Name: "capture",
				Summary: "Run a command and print stdout, stderr, and exit code as JSON",
				Args:    "<command> [args...]",
				Examples: []string{
					"aos exec capture git status",
					"aos exec capture -- ls -la",
				},
				Run: runExecCapture,
			},
			&cli.Command{
				Group: "exec", Name: "shell",
				Summary: "Run a command line through the platform shell",
				Args:    "<script>",
				Examples: []string{
					`aos exec shell "git log --oneline | head -5"`,
				},
				Run: runExecShell,
			},
		)
	})
}

// stripSeparator drops a leading `--`, which callers use to stop agentic-os
// from claiming flags that belong to the child command.
func stripSeparator(args []string) []string {
	if len(args) > 0 && args[0] == "--" {
		return args[1:]
	}
	return args
}

func runExecRun(c *cli.Ctx, args []string) error {
	args = stripSeparator(args)
	if len(args) == 0 {
		return fmt.Errorf("`exec run` needs a command")
	}
	return passthroughExit(sys.Passthrough(args[0], args[1:]...))
}

func runExecShell(c *cli.Ctx, args []string) error {
	args = stripSeparator(args)
	if len(args) == 0 {
		return fmt.Errorf("`exec shell` needs a command line")
	}
	shell := sys.Shell()
	return passthroughExit(sys.Passthrough(shell[0], append(shell[1:], args...)...))
}

func runExecCapture(c *cli.Ctx, args []string) error {
	args = stripSeparator(args)
	if len(args) == 0 {
		return fmt.Errorf("`exec capture` needs a command")
	}

	child := exec.Command(args[0], args[1:]...)
	var stdout, stderr bytes.Buffer
	child.Stdout, child.Stderr, child.Stdin = &stdout, &stderr, c.Stdin

	started := time.Now()
	err := child.Run()
	result := execResult{
		Command:  args,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(started).Milliseconds(),
		Exit:     child.ProcessState.ExitCode(),
	}
	// A command that never started has no exit code; report why instead.
	if err != nil && result.Exit <= 0 && exitCodeOf(err) == 0 {
		result.Exit = -1
		result.Error = err.Error()
	}

	enc := json.NewEncoder(c.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return err
	}
	// The JSON is the payload, so mirror the child's status in our own exit
	// code without printing a second error line.
	if result.Exit != 0 {
		return &cli.ExitError{Code: 1}
	}
	return nil
}
