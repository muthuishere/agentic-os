# Porting map — omarchy 4.0.0.alpha → agentic-os

Generated from `docs/omarchy-surface.json`, which is `omarchy commands --all --json`
captured from the reference tree.

Omarchy ships **438 commands across 74 groups**. Status here is per *group*:

- **done** — a cross-platform equivalent exists, with macOS, Windows, and Linux backends.
- **planned** — portable in principle, not written yet.
- **linux-only** — the concept is bound to Arch, Hyprland, or Wayland. These stay
  out of the Go binary; if you want them on a Linux box, drop the original scripts
  in as `agentic-os-*` plugins.

Totals: 14 done, 41 planned, 19 linux-only.

| group | omarchy commands | status |
|---|--:|---|
| `agent` | 7 | planned |
| `apply` | 3 | planned |
| `ascii` | 1 | planned |
| `audio` | 9 | done |
| `bar` | 2 | linux-only |
| `battery` | 3 | done |
| `bluetooth` | 2 | planned |
| `branding` | 3 | planned |
| `brightness` | 5 | planned |
| `capture` | 8 | done |
| `channel` | 2 | linux-only |
| `chromium` | 2 | planned |
| `clipboard` | 3 | done |
| `cmd` | 3 | planned |
| `crash` | 2 | planned |
| `debug` | 2 | done |
| `default` | 4 | planned |
| `dev` | 11 | planned |
| `disk` | 1 | planned |
| `display` | 1 | done |
| `dns` | 1 | planned |
| `done` | 1 | planned |
| `drive` | 3 | linux-only |
| `file` | 1 | planned |
| `font` | 3 | done |
| `games` | 2 | linux-only |
| `git` | 1 | planned |
| `hibernation` | 3 | linux-only |
| `hook` | 2 | planned |
| `hw` | 27 | planned |
| `hyprland` | 24 | linux-only |
| `install` | 33 | planned |
| `installed` | 2 | planned |
| `launch` | 23 | done |
| `menu` | 13 | planned |
| `migrate` | 2 | linux-only |
| `mise` | 1 | linux-only |
| `monitor` | 1 | planned |
| `network` | 5 | done |
| `notification` | 6 | planned |
| `osd` | 1 | linux-only |
| `pkg` | 9 | done |
| `plugin` | 9 | linux-only |
| `plymouth` | 7 | linux-only |
| `power` | 1 | done |
| `powerprofiles` | 3 | linux-only |
| `provision` | 3 | linux-only |
| `refresh` | 12 | planned |
| `reinstall` | 3 | planned |
| `reminder` | 1 | planned |
| `remove` | 24 | planned |
| `restart` | 16 | planned |
| `screensaver` | 1 | planned |
| `setup` | 6 | planned |
| `share` | 1 | planned |
| `shell` | 2 | linux-only |
| `show` | 2 | planned |
| `snapshot` | 1 | linux-only |
| `state` | 1 | planned |
| `sudo` | 3 | linux-only |
| `system` | 10 | done |
| `tailscale` | 2 | linux-only |
| `theme` | 32 | planned |
| `toggle` | 13 | planned |
| `transcode` | 2 | planned |
| `tui` | 3 | planned |
| `update` | 21 | planned |
| `upgrade` | 1 | planned |
| `upload` | 1 | planned |
| `version` | 4 | done |
| `voxtype` | 5 | linux-only |
| `weather` | 3 | planned |
| `webapp` | 5 | done |
| `windows` | 2 | linux-only |
