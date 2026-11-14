// Feature 020 B5/T12, plan §Q2: fixed-20 Pagefind acceptance and allowlist
// tooling. It records top-5 URLs, the expected page's rank and the critical
// top-3 result for every frozen query.
//
// With no built site this check reports `blocked` (loudly, never a silent skip).
// Supply one with DOCS_DIST, or run the file directly with --built:
//   DOCS_DIST=/abs/website/dist node --test website/tests/search-acceptance-pagefind.test.mjs
//   node website/tests/search-acceptance-pagefind.test.mjs --built /abs/website/dist
// Add --require-build (or DOCS_REQUIRE_BUILD=1) to turn a missing build into a
// hard failure. The runner never invokes `astro build` and never writes into the
// supplied dist.

import assert from 'node:assert/strict';
import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import {
	CRITICAL_QUERY_IDS,
	CRITICAL_TOP_N,
	MIN_TOP_N_HITS,
	QUERY_SET,
	QUERY_SET_VERSION,
	REQUIRED_COVERAGE_AREAS,
	TOTAL_QUERIES,
	querySetDigest,
} from './search-acceptance/queries.mjs';
import {
	FORBIDDEN_PATH_RULES,
	REPO_ROOT,
	WEBSITE_DIR,
	announceBlocked,
	buildRequired,
	collectIdentity,
	createSearcher,
	describeEvaluation,
	emitObservation,
	evaluateQueryResults,
	findForbiddenPages,
	findUnlistedPages,
	inspectSearchBundle,
	loadDocsManifest,
	manifestAllowlist,
	resolveBuildInput,
	resolvePagefind,
	resolveSearchBundle,
	searchQuerySet,
	toPageId,
} from './search-acceptance/pagefind.mjs';
import { PAGE_ID_CASES, PRIVATE_PATH_CASES, rankingCases } from './search-acceptance/fixtures.mjs';
import { walkArtifact } from './search-acceptance/dist-artifact.mjs';

const PLAN_PATH = join(REPO_ROOT, 'specs/020-documentation-ecosystem/plan.md');
const SITEMAP_PATH = join(REPO_ROOT, 'specs/020-documentation-ecosystem/sitemap.md');

