package groups

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/muthuishere/aos/internal/sys"
)

// pmsetBattery matches the battery line of `pmset -g batt`, e.g.
// ` -InternalBattery-0 (id=1234)	87%; discharging; 3:41 remaining present: true`
var pmsetBattery = regexp.MustCompile(`(\d+)%; ([^;]+);\s*([0-9:]+|\(no estimate\))?`)

func readPower() (powerState, error) {
	state := powerState{Percent: -1}
	out, err := sys.Output("pmset", "-g", "batt")
	if err != nil {
		return state, err
	}
	state.OnAC = strings.Contains(out, "'AC Power'")
	state.HasBattery = strings.Contains(out, "InternalBattery")

	if match := pmsetBattery.FindStringSubmatch(out); match != nil {
		if percent, err := strconv.Atoi(match[1]); err == nil {
			state.Percent = percent
		}
		state.Status = strings.TrimSpace(match[2])
		if remaining := strings.TrimSpace(match[3]); remaining != "" && remaining != "0:00" {
			state.Remaining = remaining
		}
	}
	return state, nil
}
