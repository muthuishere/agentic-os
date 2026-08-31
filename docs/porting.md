# Porting map — omarchy 4.0.0.alpha → aos

Generated from `docs/omarchy-surface.json`, which is `omarchy commands --all --json`
captured from the reference tree.

Omarchy ships **438 commands across 74 groups**. Status is per *group*:

- **done** — a cross-platform equivalent exists here, with macOS, Windows and Linux backends.
- **planned** — portable in principle, not written yet.
- **linux-only** — bound to Arch, Hyprland, or omarchy's own shell and theming.
  These stay where they belong. On a Linux box you can drop omarchy's own scripts
  in as `aos-*` plugins and they appear in this CLI unchanged.

Totals: 17 done, 3 planned, 54 linux-only.

Note this map only covers the omarchy surface. Much of what `aos` does — MCP
serving, adapters, the audit trail, `doctor`, remote share, headless management,
cross-platform services — has no omarchy counterpart and is not listed here.

| group | omarchy commands | status |
|---|--:|---|
| `agent` | 7 | linux-only |
| `apply` | 3 | linux-only |
| `ascii` | 1 | linux-only |
| `audio` | 9 | done |
| `bar` | 2 | linux-only |
| `battery` | 3 | done |
| `bluetooth` | 2 | planned |
| `branding` | 3 | linux-only |
| `brightness` | 5 | planned |
| `capture` | 8 | done |
| `channel` | 2 | linux-only |
| `chromium` | 2 | linux-only |
| `clipboard` | 3 | done |
| `cmd` | 3 | linux-only |
| `crash` | 2 | linux-only |
| `debug` | 2 | done |
| `default` | 4 | linux-only |
| `dev` | 11 | linux-only |
| `disk` | 1 | linux-only |
| `display` | 1 | done |
| `dns` | 1 | linux-only |
| `done` | 1 | linux-only |
| `drive` | 3 | linux-only |
| `file` | 1 | done |
| `font` | 3 | done |
| `games` | 2 | linux-only |
| `git` | 1 | linux-only |
| `hibernation` | 3 | linux-only |
| `hook` | 2 | linux-only |
| `hw` | 27 | linux-only |
| `hyprland` | 24 | linux-only |
| `install` | 33 | done |
| `installed` | 2 | linux-only |
| `launch` | 23 | done |
| `menu` | 13 | linux-only |
| `migrate` | 2 | linux-only |
| `mise` | 1 | linux-only |
| `monitor` | 1 | linux-only |
| `network` | 5 | done |
| `notification` | 6 | done |
| `osd` | 1 | linux-only |
| `pkg` | 9 | done |
| `plugin` | 9 | linux-only |
| `plymouth` | 7 | linux-only |
| `power` | 1 | done |
| `powerprofiles` | 3 | linux-only |
| `provision` | 3 | linux-only |
| `refresh` | 12 | linux-only |
| `reinstall` | 3 | linux-only |
| `reminder` | 1 | linux-only |
| `remove` | 24 | linux-only |
| `restart` | 16 | linux-only |
| `screensaver` | 1 | linux-only |
| `setup` | 6 | linux-only |
| `share` | 1 | linux-only |
| `shell` | 2 | linux-only |
| `show` | 2 | linux-only |
| `snapshot` | 1 | linux-only |
| `state` | 1 | linux-only |
| `sudo` | 3 | linux-only |
| `system` | 10 | done |
| `tailscale` | 2 | linux-only |
| `theme` | 32 | linux-only |
| `toggle` | 13 | planned |
| `transcode` | 2 | linux-only |
| `tui` | 3 | linux-only |
| `update` | 21 | linux-only |
| `upgrade` | 1 | linux-only |
| `upload` | 1 | linux-only |
| `version` | 4 | done |
| `voxtype` | 5 | linux-only |
| `weather` | 3 | linux-only |
| `webapp` | 5 | done |
| `windows` | 2 | linux-only |
