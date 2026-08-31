package groups

import "testing"

// The docs, the examples and the bundled agent skill all say `--app=Chrome`,
// because that is how people refer to the browser. macOS reports it as
// "Google Chrome", and the window backend matches App exactly and
// case-sensitively, so the documented form matched nothing at all until the
// front door started resolving it.
func TestPickAppName(t *testing.T) {
	macOS := []string{"Google Chrome", "Ghostty", "Finder", "Visual Studio Code"}

	cases := []struct {
		name  string
		names []string
		query string
		want  string
		ok    bool
	}{
		{"the bug: a short name people actually type", macOS, "Chrome", "Google Chrome", true},
		{"case does not matter", macOS, "chrome", "Google Chrome", true},
		{"a full name still works", macOS, "Google Chrome", "Google Chrome", true},
		{"surrounding space is not the user's mistake to pay for", macOS, "  chrome ", "Google Chrome", true},
		{"a word from the middle", macOS, "Studio", "Visual Studio Code", true},
		{"no match stays a miss, rather than guessing", macOS, "Safari", "", false},
		{"empty is not a wildcard", macOS, "", "", false},
		{"nothing open", nil, "Chrome", "", false},

		// An exact hit has to win, or naming the shorter of two similar apps
		// would select the longer one.
		{"exact beats substring", []string{"Code - Insiders", "Code"}, "Code", "Code", true},
		{"exact beats substring, either order", []string{"Code", "Code - Insiders"}, "Code", "Code", true},
		{"exact wins case-insensitively too", []string{"code - insiders", "Code"}, "CODE", "Code", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := pickAppName(tc.names, tc.query)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("pickAppName(%q) = %q, %v; want %q, %v", tc.query, got, ok, tc.want, tc.ok)
			}
		})
	}
}
