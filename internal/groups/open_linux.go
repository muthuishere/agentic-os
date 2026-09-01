package groups

import "github.com/muthuishere/aos/internal/sys"

func openTarget(target string) error {
	// Openers hand their stdio to the application they start, so spawn instead
	// of capturing; a captured call would wait for that application to exit.
	switch sys.FirstAvailable("xdg-open", "gio") {
	case "xdg-open":
		return sys.Spawn("xdg-open", target)
	case "gio":
		return sys.Spawn("gio", "open", target)
	}
	return errUnsupported
}

func launchApp(app string, args []string) error {
	if sys.Has(app) {
		return sys.Spawn(app, args...)
	}
	if sys.Has("gtk-launch") {
		return sys.Spawn("gtk-launch", append([]string{app}, args...)...)
	}
	return errUnsupported
}

var chromiumApps = []string{"google-chrome", "brave", "chromium", "microsoft-edge"}

func openWebapp(target string) error {
	for _, browser := range chromiumApps {
		if !sys.Has(browser) {
			continue
		}
		return sys.Spawn(browser, "--app="+target)
	}
	return openTarget(target)
}
