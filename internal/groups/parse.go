package groups

import (
	"os"
	"path/filepath"
	"strings"
)

// mergeKeyValues folds `key=value` lines from a probe command into facts,
// skipping blanks so a partially-successful probe still contributes.
func mergeKeyValues(facts map[string]string, out string) {
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if key != "" && value != "" {
			facts[key] = value
		}
	}
}

// fileExists reports whether a path names an existing regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// expandHome resolves a leading ~ so paths typed at a shell prompt behave the
// same when they arrive already-quoted.
func expandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(path, "~"), "/"))
}
