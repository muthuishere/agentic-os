package groups

import (
	"strings"
	"testing"
)

func TestServiceNameNormalizesAndValidates(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"mcp", "mcp"},
		{"agentic-os.mcp", "mcp"},
		{"  serve.mcp  ", "serve.mcp"},
		{"a_b-c.1", "a_b-c.1"},
	} {
		got, err := serviceName(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("serviceName(%q) = %q, %v", tc.in, got, err)
		}
	}

	for _, bad := range []string{"", "   ", "agentic-os.", "../evil", ".hidden", "with space", "sl/ash", "na;me"} {
		if got, err := serviceName(bad); err == nil {
			t.Fatalf("serviceName(%q) accepted as %q", bad, got)
		}
	}
}

func TestServiceLabelRoundTripStaysInNamespace(t *testing.T) {
	label := serviceLabel("mcp")
	if label != "agentic-os.mcp" {
		t.Fatalf("label = %q", label)
	}
	if name, ok := serviceShortName(label); !ok || name != "mcp" {
		t.Fatalf("shortName = %q, %v", name, ok)
	}
	// The listing filter is the safety net: nothing outside the namespace may
	// ever be claimed as ours.
	for _, foreign := range []string{"com.apple.Safari", "agentic-os", "agentic-os.", "ssh-agent"} {
		if name, ok := serviceShortName(foreign); ok {
			t.Fatalf("serviceShortName(%q) claimed %q", foreign, name)
		}
	}
}

func TestSplitServiceCommand(t *testing.T) {
	head, command := splitServiceCommand([]string{"mcp", "--now", "--", "agentic-os", "serve", "mcp", "--addr=:1"})
	if strings.Join(head, " ") != "mcp --now" {
		t.Fatalf("head = %v", head)
	}
	if strings.Join(command, " ") != "agentic-os serve mcp --addr=:1" {
		t.Fatalf("command = %v", command)
	}

	head, command = splitServiceCommand([]string{"nap", "/bin/sleep", "60"})
	if len(command) != 0 || strings.Join(head, " ") != "nap /bin/sleep 60" {
		t.Fatalf("head = %v, command = %v", head, command)
	}

	// A separator with nothing after it is an empty command, not a missing one.
	head, command = splitServiceCommand([]string{"nap", "--"})
	if len(head) != 1 || command == nil || len(command) != 0 {
		t.Fatalf("head = %v, command = %v", head, command)
	}
}

func testSpec() serviceSpec {
	return serviceSpec{
		Name:      "nap",
		Label:     "agentic-os.nap",
		Command:   []string{"/bin/sleep", "60", `say "hi" & bye`},
		Dir:       "/tmp",
		Autostart: true,
		OutLog:    "/tmp/nap.out.log",
		ErrLog:    "/tmp/nap.err.log",
	}
}

func TestRenderLaunchdPlist(t *testing.T) {
	plist := renderLaunchdPlist(testSpec())
	for _, want := range []string{
		"<string>agentic-os.nap</string>",
		"<string>/bin/sleep</string>",
		"<string>60</string>",
		"<key>WorkingDirectory</key>",
		"<string>/tmp</string>",
		"<key>RunAtLoad</key>\n\t<true/>",
		"<key>KeepAlive</key>\n\t<false/>",
	} {
		if !strings.Contains(plist, want) {
			t.Fatalf("plist missing %q:\n%s", want, plist)
		}
	}
	// XML metacharacters in an argument must not break the document.
	if !strings.Contains(plist, "&amp;") || strings.Contains(plist, `& bye`) {
		t.Fatalf("argument was not escaped:\n%s", plist)
	}

	spec := testSpec()
	spec.Autostart = false
	if !strings.Contains(renderLaunchdPlist(spec), "<key>RunAtLoad</key>\n\t<false/>") {
		t.Fatal("RunAtLoad not cleared when autostart is off")
	}
}

func TestRenderSystemdUnit(t *testing.T) {
	unit := renderSystemdUnit(testSpec())
	for _, want := range []string{
		"Description=agentic-os service nap",
		"Type=simple",
		`ExecStart="/bin/sleep" "60" "say \"hi\" & bye"`,
		"WorkingDirectory=/tmp",
		"StandardOutput=append:/tmp/nap.out.log",
		"WantedBy=default.target",
	} {
		if !strings.Contains(unit, want) {
			t.Fatalf("unit missing %q:\n%s", want, unit)
		}
	}
}

func TestSchtasksCommandQuotesOnlyWhatNeedsIt(t *testing.T) {
	got := schtasksCommand([]string{`C:\Program Files\agentic-os.exe`, "serve", "mcp", `--addr="x"`})
	want := `"C:\Program Files\agentic-os.exe" serve mcp "--addr=\"x\""`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
