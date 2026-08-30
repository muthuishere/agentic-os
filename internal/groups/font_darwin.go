package groups

import (
	"os"
	"path/filepath"
)

func fontDirs() []string {
	dirs := []string{"/System/Library/Fonts", "/Library/Fonts"}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Library", "Fonts"))
	}
	return dirs
}

// fontNames has no cheap native source here, so the directory scan is used.
func fontNames() ([]string, bool) { return nil, false }
