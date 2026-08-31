# LinkedIn — aos launch

Status: **DRAFT, not posted.** Publishing is owner-gated.
Repo: https://github.com/muthuishere/agentic-os
Site: https://muthuishere.github.io/agentic-os/

---

## Main post (recommended)

Every agent can write code. Almost none of them can use the computer.

They can't move a window, take a screenshot, read the battery, install a
package, or find out whether the machine even has a screen. So we hand them a
raw shell and hope.

I spent the last stretch building **aos** — one CLI over the machine, 95
commands across 33 groups, identical on macOS, Windows and Linux.

The part I'd push back on if someone else built it: **it leads with an agent
skill, not MCP.**

Everyone is shipping MCP servers. But an agent already has a shell. What it
lacks isn't a protocol — it's knowing the CLI exists, what the exit codes mean,
and which commands need a screen. That's a document, not a server.

So `aos install --skills` writes a skill that ships *inside* the binary. No
port. No process to supervise. Nothing to restart when the laptop wakes up. And
because it's instructions rather than a connection, it works over `ssh` into any
machine that has the binary — which is the whole story on servers.

MCP is still there, one command away, for agents that want typed tools. Same
routes, same code path. It's a preference, not a trade-off.

Three things I decided differently:

→ **Safety is structural.** A command that needs a screen is refused with a
reason on a headless box, not failed deep inside a display call. `file delete`
won't touch a filesystem root, `$HOME`, or a system directory. `serve` mints a
token per run, because loopback is not a permission.

→ **One verb everywhere.** `pkg install ripgrep` over brew, winget, scoop,
choco, apt, pacman, yay and dnf.

→ **An audit trail.** Hand an agent your machine and you will eventually want
one question answered: what did it actually do. `aos obs audit --since=24h`.

The shape is stolen, and I'll say so plainly: it's DHH's **omarchy** CLI. Best
system command centre I've used. I'm not on Arch, I work across three OSes, and
I wanted those ergonomics everywhere. If I move to omarchy full time, I'll use
omarchy's.

Free, MIT, one static binary:
go install github.com/muthuishere/agentic-os/cmd/aos@latest

https://muthuishere.github.io/agentic-os/

---

## First comment (post this yourself, right after)

The bug that taught me the most, since it's the honest part:

Every doc I wrote used `--app=Chrome`. It was in the README, in most examples,
and in the agent skill shipped inside the binary.

It could never have worked. macOS reports the app as "Google Chrome", and the
backend matched App *exactly* and case-sensitively — while `--title` had always
been a substring. So the single most-copied line in the whole project returned
"no windows matched".

I only found it because I stopped reading code and used the tool on itself:
screenshotting its own landing page. `aos capture screenshot --app=Chrome`.
Failed instantly.

Fixed in the front door, exact-match-wins so `--app=Code` still beats
"Code - Insiders", and pinned with a table test — that's the class of bug that
comes back.

Worth saying out loud: docs that were never executed are just confident
guesses.

---

## Shorter variant (if the long one underperforms)

Agents can write code. Almost none of them can use the computer.

aos: one CLI over the machine — windows, input, screenshots, files, processes,
packages, services. 95 commands, 33 groups, same on macOS, Windows and Linux.

The contrarian bit: it leads with an **agent skill**, not MCP.

An agent already has a shell. What it lacks is knowing the CLI exists and what
the exit codes mean. That's a document, not a server — no port, nothing running,
and it works over ssh. MCP is one command away when you want typed tools.

Safety is structural: needs-a-screen is refused with a reason, `file delete`
won't touch `/` or `$HOME`, `serve` is token-gated. Plus an audit trail, because
you'll want to know what the agent actually did.

Command shape borrowed, with credit, from DHH's omarchy CLI.

go install github.com/muthuishere/agentic-os/cmd/aos@latest
https://muthuishere.github.io/agentic-os/

---

## Notes for posting

- Attach `marketing/hero.png` (the landing diagram). Native image, not a link
  preview — LinkedIn suppresses reach on posts whose primary payload is an
  outbound link.
- Better still: put the link in the FIRST COMMENT and keep the post link-free.
  The `go install` line is not a link, so it survives either way.
- Best windows: Tue–Thu, 8–10am IST or 7–9pm IST.
- Reply to every comment in the first hour; that window sets the reach.
- Do not use more than 3 hashtags.

Suggested: #golang #ai #developertools
