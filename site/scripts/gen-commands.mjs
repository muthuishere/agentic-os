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
import { existsSync, mkdirSync, readdirSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(here, '..', '..');
const out = resolve(here, '..', 'src', 'content', 'docs', 'reference', 'commands.mdx');
const dataOut = resolve(here, '..', 'src', 'data', 'registry.json');

// Find the main package rather than hard-coding its name: the command has been
// renamed once already, and a docs generator that breaks on a rename is a
// generator that will quietly be replaced by a hand-written page.
function mainPackage() {
	const cmdDir = join(repoRoot, 'cmd');
	const found = readdirSync(cmdDir, { withFileTypes: true })
		.filter((d) => d.isDirectory() && existsSync(join(cmdDir, d.name, 'main.go')))
		.map((d) => `./cmd/${d.name}`);
	if (found.length !== 1) {
		console.error(
			`gen-commands: expected exactly one main package under cmd/, found ${found.length}` +
				(found.length ? `: ${found.join(', ')}` : ''),
		);
		process.exit(1);
	}
	return found[0];
}

const CMD = 'go';
const ARGS = ['run', mainPackage(), 'commands', '--json', '--all'];

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

const { commands: registered } = JSON.parse(raw);
// The route strings carry the binary's own name; take it from there rather
// than assuming it.
const bin = registered[0].route.split(' ')[0];

// Only what ships inside the binary. The CLI deliberately discovers adapters
// from the user's config directory and plugins from PATH, which means running
// it on a build machine mixes that machine's private extensions into the
// registry -- and this page is published. Filtering on `source` keeps the
// reference reproducible: it documents the product, not the laptop that built
// it.
const commands = registered.filter((c) => c.source === 'builtin');
const local = registered.length - commands.length;
if (local > 0) {
	console.log(`gen-commands: ignored ${local} adapter/plugin command(s) from this machine`);
}
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
	`This page is generated. \`site/scripts/gen-commands.mjs\` runs \`${bin} commands --json --all\`` +
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
lines.push(`${bin} commands --json --all`);
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

// The same counts, for the pages and the landing canvas to quote, so no page
// has to hard-code a number the registry can change out from under it.
mkdirSync(dirname(dataOut), { recursive: true });
writeFileSync(
	dataOut,
	JSON.stringify(
		{
			commands: commands.length,
			documented: visible.length,
			groups: names.length,
			binary: bin,
			// The whole surface, group by group. The landing canvas draws every
			// group and lists a group's real commands when one is clicked, so it
			// needs the registry itself rather than a count to quote.
			groupList: names.map((g) => {
				const cmds = groups
					.get(g)
					.filter((c) => !c.hidden)
					.sort((a, b) => a.route.localeCompare(b.route));
				return {
					name: g,
					count: cmds.length,
					gui: cmds.some((c) => c.needs_display),
					commands: cmds.map((c) => ({
						// The leaf, without the group prefix the chip already shows.
						name: c.route.split(' ').slice(2).join(' '),
						summary: c.summary,
						gui: !!c.needs_display,
						only: (c.platforms || []).length ? c.platforms : undefined,
					})),
				};
			}),
		},
		null,
		2,
	) + '\n',
);
console.log(
	`gen-commands: wrote ${visible.length} of ${commands.length} commands in ${names.length} groups -> ${out}`,
);
