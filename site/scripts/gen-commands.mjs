// Generates src/content/docs/reference/commands.mdx from the binary itself.
//
// It shells out to `go run ./cmd/agentic-os commands --json --all` in the repo
// root, so the reference page cannot drift from what the CLI actually
// registers: if a command is added, renamed or given a new example, the next
// site build picks it up. `--all` is used deliberately — the page documents the
// whole registry, not just what this build machine happens to support.
//
// Wired into `npm run build` via the `prebuild` script.

import { execFileSync } from 'node:child_process';
import { mkdirSync, writeFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(here, '..', '..');
const out = resolve(here, '..', 'src', 'content', 'docs', 'reference', 'commands.mdx');

const CMD = 'go';
const ARGS = ['run', './cmd/agentic-os', 'commands', '--json', '--all'];

let raw;
try {
	raw = execFileSync(CMD, ARGS, { cwd: repoRoot, encoding: 'utf8', maxBuffer: 32 * 1024 * 1024 });
} catch (err) {
	console.error(
		`gen-commands: could not run \`${CMD} ${ARGS.join(' ')}\` in ${repoRoot}.\n` +
			'The command reference is generated from the binary, so a Go toolchain is required to build this site.\n' +
			String(err.stderr || err.message),
	);
	process.exit(1);
}

const { commands } = JSON.parse(raw);
const visible = commands.filter((c) => !c.hidden);

const groups = new Map();
for (const c of visible) {
	if (!groups.has(c.group)) groups.set(c.group, []);
	groups.get(c.group).push(c);
}
const names = [...groups.keys()].sort();

const esc = (s) => String(s).replace(/([<>{}])/g, '\\$1');

const lines = [];
lines.push('---');
lines.push('title: Every command');
lines.push(
	'description: The complete agentic-os command registry — ' +
		`${commands.length} commands across ${names.length} groups — generated from the binary at build time.`,
);
lines.push('---');
lines.push('');
lines.push(
	`This page is generated. \`site/scripts/gen-commands.mjs\` runs \`agentic-os commands --json --all\`` +
		' against the source tree and writes this file every time the site is built, so it cannot' +
		' describe a command the binary does not have, or miss one it does.',
);
lines.push('');
const hidden = commands.length - visible.length;
lines.push(
	`**${commands.length} commands across ${names.length} groups.**` +
		(hidden ? ` ${hidden} internal one${hidden === 1 ? ' is' : 's are'} marked hidden and not listed below.` : ''),
);
lines.push('');
lines.push(
	'`needs a display` marks a command that is refused with a reason on a machine with no screen.' +
		' `platforms` appears only where a command does not exist everywhere; everything else runs on' +
		' macOS, Windows and Linux.',
);
lines.push('');
lines.push('The same data, machine-readable, comes from the binary you installed:');
lines.push('');
lines.push('```sh');
lines.push('agentic-os commands --json --all');
lines.push('```');
lines.push('');

for (const g of names) {
	const cmds = groups.get(g).sort((a, b) => a.route.localeCompare(b.route));
	lines.push(`## ${g}`);
	lines.push('');
	for (const c of cmds) {
		const usage = [c.route, c.args].filter(Boolean).join(' ');
		lines.push(`### \`${usage}\``);
		lines.push('');
		lines.push(esc(c.summary));
		lines.push('');
		const notes = [];
		if (c.needs_display) notes.push('needs a display');
		if (c.requires_sudo) notes.push('needs elevated rights');
		if (c.platforms?.length) notes.push(`platforms: ${c.platforms.join(', ')}`);
		if (c.aliases?.length) notes.push(`also: ${c.aliases.map((a) => `\`${a}\``).join(', ')}`);
		if (notes.length) {
			lines.push(notes.join(' · '));
			lines.push('');
		}
		if (c.examples?.length) {
			lines.push('```sh');
			for (const ex of c.examples) lines.push(ex);
			lines.push('```');
			lines.push('');
		}
	}
}

mkdirSync(dirname(out), { recursive: true });
writeFileSync(out, lines.join('\n'));
console.log(
	`gen-commands: wrote ${visible.length} of ${commands.length} commands in ${names.length} groups -> ${out}`,
);
