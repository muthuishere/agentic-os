# LinkedIn — aos launch

Status: **DRAFT, not posted.** Publishing is owner-gated.
Repo: https://github.com/muthuishere/aos
Site: https://muthuishere.github.io/aos/

---

## Main post (recommended)

Every agent can write code. Almost none of them can use the computer.

They can't move a window, take a screenshot, read the battery, install a package, kill a stuck process, or find out whether the machine even has a screen. So we hand them a raw shell and hope.

I built aos — one CLI over the machine. 100 commands, identical on macOS, Windows and Linux.

The part I'd argue with if someone else built it: it leads with an agent skill, not MCP.

Everyone is shipping MCP servers. But an agent already has a shell. What it lacks isn't a protocol — it's knowing the CLI exists, what the exit codes mean, and which commands need a screen. That's a document, not a server.

So `aos skill install` writes a skill that ships inside the binary. No port, no process to supervise, nothing to restart when the laptop wakes. And because it's instructions rather than a connection, it works over ssh into any machine that has the binary — which is the whole story on servers.

MCP is still one command away, for agents that want typed tools. Same routes, same code path.

Three things I decided differently:

→ Safety is structural. A command that needs a screen is refused with a reason on a headless box, not failed deep inside a display call. `file delete` won't touch a filesystem root, $HOME, or a system directory. `serve` mints a token per run, because loopback is not a permission.

→ One verb everywhere. `pkg install ripgrep` over brew, winget, scoop, choco, apt, pacman, yay and dnf.

→ An audit trail. Hand an agent your machine and you'll eventually want one question answered: what did it actually do. `aos obs audit --since=24h`.

The command shape is stolen, and I'll say so plainly: it's DHH's omarchy CLI. Best system command centre I've used. I'm not on Arch and I work across three OSes, so I wanted those ergonomics everywhere. If I move to omarchy full time, I'll use omarchy's.

MIT, one static binary, no runtime:

go install github.com/muthuishere/aos/cmd/aos@latest

https://muthuishere.github.io/aos/

---

## First comment (post right after)

The bug that taught me the most, since it's the honest part:

Every doc I wrote used `--app=Chrome`. The README, most examples, and the agent skill shipped inside the binary.

It could never have worked. macOS reports the app as "Google Chrome", and the backend matched the app name exactly and case-sensitively — while `--title` had always been a substring. So the single most-copied line in the project returned "no windows matched".

I only found it because I stopped reading code and used the tool on itself, to screenshot its own landing page. Failed instantly.

Docs that were never executed are just confident guesses.

---

## Single X post

Agents can write code. Almost none can use the computer.

aos — one CLI over the machine. 100 commands, macOS, Windows and Linux.

It ships an agent skill first, MCP second. Your agent already has a shell — it just didn't know this CLI existed.

github.com/muthuishere/aos

---

## Notes

- Attach `marketing/hero.png` as a native image; LinkedIn suppresses reach on link-first posts.
- Better still: put the URL in the first comment and keep the post link-free. The `go install` line is not a link.
- Tue-Thu, 8-10am or 7-9pm IST. Reply to everything in the first hour.
- The 10-post X thread is in `marketing/x-thread.md` if you want the long form instead.
