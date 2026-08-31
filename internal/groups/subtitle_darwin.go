package groups

import (
	"fmt"
	"strings"

	"github.com/muthuishere/agentic-os/internal/sys"
)

// showSubtitle draws a borderless, click-through NSWindow through
// AppleScriptObjC. The window never becomes key: an accessory activation policy
// keeps osascript out of the Dock and orderFrontRegardless shows the window
// without activating the app behind it, so whatever the user was typing into
// keeps the keyboard.
//
// osascript holds the window up by running its own run loop for the duration
// and then exits, which is also the teardown — there is no process to clean up
// and no window that can outlive its deadline.
func showSubtitle(req subtitleRequest) (string, error) {
	if !subtitleOverlayReady() {
		// Be loud about the downgrade. A notification is not an overlay — it
		// lands in Notification Centre, obeys Do Not Disturb, and may never be
		// seen — so a caller that silently got one would draw a false
		// conclusion about what is on the user's screen.
		script := "display notification " + appleScriptString(req.Text) + ` with title "agentic-os"`
		if _, err := sys.Osascript(script); err != nil {
			return "", err
		}
		return "notification (AppleScriptObjC overlay unavailable)", nil
	}
	// Spawn, never capture: this process lives for the whole duration.
	if err := sys.Spawn("osascript", "-e", subtitleScript(req)); err != nil {
		return "", err
	}
	return "overlay", nil
}

// subtitleOverlayReady probes the two things the overlay needs — an
// AppleScriptObjC bridge and a screen to draw on — before committing to a
// detached process whose failure nobody would ever see.
func subtitleOverlayReady() bool {
	out, err := sys.Osascript(`use framework "AppKit"
return (current application's NSScreen's mainScreen() is not missing value) as text`)
	return err == nil && out == "true"
}

func subtitleScript(req subtitleRequest) string {
	// Placement is against visibleFrame, so a bottom caption clears the Dock
	// and a top one clears the menu bar.
	var originY string
	switch req.Position {
	case "top":
		originY = "sy + sheight - wh - 60"
	case "center":
		originY = "sy + (sheight - wh) / 2"
	default:
		originY = "sy + 60"
	}

	return fmt.Sprintf(`use framework "Foundation"
use framework "AppKit"
use scripting additions

set nsapp to current application's NSApplication's sharedApplication()
-- Accessory policy: no Dock tile, no menu bar, and nothing that could steal focus.
nsapp's setActivationPolicy:1

set lbl to current application's NSTextField's alloc()'s initWithFrame:{{0, 0}, {10, 10}}
lbl's setStringValue:%s
lbl's setFont:(current application's NSFont's boldSystemFontOfSize:%d)
lbl's setTextColor:(current application's NSColor's whiteColor)
lbl's setBezeled:false
lbl's setDrawsBackground:false
lbl's setEditable:false
lbl's setSelectable:false
lbl's sizeToFit()
set {{lx, ly}, {tw, twh}} to lbl's frame()

set ww to tw + 60
set wh to twh + 36
set scr to current application's NSScreen's mainScreen()
set {{sx, sy}, {swidth, sheight}} to scr's visibleFrame()
set wx to sx + (swidth - ww) / 2
set wy to %s

set win to current application's NSWindow's alloc()'s initWithContentRect:{{wx, wy}, {ww, wh}} styleMask:0 backing:2 defer:false
win's setOpaque:false
win's setBackgroundColor:(current application's NSColor's colorWithCalibratedWhite:0.0 alpha:0.75)
-- Above normal windows and full-screen apps; click-through so the caption
-- cannot swallow a pointer event meant for what is underneath.
win's setLevel:1000
win's setIgnoresMouseEvents:true
win's setCollectionBehavior:273
win's setHasShadow:true
lbl's setFrame:{{30, 18}, {tw, twh}}
(win's contentView())'s addSubview:lbl
win's orderFrontRegardless()

current application's NSRunLoop's currentRunLoop()'s runUntilDate:(current application's NSDate's dateWithTimeIntervalSinceNow:%d)
`, appleScriptString(req.Text), req.Size, originY, req.Seconds)
}

// appleScriptString renders a Go string as an AppleScript literal.
func appleScriptString(value string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"`
}
