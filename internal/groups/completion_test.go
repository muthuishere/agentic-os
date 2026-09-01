package groups

import (
	"strings"
	"testing"

	"github.com/muthuishere/aos/internal/cli"
)

// testRegistry is a hand-built stand-in for the real command tree: a group
// default, a multi-token route, a command carrying flags, and a hidden one that
// must never reach a user's shell.
func testRegistry() *cli.Registry {
	reg := cli.NewRegistry()
	reg.Describe("audio", "Sound")
	reg.Describe("file", "Files")
	run := func(*cli.Ctx, []string) error { return nil }
	reg.Add(
		&cli.Command{Group: "audio", Summary: "Show audio", Run: run},
		&cli.Command{Group: "audio", Name: "output set default", Summary: "Set output", Run: run},
		&cli.Command{Group: "file", Name: "read", Args: "[--json] [--app=<name>] <path>", Summary: "Read", Run: run},
		&cli.Command{Group: "file", Name: "shred", Summary: "Shred", Hidden: true, Run: run},
	)
	return reg
}

func TestCompletionScriptMentionsRoutes(t *testing.T) {
	cases := []struct {
		name  string
		shell string
		want  []string
		gone  []string
	}{
		{
			name:  "bash offers groups, nested words and flags",
			shell: "bash",
			want: []string{
				"complete -F _aos_complete aos",
				"audio",
				"file",
				"output",
				"set",
				"default",
				"--json",
				"--app",
			},
			gone: []string{"shred"},
		},
		{
			name:  "zsh is a compdef script with the same tree",
			shell: "zsh",
			want:  []string{"#compdef aos", "compdef _aos aos", "audio", "--app"},
			gone:  []string{"shred"},
		},
		{
			name:  "fish drives everything off one prefix helper",
			shell: "fish",
			want:  []string{"complete -c aos", "__aos_at", "audio", "--app"},
			gone:  []string{"shred"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			script, err := completionScript(testRegistry(), tc.shell)
			if err != nil {
				t.Fatalf("completionScript(%q) error: %v", tc.shell, err)
			}
			for _, want := range tc.want {
				if !strings.Contains(script, want) {
					t.Errorf("%s script missing %q", tc.shell, want)
				}
			}
			for _, gone := range tc.gone {
				if strings.Contains(script, gone) {
					t.Errorf("%s script leaks hidden command %q", tc.shell, gone)
				}
			}
		})
	}
}

func TestCompletionScriptUnknownShell(t *testing.T) {
	if _, err := completionScript(testRegistry(), "powershell"); err == nil {
		t.Fatal("want an error for an unsupported shell")
	}
}

// The tree is what the three emitters share, so nesting is asserted here rather
// than three times over in the generated text.
func TestCompletionTree(t *testing.T) {
	nodes := map[string][]string{}
	for _, node := range completionTree(testRegistry()) {
		nodes[node.Key] = node.Words
	}

	cases := []struct {
		name string
		key  string
		want []string
	}{
		{"root lists group names", "", []string{"audio", "commands", "file"}},
		{"a group lists its next word", "audio", []string{"output"}},
		{"nesting continues past the group", "audio output", []string{"set"}},
		{"and again at the last word", "audio output set", []string{"default"}},
		{"flags are parsed out of Args", "file read", []string{"--app", "--json"}},
		{"a hidden command contributes nothing", "file", []string{"read"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.Join(nodes[tc.key], " ")
			if got != strings.Join(tc.want, " ") {
				t.Fatalf("node %q = %q; want %q", tc.key, got, strings.Join(tc.want, " "))
			}
		})
	}
}

func TestCompletionFlags(t *testing.T) {
	cases := []struct {
		name string
		args string
		want string
	}{
		{"no args, no flags", "", ""},
		{"positional only", "<path>", ""},
		{"a boolean flag", "[--json]", "--json"},
		{"a valued flag keeps only its name", "[--app=<name>]", "--app"},
		{"mixed, sorted, deduped", "[--json] [--app=<name>] [--json] <path>", "--app --json"},
		{"a bare dash is not a flag", "[-a] [--all]", "--all"},
		{"dashes inside a name are kept", "[--dry-run]", "--dry-run"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.Join(completionFlags(tc.args), " ")
			if got != tc.want {
				t.Fatalf("completionFlags(%q) = %q; want %q", tc.args, got, tc.want)
			}
		})
	}
}
