package groups

import (
	"strings"

	"github.com/muthuishere/agentic-os/internal/sys"
)

func readNetwork() (netState, error) {
	var state netState

	// `ip route get` names the interface and source address the kernel would
	// actually use, which beats guessing from the interface list.
	if out, err := sys.Output("ip", "route", "get", "1.1.1.1"); err == nil {
		fields := strings.Fields(out)
		for i, field := range fields {
			if i+1 >= len(fields) {
				break
			}
			switch field {
			case "dev":
				state.Interface = fields[i+1]
			case "src":
				state.IP = fields[i+1]
			}
		}
	}

	if sys.Has("nmcli") {
		if out, err := sys.Output("nmcli", "-t", "-f", "active,ssid,signal", "dev", "wifi"); err == nil {
			for _, line := range strings.Split(out, "\n") {
				parts := strings.Split(line, ":")
				if len(parts) >= 3 && parts[0] == "yes" {
					state.SSID = parts[1]
					state.Signal = parts[2] + "%"
					break
				}
			}
		}
	}
	if state.SSID == "" && sys.Has("iwgetid") {
		if ssid, err := sys.Output("iwgetid", "-r"); err == nil {
			state.SSID = ssid
		}
	}
	return state, nil
}
