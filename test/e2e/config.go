// Package e2e runs the command surface against real machines.
//
// Unit tests cover the registry and the parsing; they cannot tell you whether
// a screenshot on Windows is blank or whether a window really moved. This
// suite drives the built binaries on whatever machines are configured and
// asserts on what came back.
//
// The machine list is deliberately not in the repo: it names hosts and VMs
// specific to one person. Copy config.example.json to config.json (gitignored)
// and the suite picks it up; without it every test skips with an explanation
// rather than failing.
package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config is the machine list.
type Config struct {
	Targets []Target `json:"targets"`
}

// Target is one machine to exercise.
type Target struct {
	// Name labels the subtest.
	Name string `json:"name"`
	// Kind is local, ssh, or agentbus.
	Kind string `json:"kind"`
	// Binary is the locally built executable to run or ship.
	Binary string `json:"binary"`
	// Host is the ssh destination, for kind "ssh".
	Host string `json:"host"`
	// RemotePath is where the binary lands on an ssh host.
	RemotePath string `json:"remotePath"`
	// Node is the agentbus node name, for kind "agentbus".
	Node string `json:"node"`
	// Shell is the agentbus interpreter, usually cmd on Windows.
	Shell string `json:"shell"`
	// OS is the target's GOOS. Steps need it because the same intent is
	// spelled differently per platform: `echo` is a real binary on Unix and a
	// shell builtin on Windows.
	OS string `json:"os"`
	// TempDir is a writable absolute path on the target. Remote steps must use
	// absolute paths: an agentbus job runs in a fresh directory each time, so
	// a relative file written by one step does not exist for the next.
	TempDir string `json:"tempDir"`
	// HasDisplay says whether GUI steps should run. A server sets this false
	// so the suite asserts the *refusal* rather than skipping silently.
	HasDisplay bool `json:"hasDisplay"`
	// TimeoutSeconds bounds every single step. A step that hangs is a failure
	// to report, never something to wait on — the whole reason this exists.
	TimeoutSeconds int `json:"timeoutSeconds"`
}

// IsWindows reports whether steps should use Windows spellings.
func (t Target) IsWindows() bool { return t.OS == "windows" || t.Shell == "cmd" }

// Temp returns an absolute, writable directory on the target.
func (t Target) Temp() string {
	if t.TempDir != "" {
		return t.TempDir
	}
	if t.IsWindows() {
		return `C:\Windows\Temp`
	}
	return "/tmp"
}

// Path joins Temp with a file name using the target's separator.
func (t Target) Path(name string) string {
	if t.IsWindows() {
		return t.Temp() + `\` + name
	}
	return t.Temp() + "/" + name
}

// EchoArgs spells "print this word" for the target, for use with `exec capture`.
// exec capture runs a program, and on Windows `echo` is not one.
func (t Target) EchoArgs(word string) []string {
	if t.IsWindows() {
		return []string{"cmd", "/c", "echo", word}
	}
	return []string{"echo", word}
}

// StepTimeout is the per-step bound, with a sane default.
func (t Target) StepTimeout() time.Duration {
	if t.TimeoutSeconds <= 0 {
		return 60 * time.Second
	}
	return time.Duration(t.TimeoutSeconds) * time.Second
}

// ConfigPath is where the machine list lives, overridable so CI can point
// somewhere else.
func ConfigPath() string {
	if path := os.Getenv("AGENTIC_OS_E2E_CONFIG"); path != "" {
		return path
	}
	return filepath.Join("config.json")
}

// LoadConfig reads the machine list. A missing file is not an error: it means
// this checkout has no machines configured.
func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(ConfigPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse %s: %w", ConfigPath(), err)
	}
	for i, target := range config.Targets {
		if target.Name == "" || target.Kind == "" {
			return nil, fmt.Errorf("target %d needs a name and a kind", i)
		}
	}
	return &config, nil
}
