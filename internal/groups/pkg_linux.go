package groups

import "github.com/muthuishere/aos/internal/sys"

func detectPackageManager() *packageManager {
	switch {
	case sys.Has("yay"):
		return &packageManager{
			Name: "yay", Bin: "yay",
			Install: []string{"-S", "--noconfirm"},
			Remove:  []string{"-Rns", "--noconfirm"},
			Search:  []string{"-Ss"},
			List:    []string{"-Q"},
			Refresh: []string{"-Sy"},
			Upgrade: []string{"-Syu", "--noconfirm"},
		}
	case sys.Has("pacman"):
		return &packageManager{
			Name: "pacman", Bin: "pacman",
			Install: []string{"-S", "--noconfirm"},
			Remove:  []string{"-Rns", "--noconfirm"},
			Search:  []string{"-Ss"},
			List:    []string{"-Q"},
			Refresh: []string{"-Sy"},
			Upgrade: []string{"-Syu", "--noconfirm"},
		}
	case sys.Has("apt"):
		return &packageManager{
			Name: "apt", Bin: "apt",
			Install: []string{"install", "-y"},
			Remove:  []string{"remove", "-y"},
			Search:  []string{"search"},
			List:    []string{"list", "--installed"},
			Refresh: []string{"update"},
			Upgrade: []string{"upgrade", "-y"},
		}
	case sys.Has("dnf"):
		return &packageManager{
			Name: "dnf", Bin: "dnf",
			Install: []string{"install", "-y"},
			Remove:  []string{"remove", "-y"},
			Search:  []string{"search"},
			List:    []string{"list", "--installed"},
			Refresh: []string{"check-update"},
			Upgrade: []string{"upgrade", "-y"},
		}
	}
	return nil
}
