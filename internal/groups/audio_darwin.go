package groups

import (
	"fmt"
	"strconv"

	"github.com/muthuishere/agentic-os/internal/sys"
)

func getVolume() (int, error) {
	out, err := sys.Osascript("output volume of (get volume settings)")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(out)
}

func setVolume(level int) error {
	_, err := sys.Osascript(fmt.Sprintf("set volume output volume %d", level))
	return err
}

func getMute() (bool, error) {
	out, err := sys.Osascript("output muted of (get volume settings)")
	if err != nil {
		return false, err
	}
	return out == "true", nil
}

func setMute(muted bool) error {
	_, err := sys.Osascript(fmt.Sprintf("set volume output muted %t", muted))
	return err
}
