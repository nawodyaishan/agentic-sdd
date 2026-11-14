// Feature 020 B5/T12, plan §Q3: timing and deploy-artifact size capture.
//
// Cold and warm observations are kept separate:
//   cold  = Pagefind search-index provisioning, observed only when the supplied
//           build has no usable bundle (then this runner rebuilds it once, into a
//           temporary directory). The site build and dependency install are always
//           `not_run` because this runner never invokes them, and a build that
//           already ships its bundle reports provisioning as `not_run` too.
//   warm  = one untimed engine warm-up, then three repeated integrity passes and
//           three repeated 20-query search passes on the same machine, without
//           flushing OS caches.
//
// Every number is an observation with exact input identity. No time or size
// threshold is applied: plan §6-§7 budgets remain targets until the parent
// compares a repeated measurement.
//
// With no built site this check reports `blocked` (loudly, never a silent skip).
// Supply DOCS_DIST or --built exactly as in search-acceptance-pagefind.test.mjs.

import assert from 'node:assert/strict';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import { QUERY_SET, QUERY_SET_VERSION, querySetDigest } from './search-acceptance/queries.mjs';
import {
	announceBlocked,
	buildRequired,
	collectIdentity,
	createSearcher,
	emitObservation,
	inspectSearchBundle,
	loadDocsManifest,
	resolveBuildInput,
	resolvePagefind,
	resolveSearchBundle,
	searchQuerySet,
	toPageId,
} from './search-acceptance/pagefind.mjs';
import {
	measureArtifact,
	measurePageWeight,
	pickDeepPage,
	resolveAssetPath,
	summarizeSamples,
	walkArtifact,
} from './search-acceptance/dist-artifact.mjs';
import { ESCAPING_HTML, SYNTHETIC_DIST, createSyntheticDist } from './search-acceptance/fixtures.mjs';

function withSyntheticDist(t) {
	const root = mkdtempSync(join(tmpdir(), 'sdd020-perf-'));
	t.after(() => rmSync(root, { recursive: true, force: true }));
	createSyntheticDist(root);
	return root;
}

test('artifact accounting splits buckets and reports compressed JS+CSS per page', (t) => {
	const root = withSyntheticDist(t);
	const measured = measureArtifact({ distDir: root, base: '/agentic-sdd' });
	const expectedTotal = Object.values(SYNTHETIC_DIST).reduce((total, text) => total + Buffer.byteLength(text, 'utf8'), 0);

	assert.equal(measured.artifact.fileCount, Object.keys(SYNTHETIC_DIST).length);
	assert.equal(measured.artifact.totalBytes, expectedTotal);
	assert.deepEqual(measured.artifact.byBucket.pagefind, {
		files: 4,
		bytes: ['pagefind.js', 'pagefind-entry.json', 'pagefind-ui.css', 'wasm.unknown.pagefind']
			.map((name) => Buffer.byteLength(SYNTHETIC_DIST[`pagefind/${name}`], 'utf8'))
			.reduce((total, bytes) => total + bytes, 0),
	});
	assert.deepEqual(measured.artifact.byBucket['astro-assets'], {
		files: 4,
		bytes: ['site.aaa.css', 'extra.ccc.css', 'site.bbb.js', 'extra.ddd.js']
			.map((name) => Buffer.byteLength(SYNTHETIC_DIST[`_astro/${name}`], 'utf8'))
			.reduce((total, bytes) => total + bytes, 0),
	});
	assert.equal(measured.artifact.byBucket.html.files, 3);
	assert.equal(measured.artifact.byBucket.other.files, 0);

	assert.equal(measured.deepPage.relPath, 'reference/cli/index.html');
	assert.equal(measured.deepPage.selected, 'preferred');
	assert.equal(measured.pages.length, 2);
	assert.deepEqual(measured.missingAssets, []);

	const index = measured.pages.find((page) => page.pageId === 'index');
	assert.equal(index.js.files.length, 1, 'JSON-LD is not a loaded JavaScript asset');
	assert.equal(index.css.files.length, 1);
	assert.equal(index.js.raw, Buffer.byteLength(SYNTHETIC_DIST['_astro/site.bbb.js'], 'utf8'));
	assert.equal(index.css.raw, Buffer.byteLength(SYNTHETIC_DIST['_astro/site.aaa.css'], 'utf8'));
	assert.equal(index.inline.scriptBytes, Buffer.byteLength('console.log("inline")', 'utf8'));
	assert.equal(index.inline.styleBytes, Buffer.byteLength('body{color:#111}', 'utf8'));
	assert.ok(index.totals.gzip > 0);
	assert.ok(index.totals.brotli > 0);

	const deep = measured.pages.find((page) => page.pageId === 'reference/cli');
	assert.equal(deep.css.files.length, 2, 'relative and base-prefixed stylesheets both resolve');
	assert.equal(deep.js.files.length, 2);
	assert.deepEqual(
		deep.css.files.map((file) => file.relPath).sort(),
		['_astro/extra.ccc.css', '_astro/site.aaa.css'],
	);
	assert.equal(deep.inline.scriptBytes, 0);
});

