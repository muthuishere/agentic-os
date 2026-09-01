package groups

import "github.com/muthuishere/aos/internal/sys"

func detectPackageManager() *packageManager {
	if !sys.Has("brew") {
		return nil
	}
	return &packageManager{
		Name:    "homebrew",
		Bin:     "brew",
		Install: []string{"install"},
		Remove:  []string{"uninstall"},
		Search:  []string{"search"},
		List:    []string{"list"},
		Refresh: []string{"update"},
		Upgrade: []string{"upgrade"},
	}
}
