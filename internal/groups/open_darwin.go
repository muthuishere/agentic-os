package groups

import "github.com/muthuishere/agentic-os/internal/sys"

func openTarget(target string) error {
	return sys.Spawn("open", target)
}

func launchApp(app string, args []string) error {
	argv := []string{"-a", app}
	if len(args) > 0 {
		argv = append(argv, "--args")
		argv = append(argv, args...)
	}
	return sys.Spawn("open", argv...)
}

// chromiumApps lists the Chromium-family browsers that support --app windows,
// most-preferred first.
var chromiumApps = []string{
	"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
	"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
	"/Applications/Chromium.app/Contents/MacOS/Chromium",
}

func openWebapp(target string) error {
	for _, browser := range chromiumApps {
		if !fileExists(browser) {
			continue
		}
		// Detach: the app window should outlive this CLI invocation.
		return sys.Spawn(browser, "--app="+target)
	}
	// No Chromium-family browser, so fall back to a normal tab.
	return openTarget(target)
}
