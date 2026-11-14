// Pagefind acceptance harness for Feature 020 plan §Q2.
//
// The harness only reads a *supplied* built site. It never invokes `astro build`
// and never writes into the supplied dist. It prefers the Pagefind bundle the
// build already ships (the actual site search) and only rebuilds from the dist
// HTML into a temporary directory when that bundle is absent or its page count
// does not match the build. Missing build artifacts are `blocked`, never a silent
// skip and never a pass.
//
// The search call uses the pinned Pagefind version's own emitted browser API,
// served from the bundle's deployed `/pagefind/` path by a read-only fetch shim,
// so ranking is exactly the pinned engine's ranking. No second search engine is
// introduced.

import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync, lstatSync, readdirSync, readFileSync } from 'node:fs';
import { readFile } from 'node:fs/promises';
import { release as osRelease, tmpdir } from 'node:os';
import { dirname, isAbsolute, join, relative, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import {
	CRITICAL_TOP_N,
	MIN_TOP_N_HITS,
	QUERY_SET,
	TOP_N,
} from './queries.mjs';

export const WEBSITE_DIR = resolve(dirname(fileURLToPath(import.meta.url)), '..', '..');
export const REPO_ROOT = resolve(WEBSITE_DIR, '..');

// Pages emitted by the static build that are not content pages. Anything else
// outside website/docs-manifest.json is an allowlist finding, not a pass.
export const STATIC_PAGE_IDS = Object.freeze(['404']);

export const BLOCKED_GUIDANCE = [
	'Provide a built static site; this runner never builds one:',
	'  DOCS_DIST=/abs/path/to/website/dist node --test website/tests/search-acceptance-pagefind.test.mjs',
	'  node website/tests/search-acceptance-pagefind.test.mjs --built /abs/path/to/website/dist',
	'  DOCS_DIST=/abs/path/to/website/dist node website/tests/performance-baseline.test.mjs',
	'The build must be produced by the authorized parent (for example `cd website && npm run build`).',
	'Without a build the build-dependent check is reported as `blocked`, not as success.',
].join('\n');

// Paths that must never appear in the public site or in search results: specs,
// session handovers, private execution scratch (`.agentic-sdd`) and backups.
// Segment-anchored so the legitimate `reference/files-and-backups` page is not a
// false positive.
export const FORBIDDEN_PATH_RULES = Object.freeze([
	{ id: 'specs', pattern: /(^|\/)specs(\/|$)/i },
	{ id: 'handovers', pattern: /(^|\/)handovers?(\/|$)/i },
	{ id: 'agentic-sdd-state', pattern: /(^|\/)\.agentic-sdd(\/|$)/i },
	{ id: 'backups', pattern: /(^|\/)backups(\/|$)/i },
	{ id: 'private', pattern: /(^|\/)private(\/|$)/i },
]);

function parseBuiltFlag(argv) {
	let value = null;
	for (let index = 0; index < argv.length; index += 1) {
		const arg = argv[index];
		if (arg === '--built') {
			const next = argv[index + 1];
			if (next === undefined || next.startsWith('--')) {
				return { error: '--built requires a directory argument' };
			}
			value = next;
			index += 1;
		} else if (arg.startsWith('--built=')) {
			const inline = arg.slice('--built='.length);
			if (inline === '') {
				return { error: '--built= requires a directory argument' };
			}
			value = inline;
		}
	}
	return { value };
}

export function buildRequired({ argv = process.argv.slice(2), env = process.env } = {}) {
	return argv.includes('--require-build') || env.DOCS_REQUIRE_BUILD === '1';
}

function resolveInputPath(value, cwd) {
	return isAbsolute(value) ? resolve(value) : resolve(cwd, value);
}

function collectHtmlFiles(root) {
	const htmlFiles = [];
	const symlinks = [];
	const stack = [{ abs: root, rel: '' }];
	while (stack.length > 0) {
		const current = stack.pop();
		const entries = readdirSync(current.abs, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name, 'en'));
		for (const entry of entries) {
			const abs = join(current.abs, entry.name);
			const rel = current.rel === '' ? entry.name : `${current.rel}/${entry.name}`;
			const stat = lstatSync(abs);
			if (stat.isSymbolicLink() || entry.isSymbolicLink()) {
				symlinks.push(rel);
				continue;
			}
			if (stat.isDirectory()) {
				stack.push({ abs, rel });
				continue;
			}
			if (stat.isFile() && entry.name.toLowerCase().endsWith('.html')) {
				htmlFiles.push(rel);
			}
			if (htmlFiles.length > 20000) {
				throw new Error(`refusing to inspect more than 20000 HTML files under ${root}`);
			}
		}
	}
	return { htmlFiles: htmlFiles.sort(), symlinks };
}

