package e2e

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Result is what a step produced.
type Result struct {
	Stdout string
	Stderr string
	Exit   int
}

// Combined is stdout and stderr together, for assertions that do not care
// which stream a message came out on.
func (r Result) Combined() string { return r.Stdout + r.Stderr }

// Runner executes aos on one target.
type Runner struct {
	target Target
	deploy sync.Once
	err    error
}

func NewRunner(target Target) *Runner { return &Runner{target: target} }

// Run executes one command, always under the target's step timeout.
//
// A timeout is reported as an error rather than waited out. A hung remote step
// once cost half an hour of a session; the suite exists so that costs seconds.
func (r *Runner) Run(args ...string) (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.target.StepTimeout())
	defer cancel()

	switch r.target.Kind {
	case "local":
		return r.runLocal(ctx, args)
	case "ssh":
		return r.runSSH(ctx, args)
	case "agentbus":
		return r.runAgentbus(ctx, args)
	}
	return Result{}, fmt.Errorf("unknown target kind %q", r.target.Kind)
}

func (r *Runner) runLocal(ctx context.Context, args []string) (Result, error) {
	binary, err := filepath.Abs(r.target.Binary)
	if err != nil {
		return Result{}, err
	}
	if _, err := os.Stat(binary); err != nil {
		return Result{}, fmt.Errorf("build it first (task cross): %w", err)
	}
	return capture(ctx, r.target.StepTimeout(), binary, args...)
}

// runSSH ships the binary once per target, then runs commands against it.
func (r *Runner) runSSH(ctx context.Context, args []string) (Result, error) {
	r.deploy.Do(func() {
		binary, err := filepath.Abs(r.target.Binary)
		if err != nil {
			r.err = err
			return
		}
		remote := r.remotePath()
		if _, err := capture(ctx, r.target.StepTimeout(), "scp", "-q", binary, r.target.Host+":"+remote); err != nil {
			r.err = fmt.Errorf("ship binary to %s: %w", r.target.Host, err)
			return
		}
		if _, err := capture(ctx, r.target.StepTimeout(), "ssh", r.target.Host, "chmod +x "+remote); err != nil {
			r.err = err
		}
	})
	if r.err != nil {
		return Result{}, r.err
	}
	command := r.remotePath() + " " + shellJoin(args)
	return capture(ctx, r.target.StepTimeout(), "ssh", r.target.Host, command)
}

func (r *Runner) remotePath() string {
	if r.target.RemotePath != "" {
		return r.target.RemotePath
	}
	return "/tmp/agentic-os-e2e"
}

// runAgentbus ships the binary with each job — agentbus runs every job in a
// fresh directory, so there is nothing to cache.
//
// Output is redirected to files on the target and collected, rather than read
// from agentbus's own stdout: `agentbus run` prints a bounded excerpt and
// appends "[truncated; see stdout.txt]". Parsing that as the whole answer made
// valid JSON look corrupt.
func (r *Runner) runAgentbus(ctx context.Context, args []string) (Result, error) {
	binary, err := filepath.Abs(r.target.Binary)
	if err != nil {
		return Result{}, err
	}
	name := filepath.Base(binary)

	fetchDir, err := os.MkdirTemp("", "agentic-os-e2e-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(fetchDir)

	const outFile, errFile = "__e2e_stdout.txt", "__e2e_stderr.txt"
	prefix := "./"
	if r.target.Shell == "cmd" {
		prefix = ".\\"
	}
	invocation := fmt.Sprintf("%s%s %s > %s 2> %s",
		prefix, name, shellJoin(args), outFile, errFile)

	busArgs := []string{
		"run",
		"--node", r.target.Node,
		"--timeout", strconv.Itoa(int(r.target.StepTimeout().Seconds())),
		"--file", binary,
		"--collect", outFile,
		"--collect", errFile,
		"--fetch", fetchDir,
	}
	if r.target.Shell != "" {
		busArgs = append(busArgs, "--shell", r.target.Shell)
	}
	busArgs = append(busArgs, invocation)

	// agentbus exits with the remote command's code, so the result maps
	// straight through.
	outcome, err := capture(ctx, r.target.StepTimeout()+30*time.Second, "agentbus", busArgs...)
	if err != nil {
		return outcome, err
	}
	return Result{
		Stdout: readIfPresent(filepath.Join(fetchDir, outFile)),
		Stderr: readIfPresent(filepath.Join(fetchDir, errFile)),
		Exit:   outcome.Exit,
	}, nil
}

// readIfPresent returns a collected file's contents, or "" when the step
// produced nothing on that stream.
func readIfPresent(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// capture runs a command and returns its streams and exit code. A non-zero
// exit is a Result, not an error; only a failure to run at all is an error.
func capture(ctx context.Context, timeout time.Duration, name string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	cmd.WaitDelay = 2 * time.Second

	err := cmd.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String(), Exit: cmd.ProcessState.ExitCode()}

	if ctx.Err() != nil {
		return result, fmt.Errorf("timed out after %s", timeout)
	}
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		return result, err
	}
	return result, nil
}

// shellJoin quotes arguments that contain spaces, which is all the quoting the
// suite's own commands need.
func shellJoin(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.ContainsAny(arg, " \t") {
			quoted = append(quoted, `"`+arg+`"`)
			continue
		}
		quoted = append(quoted, arg)
	}
	return strings.Join(quoted, " ")
}
