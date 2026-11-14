import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
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
			sitemapIndex: readFileSync(join(outDir, 'sitemap-index.xml'), 'utf8'),
			sitemap0: readFileSync(join(outDir, 'sitemap-0.xml'), 'utf8'),
			pagefindEntryExists: existsSync(join(outDir, 'pagefind/pagefind-entry.json')),
			pagefindJsExists: existsSync(join(outDir, 'pagefind/pagefind.js')),
		};
	} finally {
		rmSync(runDir, { recursive: true, force: true });
	}
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
	const result = runDisposableBuild();
	assert.equal(result.pagefindEntryExists, true);
	assert.equal(result.pagefindJsExists, true);
	assert.match(result.indexHtml, /<link rel="canonical" href="https:\/\/nawodyaishan\.github\.io\/agentic-sdd\/"\s*\/?>/);
	assert.match(
		result.indexHtml,
		/href="https:\/\/github\.com\/nawodyaishan\/agentic-sdd\/edit\/main\/website\/src\/content\/docs\/index\.md"/
	);
	assert.match(result.indexHtml, /href="\/agentic-sdd\/_astro\//);
	assert.match(result.indexHtml, /href="\/agentic-sdd\/#foundation-validation"/);
	assert.match(result.indexHtml, /<img[^>]+src="\/agentic-sdd\/_astro\/base-path\.[^"]+\.svg"/);
	assert.match(result.sitemapIndex, /https:\/\/nawodyaishan\.github\.io\/agentic-sdd\/sitemap-0\.xml/);
	assert.match(result.sitemap0, /<loc>https:\/\/nawodyaishan\.github\.io\/agentic-sdd\/<\/loc>/);
	assert.equal(result.emittedFiles.length > 5, true);
});

test('disposable Starlight build succeeds with root base /', () => {
	const result = runDisposableBuild({ baseOverride: '/', manifestBase: '/' });
	assert.equal(result.pagefindEntryExists, true);
	assert.equal(result.pagefindJsExists, true);
	assert.match(result.indexHtml, /<link rel="canonical" href="https:\/\/nawodyaishan\.github\.io\/"\s*\/?>/);
	assert.match(result.indexHtml, /href="\/_astro\//);
	assert.match(result.indexHtml, /<img[^>]+src="\/_astro\/base-path\.[^"]+\.svg"/);
	assert.doesNotMatch(result.indexHtml, /href="\/\/_astro\//);
	assert.doesNotMatch(result.indexHtml, /"\/agentic-sdd\//);
	assert.match(result.sitemapIndex, /https:\/\/nawodyaishan\.github\.io\/sitemap-0\.xml/);
	assert.match(result.sitemap0, /<loc>https:\/\/nawodyaishan\.github\.io\/<\/loc>/);
});