function inspectDist(distDir) {
	if (!existsSync(distDir)) {
		return { kind: 'invalid', message: `built site not found: ${distDir}` };
	}
	const stat = lstatSync(distDir);
	if (!stat.isDirectory()) {
		return { kind: 'invalid', message: `built site is not a directory: ${distDir}` };
	}
	const inspected = collectHtmlFiles(distDir);
	if (inspected.symlinks.length > 0) {
		return {
			kind: 'invalid',
			message: `built site contains symlinks, which are not accepted as release input: ${inspected.symlinks.slice(0, 5).join(', ')}`,
		};
	}
	if (inspected.htmlFiles.length === 0) {
		return { kind: 'invalid', message: `built site has no HTML files: ${distDir}` };
	}
	return { kind: 'ready', distDir, htmlFiles: inspected.htmlFiles };
}

// Resolve DOCS_DIST or --built. `blocked` means no site was supplied; `invalid`
// means something was supplied but is not usable (that must fail loudly).
export function resolveBuildInput({ argv = process.argv.slice(2), env = process.env, cwd = process.cwd() } = {}) {
	const cli = parseBuiltFlag(argv);
	if (cli.error) {
		return { kind: 'invalid', message: cli.error };
	}
	const fromEnv = (env.DOCS_DIST ?? '').trim();
	const supplied = [];
	if (fromEnv !== '') {
		supplied.push({ source: 'DOCS_DIST', path: resolveInputPath(fromEnv, cwd) });
	}
	if (cli.value) {
		supplied.push({ source: '--built', path: resolveInputPath(cli.value, cwd) });
	}
	if (supplied.length === 0) {
		return { kind: 'blocked', message: 'no built static site supplied', guidance: BLOCKED_GUIDANCE };
	}
	if (supplied.length > 1 && supplied[0].path !== supplied[1].path) {
		return {
			kind: 'invalid',
			message: `conflicting build inputs: DOCS_DIST=${supplied[0].path} --built=${supplied[1].path}`,
		};
	}
	const chosen = supplied[supplied.length - 1];
	const inspected = inspectDist(chosen.path);
	if (inspected.kind !== 'ready') {
		return inspected;
	}
	return { ...inspected, source: chosen.source };
}

function decodePath(value) {
	try {
		return decodeURIComponent(value);
	} catch {
		return value;
	}
}

// Normalize a search URL to a docs page ID. Handles the configured base,
// trailing slashes, `index.html`, query strings, fragments, absolute URLs and
// percent-encoded paths.
export function toPageId(rawUrl, { base = '' } = {}) {
	let url = String(rawUrl ?? '').trim();
	if (url === '') {
		return 'index';
	}
	const absolute = /^[a-z][a-z0-9+.-]*:\/\//i.test(url);
	if (absolute) {
		url = new URL(url).pathname;
	}
	url = url.split('#')[0].split('?')[0];
	url = decodePath(url);

	const normalizedBase = `/${String(base).replace(/^\/+|\/+$/g, '')}`;
	if (normalizedBase !== '/' && (url === normalizedBase || url.startsWith(`${normalizedBase}/`))) {
		url = url.slice(normalizedBase.length);
	}
	url = url.replace(/^\/+/, '').replace(/\/+$/, '');
	url = url.replace(/(^|\/)index\.html$/, '$1').replace(/\.html$/, '');
	url = url.replace(/^\/+|\/+$/g, '');
	return url === '' ? 'index' : url;
}

export function findForbiddenPages(pageIds) {
	const violations = [];
	for (const pageId of pageIds) {
		for (const rule of FORBIDDEN_PATH_RULES) {
			if (rule.pattern.test(pageId)) {
				violations.push({ pageId, rule: rule.id });
				break;
			}
		}
	}
	return violations;
}

export function findUnlistedPages(pageIds, allowlist) {
	const unknown = new Set();
	for (const pageId of pageIds) {
		if (!allowlist.has(pageId)) {
			unknown.add(pageId);
		}
	}
	return [...unknown].sort();
}

