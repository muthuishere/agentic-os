package groups

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/muthuishere/aos/internal/sys"
)

// systemdUserDir is where a user's own units live, and the directory listing is
// our service registry: `list` reads it and keeps only the namespaced files.
func systemdUserDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "systemd", "user"), nil
}

// systemdUnit is the unit name systemctl expects, label and suffix.
func systemdUnit(label string) string { return label + ".service" }

func systemdUnitPath(label string) (string, error) {
	dir, err := systemdUserDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, systemdUnit(label)), nil
}

func systemctl(args ...string) (string, error) {
	// systemd is the service's parent, so none of these calls hand our stdio to
	// a long-lived child and capturing their output is safe.
	return sys.Output("systemctl", append([]string{"--user"}, args...)...)
}

func serviceInstall(spec serviceSpec) ([]string, error) {
	if !sys.Has("systemctl") {
		return nil, errUnsupported
	}
	path, err := systemdUnitPath(spec.Label)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(renderSystemdUnit(spec)), 0o644); err != nil {
		return nil, err
	}
	if _, err := systemctl("daemon-reload"); err != nil {
		return nil, err
	}
	notes := []string{"unit " + path}
	if spec.Autostart {
		if _, err := systemctl("enable", systemdUnit(spec.Label)); err != nil {
			return nil, err
		}
		// The part people miss: without linger, the user manager is torn down at
		// logout and the "autostart" service dies with it on a server.
		notes = append(notes,
			"run `loginctl enable-linger "+os.Getenv("USER")+"` for this to survive logout")
	}
	return notes, nil
}

func serviceUninstall(label string) error {
	path, err := systemdUnitPath(label)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("service %q is not installed", label)
	}
	unit := systemdUnit(label)
	// Both are best-effort: a service that was never started or never enabled
	// is already in the state we are asking for.
	_, _ = systemctl("stop", unit)
	_, _ = systemctl("disable", unit)
	if err := os.Remove(path); err != nil {
		return err
	}
	_, _ = systemctl("daemon-reload")
	return nil
}

func serviceStartOne(label string) error {
	if err := requireSystemdUnit(label); err != nil {
		return err
	}
	_, err := systemctl("start", systemdUnit(label))
	return err
}

func serviceStopOne(label string) error {
	if err := requireSystemdUnit(label); err != nil {
		return err
	}
	_, err := systemctl("stop", systemdUnit(label))
	return err
}

func requireSystemdUnit(label string) error {
	path, err := systemdUnitPath(label)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("service %q is not installed", label)
	}
	return nil
}

func serviceQuery(label string) (serviceInfo, error) {
	path, err := systemdUnitPath(label)
	if err != nil {
		return serviceInfo{}, err
	}
	// The unit file, not systemd, is the record of installation: a stopped
	// service is still installed and must not report as missing.
	if _, err := os.Stat(path); err != nil {
		return serviceInfo{State: serviceNotInstalled}, nil
	}
	// `is-active` exits non-zero for anything but an active unit, so its output
	// is the answer and the error is not.
	active, _ := systemctl("is-active", systemdUnit(label))
	if active != "active" && active != "activating" {
		if active == "" {
			active = "inactive"
		}
		return serviceInfo{State: serviceStopped, Detail: active}, nil
	}
	detail := ""
	if pid, err := systemctl("show", "--property=MainPID", "--value", systemdUnit(label)); err == nil {
		if pid = strings.TrimSpace(pid); pid != "" && pid != "0" {
			detail = "pid " + pid
		}
	}
	return serviceInfo{State: serviceRunning, Detail: detail}, nil
}

func serviceLabels() ([]string, error) {
	dir, err := systemdUserDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var labels []string
	for _, entry := range entries {
		label := strings.TrimSuffix(entry.Name(), ".service")
		if label == entry.Name() {
			continue
		}
		if _, ok := serviceShortName(label); ok {
			labels = append(labels, label)
		}
	}
	return labels, nil
}
