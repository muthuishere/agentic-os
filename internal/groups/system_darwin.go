package groups

import "github.com/muthuishere/aos/internal/sys"

const cgSession = "/System/Library/CoreServices/Menu Extras/User.menu/Contents/Resources/CGSession"

func systemLock() error {
	if _, err := sys.Output(cgSession, "-suspend"); err == nil {
		return nil
	}
	// Fallback for machines where CGSession is unavailable: sleep the display,
	// which locks once the "require password after sleep" default is set.
	_, err := sys.Output("pmset", "displaysleepnow")
	return err
}

func systemSleep() error {
	_, err := sys.Output("pmset", "sleepnow")
	return err
}

func systemRestart() error {
	_, err := sys.Osascript(`tell application "System Events" to restart`)
	return err
}

func systemShutdown() error {
	_, err := sys.Osascript(`tell application "System Events" to shut down`)
	return err
}

func systemLogout() error {
	_, err := sys.Osascript(`tell application "System Events" to log out`)
	return err
}

func systemInfo() (map[string]string, error) {
	facts := map[string]string{"platform": "darwin"}
	if out, err := sys.Output("sw_vers", "-productName"); err == nil {
		facts["os"] = out
	}
	if out, err := sys.Output("sw_vers", "-productVersion"); err == nil {
		facts["os_version"] = out
	}
	if out, err := sys.Output("sw_vers", "-buildVersion"); err == nil {
		facts["build"] = out
	}
	if out, err := sys.Output("uname", "-m"); err == nil {
		facts["arch"] = out
	}
	if out, err := sys.Output("hostname"); err == nil {
		facts["host"] = out
	}
	if out, err := sys.Output("sysctl", "-n", "machdep.cpu.brand_string"); err == nil {
		facts["cpu"] = out
	}
	if out, err := sys.Output("sysctl", "-n", "hw.model"); err == nil {
		facts["model"] = out
	}
	return facts, nil
}