export function loadDocsManifest(websiteDir = WEBSITE_DIR) {
	const manifestPath = join(websiteDir, 'docs-manifest.json');
	const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
	if (!Array.isArray(manifest.pages)) {
		throw new Error(`docs manifest has no pages array: ${manifestPath}`);
	}
	return manifest;
}

export function manifestAllowlist(manifest) {
	return new Set([...manifest.pages.map((page) => page.id), ...STATIC_PAGE_IDS]);
}

// Evaluate the fixed query set against recorded result URLs. Pure function: the
// live test and the fixture tests use the same decision logic.
export function evaluateQueryResults({
	results,
	querySet = QUERY_SET,
	base = '',
	topN = TOP_N,
	minTopNHits = MIN_TOP_N_HITS,
	criticalTopN = CRITICAL_TOP_N,
}) {
	const perQuery = querySet.map((entry) => {
		const urls = results[entry.id] ?? [];
		const pageIds = urls.map((url) => toPageId(url, { base }));
		const rankIndex = pageIds.findIndex((pageId) => entry.expectedPageIds.includes(pageId));
		const expectedRank = rankIndex === -1 ? null : rankIndex + 1;
		return {
			id: entry.id,
			query: entry.query,
			critical: entry.critical,
			expectedPageIds: [...entry.expectedPageIds],
			topN: pageIds.slice(0, topN),
			topNUrls: urls.slice(0, topN),
			expectedRank,
			resultCount: pageIds.length,
			hit: expectedRank !== null && expectedRank <= topN,
			criticalHit: entry.critical ? expectedRank !== null && expectedRank <= criticalTopN : null,
			pageIds,
		};
	});

	const hits = perQuery.filter((entry) => entry.hit).length;
	const criticalFailures = perQuery.filter((entry) => entry.critical && entry.criticalHit !== true);
	const emptyQueries = perQuery.filter((entry) => entry.resultCount === 0).map((entry) => entry.id);

	return {
		perQuery,
		hits,
		misses: perQuery.filter((entry) => !entry.hit).map((entry) => entry.id),
		criticalFailures: criticalFailures.map((entry) => entry.id),
		emptyQueries,
		threshold: { topN, minTopNHits, criticalTopN, total: querySet.length },
		passed: hits >= minTopNHits && criticalFailures.length === 0,
	};
}

export function describeEvaluation(evaluation) {
	const lines = evaluation.perQuery.map((entry) => {
		const rank = entry.expectedRank === null ? 'absent' : `rank ${entry.expectedRank}`;
		const flag = entry.critical ? ' critical' : '';
		return `  ${entry.id}${flag} "${entry.query}" -> ${entry.hit ? 'hit' : 'MISS'} (${rank}); top${evaluation.threshold.topN}=[${entry.topN.join(', ')}]`;
	});
	return [
		`top${evaluation.threshold.topN} hits ${evaluation.hits}/${evaluation.threshold.total} (minimum ${evaluation.threshold.minTopNHits}); critical top${evaluation.threshold.criticalTopN} failures: ${evaluation.criticalFailures.join(', ') || 'none'}`,
		...lines,
	].join('\n');
}

export function resolvePagefind(websiteDir = WEBSITE_DIR) {
	const packagePath = join(websiteDir, 'node_modules/pagefind/package.json');
	const cliPath = join(websiteDir, 'node_modules/pagefind/lib/runner/bin.cjs');
	if (!existsSync(packagePath) || !existsSync(cliPath)) {
		return null;
	}
	const version = JSON.parse(readFileSync(packagePath, 'utf8')).version;
	return { version, cliPath, packagePath };
}

function childEnv(workDir) {
	return {
		PATH: process.env.PATH ?? '',
		HOME: workDir,
		TMPDIR: process.env.TMPDIR ?? tmpdir(),
		LANG: 'C.UTF-8',
		LC_ALL: 'C.UTF-8',
		NO_COLOR: '1',
	};
}

// Build the Pagefind bundle from the supplied dist into a separate output
// directory. Timed on its own: this is search-index provisioning, not a site
// build.
export function buildSearchIndex({ cliPath, distDir, outputDir, timeout = 600000 }) {
	if (!existsSync(dirname(outputDir))) {
		throw new Error(`output parent does not exist: ${dirname(outputDir)}`);
	}
	const started = process.hrtime.bigint();
	execFileSync(process.execPath, [cliPath, '--site', distDir, '--output-path', outputDir, '--silent'], {
		cwd: dirname(outputDir),
		env: childEnv(dirname(outputDir)),
		stdio: ['ignore', 'pipe', 'pipe'],
		timeout,
		maxBuffer: 64 * 1024 * 1024,
	});
	const elapsedMs = Number(process.hrtime.bigint() - started) / 1e6;
	return { elapsedMs };
}

