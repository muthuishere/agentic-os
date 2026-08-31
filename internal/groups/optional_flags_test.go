package groups

import "testing"

// An optional-value flag is one that is meaningful bare AND with a value.
// `launch Ghostty --wait` broke with "--wait needs a value" when `wait` was
// declared as a value flag: the parser demanded a following word. The fix is
// that such a flag is simply left out of valueFlags, so bare `--wait` records
// an empty value that Int/String read as "use the default", `--wait=5000`
// still parses through the `=` form, and the flag never eats a positional.
func TestOptionalValueFlagBareValuedAndDoesNotEatAPositional(t *testing.T) {
	const defaultWait = 10000

	cases := []struct {
		name     string
		args     []string
		wantWait int
		wantRest []string
	}{
		{
			name:     "bare --wait means the default timeout",
			args:     []string{"Ghostty", "--wait"},
			wantWait: defaultWait,
			wantRest: []string{"Ghostty"},
		},
		{
			name:     "--wait=5000 parses its value",
			args:     []string{"Ghostty", "--wait=5000"},
			wantWait: 5000,
			wantRest: []string{"Ghostty"},
		},
		{
			name:     "--wait before the app must not swallow it",
			args:     []string{"--wait", "Ghostty"},
			wantWait: defaultWait,
			wantRest: []string{"Ghostty"},
		},
		{
			name:     "--wait must not swallow any of several positionals",
			args:     []string{"--wait", "Ghostty", "notes.txt"},
			wantWait: defaultWait,
			wantRest: []string{"Ghostty", "notes.txt"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Exactly how runLaunch parses: `wait` is deliberately not declared
			// as a value flag.
			set, err := parseArgs(tc.args)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if !set.Has("wait") {
				t.Fatal("--wait was not recorded at all")
			}
			wait, err := set.Int("wait", defaultWait)
			if err != nil {
				t.Fatalf("--wait: %v", err)
			}
			if wait != tc.wantWait {
				t.Errorf("wait = %d, want %d", wait, tc.wantWait)
			}
			if len(set.Rest) != len(tc.wantRest) {
				t.Fatalf("rest = %v, want %v", set.Rest, tc.wantRest)
			}
			for i, want := range tc.wantRest {
				if set.Rest[i] != want {
					t.Errorf("rest[%d] = %q, want %q", i, set.Rest[i], want)
				}
			}
			// Reject is what surfaces a typo; the flag itself must be accepted.
			if err := set.Reject("wait"); err != nil {
				t.Errorf("Reject rejected the declared flag: %v", err)
			}
		})
	}
}

// The other half of the contract: an absent optional flag is distinguishable
// from a bare one, because `launch Ghostty` must return without waiting.
func TestAbsentOptionalFlagIsNotTheSameAsABareOne(t *testing.T) {
	set, err := parseArgs([]string{"Ghostty"})
	if err != nil {
		t.Fatal(err)
	}
	if set.Has("wait") {
		t.Fatal("--wait was not given; Has must say so")
	}
	if ptr, err := set.IntPtr("wait"); err != nil || ptr != nil {
		t.Fatalf("IntPtr = %v, %v; an absent flag is nil", ptr, err)
	}
}

// A bad explicit value must be a clear error, not a silent fall back to the
// default — an agent that types `--wait=soon` needs to be told.
func TestOptionalFlagWithANonNumericValueIsAnError(t *testing.T) {
	set, err := parseArgs([]string{"Ghostty", "--wait=soon"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Int("wait", 10000); err == nil {
		t.Fatal("want an error for --wait=soon")
	}
}

// The contrast case, kept next to the regression so the distinction stays
// visible: a genuine value flag DOES take the following word.
func TestDeclaredValueFlagStillTakesTheNextWord(t *testing.T) {
	set, err := parseArgs([]string{"--timeout", "5000", "Ghostty"}, "timeout")
	if err != nil {
		t.Fatal(err)
	}
	if got := set.String("timeout", ""); got != "5000" {
		t.Fatalf("timeout = %q", got)
	}
	if len(set.Rest) != 1 || set.Rest[0] != "Ghostty" {
		t.Fatalf("rest = %v", set.Rest)
	}
}
