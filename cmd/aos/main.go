// Command aos is a cross-platform command center for the machine you are
// on — the omarchy CLI's shape, running natively on macOS, Windows, and Linux.
package main

import (
	"os"

	"github.com/muthuishere/agentic-os/internal/cli"
	"github.com/muthuishere/agentic-os/internal/groups"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	registry := cli.NewRegistry()
	groups.Register(registry)

	ctx := cli.NewCtx(registry, version)
	// A virtual display started by `headless start` is this machine's display
	// as far as the GUI commands are concerned, so adopt it before anything
	// checks whether one exists.
	groups.AdoptManagedDisplay(ctx.Env)
	// User extensions, lowest precedence first is irrelevant here: both refuse
	// to shadow a builtin, so the shipped commands always mean what they say.
	cli.LoadAdapters(registry, ctx.Env)
	cli.DiscoverPlugins(registry, ctx.Env)

	os.Exit(cli.Run(ctx, os.Args[1:]))
}
