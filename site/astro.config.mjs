// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLlmsTxt from 'starlight-llms-txt';
import { readFileSync } from 'node:fs';

// Written by scripts/gen-commands.mjs, which runs the binary. Every count on
// this site comes from here, so no page can quote a number the registry has
// moved past. `npm run build` regenerates it before Astro starts.
const registry = JSON.parse(readFileSync('./src/data/registry.json', 'utf8'));
const surface = `${registry.commands} commands across ${registry.groups} groups`;

// Project GitHub Pages: https://muthuishere.github.io/agentic-os
// The landing page is src/pages/index.astro — a full-viewport canvas, outside
// the Starlight layout on purpose. Starlight owns everything under /docs-ish
// slugs below it.
//
// starlight-sidebar-topics is deliberately NOT used here: it earns its place
// when a site has several parallel trees (toolnexus has one per language). This
// site has one flat set of pages, and a topic picker over a single topic is
// chrome with nothing behind it.
export default defineConfig({
	site: 'https://muthuishere.github.io',
	base: '/agentic-os',
	integrations: [
		starlight({
			title: 'agentic-os',
			description:
				`A computer-use MCP server that is also a CLI a person can type. ${surface}, on macOS, Windows and Linux.`,
			// The audience for this product is substantially agents, so the
			// llms.txt is load-bearing, not decoration.
			plugins: [
				starlightLlmsTxt({
					projectName: 'agentic-os',
					description:
						`A computer-use MCP server that is also a CLI a person can type: ${surface} covering windows, input, screen capture, files, processes, packages, network, audio, clipboard, services and messaging, on macOS, Windows and Linux.`,
					details:
						'Every command is an MCP tool and every MCP tool runs the identical code path a person gets at a prompt, so what you test by hand is what the agent gets. Safety is structural: commands that need a screen are refused with a reason on a headless box, `file delete` will not touch a filesystem root or $HOME or a system directory, `serve` binds loopback, services are namespaced, and telemetry records argument counts rather than argument values. One verb set means the same thing everywhere (`pkg install` over brew/winget/scoop/choco/apt/pacman/yay/dnf). It extends without forking through JSON adapters and PATH executables, both of which become MCP tools and neither of which can shadow a built-in.',
				}),
			],
			customCss: ['@fontsource-variable/inter', './src/styles/theme.css'],
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/muthuishere/agentic-os' },
			],
			sidebar: [
				{
					label: 'Start here',
					items: [
						{ label: 'What it is', slug: 'what-it-is' },
						{ label: "What it's not", slug: 'what-it-is-not' },
						{ label: 'Quickstart', slug: 'quickstart' },
					],
				},
				{
					label: 'Using it',
					items: [
						{ label: 'The command surface', slug: 'command-surface' },
						{ label: 'Using it from an agent (MCP)', slug: 'mcp' },
						{ label: 'Safety model', slug: 'safety' },
						{ label: 'Headless machines', slug: 'headless' },
						{ label: 'Extending it', slug: 'extending' },
						{ label: 'Watching a machine', slug: 'remote-share' },
						{ label: 'Observability', slug: 'observability' },
					],
				},
				{
					label: 'Reference',
					items: [{ label: 'Every command', slug: 'reference/commands' }],
				},
			],
		}),
	],
});
