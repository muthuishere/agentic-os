package groups

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/muthuishere/agentic-os/internal/sys"
)

// launchAgentsDir is where launchd looks for per-user agents, and it loads
// every plist in it at login. That directory listing is also our service
// registry: `list` reads it and keeps only the namespaced files.
func launchAgentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents"), nil
}

func servicePlistPath(label string) (string, error) {
	dir, err := launchAgentsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, label+".plist"), nil
}

// launchdTarget names a job in the per-user GUI domain. `bootstrap`/`bootout`
// against gui/<uid> replace the deprecated `load`/`unload`, which guessed the
// domain and quietly did the wrong thing over SSH.
func launchdTarget(label string) string {
	return fmt.Sprintf("gui/%d/%s", os.Getuid(), label)
}

func launchdDomain() string { return fmt.Sprintf("gui/%d", os.Getuid()) }

func serviceInstall(spec serviceSpec) ([]string, error) {
	path, err := servicePlistPath(spec.Label)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(renderLaunchdPlist(spec)), 0o644); err != nil {
		return nil, err
	}
	notes := []string{"plist " + path}
	if spec.Autostart {
		notes = append(notes, "launchd loads ~/Library/LaunchAgents at login, so autostart applies from the next one")
	}
	return notes, nil
}

func serviceUninstall(label string) error {
	path, err := servicePlistPath(label)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("service %q is not installed", label)
	}
	// A job that was never bootstrapped makes bootout fail; that is the state
	// we want anyway, so only the file removal has to succeed.
	_, _ = sys.Output("launchctl", "bootout", launchdTarget(label))
	return os.Remove(path)
}

func serviceStartOne(label string) error {
	path, err := servicePlistPath(label)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("service %q is not installed", label)
	}
	// Bootstrapping an already-bootstrapped job is an error, so its failure is
	// not interesting; kickstart is the call that has to work. Neither hands
	// the job our stdio — launchd is the service's parent, not this process —
	// so capturing their output cannot wedge on a long-lived child.
	_, _ = sys.Output("launchctl", "bootstrap", launchdDomain(), path)
	if _, err := sys.Output("launchctl", "kickstart", launchdTarget(label)); err != nil {
		return err
	}
	return nil
}

func serviceStopOne(label string) error {
	info, err := serviceQuery(label)
	if err != nil {
		return err
	}
	switch info.State {
	case serviceNotInstalled:
		return fmt.Errorf("service %q is not installed", label)
	case serviceStopped:
		return nil
	}
	// bootout unloads the job rather than signalling it, which is what makes a
	// stop stick: a merely killed job with RunAtLoad set comes back.
	_, err = sys.Output("launchctl", "bootout", launchdTarget(label))
	return err
}

func serviceQuery(label string) (serviceInfo, error) {
	path, err := servicePlistPath(label)
	if err != nil {
		return serviceInfo{}, err
	}
	// The plist, not launchd, is the record of installation: a stopped service
	// is booted out of the domain and would otherwise read as never created.
	if _, err := os.Stat(path); err != nil {
		return serviceInfo{State: serviceNotInstalled}, nil
	}
	out, err := sys.Output("launchctl", "print", launchdTarget(label))
	if err != nil {
		return serviceInfo{State: serviceStopped}, nil
	}
	if pid := launchdField(out, "pid"); pid != "" {
		return serviceInfo{State: serviceRunning, Detail: "pid " + pid}, nil
	}
	if code := launchdField(out, "last exit code"); code != "" && code != "0" {
		return serviceInfo{State: serviceStopped, Detail: "last exit " + code}, nil
	}
	return serviceInfo{State: serviceStopped}, nil
}

// launchdField pulls one `key = value` line out of `launchctl print` output.
func launchdField(out, key string) string {
	for _, line := range strings.Split(out, "\n") {
		name, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || strings.TrimSpace(name) != key {
			continue
		}
		return strings.TrimSpace(value)
	}
	return ""
}

func serviceLabels() ([]string, error) {
	dir, err := launchAgentsDir()
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
		label := strings.TrimSuffix(entry.Name(), ".plist")
		if label == entry.Name() {
			continue
		}
		if _, ok := serviceShortName(label); ok {
			labels = append(labels, label)
		}
	}
	return labels, nil
}
