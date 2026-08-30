package sys

import (
	"runtime"
	"testing"
	"time"
)

// TestOutputDoesNotWaitOnAnInheritedPipe reproduces the bug that hung `launch`
// on Windows: a launcher exits immediately but hands its stdout handle to a
// long-lived child, and a captured Run waits for the pipe, not the process.
// WaitDelay has to bound that wait.
func TestOutputDoesNotWaitOnAnInheritedPipe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell to background a pipe-holding child")
	}

	started := time.Now()
	// `sh` exits at once; the backgrounded sleep keeps stdout open for 30s.
	_, _ = Output("sh", "-c", "sleep 30 & echo launched")
	elapsed := time.Since(started)

	// Without WaitDelay this takes the full 30 seconds.
	if elapsed > 10*time.Second {
		t.Fatalf("Output waited %v on a pipe held by a grandchild; WaitDelay is not bounding it", elapsed)
	}
}

// TestSpawnReturnsImmediately is the other half of the fix: launching an
// application must not wait for it at all.
func TestSpawnReturnsImmediately(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX sleep")
	}

	started := time.Now()
	if err := Spawn("sleep", "30"); err != nil {
		t.Fatalf("spawn: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("Spawn waited %v; it must not wait for the process at all", elapsed)
	}
}

func TestSpawnReportsAMissingProgram(t *testing.T) {
	if err := Spawn("agentic-os-no-such-program"); err == nil {
		t.Fatal("want an error when the program does not exist")
	}
}
