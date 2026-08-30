package groups

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/muthuishere/agentic-os/internal/sys"
)

// defaultSink is wireplumber's name for the current default output.
const defaultSink = "@DEFAULT_AUDIO_SINK@"

func getVolume() (int, error) {
	if sys.Has("wpctl") {
		out, err := sys.Output("wpctl", "get-volume", defaultSink)
		if err != nil {
			return 0, err
		}
		// "Volume: 0.65" — or "Volume: 0.65 [MUTED]".
		fields := strings.Fields(out)
		if len(fields) < 2 {
			return 0, fmt.Errorf("unexpected wpctl output %q", out)
		}
		scalar, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return 0, err
		}
		return int(math.Round(scalar * 100)), nil
	}
	if sys.Has("pactl") {
		out, err := sys.Output("pactl", "get-sink-volume", "@DEFAULT_SINK@")
		if err != nil {
			return 0, err
		}
		for _, field := range strings.Fields(out) {
			if strings.HasSuffix(field, "%") {
				return strconv.Atoi(strings.TrimSuffix(field, "%"))
			}
		}
	}
	return 0, errUnsupported
}

func setVolume(level int) error {
	if sys.Has("wpctl") {
		_, err := sys.Output("wpctl", "set-volume", defaultSink, fmt.Sprintf("%.2f", float64(level)/100))
		return err
	}
	if sys.Has("pactl") {
		_, err := sys.Output("pactl", "set-sink-volume", "@DEFAULT_SINK@", strconv.Itoa(level)+"%")
		return err
	}
	return errUnsupported
}

func getMute() (bool, error) {
	if sys.Has("wpctl") {
		out, err := sys.Output("wpctl", "get-volume", defaultSink)
		if err != nil {
			return false, err
		}
		return strings.Contains(out, "[MUTED]"), nil
	}
	if sys.Has("pactl") {
		out, err := sys.Output("pactl", "get-sink-mute", "@DEFAULT_SINK@")
		if err != nil {
			return false, err
		}
		return strings.Contains(out, "yes"), nil
	}
	return false, errUnsupported
}

func setMute(muted bool) error {
	value := "0"
	if muted {
		value = "1"
	}
	if sys.Has("wpctl") {
		_, err := sys.Output("wpctl", "set-mute", defaultSink, value)
		return err
	}
	if sys.Has("pactl") {
		_, err := sys.Output("pactl", "set-sink-mute", "@DEFAULT_SINK@", value)
		return err
	}
	return errUnsupported
}
