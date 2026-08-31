package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testEnv(claude, agents string) func(string) string {
	return func(key string) string {
		switch key {
		case "CLAUDE_SKILLS_DIR":
			return claude
		case "AGENTS_SKILLS_DIR":
			return agents
		}
		return ""
	}
}

func TestInstallWritesToEveryHost(t *testing.T) {
	claude, agents := t.TempDir(), t.TempDir()
	env := testEnv(claude, agents)

	results, err := Install(env)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("want a result per host, got %+v", results)
	}
	for _, result := range results {
		if result.Action != "installed" {
			t.Errorf("%s: action = %q, want installed", result.Host, result.Action)
		}
		if _, err := os.Stat(filepath.Join(result.Path, "SKILL.md")); err != nil {
			t.Errorf("%s: SKILL.md missing: %v", result.Host, err)
		}
	}
}

// TestInstallIsIdempotentAndReplaces guards the update path: a second install
// must not fail, and must not leave a file from an older version behind, since
// an agent would read it as though it were current.
func TestInstallIsIdempotentAndReplaces(t *testing.T) {
	claude, agents := t.TempDir(), t.TempDir()
	env := testEnv(claude, agents)

	if _, err := Install(env); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(claude, Name, "STALE.md")
	if err := os.WriteFile(stale, []byte("from an older version"), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := Install(env)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if results[0].Action != "updated" {
		t.Errorf("action = %q, want updated", results[0].Action)
	}
	if _, err := os.Stat(stale); err == nil {
		t.Error("a stale file survived reinstall")
	}
}

// TestUninstallIsSafeToRepeat: a host that never had the skill is reported as
// absent rather than failing.
func TestUninstallIsSafeToRepeat(t *testing.T) {
	claude, agents := t.TempDir(), t.TempDir()
	env := testEnv(claude, agents)

	if _, err := Install(env); err != nil {
		t.Fatal(err)
	}
	first, err := Uninstall(env)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range first {
		if result.Action != "removed" {
			t.Errorf("%s: action = %q, want removed", result.Host, result.Action)
		}
	}

	second, err := Uninstall(env)
	if err != nil {
		t.Fatalf("uninstalling twice must not fail: %v", err)
	}
	for _, result := range second {
		if result.Action != "absent" {
			t.Errorf("%s: action = %q, want absent", result.Host, result.Action)
		}
	}
}

// TestBundledSkillIsUsable checks the embedded content is a real skill file:
// agents key off the frontmatter, so a malformed header makes the skill
// invisible rather than broken, which is much harder to notice.
func TestBundledSkillIsUsable(t *testing.T) {
	content, err := Content()
	if err != nil {
		t.Fatalf("content: %v", err)
	}
	if !strings.HasPrefix(content, "---\n") {
		t.Fatal("skill does not start with frontmatter")
	}
	header, _, ok := strings.Cut(strings.TrimPrefix(content, "---\n"), "\n---")
	if !ok {
		t.Fatal("frontmatter is not terminated")
	}
	if !strings.Contains(header, "name: "+Name) {
		t.Errorf("frontmatter must name the skill %q:\n%s", Name, header)
	}
	if !strings.Contains(header, "description:") {
		t.Error("frontmatter needs a description; it is what decides whether the skill triggers")
	}
	if len(content) < 500 {
		t.Errorf("skill is suspiciously short (%d bytes)", len(content))
	}
}
