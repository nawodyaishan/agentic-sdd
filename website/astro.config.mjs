import { fileURLToPath } from 'node:url';
import { defineConfig } from 'astro/config';
import { unified } from '@astrojs/markdown-remark';
import starlight from '@astrojs/starlight';
import remarkLinkRoutes from './plugins/remark-link-routes.mjs';
import { loadManifest } from './scripts/lib/manifest.mjs';

const repoRoot = fileURLToPath(new URL('..', import.meta.url));
const manifestPath = fileURLToPath(new URL('./docs-manifest.json', import.meta.url));
const manifest = loadManifest(manifestPath, { repoRoot });
const repoUrl = manifest.repository;
const cacheRoot = process.env.ASTRO_CACHE_DIR || './.cache/astro';
const viteCacheRoot = process.env.VITE_CACHE_DIR || './.cache/vite';
const outDir = process.env.ASTRO_OUT_DIR || './dist';
const assetBase = manifest.base.endsWith('/') ? manifest.base : `${manifest.base}/`;

export default defineConfig({
	site: manifest.site,
	base: manifest.base,
	output: 'static',
	trailingSlash: 'always',
	outDir,
	cacheDir: cacheRoot,
	vite: {
		cacheDir: viteCacheRoot,
	},
	markdown: {
		processor: unified({
			remarkPlugins: [[remarkLinkRoutes, { manifestPath, repoRoot }]],
		}),
	},
	integrations: [
		starlight({
			title: 'Agentic-SDD',
			description:
				'Spec-driven engineering skills and a preview-first installer for coding agents, with explicit human approval and bounded verification.',
			pagefind: true,
			routeMiddleware: './src/starlightRouteData.ts',
			head: [
				{
					tag: 'link',
					attrs: { rel: 'icon', type: 'image/png', sizes: '48x48', href: `${assetBase}favicon-48.png` },
				},
				{
					tag: 'link',
					attrs: { rel: 'icon', type: 'image/png', sizes: '192x192', href: `${assetBase}favicon-192.png` },
				},
			],
			customCss: ['./src/styles/focus-visible.css'],
			sidebar: [
				{ label: 'Start here', items: [{ label: 'Overview', slug: 'index' }] },
				{
					label: 'Introduction',
					items: [
						{ label: 'Core concepts', slug: 'introduction/concepts' },
						{ label: 'Clients and destinations', slug: 'introduction/clients' },
						{ label: 'Related approaches', slug: 'introduction/related-approaches' },
					],
				},
				{
					label: 'Getting started',
					items: [
						{ label: 'Choose your setup', slug: 'getting-started/choose-setup' },
						{ label: 'Installation', slug: 'getting-started/installation' },
						{ label: 'First five minutes', slug: 'getting-started/first-five-minutes' },
					],
				},
				{
					label: 'Tutorials',
					items: [
						{ label: 'Direct fix', slug: 'tutorials/direct-fix' },
						{ label: 'First feature', slug: 'tutorials/first-feature' },
						{ label: 'Specialists', slug: 'tutorials/specialists' },
						{ label: 'Research and debugging', slug: 'tutorials/research-and-debugging' },
					],
				},
				{
					label: 'How-to',
					items: [
						{ label: 'Manage skills', slug: 'how-to/skills' },
						{ label: 'Updates and recovery', slug: 'how-to/updates-and-recovery' },
						{ label: 'Orchestration', slug: 'how-to/orchestration' },
						{ label: 'Fleet', slug: 'how-to/fleet' },
						{ label: 'MCP', slug: 'how-to/mcp' },
						{ label: 'Execution', slug: 'how-to/execution' },
						{ label: 'Customization', slug: 'how-to/customization' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'CLI', slug: 'reference/cli' },
						{ label: 'TUI', slug: 'reference/tui' },
						{ label: 'Files and backups', slug: 'reference/files-and-backups' },
						{ label: 'Skills', slug: 'reference/skills' },
						{ label: 'Profiles', slug: 'reference/profiles' },
						{ label: 'Configuration', slug: 'reference/configuration' },
						{ label: 'Fleet', slug: 'reference/fleet' },
						{ label: 'MCP', slug: 'reference/mcp' },
						{ label: 'Execution', slug: 'reference/execution' },
					],
				},
				{
					label: 'Architecture',
					items: [
						{ label: 'Overview', slug: 'architecture/overview' },
						{ label: 'Interfaces', slug: 'architecture/interfaces' },
						{ label: 'Workflows', slug: 'architecture/workflows' },
						{ label: 'Evidence', slug: 'architecture/evidence' },
					],
				},
				{
					label: 'Operations',
					items: [
						{ label: 'Troubleshooting', slug: 'operations/troubleshooting' },
						{ label: 'Known limitations', slug: 'operations/known-limitations' },
					],
				},
				{ label: 'Security', items: [{ label: 'Trust model', slug: 'security/trust' }] },
				{
					label: 'Contributing',
					items: [
						{ label: 'Development setup', slug: 'contributing/development' },
						{ label: 'Testing', slug: 'contributing/testing' },
						{ label: 'Docs and changes', slug: 'contributing/docs-and-changes' },
					],
				},
				{
					label: 'Releases',
					items: [
						{ label: 'Compatibility', slug: 'releases/compatibility' },
						{ label: 'Migration', slug: 'releases/migration' },
					],
				},
			],
			editLink: {
				baseUrl: `${repoUrl}/edit/main/website/`,
			},
			social: [
				{
					icon: 'github',
					label: 'GitHub',
					href: repoUrl,
				},
			],
		}),
	],
});
