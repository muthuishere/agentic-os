package cli

import "testing"

// routeRegistry has a group default command, a deep route, and an alias, which
// are the three ways Resolve can reach a command.
func routeRegistry() *Registry {
	r := NewRegistry()
	r.Describe("audio", "Audio")
	r.Describe("launch", "Launch")
	noop := func(*Ctx, []string) error { return nil }
	r.Add(
		&Command{Group: "audio", Name: "output", Summary: "Output", Run: noop},
		&Command{Group: "audio", Name: "output set", Summary: "Set output", Run: noop},
		&Command{Group: "audio", Name: "output set default", Summary: "Default sink", Run: noop},
		&Command{Group: "launch", Summary: "Launch an app", Aliases: []string{"start"}, Run: noop},
	)
	return r
}

// The longest matching prefix wins even when every shorter prefix is also a
// real route: `audio output set default` must not resolve to `audio output`
// with "set default" handed over as positional arguments.
func TestResolveTakesTheLongestPrefixWhenEveryPrefixIsARoute(t *testing.T) {
	cases := []struct {
		args      []string
		wantRoute string
		wantRest  []string
	}{
		{[]string{"audio", "output"}, "audio output", nil},
		{[]string{"audio", "output", "set"}, "audio output set", nil},
		{[]string{"audio", "output", "set", "default"}, "audio output set default", nil},
	}
	for _, tc := range cases {
		cmd, rest, err := routeRegistry().Resolve(tc.args)
		if err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		if cmd.Route() != tc.wantRoute {
			t.Errorf("%v resolved to %q, want %q", tc.args, cmd.Route(), tc.wantRoute)
		}
		if len(rest) != len(tc.wantRest) {
			t.Errorf("%v left rest %v, want %v", tc.args, rest, tc.wantRest)
		}
	}
}

// A positional that happens to repeat a route word must stay a positional: the
// prefix match is over whole words, and it stops at the deepest real route.
func TestResolveKeepsAPositionalThatLooksLikeARouteWord(t *testing.T) {
	cmd, rest, err := routeRegistry().Resolve([]string{"audio", "output", "set", "default", "default"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Route() != "audio output set default" {
		t.Fatalf("route = %q", cmd.Route())
	}
	if len(rest) != 1 || rest[0] != "default" {
		t.Fatalf("rest = %v, want the trailing positional", rest)
	}
}

// A group default command (empty Name) is reached by the bare group word, and
// everything after it is arguments, not route words.
func TestResolveGroupDefaultTakesTheRestAsArguments(t *testing.T) {
	cmd, rest, err := routeRegistry().Resolve([]string{"launch", "Chrome", "--wait"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Route() != "launch" {
		t.Fatalf("route = %q", cmd.Route())
	}
	if len(rest) != 2 || rest[0] != "Chrome" || rest[1] != "--wait" {
		t.Fatalf("rest = %v", rest)
	}
}

// Routes are matched before aliases, so adding a shortcut can never quietly
// take over a real route with the same first word.
func TestResolvePrefersARouteOverAnAlias(t *testing.T) {
	r := routeRegistry()
	r.Describe("start", "A real group")
	r.Add(&Command{Group: "start", Name: "here", Summary: "Real route", Run: func(*Ctx, []string) error { return nil }})

	cmd, _, err := r.Resolve([]string{"start", "here"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Route() != "start here" {
		t.Fatalf("alias shadowed the longer route: got %q", cmd.Route())
	}

	// The alias still works on its own.
	cmd, rest, err := r.Resolve([]string{"start", "Chrome"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Route() != "launch" || len(rest) != 1 || rest[0] != "Chrome" {
		t.Fatalf("alias route = %q rest = %v", cmd.Route(), rest)
	}
}

// A flag never becomes part of a route, however early it appears.
func TestResolveDoesNotTreatFlagsAsRouteWords(t *testing.T) {
	cmd, rest, err := routeRegistry().Resolve([]string{"audio", "output", "--json", "set"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Route() != "audio output" {
		t.Fatalf("route = %q", cmd.Route())
	}
	if len(rest) != 2 || rest[0] != "--json" {
		t.Fatalf("rest = %v", rest)
	}
}
