package groups

import (
	"os"
	"path/filepath"
)

func fontDirs() []string {
	dirs := []string{"/usr/share/fonts", "/usr/local/share/fonts"}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs,
			filepath.Join(home, ".local", "share", "fonts"),
			filepath.Join(home, ".fonts"),
		)
	}
	return dirs
}

// fontNames has no cheap native source here, so the directory scan is used.
func fontNames() ([]string, bool) { return nil, false }