export function inspectSearchBundle(bundleDir) {
	const required = ['pagefind.js', 'pagefind-entry.json', 'pagefind-ui.css'];
	const missing = required.filter((name) => !existsSync(join(bundleDir, name)));
	const wasm = readdirSync(bundleDir).filter((name) => name.startsWith('wasm.') && name.endsWith('.pagefind'));
	const entryPath = join(bundleDir, 'pagefind-entry.json');
	const entry = existsSync(entryPath) ? JSON.parse(readFileSync(entryPath, 'utf8')) : null;
	const languages = entry?.languages ?? {};
	const pageCount = Object.values(languages).reduce((total, language) => total + Number(language.page_count ?? 0), 0);
	return {
		missing,
		wasmFiles: wasm.sort(),
		pageCount,
		version: entry?.version ?? null,
		languages: Object.keys(languages).sort(),
	};
}

// Prefer the Pagefind bundle the supplied build already ships: that is the
// actual site search. Fall back to rebuilding from the dist HTML only when the
// bundle is absent or its page inventory does not match the build, and record
// which happened so the source of the ranking is never ambiguous.
export function resolveSearchBundle({ distDir, workDir, cliPath, htmlCount }) {
	const shippedDir = join(distDir, 'pagefind');
	const shippedEntry = join(shippedDir, 'pagefind-entry.json');
	if (existsSync(join(shippedDir, 'pagefind.js')) && existsSync(shippedEntry)) {
		const shipped = inspectSearchBundle(shippedDir);
		const pageCountDelta = Number(shipped.pageCount) - htmlCount;
		if (shipped.missing.length === 0 && Math.abs(pageCountDelta) <= 1) {
			return {
				bundleDir: shippedDir,
				source: 'built-dist',
				provisioning: {
					status: 'not_run',
					reason: 'the supplied build already ships its Pagefind bundle; provisioning happened during the parent build',
				},
				shippedPageCount: shipped.pageCount,
				shippedBundlePresent: true,
				shippedBundleMissing: shipped.missing,
				pageCountDelta,
			};
		}
		const rebuiltDir = join(workDir, 'pagefind');
		const built = buildSearchIndex({ cliPath, distDir, outputDir: rebuiltDir });
		return {
			bundleDir: rebuiltDir,
			source: 'rebuilt-from-html',
			provisioning: {
				status: 'observed',
				elapsedMs: Number(built.elapsedMs.toFixed(3)),
				reason:
					shipped.missing.length > 0
						? 'the shipped bundle is incomplete, so ranking was rebuilt from the dist HTML'
						: 'the shipped bundle page count did not match the build inventory, so ranking was rebuilt from the dist HTML',
			},
			shippedPageCount: shipped.pageCount,
			shippedBundlePresent: true,
			shippedBundleMissing: shipped.missing,
			pageCountDelta,
			shippedBundleMismatch: true,
		};
	}
	const rebuiltDir = join(workDir, 'pagefind');
	const built = buildSearchIndex({ cliPath, distDir, outputDir: rebuiltDir });
	return {
		bundleDir: rebuiltDir,
		source: 'rebuilt-from-html',
		provisioning: {
			status: 'observed',
			elapsedMs: Number(built.elapsedMs.toFixed(3)),
			reason: 'the supplied build has no Pagefind bundle; ranking was built offline from the dist HTML',
		},
		shippedPageCount: null,
		shippedBundlePresent: false,
		shippedBundleMissing: [],
	};
}

function mimeFor(filePath) {
	if (filePath.endsWith('.wasm')) return 'application/wasm';
	if (filePath.endsWith('.json')) return 'application/json';
	if (filePath.endsWith('.js')) return 'text/javascript';
	if (filePath.endsWith('.css')) return 'text/css';
	return 'application/octet-stream';
}

let activeSearcher = null;

// The browser API derives result URLs from the bundle's web path and the site
// root above it, so the shim presents the bundle at the deployed
// `/pagefind/` path instead of a file:// path (which would mangle URLs). Only
// files inside the chosen bundle directory are served.
export const SEARCH_BUNDLE_BASE_PATH = '/pagefind/';

