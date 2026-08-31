# agentic-os

**One CLI for the machine you are sitting at — and the API other agents connect to.**

It started as the [omarchy CLI](https://learn.omacom.io/2/the-omarchy-manual/115/omarchy-cli)'s
shape (`<group> <command>`, discoverable help, a machine-readable command index,
drop-in plugins) rebuilt in Go so it runs the same on **macOS, Windows, and Linux**.
Then every command became an **MCP tool**, so any agent — Claude Code, Codex, or
your own — can drive the machine through the exact code path you type at a prompt.

```console
$ agentic-os system info
$ agentic-os window move Ghostty --zone=1B
$ agentic-os capture screenshot --app=Chrome --out=/tmp/shot.png
$ agentic-os pkg install ripgrep          # brew / winget / pacman / apt, same verb
$ agentic-os msg send --channel=ops "build passed"
$ agentic-os serve mcp                    # now every command above is an agent tool
```

## Install

```sh
go install github.com/muthuishere/agentic-os/cmd/agentic-os@latest
```

One binary, no runtime and no package manager of its own. The macOS window
backend is CoreGraphics through cgo, so a Mac build needs `CGO_ENABLED=1` — the
default when building natively. Linux and Windows are pure Go and cross-compile
from anywhere.

## Agents connect over MCP

```sh
agentic-os serve mcp                      # streamable HTTP at 127.0.0.1:14320/mcp
claude mcp add --transport http agentic-os http://127.0.0.1:14320/mcp
```

Every non-hidden, non-blocking command on this platform becomes one MCP tool
(`window_move`, `capture_screenshot`, `exec_capture`, …) carrying the same
summary, usage, and examples the CLI documents. There is no second
implementation: a tool call runs the identical `Runner` the terminal does.

```sh
agentic-os serve tools                    # preview the catalogue
agentic-os serve mcp --groups=window,capture,exec   # expose a subset
```

The MCP layer is [toolnexus](https://github.com/muthuishere/toolnexus)'s
`Toolkit.Serve`, so the protocol work is conformance-tested rather than
hand-rolled. It binds loopback by default — serving this registry is remote
control of the machine, so reaching it from elsewhere should be a deliberate act
(an SSH tunnel, or an explicit `--addr`).

## Headless machines

Most of agentic-os needs no screen at all. On a server, in CI, or in a container,
`exec`, `file`, `msg`, `pkg`, `network`, `system`, `power`, `battery`, `font`,
`debug`, and `serve` work exactly as they do on a laptop.

The desktop commands do need a display, and they say so rather than failing deep
inside a display-server call:

```console
$ agentic-os window list
agentic-os: "window list" needs a display, and this machine has none
Start one with `agentic-os headless start`, or check `agentic-os headless status`.
```

On Linux, give it one:

```sh
agentic-os headless start --size=1920x1080 --wm   # Xvfb + a lightweight WM
agentic-os window list                            # the display is adopted automatically
agentic-os headless stop
```

`headless start` waits for the window manager to claim the screen before
returning, so the next command works instead of losing a race with it. Later runs
of `agentic-os` adopt the managed display on their own — no `export DISPLAY`. An
environment that already names a display always wins, so this never redirects a
real session.

macOS and Windows have no Xvfb equivalent: there, the desktop commands need a
real logged-in session, and `headless status` says so.

### Knowing what will work

Every command declares whether it needs a display, and the whole CLI respects it:

```sh
agentic-os headless status       # is there a display, and where did it come from
agentic-os commands              # screenless machines list only what will run
agentic-os commands --all        # `g` marks a command waiting on a display
agentic-os commands --json       # each entry carries "needs_display"
agentic-os serve mcp --gui=off   # expose only the screenless tools
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
| **Watch** | `watch` — long-running monitors (clipboard, focused window) that print one JSON line per change |
| **Agents** | `serve` — expose all of the above over MCP; `headless` — run the desktop commands with no screen |

Window, monitor, screenshot, and input work is
[windowctl](https://github.com/muthuishere/windowctl), which already solved it on
all three platforms; agentic-os is the command surface over it.

## Reacting to what happens

`watch` commands run until interrupted and print one compact JSON line per
event, so an agent can pipe one and act on each line.

```sh
agentic-os watch clipboard              # {"event":"clipboard","at":…,"seq":1,"length":34,"digest":"7a37ded45b5e"}
agentic-os watch clipboard --content    # include the text — opt-in, clipboards hold passwords
agentic-os watch window --max=1         # wait for the next focus change, then exit
```

The first sample is only a baseline, so `--max=1` means "the next change", which
is also what makes a watcher scriptable. Being blocking, they are deliberately
absent from the MCP tool list — an agent reads them by running the CLI.

## How it is put together

| Layer | Where | What it does |
|---|---|---|
| Registry | `internal/cli` | Command tree, longest-prefix route resolution, help, `commands --json`, `Invoke` |
| Plugins | `internal/cli/plugins.go` | Discovers external `agentic-os-<route>` executables |
| Platform helpers | `internal/sys` | Bounded shell-outs, `osascript`, PowerShell |
| Groups | `internal/groups` | One file per group, plus `_darwin.go` / `_windows.go` / `_linux.go` backends |

A group declares its commands once, cross-platform; each backend file supplies
the machine-specific work. A command that genuinely cannot exist somewhere sets
`Platforms` — it still shows in help, marked, and exits 2 with a clear message
rather than pretending not to exist.

### Routes are multi-token

Like omarchy, a route is as many words as it needs: `audio output set default`
resolves by longest prefix, and everything after it is the command's own args.

### Plugins

Any executable named `agentic-os-<group>-<name>` in `$AGENTIC_OS_BIN_DIR`,
`~/.config/agentic-os/bin`, or on `PATH` joins the CLI — and therefore the MCP
tool list. It describes itself with comment headers in its first 80 lines:

```sh
#!/bin/sh
# agentic-os:summary=Do the thing
# agentic-os:args=<target>
# agentic-os:examples=agentic-os demo do thing a | agentic-os demo do thing b
# agentic-os:platforms=darwin | linux
# agentic-os:route=theme bg-switcher   # when a word contains a hyphen
```

Built-in commands win over a plugin with the same route, so shipping a Go
implementation transparently replaces a script.

## Introspection

```sh
agentic-os commands            # what this machine can run
agentic-os commands --all      # every registered command, including other platforms
agentic-os commands --json     # machine-readable index
agentic-os commands --check    # registry lint; non-zero when something is off
agentic-os debug               # platform, tool availability, plugin dirs
```

## Is it working?

```sh
agentic-os doctor          # functional checks, each with a fix
```

`doctor` does not check whether binaries are on PATH — it round-trips the
clipboard, captures a real screenshot and inspects the bytes, asks the window
backend for a list, and pings the messenger hub. Anything not `ok` prints what
to run to fix it.

## What has this machine been asked to do?

Every invocation — typed at a terminal or called as an MCP tool — records one
OpenTelemetry-shaped span. Nothing is sent anywhere; the record simply exists,
in the right shape, for the day you want it.

```sh
agentic-os obs stats                 # calls, failures, p50/p95 per route
agentic-os obs tail --limit=20       # the most recent work
agentic-os obs export --since=1h     # OTLP JSON, ready for a collector
agentic-os obs path                  # where it is written
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

## Status

`docs/porting.md` maps the omarchy 4.0.0.alpha surface group by group. Everything
outside that map — `window`, `mouse`, `key`, `exec`, `file`, `msg`, `serve` — is
new here: the parts an agent needs that a desktop CLI never had.
