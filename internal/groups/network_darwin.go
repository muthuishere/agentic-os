package groups

import (
	"regexp"
	"strings"

	"github.com/muthuishere/aos/internal/sys"
)

var (
	routeInterface = regexp.MustCompile(`interface:\s*(\S+)`)
	// macOS 14.4 dropped the SSID from `networksetup`/`airport`; `ipconfig
	// getsummary` still reports it, as ` SSID : <name>`.
	ipconfigSSID = regexp.MustCompile(`(?m)^\s*SSID\s*:\s*(.+)$`)
	airportRSSI  = regexp.MustCompile(`(?m)^\s*agrCtlRSSI:\s*(-?\d+)`)
)

const airportBinary = "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"

func readNetwork() (netState, error) {
	var state netState

	if out, err := sys.Output("route", "-n", "get", "default"); err == nil {
		if match := routeInterface.FindStringSubmatch(out); match != nil {
			state.Interface = match[1]
		}
	}
	if state.Interface != "" {
		if ip, err := sys.Output("ipconfig", "getifaddr", state.Interface); err == nil {
			state.IP = ip
		}
		if out, err := sys.Output("ipconfig", "getsummary", state.Interface); err == nil {
			if match := ipconfigSSID.FindStringSubmatch(out); match != nil {
				state.SSID = strings.TrimSpace(match[1])
			}
		}
	}
	if state.SSID == "" && state.Interface != "" {
		if out, err := sys.Output("networksetup", "-getairportnetwork", state.Interface); err == nil {
			if _, name, ok := strings.Cut(out, ": "); ok {
				state.SSID = strings.TrimSpace(name)
			}
		}
	}
	if state.SSID != "" {
		if out, err := sys.Output(airportBinary, "-I"); err == nil {
			if match := airportRSSI.FindStringSubmatch(out); match != nil {
				state.Signal = match[1] + " dBm"
			}
		}
	}
	return state, nil
}
