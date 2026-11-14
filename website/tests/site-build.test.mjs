import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import {
	cpSync,
	existsSync,
	lstatSync,
	mkdirSync,
	mkdtempSync,
	readdirSync,
	readFileSync,
	rmSync,
	symlinkSync,
	writeFileSync,
} from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { computeReviewDigest } from '../scripts/lib/manifest.mjs';

const testDir = dirname(fileURLToPath(import.meta.url));
const websiteDir = resolve(testDir, '..');

function resolveNodeModules() {
	const candidate = resolve(websiteDir, 'node_modules');
	if (!existsSync(candidate)) {
		throw new Error(`Missing provisioned node_modules at ${candidate}`);
	}
	return candidate;
}

function walkFiles(dir) {
	const out = [];
	for (const entry of readdirSync(dir, { withFileTypes: true })) {
		const full = join(dir, entry.name);
		const st = lstatSync(full);
		assert.equal(st.isSymbolicLink(), false, `unexpected symlink in build output: ${full}`);
		if (st.isDirectory()) {
			out.push(...walkFiles(full));
		} else {
			assert.equal(st.nlink, 1, `unexpected hardlink in build output: ${full}`);
			out.push(full);
		}
	}
	return out;
}

function copySite(runDir, manifestBase) {
	const siteDir = join(runDir, 'website');
	cpSync(websiteDir, siteDir, {
		recursive: true,
		filter: (source) => {
			const name = source.slice(websiteDir.length + 1);
			return name !== 'node_modules' && name !== '.cache' && name !== '.astro' && name !== 'dist';
		},
	});
	symlinkSync(resolveNodeModules(), join(siteDir, 'node_modules'), 'dir');
	// The base-path image/anchor fixture no longer lives in the real homepage.
	// Inject an equivalent disposable page into the scratch copy so hashed-asset
	// and root-base behaviour stay covered without leaking into <main>, Pagefind
	// or search snippets of the shipped site.
	const fixtureSource = 'website/src/content/docs/base-path-fixture.md';
	const fixtureText = `---
title: Base path fixture
description: Disposable base-path validation fixture injected by site-build tests.
---

## Foundation validation

This [Foundation validation](./base-path-fixture.md#foundation-validation) anchor link and image fixture validates base-path URL handling in disposable site builds.

![Base-path fixture](../../assets/fixtures/base-path.svg)
`;
	writeFileSync(join(siteDir, 'src/content/docs/base-path-fixture.md'), fixtureText);
	const fixtureSha256 = createHash('sha256').update(fixtureText).digest('hex');
	const manifestPath = join(siteDir, 'docs-manifest.json');
	const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
	manifest.pages.push({
		id: 'base-path-fixture',
		source: fixtureSource,
		type: 'explanation',
		audience: ['new user'],
		class: 'L',
		sources: [{ path: fixtureSource, sha256: fixtureSha256 }],
		reviewDigest: computeReviewDigest([{ path: fixtureSource, sha256: fixtureSha256 }]),
		appliesTo: 'unreleased',
		reviewStatus: 'unreviewed',
	});
	writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);
	if (manifestBase !== undefined) {
		const manifestPath = join(siteDir, 'docs-manifest.json');
		const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
		manifest.base = manifestBase;
		writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);
	}
	return siteDir;
}

function runDisposableBuild({ baseOverride, manifestBase } = {}) {
	const scratchBase = join(resolve(websiteDir, '..'), '.cache');
	mkdirSync(scratchBase, { recursive: true });
	const runDir = mkdtempSync(join(scratchBase, 'site-test-'));
	const outDir = join(runDir, 'dist');
	const astroCacheDir = join(runDir, 'cache-astro');
	const viteCacheDir = join(runDir, 'cache-vite');

	try {
		const siteDir = copySite(runDir, manifestBase);
		const astroBin = join(siteDir, 'node_modules/.bin/astro');
		const args = ['build', '--outDir', outDir];
		if (baseOverride !== undefined) {
			args.push('--base', baseOverride);
		}

		execFileSync(astroBin, args, {
			cwd: siteDir,
			env: {
				PATH: process.env.PATH,
				HOME: runDir,
				CI: 'true',
				ASTRO_TELEMETRY_DISABLED: '1',
				ASTRO_CACHE_DIR: astroCacheDir,
				VITE_CACHE_DIR: viteCacheDir,
				ASTRO_OUT_DIR: outDir,
			},
			stdio: 'pipe',
		});

		const emittedFiles = walkFiles(outDir);
		return {
			emittedFiles,
			indexHtml: readFileSync(join(outDir, 'index.html'), 'utf8'),
			fixtureHtml: readFileSync(join(outDir, 'base-path-fixture/index.html'), 'utf8'),
			notFoundHtml: readFileSync(join(outDir, '404.html'), 'utf8'),
			deepPageHtml: readFileSync(join(outDir, 'reference/cli/index.html'), 'utf8'),
			sitemapIndex: readFileSync(join(outDir, 'sitemap-index.xml'), 'utf8'),
			sitemap0: readFileSync(join(outDir, 'sitemap-0.xml'), 'utf8'),
			pagefindEntryExists: existsSync(join(outDir, 'pagefind/pagefind-entry.json')),
			pagefindJsExists: existsSync(join(outDir, 'pagefind/pagefind.js')),
		};
	} finally {
		rmSync(runDir, { recursive: true, force: true });
	}
}

