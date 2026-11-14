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
			description: 'Documentation for Agentic-SDD.',
			pagefind: true,
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
