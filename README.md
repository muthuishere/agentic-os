# agentic-os

**A computer-use MCP server that is also a CLI you can type — on macOS, Windows,
and Linux.**

Give an agent your machine: windows, input, screenshots, files, processes,
packages, network, audio, messaging. Every command is an MCP tool, and every MCP
tool is a command a person can run — one surface, one code path, so what you
test by hand is exactly what the agent gets.

Two things the [2026 survey of desktop-automation MCP servers](https://chatforest.com/guides/best-desktop-automation-mcp-servers/)
found missing across 25+ of them, and what this is built around:

**Safety is structural, not a disclaimer.** Commands that need a screen are
refused with a reason on a headless box instead of failing somewhere deep.
`file delete` will not touch a filesystem root, `$HOME`, or a system directory.
`serve` binds loopback. Services are namespaced so removing one can never reach
a system service. Telemetry records how many arguments a command got, never what
they were.

**One verb set that actually means the same thing everywhere.** `pkg install`
over brew, winget, scoop, choco, apt, pacman, yay and dnf. `--app=Chrome`
matching the same window on all three platforms — which took fixing Windows and
working around Linux to make true.

And it runs where there is no screen at all: most of the surface needs no
display, and on Linux `headless start` provides one when something does.

```console
$ aos system info
$ aos window move Chrome --zone=1B
$ aos capture screenshot --app=Chrome --out=/tmp/shot.png
$ aos pkg install ripgrep          # brew / winget / pacman / apt, same verb
$ aos obs audit --since=24h            # what an agent did on this machine
$ aos serve mcp                    # now every command above is an agent tool
```

## Install

No Go? The script installs it first, then builds, then installs the agent skill.
Everything lands in your home directory; nothing needs root.

```sh
curl -fsSL https://raw.githubusercontent.com/muthuishere/agentic-os/main/install.sh | sh
```

Reading a script before piping it into a shell is the right instinct:

```sh
curl -fsSL https://raw.githubusercontent.com/muthuishere/agentic-os/main/install.sh -o install.sh
less install.sh && sh install.sh
```

With Go already installed:

```sh
go install github.com/muthuishere/agentic-os/cmd/aos@latest
aos install --skills # teach Claude Code and other agents to use it
```

`install --skills` writes the agent skill bundled inside the binary to
both `~/.claude/skills/` and `~/.agents/skills/`, so the instructions an agent
reads can never describe a different version than the one installed. `skill show`
prints it without installing; `uninstall --skills` removes it.

One binary, no runtime and no package manager of its own. The macOS window
backend is CoreGraphics through cgo, so a Mac build needs `CGO_ENABLED=1` — the
default when building natively. Linux and Windows are pure Go and cross-compile
from anywhere.

## Agents connect over MCP

```sh
aos serve mcp
# aos mcp  http://127.0.0.1:14320/mcp
# token    9c23b0efb0a002fe536a579adfd7cdb1
# connect  claude mcp add --transport http aos http://127.0.0.1:14320/mcp \
#            --header "Authorization: Bearer 9c23b0efb0a002fe536a579adfd7cdb1"
```

Loopback is not a permission — every process on the machine shares it, and this
surface drives windows, input and the filesystem. The server mints a token per
run and refuses anything without it, as a bearer header or `?t=`. Set
`AGENTIC_OS_MCP_TOKEN` (or `--token`) to pin one so a client configured once
survives a restart.

Every non-hidden, non-blocking command on this platform becomes one MCP tool
(`window_move`, `capture_screenshot`, `exec_capture`, …) carrying the same
summary, usage, and examples the CLI documents. There is no second
implementation: a tool call runs the identical `Runner` the terminal does.

```sh
aos serve tools                    # preview the catalogue
aos serve mcp --groups=window,capture,exec   # expose a subset
```

The MCP layer is [toolnexus](https://github.com/muthuishere/toolnexus)'s
`Toolkit.Serve`, so the protocol work is conformance-tested rather than
hand-rolled. It binds loopback by default — serving this registry is remote
control of the machine, so reaching it from elsewhere should be a deliberate act
(an SSH tunnel, or an explicit `--addr`).

## Keeping it running

A server an agent connects to should not depend on a terminal staying open, so
any command — `serve mcp` above all — can be handed to the machine's own service
manager:

```sh
aos service create mcp --autostart --now -- aos serve mcp
aos service status mcp     # exits non-zero unless it is running
aos service list
aos service remove mcp
```

That is a launchd user agent on macOS, a `systemd --user` unit on Linux, and a
Scheduled Task on Windows — per-user throughout, so nothing here asks for admin
or sudo. Every service aos creates is namespaced `agentic-os.<name>`, and
`list` and `remove` only ever see that namespace: this CLI cannot delete a
service it did not create.

## Headless machines

Most of aos needs no screen at all. On a server, in CI, or in a container,
`exec`, `file`, `msg`, `pkg`, `network`, `system`, `power`, `battery`, `font`,
`debug`, and `serve` work exactly as they do on a laptop.

The desktop commands do need a display, and they say so rather than failing deep
inside a display-server call:

```console
$ aos window list
agentic-os: "window list" needs a display, and this machine has none
Start one with `aos headless start`, or check `aos headless status`.
```

On Linux, give it one:

```sh
aos headless start --size=1920x1080 --wm   # Xvfb + a lightweight WM
aos window list                            # the display is adopted automatically
aos headless stop
```

`headless start` waits for the window manager to claim the screen before
returning, so the next command works instead of losing a race with it. Later runs
of `aos` adopt the managed display on their own — no `export DISPLAY`. An
environment that already names a display always wins, so this never redirects a
real session.

macOS and Windows have no Xvfb equivalent: there, the desktop commands need a
real logged-in session, and `headless status` says so.

### Knowing what will work

Every command declares whether it needs a display, and the whole CLI respects it:

```sh
aos headless status       # is there a display, and where did it come from
aos commands              # screenless machines list only what will run
aos commands --all        # `g` marks a command waiting on a display
aos commands --json       # each entry carries "needs_display"
aos serve mcp --gui=off   # expose only the screenless tools
```

`serve` defaults to `--gui=auto`: GUI tools appear only when this machine
actually has a display. Offering an agent a tool that cannot run is worse than
not offering it — it will try, fail, and try again.

## Command groups

| | |
|---|---|
| **Desktop** | `window` (list · focus · move to zone/split/coords · resize · wait · arrange a saved layout), `display`, `mouse`, `key`, `permission` |
| **Machine** | `system` (lock · sleep · restart · shutdown · logout · info), `power`, `battery`, `network`, `audio`, `font`, `pkg`, `debug` |
| **Content** | `capture`, `clipboard`, `file`, `exec`, `open`, `launch`, `webapp`, `subtitle` |
| **Comms** | `msg` — send, poll, and follow the local messenger hub |
| **Remote** | `remote` — hand this screen and its input to a browser on the LAN, for as long as the command runs |
| **Watch** | `watch` — long-running monitors (clipboard, focused window) that print one JSON line per change |
| **Agents** | `serve` — expose all of the above over MCP; `service` — keep any of it running as a per-user OS service; `headless` — run the desktop commands with no screen |

Windows, monitors, input and screenshots are handled by
[windowctl](https://github.com/muthuishere/windowctl); MCP serving by
[toolnexus](https://github.com/muthuishere/toolnexus). Both are libraries this
depends on — aos is the front door.

## Reacting to what happens

`watch` commands run until interrupted and print one compact JSON line per
event, so an agent can pipe one and act on each line.

```sh
aos watch clipboard              # {"event":"clipboard","at":…,"seq":1,"length":34,"digest":"7a37ded45b5e"}
aos watch clipboard --content    # include the text — opt-in, clipboards hold passwords
aos watch window --max=1         # wait for the next focus change, then exit
```

The first sample is only a baseline, so `--max=1` means "the next change", which
is also what makes a watcher scriptable. Being blocking, they are deliberately
absent from the MCP tool list — an agent reads them by running the CLI.

## How it is put together

| Layer | Where | What it does |
|---|---|---|
| Registry | `internal/cli` | Command tree, longest-prefix route resolution, help, `commands --json`, `Invoke` |
| Plugins | `internal/cli/plugins.go` | Discovers external `aos-<route>` executables |
| Platform helpers | `internal/sys` | Bounded shell-outs, `osascript`, PowerShell |
| Groups | `internal/groups` | One file per group, plus `_darwin.go` / `_windows.go` / `_linux.go` backends |

A group declares its commands once, cross-platform; each backend file supplies
the machine-specific work. A command that genuinely cannot exist somewhere sets
`Platforms` — it still shows in help, marked, and exits 2 with a clear message
rather than pretending not to exist.

### Routes are multi-token

Like omarchy, a route is as many words as it needs: `audio output set default`
resolves by longest prefix, and everything after it is the command's own args.

### Adapters — add a command without writing a program

Most additions are a command line someone already runs. Drop a JSON file in
`~/.config/agentic-os/adapters/` and it becomes a group:

```json
{
  "group": "notes",
  "description": "Personal notes",
  "commands": [
    { "name": "today", "summary": "Open today's note",
      "run": "$EDITOR ~/notes/$(date +%F).md", "platforms": ["darwin", "linux"] }
  ]
}
```

```sh
aos adapters example --write   # a working starter file
aos adapters list              # what this machine has loaded
aos adapters path
```

`run` goes through the platform shell with the user's arguments appended and
quoted. A command can declare `platforms`, `needsDisplay`, and `blocking`, so an
adapter is gated and excluded from the MCP tool list on exactly the same terms
as a built-in one — and adapter commands become MCP tools automatically.

An adapter can never shadow a built-in command: a file in the config directory
must not be able to change what a shipped command does. A malformed adapter is
reported by `commands --check` rather than silently ignored.

### Plugins — when you need a program

Any executable named `aos-<group>-<name>` in `$AGENTIC_OS_BIN_DIR`,
`~/.config/agentic-os/bin`, or on `PATH` joins the CLI — and therefore the MCP
tool list. It describes itself with comment headers in its first 80 lines:

```sh
#!/bin/sh
# aos:summary=Do the thing
# agentic-os:args=<target>
# agentic-os:examples=aos demo do thing a | aos demo do thing b
# agentic-os:platforms=darwin | linux
# agentic-os:route=theme bg-switcher   # when a word contains a hyphen
```

Built-in commands win over a plugin with the same route, so shipping a Go
implementation transparently replaces a script.

## Introspection

```sh
aos commands            # what this machine can run
aos commands --all      # every registered command, including other platforms
aos commands --json     # machine-readable index
aos commands --check    # registry lint; non-zero when something is off
aos debug               # platform, tool availability, plugin dirs
```

## Is it working?

```sh
aos doctor          # functional checks, each with a fix
```

`doctor` does not check whether binaries are on PATH — it round-trips the
clipboard, captures a real screenshot and inspects the bytes, asks the window
backend for a list, and pings the messenger hub. Anything not `ok` prints what
to run to fix it.

## Audit trail

Every invocation — typed at a terminal or called as an MCP tool — records one
OpenTelemetry-shaped span, locally. When you hand an agent your machine, the
question you will eventually want answered is *what did it actually do*:

```sh
aos obs audit --since=24h        # everything, oldest first
aos obs audit --source=mcp       # only what agents asked for
aos obs audit --failed           # only what went wrong
```

```
2026-09-01 01:10:28  mcp   window move                    41ms  ok
2026-09-01 01:10:33  cli   file delete                     2ms  FAILED exit 1
```

And whether it is earning its place:

```sh
aos obs summary                  # calls, routes, cli vs mcp, most used
aos obs stats                    # p50/p95 and failure rate per route
aos obs tail --limit=20          # the last few
aos obs export --since=1h        # OTLP JSON, ready for a collector
aos obs path                     # where it is written
```

Spans record the route, source (`cli` or `mcp`), exit code and duration — and
the *count* of arguments, never their contents, which routinely carry paths and
message bodies. Set `AGENTIC_OS_TELEMETRY=off` to disable, or
`AGENTIC_OS_TELEMETRY_DIR` to relocate.

## Testing on real machines

Unit tests cover the registry and parsing. They cannot tell you whether a
screenshot on Windows is blank or whether a headless server refuses a GUI
command properly, so there is a suite that drives the built binaries on actual
machines — locally, over SSH, and through an agentbus node.

```sh
cp test/e2e/config.example.json test/e2e/config.json   # gitignored; name your machines
task e2e
```

Every step has its own timeout, because the failure that hurts is a hang, not a
wrong answer. On a machine with `hasDisplay: false` the suite asserts the
*refusal* — `window list` must exit 2 saying it needs a display — rather than
skipping quietly.

## Credit, and why this exists

The command shape is [omarchy](https://omarchy.org)'s — DHH's opinionated
Arch + Hyprland desktop. Its CLI is the best system command centre I have used:
`<group> <command>` routes, help that is actually discoverable, a
machine-readable command index, and plugins that are just executables with a
comment header. All of that is theirs, and it is worth studying directly.

I am not on Arch. I work across macOS, Windows and a pile of Linux servers, and
I wanted those ergonomics on all of them. **If I move to omarchy full time I
will use omarchy's CLI — it is better on the machine it was built for, and it
has years of taste in it that this does not.** Until then, this is how I get the
same shape everywhere else, plus the parts an agent needs that a desktop CLI
never had to care about.

This is not a port. Roughly half of omarchy's surface is Arch- and
Hyprland-specific and belongs exactly where it is.

### What omarchy has that this does not

`theme` (32 commands — the whole theming system, and the reason omarchy looks
the way it does), `hyprland`, `plymouth`, `drive` and LUKS, `powerprofiles`,
`hibernation`, `snapshot`, the omarchy `bar` / `plugin` / `shell`, its `menu`
and `tui` launchers, `install` of 33 Arch packages, `mise`, `tailscale`,
`branch`, `channel`, `migrate`. On a Linux box you can drop omarchy's own
scripts in as `aos-*` plugins and they will show up here.

### What this has that omarchy does not

Windows and macOS at all. An MCP server, so an agent can drive the machine.
JSON adapters. An audit trail and usage stats. `doctor`. A LAN screen share.
Headless operation with a managed virtual display. Cross-platform services.

`docs/porting.md` maps the omarchy 4.0.0.alpha surface group by group.

