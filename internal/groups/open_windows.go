package groups

import "github.com/muthuishere/agentic-os/internal/sys"

func openTarget(target string) error {
	// The empty string is start's window-title argument; without it a quoted
	// target is mistaken for the title and nothing opens. Spawn rather than
	// capture: `start` passes its stdio handles to whatever it opens.
	return sys.Spawn("cmd", "/c", "start", "", target)
}

func launchApp(app string, args []string) error {
	return sys.Spawn("cmd", append([]string{"/c", "start", "", app}, args...)...)
}

var chromiumApps = []string{
	`C:\Program Files\Google\Chrome\Application\chrome.exe`,
	`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
	`C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`,
	`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
}

func openWebapp(target string) error {
	for _, browser := range chromiumApps {
		if !fileExists(browser) {
			continue
		}
		return sys.Spawn(browser, "--app="+target)
	}
	return openTarget(target)
}
