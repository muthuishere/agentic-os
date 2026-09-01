package groups

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/muthuishere/aos/internal/sys"
)

// fontRegistryKey maps installed fonts to their real display names, which the
// file stems do not give away: Candara Bold ships as `Candarab.ttf`.
const fontRegistryKey = `HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`

// fontTypeSuffix is the format tag Windows appends to a registry font name,
// as in "Arial (TrueType)".
var fontTypeSuffix = regexp.MustCompile(`\s*\((TrueType|OpenType|VGA res|All res)\)\s*$`)

func fontNames() ([]string, bool) {
	out, err := sys.PowerShell(
		"(Get-ItemProperty '" + fontRegistryKey + "').PSObject.Properties | " +
			"Where-Object { $_.Name -notlike 'PS*' } | ForEach-Object { $_.Name }")
	if err != nil {
		return nil, false
	}

	var names []string
	for _, line := range strings.Split(out, "\n") {
		if name := fontTypeSuffix.ReplaceAllString(strings.TrimSpace(line), ""); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil, false
	}
	return names, true
}

func fontDirs() []string {
	var dirs []string
	if windir := os.Getenv("WINDIR"); windir != "" {
		dirs = append(dirs, filepath.Join(windir, "Fonts"))
	} else {
		dirs = append(dirs, `C:\Windows\Fonts`)
	}
	// Per-user fonts installed without admin rights live under LOCALAPPDATA.
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		dirs = append(dirs, filepath.Join(local, "Microsoft", "Windows", "Fonts"))
	}
	return dirs
}