test('asset resolution refuses escaping paths and ignores external resources', (t) => {
	const root = withSyntheticDist(t);
	writeFileSync(join(root, 'escape.html'), ESCAPING_HTML);

	assert.throws(
		() => measurePageWeight({ distDir: root, base: '/agentic-sdd', relPath: 'escape.html' }),
		/escapes the built site/,
	);
	assert.equal(resolveAssetPath('https://example.com/app.js', { distDir: root, base: '/agentic-sdd' }), null);
	assert.equal(resolveAssetPath('//cdn.example.com/app.js', { distDir: root, base: '/agentic-sdd' }), null);
	assert.equal(resolveAssetPath('data:text/javascript,1', { distDir: root, base: '/agentic-sdd' }), null);
	assert.throws(
		() => resolveAssetPath('/agentic-sdd/../../etc/passwd', { distDir: root, base: '/agentic-sdd' }),
		/escapes the built site/,
	);
});

test('three-sample summaries report count, minimum, median and maximum', () => {
	assert.deepEqual(summarizeSamples([3, 1, 2]), { count: 3, min: 1, median: 2, max: 3 });
	assert.deepEqual(summarizeSamples([4, 1]), { count: 2, min: 1, median: 2.5, max: 4 });
	assert.deepEqual(summarizeSamples([]), { count: 0, min: null, median: null, max: null });
});