// Each SEO-relevant scenario builds once but is asserted by more than one test.
const scenarioBuilds = new Map();

function scenarioBuild(name, options) {
	if (!scenarioBuilds.has(name)) scenarioBuilds.set(name, runDisposableBuild(options));
	return scenarioBuilds.get(name);
}

function escapeRegExp(value) {
	return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function jsonLdBlocks(html) {
	return [...html.matchAll(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/g)].map((match) =>
		JSON.parse(match[1]),
	);
}

function findJsonLd(html, type) {
	const blocks = jsonLdBlocks(html).filter((block) => block['@type'] === type);
	assert.equal(blocks.length, 1, `expected exactly one ${type} JSON-LD block`);
	return blocks[0];
}

test('yaml package exports parse and parseDocument for Wave 2 interop', async () => {
	const nodeModulesDir = resolveNodeModules();
	const yamlPkgUrl = new URL(`file://${join(nodeModulesDir, 'yaml/dist/index.js')}`);
	const yaml = await import(yamlPkgUrl.href);
	assert.equal(typeof yaml.parse, 'function');
	assert.equal(typeof yaml.parseDocument, 'function');
	assert.deepEqual(yaml.parse('name: agentic-sdd\n'), { name: 'agentic-sdd' });
});

test('disposable Starlight build succeeds with project base /agentic-sdd', () => {
	const result = scenarioBuild('project');
	assert.equal(result.pagefindEntryExists, true);
	assert.equal(result.pagefindJsExists, true);
	assert.match(result.indexHtml, /<link rel="canonical" href="https:\/\/nawodyaishan\.github\.io\/agentic-sdd\/"\s*\/?>/);
	assert.match(
		result.indexHtml,
		/href="https:\/\/github\.com\/nawodyaishan\/agentic-sdd\/edit\/main\/website\/src\/content\/docs\/index\.md"/
	);
	assert.match(result.indexHtml, /href="\/agentic-sdd\/_astro\//);
	assert.match(result.fixtureHtml, /href="[^"]*#foundation-validation"/);
	assert.match(result.fixtureHtml, /<img[^>]+src="\/agentic-sdd\/_astro\/base-path\.[^"]+\.svg"/);
	assert.doesNotMatch(result.indexHtml, /foundation-validation|base-path/);
	assert.match(result.sitemapIndex, /https:\/\/nawodyaishan\.github\.io\/agentic-sdd\/sitemap-0\.xml/);
	assert.match(result.sitemap0, /<loc>https:\/\/nawodyaishan\.github\.io\/agentic-sdd\/<\/loc>/);
	assert.equal(result.emittedFiles.length > 5, true);
});

test('disposable Starlight build succeeds with root base /', () => {
	const result = scenarioBuild('root', { baseOverride: '/', manifestBase: '/' });
	assert.equal(result.pagefindEntryExists, true);
	assert.equal(result.pagefindJsExists, true);
	assert.match(result.indexHtml, /<link rel="canonical" href="https:\/\/nawodyaishan\.github\.io\/"\s*\/?>/);
	assert.match(result.indexHtml, /href="\/_astro\//);
	assert.match(result.fixtureHtml, /<img[^>]+src="\/_astro\/base-path\.[^"]+\.svg"/);
	assert.doesNotMatch(result.indexHtml, /href="\/\/_astro\//);
	assert.doesNotMatch(result.indexHtml, /"\/agentic-sdd\//);
	assert.match(result.sitemapIndex, /https:\/\/nawodyaishan\.github\.io\/sitemap-0\.xml/);
	assert.match(result.sitemap0, /<loc>https:\/\/nawodyaishan\.github\.io\/<\/loc>/);
});

const SEO_CASES = [
	{
		scenario: 'project',
		base: '/agentic-sdd',
		documentUrl: 'https://nawodyaishan.github.io/agentic-sdd/',
		deepUrl: 'https://nawodyaishan.github.io/agentic-sdd/reference/cli/',
		ogImage: 'https://nawodyaishan.github.io/agentic-sdd/og-card.png',
	},
	{
		scenario: 'root',
		base: '/',
		documentUrl: 'https://nawodyaishan.github.io/',
		deepUrl: 'https://nawodyaishan.github.io/reference/cli/',
		ogImage: 'https://nawodyaishan.github.io/og-card.png',
	},
];

test('SEO head metadata, JSON-LD and raster icons are emitted for both bases', () => {
	for (const { scenario, base, documentUrl, deepUrl, ogImage } of SEO_CASES) {
		const result = scenario === 'project' ? scenarioBuild('project') : scenarioBuild('root');
		const assetPrefix = base === '/' ? '/' : `${base}/`;
		const names = new Set(result.emittedFiles.map((file) => file.split('/').pop()));

		for (const asset of ['favicon.svg', 'favicon-48.png', 'favicon-192.png', 'og-card.png']) {
			assert.equal(names.has(asset), true, `${scenario}: ${asset} must be emitted`);
		}

		// SEO-01: the SVG favicon plus the raster icons it was converted to are referenced under the base.
		assert.match(result.indexHtml, new RegExp(`<link rel="shortcut icon" href="${assetPrefix}favicon\\.svg"`));
		assert.match(result.indexHtml, new RegExp(`<link rel="icon" type="image/png" sizes="48x48" href="${assetPrefix}favicon-48\\.png"`));
		assert.match(result.indexHtml, new RegExp(`<link rel="icon" type="image/png" sizes="192x192" href="${assetPrefix}favicon-192\\.png"`));

		// SEO-02: one descriptive home title, an unchanged brand heading and og:type=website.
		assert.equal((result.indexHtml.match(/<title>/g) ?? []).length, 1, `${scenario}: single <title>`);
		assert.match(result.indexHtml, /<title>Agentic-SDD — spec-driven engineering skills and CLI<\/title>/);
		assert.doesNotMatch(result.indexHtml, /<title>Agentic-SDD \| Agentic-SDD<\/title>/);
		assert.match(result.indexHtml, /<meta property="og:title" content="Agentic-SDD"\/>/);
		assert.match(result.indexHtml, /<meta property="og:type" content="website"\/>/);
		assert.match(result.indexHtml, /<h1[^>]*>Agentic-SDD<\/h1>/);

		// SEO-03: absolute social image URLs derived from Astro.site plus the configured base.
		assert.match(result.indexHtml, new RegExp(`<meta property="og:image" content="${escapeRegExp(ogImage)}"`));
		assert.match(result.deepPageHtml, new RegExp(`<meta property="og:image" content="${escapeRegExp(ogImage)}"`));
		assert.match(result.indexHtml, /<meta property="og:image:width" content="1200"\/>/);
		assert.match(result.indexHtml, /<meta property="og:image:height" content="630"\/>/);
		assert.match(result.indexHtml, /<meta property="og:image:alt" content="Agentic-SDD — spec-driven engineering skills and CLI"\/>/);
		assert.match(result.indexHtml, new RegExp(`<meta name="twitter:image" content="${escapeRegExp(ogImage)}"`));

		// 404: not indexed, no self-canonical, no og:url or social image.
		assert.match(result.notFoundHtml, /<meta name="robots" content="noindex"\/>/);
		assert.doesNotMatch(result.notFoundHtml, /<link rel="canonical"/);
		assert.doesNotMatch(result.notFoundHtml, /<meta property="og:url"/);
		assert.doesNotMatch(result.notFoundHtml, /<meta property="og:image"/);
		assert.doesNotMatch(result.notFoundHtml, /<meta name="twitter:image"/);

		// SEO-04: home carries TechArticle only; a deep page adds a breadcrumb that matches the sidebar.
		assert.equal(jsonLdBlocks(result.indexHtml).length, 1, `${scenario}: home JSON-LD`);
		const homeArticle = findJsonLd(result.indexHtml, 'TechArticle');
		assert.equal(homeArticle.url, documentUrl);
		assert.equal(homeArticle.inLanguage, 'en');

		const breadcrumb = findJsonLd(result.deepPageHtml, 'BreadcrumbList');
		assert.equal(breadcrumb.itemListElement.length, 3);
		assert.deepEqual(
			breadcrumb.itemListElement.map((item) => item.name),
			['Start here', 'Reference', 'CLI reference'],
		);
		assert.equal(breadcrumb.itemListElement[0].item, documentUrl);
		assert.equal(breadcrumb.itemListElement[2].item, deepUrl);
		assert.match(result.deepPageHtml, new RegExp(`<link rel="canonical" href="${escapeRegExp(deepUrl)}"`));

		const article = findJsonLd(result.deepPageHtml, 'TechArticle');
		assert.equal(article.url, deepUrl);
		assert.equal(article.description, 'Command families, flags, defaults, exit codes, and output contracts for asdx-core, asdx, and the guide, fleet, MCP, and execution subcommands.');
		assert.equal(article.isPartOf.url, 'https://github.com/nawodyaishan/agentic-sdd');
	}
});
