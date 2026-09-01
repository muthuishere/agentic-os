package groups

import (
	"strings"

	"github.com/muthuishere/aos/internal/sys"
)

func readNetwork() (netState, error) {
	var state netState

	out, err := sys.PowerShell(`
$route = Get-NetRoute -DestinationPrefix '0.0.0.0/0' -ErrorAction SilentlyContinue |
  Sort-Object RouteMetric | Select-Object -First 1
if ($route) {
  $if = Get-NetAdapter -InterfaceIndex $route.InterfaceIndex -ErrorAction SilentlyContinue
  if ($if) { "interface=" + $if.Name }
  $ip = Get-NetIPAddress -InterfaceIndex $route.InterfaceIndex -AddressFamily IPv4 -ErrorAction SilentlyContinue |
    Select-Object -First 1
  if ($ip) { "ip=" + $ip.IPAddress }
}
`)
	if err == nil {
		facts := map[string]string{}
		mergeKeyValues(facts, out)
		state.Interface, state.IP = facts["interface"], facts["ip"]
	}

	// netsh is the only reliable SSID source; it prints nothing useful when no
	// wireless adapter exists, so a failure here is not an error overall.
	if wlan, err := sys.Output("netsh", "wlan", "show", "interfaces"); err == nil {
		for _, line := range strings.Split(wlan, "\n") {
			key, value, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			key, value = strings.TrimSpace(key), strings.TrimSpace(value)
			switch key {
			case "SSID":
				state.SSID = value
			case "Signal":
				state.Signal = value
			}
		}
	}
	return state, nil
}
