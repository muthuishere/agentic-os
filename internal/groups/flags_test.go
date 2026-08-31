package groups

import "testing"

func TestParseArgsFormsAndPositionals(t *testing.T) {
	set, err := parseArgs([]string{"--zone=1B", "--monitor", "2", "--double", "Chrome"}, "zone", "monitor")
	if err != nil {
		t.Fatal(err)
	}
	if got := set.String("zone", ""); got != "1B" {
		t.Fatalf("zone = %q", got)
	}
	monitor, err := set.IntPtr("monitor")
	if err != nil || monitor == nil || *monitor != 2 {
		t.Fatalf("monitor = %v, %v", monitor, err)
	}
	if !set.Has("double") {
		t.Fatal("boolean flag not recorded")
	}
	if len(set.Rest) != 1 || set.Rest[0] != "Chrome" {
		t.Fatalf("rest = %v", set.Rest)
	}
}

func TestParseArgsBooleanDoesNotEatPositional(t *testing.T) {
	set, err := parseArgs([]string{"--copy", "region"}, "out")
	if err != nil {
		t.Fatal(err)
	}
	if !set.Has("copy") || len(set.Rest) != 1 || set.Rest[0] != "region" {
		t.Fatalf("copy=%v rest=%v", set.Has("copy"), set.Rest)
	}
}

func TestRejectUnknownFlag(t *testing.T) {
	set, err := parseArgs([]string{"--zne=1B"}, "zone")
	if err != nil {
		t.Fatal(err)
	}
	if err := set.Reject("zone"); err == nil {
		t.Fatal("want an error for the typo'd flag")
	}
}

func TestMissingValueForValueFlag(t *testing.T) {
	if _, err := parseArgs([]string{"--monitor"}, "monitor"); err == nil {
		t.Fatal("want an error when a value flag ends the args")
	}
}

func TestParseRectAndLineRange(t *testing.T) {
	rect, err := parseRect("10,20,800,600")
	if err != nil || rect.X != 10 || rect.Y != 20 || rect.W != 800 || rect.H != 600 {
		t.Fatalf("rect = %+v, %v", rect, err)
	}
	if _, err := parseRect("10,20,800"); err == nil {
		t.Fatal("want an error for a three-part rect")
	}
	if from, to, err := parseLineRange("10:40"); err != nil || from != 10 || to != 40 {
		t.Fatalf("range = %d:%d, %v", from, to, err)
	}
	if _, _, err := parseLineRange("40:10"); err == nil {
		t.Fatal("want an error when the range runs backwards")
	}
}

func TestToolNameReplacesSpaces(t *testing.T) {
	if got := toolName("audio output set default"); got != "audio_output_set_default" {
		t.Fatalf("toolName = %q", got)
	}
}

func TestToStringSliceShapes(t *testing.T) {
	got, err := toStringSlice([]any{"a", "b"})
	if err != nil || len(got) != 2 || got[0] != "a" {
		t.Fatalf("got %v, %v", got, err)
	}
	if _, err := toStringSlice([]any{1}); err == nil {
		t.Fatal("want an error for a non-string element")
	}
	if got, _ := toStringSlice(nil); got != nil {
		t.Fatalf("nil should yield no args, got %v", got)
	}
}
