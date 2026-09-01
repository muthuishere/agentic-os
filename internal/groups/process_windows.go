package groups

import (
	"strconv"

	"github.com/muthuishere/aos/internal/sys"
)

// listProcesses reads every process through Get-Process. Windows has no cheap
// per-process CPU percentage, so the CPU column carries total CPU seconds; see
// processInfo. The owning user needs -IncludeUserName, which needs an elevated
// shell, so it is left out rather than making every listing prompt.
func listProcesses() ([]processInfo, error) {
	out, err := sys.PowerShell(
		`Get-Process | ForEach-Object { "{0}` + "`t" + `{1}` + "`t" + `{2}` + "`t" + `{3}" -f $_.Id, $_.ProcessName, [double]$_.CPU, $_.WorkingSet64 }`)
	if err != nil {
		return nil, err
	}
	return parseWindowsProcessOutput(out), nil
}

func killProcess(pid int, force bool) error {
	args := []string{"/PID", strconv.Itoa(pid)}
	if force {
		args = append(args, "/F")
	}
	_, err := sys.Output("taskkill", args...)
	return err
}
