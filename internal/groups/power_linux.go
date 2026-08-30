package groups

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const powerSupplyDir = "/sys/class/power_supply"

func readPower() (powerState, error) {
	state := powerState{Percent: -1}
	entries, err := os.ReadDir(powerSupplyDir)
	if err != nil {
		return state, err
	}
	for _, entry := range entries {
		dir := filepath.Join(powerSupplyDir, entry.Name())
		switch readSysfs(dir, "type") {
		case "Battery":
			state.HasBattery = true
			if percent, err := strconv.Atoi(readSysfs(dir, "capacity")); err == nil {
				state.Percent = percent
			}
			if status := readSysfs(dir, "status"); status != "" {
				state.Status = strings.ToLower(status)
			}
		case "Mains", "USB":
			if readSysfs(dir, "online") == "1" {
				state.OnAC = true
			}
		}
	}
	if !state.HasBattery {
		state.OnAC = true
	}
	return state, nil
}

func readSysfs(dir, name string) string {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
