package groups

import "github.com/muthuishere/aos/internal/sys"

func detectPackageManager() *packageManager {
	if sys.Has("winget") {
		return &packageManager{
			Name: "winget",
			Bin:  "winget",
			// --accept-*-agreements keeps install non-interactive; without them
			// winget stops on a prompt that has no obvious answer in a script.
			Install: []string{"install", "--accept-package-agreements", "--accept-source-agreements"},
			Remove:  []string{"uninstall"},
			Search:  []string{"search"},
			List:    []string{"list"},
			Refresh: []string{"source", "update"},
			Upgrade: []string{"upgrade", "--all", "--accept-package-agreements", "--accept-source-agreements"},
		}
	}
	if sys.Has("scoop") {
		return &packageManager{
			Name:    "scoop",
			Bin:     "scoop",
			Install: []string{"install"},
			Remove:  []string{"uninstall"},
			Search:  []string{"search"},
			List:    []string{"list"},
			Refresh: []string{"update"},
			Upgrade: []string{"update", "*"},
		}
	}
	if sys.Has("choco") {
		return &packageManager{
			Name:    "chocolatey",
			Bin:     "choco",
			Install: []string{"install", "-y"},
			Remove:  []string{"uninstall", "-y"},
			Search:  []string{"search"},
			List:    []string{"list", "--local-only"},
			Upgrade: []string{"upgrade", "all", "-y"},
		}
	}
	return nil
}
