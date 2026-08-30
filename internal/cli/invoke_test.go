package cli

import (
	"strings"
	"testing"
)

func TestInvokeCapturesOutputAndStdin(t *testing.T) {
	r := NewRegistry()
	r.Describe("demo", "Demo")
	r.Add(&Command{Group: "demo", Name: "echo", Summary: "Echo", Run: func(c *Ctx, args []string) error {
		c.Printf("args=%s\n", strings.Join(args, ","))
		return nil
	}})

	base := &Ctx{Registry: r, Env: func(string) string { return "" }, GOOS: "darwin", Version: "test"}
	got := Invoke(base, []string{"demo", "echo", "a", "b"}, "")
	if got.Exit != 0 || got.Stdout != "args=a,b\n" {
		t.Fatalf("%+v", got)
	}
}

func TestInvokeReportsFailureWithoutTouchingProcess(t *testing.T) {
	r := NewRegistry()
	base := &Ctx{Registry: r, Env: func(string) string { return "" }, GOOS: "darwin", Version: "test"}
	got := Invoke(base, []string{"nope"}, "")
	if got.Exit == 0 || !strings.Contains(got.Stderr, "unknown command") {
		t.Fatalf("%+v", got)
	}
}
