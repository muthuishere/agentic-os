package groups

import (
	"strconv"

	"github.com/muthuishere/aos/internal/sys"
)

// listProcesses reads every process from `ps`. The `=` suffixes suppress the
// header, so parsePsOutput never has to skip one.
func listProcesses() ([]processInfo, error) {
	out, err := sys.Output("ps", "-axo", "pid=,pcpu=,rss=,user=,comm=")
	if err != nil {
		return nil, err
	}
	return parsePsOutput(out), nil
}

func killProcess(pid int, force bool) error {
	signal := "-TERM"
	if force {
		signal = "-KILL"
	}
	_, err := sys.Output("kill", signal, strconv.Itoa(pid))
	return err
}
