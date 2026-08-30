package groups

import "github.com/muthuishere/agentic-os/internal/sys"

func systemLock() error {
	if sys.Has("loginctl") {
		if _, err := sys.Output("loginctl", "lock-session"); err == nil {
			return nil
		}
	}
	if locker := sys.FirstAvailable("hyprlock", "swaylock", "i3lock", "xdg-screensaver"); locker != "" {
		return sys.Passthrough(locker)
	}
	return errUnsupported
}

func systemSleep() error {
	_, err := sys.Output("systemctl", "suspend")
	return err
}

func systemRestart() error {
	_, err := sys.Output("systemctl", "reboot")
	return err
}

func systemShutdown() error {
	_, err := sys.Output("systemctl", "poweroff")
	return err
}

func systemLogout() error {
	if _, err := sys.Output("loginctl", "terminate-user", ""); err == nil {
		return nil
	}
	_, err := sys.Output("loginctl", "terminate-session", "")
	return err
}

func systemInfo() (map[string]string, error) {
	facts := map[string]string{"platform": "linux"}
	if out, err := sys.Output("sh", "-c", ". /etc/os-release && echo \"os=$NAME\nos_version=$VERSION_ID\""); err == nil {
		mergeKeyValues(facts, out)
	}
	if out, err := sys.Output("uname", "-r"); err == nil {
		facts["kernel"] = out
	}
	if out, err := sys.Output("uname", "-m"); err == nil {
		facts["arch"] = out
	}
	if out, err := sys.Output("hostname"); err == nil {
		facts["host"] = out
	}
	return facts, nil
}
