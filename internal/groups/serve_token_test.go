package groups

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/muthuishere/agentic-os/internal/cli"
)

// TestTokenMatches pins the gate on the MCP surface. Loopback is not a
// permission — every process on the machine shares it — and this surface drives
// windows, input and the filesystem.
func TestTokenMatches(t *testing.T) {
	const token = "0123456789abcdef0123"

	cases := []struct {
		name   string
		set    func(*http.Request)
		accept bool
	}{
		{"bearer header", func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) }, true},
		{"query parameter", func(r *http.Request) { r.URL.RawQuery = "t=" + token }, true},
		{"no credential", func(r *http.Request) {}, false},
		{"wrong token", func(r *http.Request) { r.Header.Set("Authorization", "Bearer nope") }, false},
		{"empty bearer", func(r *http.Request) { r.Header.Set("Authorization", "Bearer ") }, false},
		{"prefix of the token", func(r *http.Request) { r.URL.RawQuery = "t=" + token[:8] }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			tc.set(r)
			if got := tokenMatches(r, token); got != tc.accept {
				t.Errorf("tokenMatches = %v, want %v", got, tc.accept)
			}
		})
	}
}

func TestResolveServeToken(t *testing.T) {
	env := func(pairs map[string]string) func(string) string {
		return func(key string) string { return pairs[key] }
	}

	// A flag beats the environment.
	set, _ := parseArgs([]string{"--token=flagflagflagflag"}, "token")
	c := &cli.Ctx{Env: env(map[string]string{serveTokenEnv: "envenvenvenvenvenv"})}
	if got, err := resolveServeToken(c, set); err != nil || got != "flagflagflagflag" {
		t.Fatalf("got %q, %v", got, err)
	}

	// The environment is used when no flag is given, so a client configured
	// once survives a restart.
	set, _ = parseArgs(nil, "token")
	if got, err := resolveServeToken(c, set); err != nil || got != "envenvenvenvenvenv" {
		t.Fatalf("got %q, %v", got, err)
	}

	// With neither, a fresh random token per run, and two runs must differ.
	c = &cli.Ctx{Env: env(nil)}
	first, err := resolveServeToken(c, set)
	if err != nil || len(first) < 32 {
		t.Fatalf("generated token %q, %v", first, err)
	}
	second, _ := resolveServeToken(c, set)
	if first == second {
		t.Fatal("a generated token must not repeat between runs")
	}

	// A short token is refused: it is the only thing gating the machine.
	short, _ := parseArgs([]string{"--token=tooshort"}, "token")
	if _, err := resolveServeToken(c, short); err == nil || !strings.Contains(err.Error(), "16") {
		t.Fatalf("want a length error, got %v", err)
	}
}
