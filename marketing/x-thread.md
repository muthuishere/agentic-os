# X / Twitter thread — aos launch

Status: **DRAFT, not posted.**

---

**1/**
Every agent can write code. Almost none of them can use the computer.

Move a window. Take a screenshot. Check the battery. Install a package. Know
whether the machine even has a screen.

So I built aos — one CLI over the machine. 95 commands, 33 groups, same on
macOS, Windows and Linux.

**2/**
The contrarian part: it leads with an agent skill, not MCP.

Everyone is shipping MCP servers. But an agent already has a shell.

What it lacks isn't a protocol. It's knowing the CLI exists, what the exit codes
mean, and which commands need a screen.

That's a document, not a server.

**3/**
`aos install --skills` writes a skill that ships *inside* the binary.

No port. No process to supervise. Nothing to restart when the laptop wakes.

And because it's instructions rather than a connection, it works over ssh into
any machine that has the binary. That's the whole story on servers.

**4/**
MCP is still one command away for agents that want typed tools.

`aos serve mcp`

Same routes, same code path, token-gated because loopback is not a permission.

It's a preference, not a trade-off.

**5/**
Safety is structural, not a disclaimer:

• needs a screen → refused with a reason on a headless box
• `file delete` won't touch `/`, `$HOME`, or a system dir
• services are namespaced; it can't remove one it didn't create
• telemetry records argument *counts*, never values

**6/**
One verb that means the same thing everywhere:

`aos pkg install ripgrep`

over brew, winget, scoop, choco, apt, pacman, yay and dnf.

**7/**
Hand an agent your machine and one question eventually matters: what did it
actually do.

`aos obs audit --since=24h`

Every invocation — typed or called as a tool — is one OTel-shaped span, local.

**8/**
The bug that taught me most:

Every doc used `--app=Chrome`. README, examples, the shipped skill.

It could never work. macOS says "Google Chrome" and the backend matched App
exactly, while --title was a substring.

Found it using the tool on itself. Docs that were never executed are guesses.

**9/**
The command shape is stolen and I'll say so: DHH's omarchy CLI. Best system
command centre I've used.

I'm not on Arch and I work across three OSes. If I move to omarchy full time,
I'll use omarchy's.

**10/**
MIT, one static binary, no runtime:

go install github.com/muthuishere/aos/cmd/aos@latest

https://muthuishere.github.io/aos/
