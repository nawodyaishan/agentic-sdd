// Frozen Feature 020 plan §Q2 acceptance query set. The exact queries, expected
// page IDs and critical flags are asserted against
// specs/020-documentation-ecosystem/plan.md by search-acceptance-pagefind.test.mjs,
// so silent drift in either place fails the fixture check. Twenty queries are a
// directional acceptance fixture, not a population benchmark.

import { createHash } from 'node:crypto';

export const QUERY_SET_VERSION = '020-q2-v1';

// [query, expected page IDs, critical, coverage areas from the B5/T12 acceptance brief]
const rows = [
	['install on linux without homebrew', ['getting-started/installation'], true, ['install']],
	['preview without installing anything', ['getting-started/first-five-minutes', 'getting-started/installation'], true, ['preview']],
	['what is a direct fix', ['introduction/concepts', 'tutorials/direct-fix'], false, []],
	['how many documents does a feature need', ['tutorials/first-feature', 'introduction/concepts'], false, []],
	['claude code skills folder', ['introduction/clients', 'how-to/skills'], true, ['clients', 'skills']],
	['opencode agent profiles', ['reference/profiles'], false, []],
	['model fallback not working', ['operations/troubleshooting', 'how-to/orchestration'], false, []],
	['restore previous skills backup', ['how-to/updates-and-recovery', 'reference/files-and-backups'], true, ['backups']],
	['asdx-core vs asdx', ['reference/cli', 'releases/migration'], false, ['cli-flags']],
	['mcp server authentication', ['how-to/mcp', 'reference/mcp'], false, ['mcp']],
	['execution run cleanup', ['how-to/execution', 'reference/execution'], false, ['execution-runs']],
	['fleet config file location', ['reference/fleet', 'reference/configuration'], false, ['fleet']],
	['is my data sent anywhere', ['security/trust'], true, []],
	['what is not supported yet', ['operations/known-limitations', 'releases/compatibility'], true, []],
	['keyboard shortcuts in the tui', ['reference/tui'], false, ['tui-routes']],
	['add a new documentation page', ['contributing/docs-and-changes'], false, []],
	['run only the core tests', ['contributing/testing'], false, []],
	['which files get backed up', ['reference/files-and-backups'], false, ['backups']],
	['approval before coding', ['introduction/concepts', 'tutorials/first-feature'], false, []],
	['install skills for antigravity', ['introduction/clients', 'how-to/skills'], true, ['clients', 'skills']],
];

export const QUERY_SET = Object.freeze(
	rows.map(([query, expectedPageIds, critical, coverage], index) =>
		Object.freeze({
			id: `q${String(index + 1).padStart(2, '0')}`,
			query,
			expectedPageIds: Object.freeze([...expectedPageIds]),
			critical,
			coverage: Object.freeze([...coverage]),
		}),
	),
);

// The ten acceptance areas named by the Feature 020 B5/T12 brief. Every area must
// appear on at least one query so the fixed set cannot silently drop a topic.
export const REQUIRED_COVERAGE_AREAS = Object.freeze([
	'install',
	'preview',
	'clients',
	'skills',
	'cli-flags',
	'tui-routes',
	'backups',
	'execution-runs',
	'fleet',
	'mcp',
]);

export const TOTAL_QUERIES = 20;
export const TOP_N = 5;
export const MIN_TOP_N_HITS = 18;
export const CRITICAL_TOP_N = 3;

export const CRITICAL_QUERY_IDS = Object.freeze(QUERY_SET.filter((entry) => entry.critical).map((entry) => entry.id));

// Stable identity for an observation. Includes the coverage annotations so a
// fixture edit cannot pass unnoticed.
export function querySetDigest() {
	const canonical = QUERY_SET.map((entry) => ({
		id: entry.id,
		query: entry.query,
		expectedPageIds: [...entry.expectedPageIds],
		critical: entry.critical,
		coverage: [...entry.coverage],
	}));
	return createHash('sha256').update(JSON.stringify(canonical)).digest('hex');
}