test('cold and warm timing plus size observation for a supplied build', async (t) => {
	const gate = resolveBuildInput();
	if (gate.kind === 'blocked') {
		announceBlocked('performance', gate);
		if (buildRequired()) {
			assert.fail(`blocked: ${gate.message}\n${gate.guidance}`);
		}
		t.skip(`blocked: ${gate.message}`);
		return;
	}
	assert.equal(gate.kind, 'ready', gate.kind === 'invalid' ? gate.message : `unexpected build input state: ${gate.kind}`);

	const pagefind = resolvePagefind();
	assert.ok(pagefind, 'Pagefind is not provisioned under website/node_modules; provision the pinned dependencies before measuring');
	const manifest = loadDocsManifest();
	const distArtifact = walkArtifact(gate.distDir);
	const workDir = mkdtempSync(join(tmpdir(), 'sdd020-perf-live-'));
	t.after(() => rmSync(workDir, { recursive: true, force: true }));

	const resolved = resolveSearchBundle({
		distDir: gate.distDir,
		workDir,
		cliPath: pagefind.cliPath,
		htmlCount: gate.htmlFiles.length,
	});
	const bundle = inspectSearchBundle(resolved.bundleDir);
	assert.deepEqual(bundle.missing, [], `Pagefind bundle is incomplete: ${bundle.missing.join(', ')}`);
	const bundleBytes = walkArtifact(resolved.bundleDir).totalBytes;

	const integrityPass = () => {
		const started = process.hrtime.bigint();
		const artifact = walkArtifact(gate.distDir);
		const deep = pickDeepPage(artifact.htmlFiles);
		const pages = ['index.html', ...(deep.relPath ? [deep.relPath] : [])]
			.filter((relPath) => artifact.htmlFiles.includes(relPath))
			.map((relPath) => measurePageWeight({ distDir: gate.distDir, base: manifest.base, relPath }));
		return {
			elapsedMs: Number(process.hrtime.bigint() - started) / 1e6,
			artifact,
			deep,
			pages,
			missingAssets: pages.flatMap((page) => page.missing.map((relPath) => ({ page: page.pageId, relPath }))),
		};
	};

	const integrityPasses = [integrityPass(), integrityPass(), integrityPass()];
	assert.equal(new Set(integrityPasses.map((pass) => pass.artifact.digest)).size, 1, 'artifact digest changed between warm passes');
	const sizePass = integrityPasses[integrityPasses.length - 1];

	const searcher = await createSearcher({ bundleDir: resolved.bundleDir });
	t.after(() => searcher.close());
	const warmupStarted = process.hrtime.bigint();
	await searchQuerySet(searcher);
	const warmupMs = Number(process.hrtime.bigint() - warmupStarted) / 1e6;
	const queryPasses = [];
	for (let pass = 0; pass < 3; pass += 1) {
		const started = process.hrtime.bigint();
		const { timings } = await searchQuerySet(searcher);
		queryPasses.push({ elapsedMs: Number(process.hrtime.bigint() - started) / 1e6, timings });
	}

	const perQueryMedians = QUERY_SET.map((entry) => ({
		id: entry.id,
		medianMs: summarizeSamples(queryPasses.map((pass) => pass.timings[entry.id])).median,
	}));
	const perQuerySamples = QUERY_SET.flatMap((entry) => queryPasses.map((pass) => pass.timings[entry.id]));

	const observation = {
		kind: 'performance-baseline',
		status: 'observed',
		querySet: {
			version: QUERY_SET_VERSION,
			digest: querySetDigest(),
			count: QUERY_SET.length,
		},
		input: {
			source: gate.source,
			...collectIdentity({ distDir: gate.distDir, pagefind, dist: distArtifact }),
			htmlPages: gate.htmlFiles.length,
		},
		observationMethod: 'Pagefind bundle inspection plus repeated Node-side passes; no network, no site build, no cache flush',
		cold: {
			siteBuild: {
				status: 'not_run',
				reason: 'this runner never invokes astro build; the build must be supplied and timed by the authorized parent',
			},
			dependencyProvisioning: {
				status: 'not_run',
				reason: 'website/node_modules is treated as pre-provisioned; installs are out of scope',
			},
			searchIndexProvisioning: resolved.provisioning,
			searchBundleSource: resolved.source,
		},
		warm: {
			method: 'repeated runs on the same machine without flushing OS caches',
			integrityPassMs: summarizeSamples(integrityPasses.map((pass) => pass.elapsedMs)),
			queryPassMs: summarizeSamples(queryPasses.map((pass) => pass.elapsedMs)),
			queryLatencyMs: summarizeSamples(perQuerySamples),
			perQueryMedianMs: perQueryMedians,
			warmupPassMs: Number(warmupMs.toFixed(3)),
			samples: {
				integrity: integrityPasses.map((pass) => Number(pass.elapsedMs.toFixed(3))),
				queryPass: queryPasses.map((pass) => Number(pass.elapsedMs.toFixed(3))),
			},
		},
		size: {
			compressedScope: 'external JS and CSS files referenced by the page; inline script/style bytes are reported separately',
			artifact: {
				fileCount: sizePass.artifact.fileCount,
				totalBytes: sizePass.artifact.totalBytes,
				htmlFiles: sizePass.artifact.htmlFiles.length,
				byBucket: sizePass.artifact.byBucket,
			},
			searchBundle: {
				source: resolved.source,
				shippedBundlePresent: resolved.shippedBundlePresent,
				shippedBundleMissing: resolved.shippedBundleMissing,
				totalBytes: bundleBytes,
				pageCount: bundle.pageCount,
				wasmFiles: bundle.wasmFiles,
			},
			pages: sizePass.pages.map((page) => ({
				pageId: page.pageId,
				relPath: page.relPath,
				htmlBytes: page.htmlBytes,
				js: { fileCount: page.js.files.length, raw: page.js.raw, gzip: page.js.gzip, brotli: page.js.brotli },
				css: { fileCount: page.css.files.length, raw: page.css.raw, gzip: page.css.gzip, brotli: page.css.brotli },
				inline: page.inline,
				totals: page.totals,
				files: [...page.js.files, ...page.css.files].map((file) => ({
					relPath: file.relPath,
					raw: file.raw,
					gzip: file.gzip,
					brotli: file.brotli,
				})),
			})),
			deepPageSelection: sizePass.deep.selected,
			missingAssets: sizePass.missingAssets,
		},
		budget: {
			status: 'not_evaluated',
			note: 'plan §7 target sizes are not thresholds here; the parent compares a repeated measurement and explains any tradeoff',
		},
		notRun: [
			'site build time (astro build is never invoked by this runner)',
			'dependency provisioning time (install is out of scope)',
			'cold-disk measurement (OS caches are not flushed)',
			'Lighthouse or other third-party audit scores',
			'live deployment asset transfer (site is not published)',
		],
	};
	emitObservation('performance-baseline', observation);

	assert.equal(observation.warm.integrityPassMs.count, 3);
	assert.equal(observation.warm.queryPassMs.count, 3);
	assert.equal(observation.warm.queryLatencyMs.count, QUERY_SET.length * 3);
	for (const summary of [observation.warm.integrityPassMs, observation.warm.queryPassMs, observation.warm.queryLatencyMs]) {
		for (const key of ['min', 'median', 'max']) {
			assert.equal(Number.isFinite(summary[key]), true, `warm ${key} is not a measured number: ${JSON.stringify(summary)}`);
			assert.ok(summary[key] >= 0, `warm ${key} is negative: ${summary[key]}`);
		}
	}
	assert.ok(observation.size.artifact.totalBytes > 0);
	assert.ok(observation.size.artifact.fileCount > 0);
	assert.ok(observation.size.searchBundle.totalBytes > 0);
	assert.ok(['not_run', 'observed'].includes(observation.cold.searchIndexProvisioning.status));
	if (observation.cold.searchIndexProvisioning.status === 'observed') {
		assert.ok(observation.cold.searchIndexProvisioning.elapsedMs >= 0);
	}
	const expectedPageIds = [];
	if (sizePass.artifact.htmlFiles.includes('index.html')) {
		expectedPageIds.push('index');
	}
	if (sizePass.deep.relPath) {
		expectedPageIds.push(toPageId(`/${sizePass.deep.relPath}`));
	}
	assert.deepEqual(
		observation.size.pages.map((page) => page.pageId),
		expectedPageIds,
		'the size observation must cover the index page and one deep page',
	);
	assert.deepEqual(observation.size.missingAssets, []);
	for (const page of observation.size.pages) {
		for (const [kind, group] of [['js', page.js], ['css', page.css]]) {
			assert.ok(group.raw >= 0 && group.gzip >= 0 && group.brotli >= 0, `${page.pageId} ${kind} sizes must be measured`);
			if (group.raw > 0) {
				assert.ok(group.gzip > 0 && group.brotli > 0, `${page.pageId} ${kind} compression was not recorded`);
			}
		}
	}
});
