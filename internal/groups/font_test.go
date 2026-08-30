package groups

import "testing"

func TestFontFamilyStripsStyles(t *testing.T) {
	cases := map[string]string{
		"JetBrainsMono-BoldItalic.ttf":      "JetBrainsMono",
		"JetBrainsMonoNerdFont-Regular.ttf": "JetBrainsMonoNerdFont",
		"Inter_ExtraLightItalic.otf":        "Inter",
		"SF-Pro-Text-Semibold.otf":          "SF Pro Text",
		"Andale Mono.ttf":                   "Andale Mono",
		"Bold.ttf":                          "Bold",
	}
	for input, want := range cases {
		if got := fontFamily(input); got != want {
			t.Errorf("fontFamily(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMergeKeyValuesSkipsBlanks(t *testing.T) {
	facts := map[string]string{"platform": "test"}
	mergeKeyValues(facts, "os=Windows\n\nversion=\n  build = 123 \nnot-a-pair\n")
	if facts["os"] != "Windows" || facts["build"] != "123" {
		t.Fatalf("bad merge: %v", facts)
	}
	if _, ok := facts["version"]; ok {
		t.Fatalf("empty value should be skipped: %v", facts)
	}
}
