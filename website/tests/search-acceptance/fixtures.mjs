// Offline fixtures for the search/performance tooling. Everything is a literal
// string or a temporary directory: no build, no network, no real credentials.

import { mkdirSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';

export const SYNTHETIC_DIST = Object.freeze({
	'index.html': [
		'<!doctype html>',
		'<html lang="en"><head>',
		'<link rel="stylesheet" href="/agentic-sdd/_astro/site.aaa.css">',
		'<style>body{color:#111}</style>',
		'<script type="application/ld+json">{"@type":"WebSite"}</script>',
		'</head><body><h1>Home</h1>',
		'<script src="/agentic-sdd/_astro/site.bbb.js"></script>',
		'<script>console.log("inline")</script>',
		'</body></html>',
	].join('\n'),
	'reference/cli/index.html': [
		'<!doctype html>',
		'<html lang="en"><head>',
		'<link rel="stylesheet" href="/agentic-sdd/_astro/site.aaa.css">',
		'<link rel="stylesheet" href="../../_astro/extra.ccc.css">',
		'</head><body><h1>CLI</h1>',
		'<script src="/agentic-sdd/_astro/site.bbb.js"></script>',
		'<script src="/agentic-sdd/_astro/extra.ddd.js"></script>',
		'</body></html>',
	].join('\n'),
	'404.html': '<!doctype html><html lang="en"><body><h1>Not found</h1></body></html>',
	'_astro/site.aaa.css': 'body{margin:0}\n',
	'_astro/extra.ccc.css': '.cli{display:block}\n',
	'_astro/site.bbb.js': 'console.log("site")\n',
	'_astro/extra.ddd.js': 'console.log("extra")\n',
	'pagefind/pagefind.js': '// synthetic pagefind entry\nexport {}\n',
	'pagefind/pagefind-entry.json': '{"version":"synthetic","languages":{"unknown":{"page_count":2}}}',
	'pagefind/pagefind-ui.css': '.pagefind{display:block}\n',
	'pagefind/wasm.unknown.pagefind': 'synthetic wasm placeholder',
});

// Writes the literal tree above under `root` and returns the paths plus the
// literal contents so a test can compute expected byte counts independently.
export function createSyntheticDist(root) {
	const written = {};
	for (const [relPath, content] of Object.entries(SYNTHETIC_DIST)) {
		const absPath = join(root, relPath);
		mkdirSync(dirname(absPath), { recursive: true });
		writeFileSync(absPath, content);
		written[relPath] = { absPath, content };
	}
	return written;
}

// Builds a result map for the frozen query set. `rankFor(entry, index)` returns
// the 1-based position of the first expected page, or null for "absent". Filler
// URLs exercise base-prefix and trailing-slash normalization.
export function buildResultMap({ querySet, rankFor, base = '/agentic-sdd' }) {
	const results = {};
	for (const [index, entry] of querySet.entries()) {
		const rank = rankFor(entry, index);
		const urls = [];
		if (rank !== null) {
			for (let position = 1; position < rank; position += 1) {
				urls.push(`${base}/filler/${entry.id}-${position}/`);
			}
			const expected = entry.expectedPageIds[0];
			urls.push(`${base}/${expected === 'index' ? '' : `${expected}/`}`);
		}
		results[entry.id] = urls;
	}
	return results;
}

export function rankingCases(querySet) {
	const nonCriticalIds = querySet.filter((entry) => !entry.critical).map((entry) => entry.id);
	const criticalIds = querySet.filter((entry) => entry.critical).map((entry) => entry.id);
	const missed = (ids, count) => new Set(ids.slice(0, count));

	return [
		{
			name: 'all expected pages in top 5 and every critical page in top 3',
			results: buildResultMap({ querySet, rankFor: (entry) => (entry.critical ? 1 : 2) }),
			expect: { passed: true, hits: querySet.length, criticalFailures: [], misses: [] },
		},
		{
			name: 'exactly 18 top-5 hits with critical pages at rank 3 still passes',
			results: buildResultMap({
				querySet,
				rankFor: (entry) => {
					if (entry.critical) return 3;
					return missed(nonCriticalIds, 2).has(entry.id) ? null : 5;
				},
			}),
			expect: { passed: true, hits: 18, criticalFailures: [], misses: nonCriticalIds.slice(0, 2) },
		},
		{
			name: '17 top-5 hits fails the 18/20 threshold',
			results: buildResultMap({
				querySet,
				rankFor: (entry) => {
					if (entry.critical) return 3;
					return missed(nonCriticalIds, 3).has(entry.id) ? null : 5;
				},
			}),
			expect: { passed: false, hits: 17, criticalFailures: [], misses: nonCriticalIds.slice(0, 3) },
		},
		{
			name: 'a critical page at rank 4 fails even with 20 top-5 hits',
			results: buildResultMap({ querySet, rankFor: (entry) => (entry.id === criticalIds[0] ? 4 : 2) }),
			expect: { passed: false, hits: querySet.length, criticalFailures: [criticalIds[0]] },
		},
		{
			name: 'an absent critical page fails',
			results: buildResultMap({ querySet, rankFor: (entry) => (entry.id === criticalIds[0] ? null : 2) }),
			expect: { passed: false, hits: querySet.length - 1, criticalFailures: [criticalIds[0]] },
		},
		{
			name: 'an expected page at rank 9 is recorded but is not a top-5 hit',
			results: buildResultMap({ querySet, rankFor: (entry) => (entry.id === nonCriticalIds[0] ? 9 : 2) }),
			expect: { passed: true, hits: querySet.length - 1, expectedRank: { id: nonCriticalIds[0], rank: 9 } },
		},
	];
}

export const PAGE_ID_CASES = Object.freeze([
	['/agentic-sdd/reference/cli/', '/agentic-sdd', 'reference/cli'],
	['/agentic-sdd/getting-started/installation/index.html', '/agentic-sdd', 'getting-started/installation'],
	['/getting-started/installation/', '', 'getting-started/installation'],
	['https://nawodyaishan.github.io/agentic-sdd/reference/mcp/?q=1#search', '/agentic-sdd', 'reference/mcp'],
	['/agentic-sdd/', '/agentic-sdd', 'index'],
	['index.html', '', 'index'],
	['/agentic-sdd/404.html', '/agentic-sdd', '404'],
	['/agentic-sdd/reference/files-and-backups/', '/agentic-sdd', 'reference/files-and-backups'],
	['/agentic-sdd/getting%2Dstarted/installation/', '/agentic-sdd', 'getting-started/installation'],
]);

export const PRIVATE_PATH_CASES = Object.freeze({
	forbidden: [
		['specs/020-documentation-ecosystem/spec', 'specs'],
		['docs/handovers/2026-11-14_020', 'handovers'],
		['.agentic-sdd/runs/20261114', 'agentic-sdd-state'],
		['backups/2026-11-14', 'backups'],
		['private/notes', 'private'],
		['reference/backups', 'backups'],
	],
	allowed: [
		'reference/files-and-backups',
		'operations/known-limitations',
		'how-to/updates-and-recovery',
		'index',
		'reference/cli',
	],
});

// A page that tries to escape the dist with a parent-relative asset reference.
export const ESCAPING_HTML = '<html><body><script src="../../../../etc/passwd"></script></body></html>';