export async function createSearcher({ bundleDir }) {
	if (activeSearcher !== null) {
		throw new Error('a Pagefind searcher is already active in this process');
	}
	const bundleRoot = resolve(bundleDir);
	const bundleEntryUrl = pathToFileURL(join(bundleRoot, 'pagefind.js')).href;
	const originalFetch = globalThis.fetch;

	globalThis.fetch = async (input) => {
		const raw = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
		const url = new URL(raw, 'https://pagefind.invalid');
		if (url.origin !== 'https://pagefind.invalid' || !url.pathname.startsWith(SEARCH_BUNDLE_BASE_PATH)) {
			throw new Error(`searcher refused resource outside ${SEARCH_BUNDLE_BASE_PATH}: ${raw}`);
		}
		const relativePath = decodePath(url.pathname.slice(SEARCH_BUNDLE_BASE_PATH.length));
		const filePath = resolve(bundleRoot, relativePath);
		const relativeToRoot = relative(bundleRoot, filePath);
		if (relativeToRoot.startsWith('..') || isAbsolute(relativeToRoot)) {
			throw new Error(`searcher refused resource outside the bundle: ${raw}`);
		}
		const bytes = await readFile(filePath);
		return new Response(bytes, { status: 200, headers: { 'content-type': mimeFor(filePath) } });
	};

	try {
		const module = await import(bundleEntryUrl);
		const api = module.createInstance({ basePath: SEARCH_BUNDLE_BASE_PATH, noWorker: true });
		await api.init();
		activeSearcher = api;
		return {
			async search(term) {
				const response = await api.search(term);
				const fragments = await Promise.all(response.results.map((result) => result.data()));
				return fragments.map((fragment) => fragment.url);
			},
			async close() {
				try {
					await api.destroy?.();
				} finally {
					activeSearcher = null;
					globalThis.fetch = originalFetch;
				}
			},
		};
	} catch (error) {
		globalThis.fetch = originalFetch;
		throw error;
	}
}

export async function searchQuerySet(searcher, querySet = QUERY_SET) {
	const results = {};
	const timings = {};
	for (const entry of querySet) {
		const started = process.hrtime.bigint();
		results[entry.id] = await searcher.search(entry.query);
		timings[entry.id] = Number(process.hrtime.bigint() - started) / 1e6;
	}
	return { results, timings };
}

function tryGit(args, cwd) {
	try {
		return execFileSync('git', ['-C', cwd, ...args], {
			stdio: ['ignore', 'pipe', 'ignore'],
			timeout: 10000,
			encoding: 'utf8',
		}).trim();
	} catch {
		return null;
	}
}

export function fileSha256(path) {
	if (!existsSync(path)) {
		return null;
	}
	return createHash('sha256').update(readFileSync(path)).digest('hex');
}

// Exact input identity for an observation. Anything not actually observed is
// null, not a guessed value.
export function collectIdentity({ distDir, pagefind, dist }) {
	const dirty = tryGit(['status', '--porcelain'], REPO_ROOT);
	return {
		distPath: distDir,
		distDigest: dist.digest,
		distFileCount: dist.fileCount,
		distTotalBytes: dist.totalBytes,
		pagefindVersion: pagefind?.version ?? null,
		pagefindCli: pagefind ? relative(REPO_ROOT, pagefind.cliPath) : null,
		node: process.version,
		platform: `${process.platform}-${process.arch}`,
		osRelease: osRelease(),
		sourceCommit: tryGit(['rev-parse', 'HEAD'], REPO_ROOT),
		sourceDirty: dirty === null ? null : dirty !== '',
		lockfileSha256: fileSha256(join(WEBSITE_DIR, 'package-lock.json')),
		docsManifestSha256: fileSha256(join(WEBSITE_DIR, 'docs-manifest.json')),
	};
}

// stdout-only observation records. Nothing is written to the repository.
export function emitObservation(kind, observation) {
	console.log(`OBSERVATION ${kind} ${JSON.stringify(observation)}`);
}

export function emitStatus(kind, status, detail) {
	console.log(`STATUS ${kind}: ${status}${detail ? ` (${detail})` : ''}`);
}

export function announceBlocked(kind, gate) {
	emitStatus(kind, 'blocked', gate.message);
	console.log(`BLOCKED ${kind}: ${gate.message}`);
	console.log(gate.guidance ?? BLOCKED_GUIDANCE);
}