function parsePlanQuerySet(planMarkdown) {
	const start = planMarkdown.indexOf('### Q2');
	const end = planMarkdown.indexOf('### Q3');
	assert.notEqual(start, -1, 'plan §Q2 heading not found in plan.md');
	assert.notEqual(end, -1, 'plan §Q3 heading not found in plan.md');
	const rows = [];
	for (const line of planMarkdown.slice(start, end).split('\n')) {
		if (!line.startsWith('|')) continue;
		const cells = line.split('|').slice(1, -1).map((cell) => cell.trim());
		if (cells.length !== 3) continue;
		const [queryCell, expectedCell, criticalCell] = cells;
		if (queryCell.toLowerCase() === 'query' || /^:?-+:?$/.test(queryCell)) continue;
		const expectedPageIds = [...expectedCell.matchAll(/`([^`]+)`/g)].map((match) => match[1]);
		if (expectedPageIds.length === 0) continue;
		rows.push({
			query: queryCell.replace(/\s+/g, ' ').trim(),
			expectedPageIds,
			critical: criticalCell === 'C',
		});
	}
	return rows;
}

function parseSitemapClasses(sitemapMarkdown) {
	const classes = new Map();
	for (const line of sitemapMarkdown.split('\n')) {
		const match = line.match(/^\|\s*`([^`]+)`\s*\/\s*([LC])\s*\//);
		if (match) {
			classes.set(match[1], match[2]);
		}
	}
	return classes;
}

test('frozen query set matches plan §Q2 exactly and covers every acceptance area', () => {
	const planMarkdown = readFileSync(PLAN_PATH, 'utf8');
	const planned = parsePlanQuerySet(planMarkdown);
	assert.equal(planned.length, TOTAL_QUERIES, `plan §Q2 must list exactly ${TOTAL_QUERIES} queries`);
	assert.deepEqual(
		QUERY_SET.map((entry) => ({
			query: entry.query,
			expectedPageIds: [...entry.expectedPageIds],
			critical: entry.critical,
		})),
		planned,
		'the frozen query set drifted from plan §Q2; update both deliberately',
	);
	assert.equal(QUERY_SET.length, TOTAL_QUERIES);
	assert.equal(new Set(QUERY_SET.map((entry) => entry.id)).size, TOTAL_QUERIES);
	assert.equal(new Set(QUERY_SET.map((entry) => entry.query)).size, TOTAL_QUERIES);
	assert.equal(CRITICAL_QUERY_IDS.length, 7, 'plan §Q2 marks seven safety/onboarding-critical queries');

	for (const area of REQUIRED_COVERAGE_AREAS) {
		assert.ok(
			QUERY_SET.some((entry) => entry.coverage.includes(area)),
			`no frozen query covers the required acceptance area: ${area}`,
		);
	}
	assert.match(QUERY_SET_VERSION, /^020-q2-/);
	assert.match(querySetDigest(), /^[0-9a-f]{64}$/);
});

test('every expected page ID is a declared launch or current-feature sitemap page', () => {
	const sitemapMarkdown = readFileSync(SITEMAP_PATH, 'utf8');
	const classes = parseSitemapClasses(sitemapMarkdown);
	assert.ok(classes.size > 20, 'sitemap L/C inventory could not be parsed');
	for (const entry of QUERY_SET) {
		for (const pageId of entry.expectedPageIds) {
			assert.ok(
				classes.has(pageId),
				`${entry.id} expects "${pageId}", which is not an L/C page in sitemap.md`,
			);
		}
	}
});

test('page IDs normalize base prefixes, index files, fragments and encoded paths', () => {
	for (const [rawUrl, base, expected] of PAGE_ID_CASES) {
		assert.equal(toPageId(rawUrl, { base }), expected, `toPageId(${rawUrl}, base=${base})`);
	}
	assert.equal(toPageId('/agentic-sdd/reference/cli/#anchor', { base: '/agentic-sdd' }), 'reference/cli');
	assert.equal(toPageId('', {}), 'index');
});

test('ranking evaluation enforces 18/20 top-5 hits and critical top-3', () => {
	assert.equal(MIN_TOP_N_HITS, 18);
	assert.equal(CRITICAL_TOP_N, 3);
	const cases = rankingCases(QUERY_SET);
	assert.ok(cases.length >= 5);

	for (const testCase of cases) {
		const evaluation = evaluateQueryResults({ results: testCase.results, base: '/agentic-sdd' });
		const label = `${testCase.name}: ${describeEvaluation(evaluation)}`;
		assert.equal(evaluation.passed, testCase.expect.passed, label);
		assert.equal(evaluation.hits, testCase.expect.hits, label);
		if (testCase.expect.criticalFailures) {
			assert.deepEqual(evaluation.criticalFailures, testCase.expect.criticalFailures, label);
		}
		if (testCase.expect.misses) {
			assert.deepEqual(evaluation.misses, testCase.expect.misses, label);
		}
		if (testCase.expect.expectedRank) {
			const record = evaluation.perQuery.find((entry) => entry.id === testCase.expect.expectedRank.id);
			assert.equal(record.expectedRank, testCase.expect.expectedRank.rank, label);
			assert.equal(record.hit, false, label);
		}
	}
});

test('private and history page IDs are refused without blocking legitimate pages', () => {
	assert.deepEqual(
		findForbiddenPages(PRIVATE_PATH_CASES.forbidden.map(([pageId]) => pageId)),
		PRIVATE_PATH_CASES.forbidden.map(([pageId, rule]) => ({ pageId, rule })),
	);
	assert.deepEqual(findForbiddenPages(PRIVATE_PATH_CASES.allowed), []);
	assert.equal(FORBIDDEN_PATH_RULES.length, 5);
});

test('built-site Pagefind acceptance: 18/20 top-5, critical top-3, no private results', async (t) => {
	const gate = resolveBuildInput();
	if (gate.kind === 'blocked') {
		announceBlocked('search-acceptance', gate);
		if (buildRequired()) {
			assert.fail(`blocked: ${gate.message}\n${gate.guidance}`);
		}
		t.skip(`blocked: ${gate.message}`);
		return;
	}
	assert.equal(gate.kind, 'ready', gate.kind === 'invalid' ? gate.message : `unexpected build input state: ${gate.kind}`);

	const pagefind = resolvePagefind();
	assert.ok(
		pagefind,
		`Pagefind is not provisioned at ${WEBSITE_DIR}/node_modules; provision the pinned website dependencies before measuring`,
	);

	const manifest = loadDocsManifest();
	const distArtifact = walkArtifact(gate.distDir);
	const workDir = mkdtempSync(join(tmpdir(), 'sdd020-search-'));
	t.after(() => rmSync(workDir, { recursive: true, force: true }));

	const resolved = resolveSearchBundle({
		distDir: gate.distDir,
		workDir,
		cliPath: pagefind.cliPath,
		htmlCount: gate.htmlFiles.length,
	});
	const bundle = inspectSearchBundle(resolved.bundleDir);
	assert.deepEqual(
		resolved.shippedBundleMissing,
		[],
		`the supplied build ships an incomplete Pagefind bundle (broken search assets): ${resolved.shippedBundleMissing.join(', ')}`,
	);
	assert.deepEqual(bundle.missing, [], `Pagefind bundle is incomplete: ${bundle.missing.join(', ')}`);
	assert.ok(bundle.wasmFiles.length > 0, 'Pagefind bundle has no wasm payload');
	assert.equal(bundle.version, pagefind.version, 'Pagefind entry version does not match the pinned install');

	const searcher = await createSearcher({ bundleDir: resolved.bundleDir });
	t.after(() => searcher.close());
	const { results, timings } = await searchQuerySet(searcher);
	const evaluation = evaluateQueryResults({ results, base: manifest.base });

	const resultPageIds = [...new Set(Object.values(results).flat().map((url) => toPageId(url, { base: manifest.base })))];
	const distPageIds = gate.htmlFiles.map((relPath) => toPageId(`/${relPath}`));
	const allowlist = manifestAllowlist(manifest);
	const observation = {
		kind: 'search-acceptance',
		status: 'observed',
		querySet: {
			version: QUERY_SET_VERSION,
			digest: querySetDigest(),
			count: QUERY_SET.length,
			criticalCount: CRITICAL_QUERY_IDS.length,
		},
		input: {
			source: gate.source,
			...collectIdentity({ distDir: gate.distDir, pagefind, dist: distArtifact }),
			htmlPages: gate.htmlFiles.length,
		},
		searchBundle: {
			source: resolved.source,
			provisioning: resolved.provisioning,
			shippedPageCount: resolved.shippedPageCount,
			shippedBundlePresent: resolved.shippedBundlePresent,
			shippedBundleMissing: resolved.shippedBundleMissing,
			shippedBundleMismatch: resolved.shippedBundleMismatch ?? false,
			version: bundle.version,
			languages: bundle.languages,
			pageCount: bundle.pageCount,
			wasmFiles: bundle.wasmFiles,
			output: resolved.source === 'built-dist' ? 'read-only shipped bundle inside the supplied dist' : 'temporary directory, never the supplied dist',
		},
		observationMethod: 'pinned Pagefind bundle searched offline through its emitted JS API; no network, no site build',
		thresholds: evaluation.threshold,
		results: evaluation.perQuery.map((entry) => ({
			id: entry.id,
			query: entry.query,
			critical: entry.critical,
			expectedPageIds: entry.expectedPageIds,
			top5: entry.topN,
			top5Urls: entry.topNUrls,
			expectedRank: entry.expectedRank,
			resultCount: entry.resultCount,
			elapsedMs: Number(timings[entry.id].toFixed(3)),
		})),
		summary: {
			hits: evaluation.hits,
			misses: evaluation.misses,
			criticalFailures: evaluation.criticalFailures,
			emptyQueries: evaluation.emptyQueries,
			passed: evaluation.passed,
		},
		forbiddenResultPages: findForbiddenPages(resultPageIds),
		forbiddenDistPages: findForbiddenPages(distPageIds),
		forbiddenArtifactPaths: findForbiddenPages(distArtifact.files.map((file) => file.relPath)),
		allowlistViolations: findUnlistedPages(distPageIds, allowlist),
		notRun: [
			'live deployment (site is not published; has_pages:false observation belongs to the parent)',
			'browser keyboard search usability (manual observation, not an automated ranking test)',
			'non-fixture user queries (the fixed 20 are directional, not a population benchmark)',
		],
	};
	emitObservation('search-acceptance', observation);

	assert.ok(bundle.pageCount >= 1, 'Pagefind indexed no pages');
	assert.ok(
		bundle.pageCount <= gate.htmlFiles.length,
		`Pagefind indexed ${bundle.pageCount} pages but the build only has ${gate.htmlFiles.length} HTML files`,
	);
	assert.deepEqual(observation.forbiddenDistPages, [], 'the built site contains private/history pages');
	assert.deepEqual(observation.forbiddenArtifactPaths, [], 'the build artifact contains private/history files');
	assert.deepEqual(observation.forbiddenResultPages, [], 'search results expose private/history pages');
	assert.deepEqual(observation.allowlistViolations, [], 'the built site contains pages outside docs-manifest.json');
	assert.equal(evaluation.passed, true, describeEvaluation(evaluation));
});
