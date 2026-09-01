package groups

import (
	"strconv"
	"strings"

	"github.com/muthuishere/aos/internal/sys"
)

// batteryStatusNames maps Win32_Battery.BatteryStatus codes to words. Code 2
// means "AC connected", which is also how a desktop with no battery reports.
var batteryStatusNames = map[int]string{
	1: "discharging", 2: "charged", 3: "fully charged", 4: "low",
	5: "critical", 6: "charging", 7: "charging (high)", 8: "charging (low)",
	9: "charging (critical)", 10: "undefined", 11: "partially charged",
}

func readPower() (powerState, error) {
	state := powerState{Percent: -1}
	out, err := sys.PowerShell(`
$b = Get-CimInstance Win32_Battery | Select-Object -First 1
if ($b) {
  "present=true"
  "percent=" + $b.EstimatedChargeRemaining
  "status=" + $b.BatteryStatus
  if ($b.EstimatedRunTime -and $b.EstimatedRunTime -lt 71582788) { "minutes=" + $b.EstimatedRunTime }
} else { "present=false" }
`)
	if err != nil {
		return state, err
	}

	facts := map[string]string{}
	mergeKeyValues(facts, out)
	state.HasBattery = facts["present"] == "true"
	if percent, err := strconv.Atoi(facts["percent"]); err == nil {
		state.Percent = percent
	}
	if code, err := strconv.Atoi(facts["status"]); err == nil {
		state.Status = batteryStatusNames[code]
		state.OnAC = code != 1
	} else {
		// No battery at all means a desktop, which is always on mains.
		state.OnAC = !state.HasBattery
	}
	if minutes, err := strconv.Atoi(facts["minutes"]); err == nil && minutes > 0 {
		state.Remaining = strconv.Itoa(minutes/60) + ":" + leftPad(minutes%60)
	}
	return state, nil
}

func leftPad(n int) string {
	s := strconv.Itoa(n)
	if len(s) < 2 {
		return strings.Repeat("0", 2-len(s)) + s
	}
	return s
}
