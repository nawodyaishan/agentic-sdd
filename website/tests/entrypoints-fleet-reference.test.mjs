// B8/T19 external fleet reference contract (AC-29–AC-31, plan §13 / C8).
//
// The fleet documentation now features an external companion repository as a
// real-world blueprint, so these checks are about destination exactness,
// curated-prose placement, provenance/safety framing and copy-paste safety
// rather than prose style.
//
// Source-level and offline: reads the six target site pages, README.md and the
// C1 manifest; Markdown structure comes from the existing collectBaseline
// helper. It never fetches github.com, builds the site, executes a documented
// command or writes outside a unique temporary fixture directory. Passing here
// proves the Markdown contract only: not rendered quality, not external-repo
// availability, and not human batch acceptance.

import assert from 'node:assert/strict';
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { collectBaseline, loadManifest } from '../scripts/lib/manifest.mjs';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const websiteDir = path.resolve(testDir, '..');
const repoRoot = path.resolve(websiteDir, '..');
const manifestPath = path.join(websiteDir, 'docs-manifest.json');

const manifest = loadManifest(manifestPath, { repoRoot });

// C8: one authoritative URL, no deep blob/commit links that rot when the
// external repository advances.
const FLEET_URL = 'https://github.com/nawodyaishan/agentic-sdd-fleet';
const FLEET_NAME = 'agentic-sdd-fleet';

const HOW_TO = 'website/src/content/docs/how-to/fleet.md';
const REFERENCE = 'website/src/content/docs/reference/fleet.md';
const CONFIGURATION = 'website/src/content/docs/reference/configuration.md';
const CUSTOMIZATION = 'website/src/content/docs/how-to/customization.md';
const SPECIALISTS = 'website/src/content/docs/tutorials/specialists.md';
const TUI = 'website/src/content/docs/reference/tui.md';
const README = 'README.md';

const TARGET_PAGES = [HOW_TO, REFERENCE, CONFIGURATION, CUSTOMIZATION, SPECIALISTS, TUI];
const TARGET_FILES = [...TARGET_PAGES, README];
// These two pages mix generated tables with curated prose; C8 allows edits only
// in the curated part.
const GENERATED_PAGES = [REFERENCE, CONFIGURATION];

const EXPECTED_PAGE_IDS = new Map([
	[HOW_TO, 'how-to/fleet'],
	[REFERENCE, 'reference/fleet'],
	[CONFIGURATION, 'reference/configuration'],
	[CUSTOMIZATION, 'how-to/customization'],
	[SPECIALISTS, 'tutorials/specialists'],
	[TUI, 'reference/tui'],
]);

// ---------------------------------------------------------------------------
// Source loading
// ---------------------------------------------------------------------------

const structureCache = new Map();

function structureOf(relative) {
	if (!structureCache.has(relative)) {
		structureCache.set(relative, collectBaseline(repoRoot, [relative]).files[0]);
	}
	return structureCache.get(relative);
}

function actual(relative) {
	return {
		path: relative,
		text: readFileSync(path.join(repoRoot, relative), 'utf8'),
		structure: structureOf(relative),
	};
}

// Mutation fixtures are authored synthetic Markdown parsed with the same
// collectBaseline helper as the real files. The unique temporary directory keeps
// the fixture isolated and cleanup scoped.
function fixtureDoc(text, relative) {
	const root = mkdtempSync(path.join(os.tmpdir(), 'spec020-b8-fleet-'));
	try {
		const target = path.join(root, relative);
		mkdirSync(path.dirname(target), { recursive: true });
		writeFileSync(target, text);
		return { path: relative, text, structure: collectBaseline(root, [relative]).files[0] };
	} finally {
		rmSync(root, { recursive: true, force: true });
	}
}

// ---------------------------------------------------------------------------
// Line views: fenced code and generated regions are not curated prose
// ---------------------------------------------------------------------------

function analyzeLines(text) {
	const lines = text.replace(/^\uFEFF/, '').split(/\r?\n/);
	const code = new Set();
	let fence = null;
	for (let index = 0; index < lines.length; index += 1) {
		const line = lines[index];
		if (fence === null) {
			const opening = /^ {0,3}(`{3,}|~{3,})(.*)$/.exec(line);
			if (opening !== null && !(opening[1][0] === '`' && opening[2].includes('`'))) {
				fence = { character: opening[1][0], length: opening[1].length };
				code.add(index);
			}
			continue;
		}
		code.add(index);
		const closing = /^ {0,3}(`{3,}|~{3,})[ \t]*$/.exec(line);
		if (closing !== null && closing[1][0] === fence.character && closing[1].length >= fence.length) {
			fence = null;
		}
	}

	const generated = new Set();
	let open = null;
	for (let index = 0; index < lines.length; index += 1) {
		if (code.has(index)) continue;
		const start = /^ {0,3}<!--\s*generated:([^\s:]+):start\s*-->[ \t]*$/.exec(lines[index]);
		if (start !== null && open === null) {
			open = { name: start[1], line: index };
			continue;
		}
		const end = /^ {0,3}<!--\s*generated:([^\s:]+):end\s*-->[ \t]*$/.exec(lines[index]);
		if (end !== null && open !== null && end[1] === open.name) {
			for (let cursor = open.line; cursor <= index; cursor += 1) generated.add(cursor);
			open = null;
		}
	}
	if (open !== null) {
		for (let cursor = open.line; cursor < lines.length; cursor += 1) generated.add(cursor);
	}

	return { lines, code, generated };
}

function curatedEntries(view) {
	const entries = [];
	for (let index = 0; index < view.lines.length; index += 1) {
		if (view.code.has(index) || view.generated.has(index)) continue;
		entries.push({ index, text: view.lines[index] });
	}
	return entries;
}

function curatedText(view) {
	return curatedEntries(view)
		.map((entry) => entry.text)
		.join('\n');
}

function mentions(line) {
	return line.includes(FLEET_URL) || line.includes(FLEET_NAME);
}

function fencedBlocks(text) {
	const lines = text.replace(/^\uFEFF/, '').split(/\r?\n/);
	const blocks = [];
	let fence = null;
	let current = null;
	for (let index = 0; index < lines.length; index += 1) {
		const line = lines[index];
		if (fence === null) {
			const opening = /^ {0,3}(`{3,}|~{3,})(.*)$/.exec(line);
			if (opening !== null && !(opening[1][0] === '`' && opening[2].includes('`'))) {
				fence = { character: opening[1][0], length: opening[1].length };
				current = { start: index + 1, lines: [] };
			}
			continue;
		}
		const closing = /^ {0,3}(`{3,}|~{3,})[ \t]*$/.exec(line);
		if (closing !== null && closing[1][0] === fence.character && closing[1].length >= fence.length) {
			blocks.push(current);
			current = null;
			fence = null;
			continue;
		}
		current.lines.push({ line: index + 1, text: line });
	}
	if (current !== null) blocks.push(current);
	return blocks;
}

// ---------------------------------------------------------------------------
// Detectors
// ---------------------------------------------------------------------------

function analyzeFleetLink(doc, options = {}) {
	const findings = [];
	const view = analyzeLines(doc.text);
	const links = doc.structure.links.filter((link) => link.href === FLEET_URL);
	if (links.length === 0) findings.push('missing-fleet-url');

	const curated = links.filter((link) => !view.code.has(link.line - 1) && !view.generated.has(link.line - 1));
	if (options.curated === true && links.length > 0 && curated.length === 0) {
		findings.push('fleet-url-not-in-curated-prose');
	}

	for (const link of doc.structure.links) {
		if (link.href !== FLEET_URL && link.href.startsWith(FLEET_URL)) {
			findings.push(`fleet-deep-link:${link.href}`);
		}
	}
	return [...new Set(findings)];
}

// Curated-only placement for the two pages that carry generated tables.
function analyzeCuratedPlacement(doc) {
	const findings = analyzeFleetLink(doc, { curated: true });
	const view = analyzeLines(doc.text);
	if (view.generated.size === 0) findings.push('missing-generated-markers');
	const curated = curatedText(view);
	if (!curated.includes(FLEET_URL) && !curated.includes(FLEET_NAME)) {
		findings.push('missing-curated-fleet-reference');
	}
	return [...new Set(findings)];
}

// A reference that exists only inside a generated block is erased by the next
// `reference.mjs` regeneration, so it is never curated documentation.
function analyzeGeneratedIntrusion(doc) {
	const view = analyzeLines(doc.text);
	const findings = [];
	const curated = curatedText(view);
	if (!curated.includes(FLEET_URL) && !curated.includes(FLEET_NAME)) {
		findings.push('missing-curated-fleet-reference');
	}
	const inside = [];
	for (let index = 0; index < view.lines.length; index += 1) {
		if (!view.generated.has(index) || view.code.has(index)) continue;
		if (mentions(view.lines[index])) inside.push(index + 1);
	}
	if (inside.length > 0) findings.push(`fleet-reference-inside-generated-block:line-${inside[0]}`);
	return findings;
}

const NON_UPSTREAM =
	/\b(?:external|third[- ]party|independent|personal|community|companion)\b|\bnot (?:part of|maintained by|owned by|affiliated|upstream)\b|\bseparate (?:from|repository)\b|\bnot upstream\b/i;

const CREDENTIALS_STATEMENT =
	/\bno credentials\b|\bwithout (?:any )?credentials\b|\bnever (?:commit|store|include|put|add|paste)\b[^.\n]{0,80}\b(?:credentials?|secrets?|tokens?|api[- ]?keys?)\b|\b(?:credentials?|secrets?|tokens?)\b[^.\n]{0,40}\b(?:stay|stays|remain|remains|belong)\b[^.\n]{0,20}\b(?:out|outside|separate)\b|\bkeep\b[^.\n]{0,40}\b(?:credentials?|secrets?|tokens?)\b[^.\n]{0,30}\bout\b/i;

const CONFIG_ONLY = /\bconfiguration[- ]only\b/i;

function isRealHome(value) {
	return /^(?:\$HOME\b|\$\{HOME\}|~(?:\/|$)|\/home\/|\/Users\/|\/root(?:\/|$)|[A-Za-z]:\\Users\\)/.test(value);
}

function fencedHomeValues(view) {
	const values = [];
	for (let index = 0; index < view.lines.length; index += 1) {
		if (!view.code.has(index)) continue;
		const match = /--home\s+(?:"([^"]+)"|'([^']+)'|(\S+))/.exec(view.lines[index]);
		if (match !== null) values.push(match[1] ?? match[2] ?? match[3]);
	}
	return values;
}

function isolationEvidence(doc) {
	const view = analyzeLines(doc.text);
	const curated = curatedText(view);
	const isolatedHome = fencedHomeValues(view).some((value) => !isRealHome(value));
	const fleetConfig = /--fleet-config\b/.test(doc.text);
	const readOnly =
		/\bread[- ]only\b|\bpreview[- ]only\b|\bnever (?:scans|writes)\b|\bdoes not scan\b/i.test(curated);
	return isolatedHome || fleetConfig || readOnly;
}

// Provenance and safety framing. The mention must exist before the framing
// checks are meaningful; the four secondary pages only need the external/personal
// disclaimer next to the mention, while the case-study page carries the full set.
function analyzeProvenance(doc, options) {
	const view = analyzeLines(doc.text);
	const entries = curatedEntries(view);
	const curated = entries.map((entry) => entry.text).join('\n');
	if (!curated.includes(FLEET_URL) && !curated.includes(FLEET_NAME)) {
		return ['missing-fleet-reference'];
	}

	const findings = [];
	const positions = entries
		.map((entry, position) => (mentions(entry.text) ? position : -1))
		.filter((position) => position >= 0);
	const window = new Set();
	for (const position of positions) {
		for (let offset = -6; offset <= 6; offset += 1) {
			const target = position + offset;
			if (target >= 0 && target < entries.length) window.add(target);
		}
	}
	const near = [...window]
		.sort((left, right) => left - right)
		.map((position) => entries[position].text)
		.join('\n');
	const scope = options.wide === true ? curated : near;

	if (options.nonUpstream === true && !NON_UPSTREAM.test(scope)) {
		findings.push('missing-non-upstream-disclaimer');
	}
	if (options.credentials === true) {
		const stated = CREDENTIALS_STATEMENT.test(scope) || (options.configOnlyOk === true && CONFIG_ONLY.test(scope));
		if (!stated) findings.push('missing-no-credentials-statement');
	}
	if (options.isolation === true && !isolationEvidence(doc)) {
		findings.push('missing-preview-or-isolated-home');
	}
	return findings;
}

// A fenced command may continue over several lines, so look ahead for its flags
// before deciding it lacks an isolated home.
function commandTextAt(view, index) {
	const parts = [view.lines[index]];
	for (let cursor = index + 1; cursor < view.lines.length && parts.length < 4; cursor += 1) {
		if (!view.code.has(cursor)) break;
		const next = view.lines[cursor];
		if (next.trim() === '') break;
		if (/\basdx(?:-core)?\s+/.test(next)) break;
		parts.push(next);
	}
	return parts.join(' ');
}

// `fleet apply` and `fleet restore` fall back to the OS user home, so a fenced
// write command must carry an isolated `--home`; `--adopt` must never be
// combined with a real-home write. Prose that merely explains the flags is not
// an instruction and is deliberately not matched.
function analyzeCommands(doc) {
	const view = analyzeLines(doc.text);
	const findings = new Set();
	for (let index = 0; index < view.lines.length; index += 1) {
		const line = view.lines[index];
		if (!/\bfleet\s+(?:apply|restore)\b/.test(line)) continue;
		if (!view.code.has(index)) continue;
		const command = commandTextAt(view, index);
		const home = /--home\s+(?:"([^"]+)"|'([^']+)'|(\S+))/.exec(command);
		const value = home === null ? null : (home[1] ?? home[2] ?? home[3]);
		const isolated = value !== null && !isRealHome(value);
		if (!isolated) findings.add(`fleet-write-without-isolated-home:line-${index + 1}`);
		if (/--adopt\b/.test(command) && !isolated) findings.add(`unsafe-real-home-adopt:line-${index + 1}`);
	}
	return [...findings];
}

function instanceCount(body) {
	try {
		const parsed = JSON.parse(body.trim());
		if (parsed !== null && typeof parsed === 'object' && parsed.instances !== null && typeof parsed.instances === 'object') {
			return Array.isArray(parsed.instances) ? parsed.instances.length : Object.keys(parsed.instances).length;
		}
	} catch {
		// Not strict JSON; fall through to the structural count.
	}
	if (!/"instances"\s*:/.test(body)) return 0;
	const keys = [...body.matchAll(/^\s*"([a-z0-9][a-z0-9-]*)"\s*:\s*\{/gim)].map((match) => match[1]);
	return keys.filter((key) => key !== 'instances').length;
}

// AC-29: the external repository is a link target, never a bundled copy.
function analyzeFullFleetCopy(doc) {
	const findings = [];
	for (const block of fencedBlocks(doc.text)) {
		const body = block.lines.map((line) => line.text).join('\n');
		if (!/"instances"\s*:/.test(body)) continue;
		const count = instanceCount(body);
		if (count > 15) findings.push(`full-fleet-json-copy:line-${block.start}:${count}-instances`);
	}
	return findings;
}

function assertCorpusReferences() {
	const referencing = TARGET_FILES.filter((relative) => {
		const text = readFileSync(path.join(repoRoot, relative), 'utf8');
		return text.includes(FLEET_URL) || text.includes(FLEET_NAME);
	});
	assert.ok(
		referencing.length > 0,
		`the external fleet reference is absent from all ${TARGET_FILES.length} target files; there is nothing safe to verify yet`,
	);
}

// ---------------------------------------------------------------------------
// Authored fixtures (positive shape and one-contract mutations)
// ---------------------------------------------------------------------------

const SAFE_FLEET_COMMANDS = [
	'```sh',
	'asdx-core fleet status --config /path/to/fleet.json --home /path/to/isolated-home',
	'asdx-core fleet plan --config /path/to/fleet.json --home /path/to/isolated-home',
	'```',
];

const HOW_TO_FRAMING = [
	`[agentic-sdd-fleet](${FLEET_URL}) is an external, personal example repository observed on 2026-11-16.`,
	'It is not part of Agentic-SDD, it contains configuration only, and it stores no credentials or secrets.',
].join('\n');

function validHowTo() {
	return [
		'# Set up an agent fleet',
		'',
		'## Study a real fleet repository',
		'',
		HOW_TO_FRAMING,
		'',
		'Inspect it read-only, or write only to an isolated home:',
		'',
		...SAFE_FLEET_COMMANDS,
	].join('\n');
}

function validReference() {
	return [
		'# Fleet reference',
		'',
		'<!-- generated:reference/fleet:start -->',
		'Source-derived fleet facts for the current checkout. Only `schemaVersion` 1 is accepted.',
		'<!-- generated:reference/fleet:end -->',
		'',
		`The [agentic-sdd-fleet](${FLEET_URL}) repository is an external, personal example, separate from the offline fixtures under website/tests/examples/fleet/.`,
	].join('\n');
}

function validReadme() {
	return [
		'# Agentic SDD',
		'',
		`Fleet: [agentic-sdd-fleet](${FLEET_URL}) is an external 27-instance \`fleet.json\` example repository (configuration only, no credentials).`,
		'',
		'```sh',
		'preview_home=$(mktemp -d)',
		'asdx-core preview --home "$preview_home"',
		'```',
	].join('\n');
}

function manyInstanceDoc(count) {
	const instances = {};
	for (let index = 0; index < count; index += 1) {
		instances[`lane-${index}`] = { source: 'agent:sdd-orchestrator' };
	}
	return [
		'# Fleet reference',
		'',
		`The [agentic-sdd-fleet](${FLEET_URL}) repository is an external, personal example.`,
		'',
		'```json',
		JSON.stringify({ schemaVersion: 1, instances }, null, 2),
		'```',
	].join('\n');
}

// ---------------------------------------------------------------------------
// Preconditions and exact destination
// ---------------------------------------------------------------------------

test('the six fleet target pages are registered C1 pages', () => {
	const sources = new Map(manifest.pages.map((page) => [page.source, page.id]));
	for (const relative of TARGET_PAGES) {
		assert.equal(
			sources.get(relative),
			EXPECTED_PAGE_IDS.get(relative),
			`${relative} must stay registered in website/docs-manifest.json as ${EXPECTED_PAGE_IDS.get(relative)}`,
		);
	}
});

for (const relative of TARGET_FILES) {
	test(`${relative} links the exact external fleet repository outside code fences`, () => {
		const findings = analyzeFleetLink(actual(relative));
		assert.deepEqual(
			findings,
			[],
			`${relative} must carry a normal Markdown link to ${FLEET_URL} outside code fences; got ${findings.join(', ')}`,
		);
	});
}

for (const relative of GENERATED_PAGES) {
	test(`${relative} keeps the fleet reference in curated prose outside generated markers`, () => {
		const findings = analyzeCuratedPlacement(actual(relative));
		assert.deepEqual(
			findings,
			[],
			`${relative} must link ${FLEET_URL} in curated prose outside <!-- generated:...:start/end -->; got ${findings.join(', ')}`,
		);
	});

	test(`${relative} keeps every fleet-repository mention outside generated markers`, () => {
		const findings = analyzeGeneratedIntrusion(actual(relative));
		assert.deepEqual(
			findings,
			[],
			`${relative} must not rely on generated blocks for the external reference; got ${findings.join(', ')}`,
		);
	});
}

// ---------------------------------------------------------------------------
// Provenance and safety boundaries
// ---------------------------------------------------------------------------

test('how-to/fleet.md frames the external repository as non-upstream, credential-free and preview-first', () => {
	const findings = analyzeProvenance(actual(HOW_TO), {
		nonUpstream: true,
		credentials: true,
		isolation: true,
		wide: true,
	});
	assert.deepEqual(
		findings,
		[],
		`${HOW_TO} must state the external/personal status, that no credentials belong there, and preview or isolated --home usage; got ${findings.join(', ')}`,
	);
});

test('every page and the README frames the external repository as external and personal', () => {
	const findings = TARGET_FILES.flatMap((relative) =>
		analyzeProvenance(actual(relative), { nonUpstream: true }).map((finding) => `${relative}: ${finding}`),
	);
	assert.deepEqual(
		findings,
		[],
		`each file that references ${FLEET_NAME} must say it is external/personal and not upstream; got ${findings.join(', ')}`,
	);
});

test('README states that the external fleet repository is configuration only with no credentials', () => {
	const findings = analyzeProvenance(actual(README), { nonUpstream: true, credentials: true, configOnlyOk: true });
	assert.deepEqual(
		findings,
		[],
		`${README} must keep the external fleet example credential-free (configuration only, no credentials); got ${findings.join(', ')}`,
	);
});

test('no target file instructs a real-home fleet apply, restore or adopt', () => {
	assertCorpusReferences();
	const findings = TARGET_FILES.flatMap((relative) =>
		analyzeCommands(actual(relative)).map((finding) => `${relative}: ${finding}`),
	);
	assert.deepEqual(
		findings,
		[],
		`every fenced fleet apply/restore needs an isolated --home and no real-home --adopt; got ${findings.join(', ')}`,
	);
});

test('no target file pastes a full multi-instance fleet.json copy', () => {
	assertCorpusReferences();
	const findings = TARGET_FILES.flatMap((relative) =>
		analyzeFullFleetCopy(actual(relative)).map((finding) => `${relative}: ${finding}`),
	);
	assert.deepEqual(
		findings,
		[],
		`docs must link the external repository instead of copying more than 15 instances; got ${findings.join(', ')}`,
	);
});

test('no target file uses a deep or non-canonical external fleet URL', () => {
	assertCorpusReferences();
	const findings = TARGET_FILES.flatMap((relative) =>
		analyzeFleetLink(actual(relative))
			.filter((finding) => finding.startsWith('fleet-deep-link:'))
			.map((finding) => `${relative}: ${finding}`),
	);
	assert.deepEqual(
		findings,
		[],
		`only the bare ${FLEET_URL} is authoritative; deep blob/commit/clone URLs rot; got ${findings.join(', ')}`,
	);
});

// ---------------------------------------------------------------------------
// Mutation fixtures: the detectors must reject what the contract forbids
// ---------------------------------------------------------------------------

test('fixture: a compliant how-to page satisfies every detector', () => {
	const doc = fixtureDoc(validHowTo(), HOW_TO);
	const findings = [
		...analyzeFleetLink(doc),
		...analyzeProvenance(doc, { nonUpstream: true, credentials: true, isolation: true, wide: true }),
		...analyzeCommands(doc),
		...analyzeFullFleetCopy(doc),
	];
	assert.deepEqual(findings, [], `valid how-to fixture must pass; got ${findings.join(', ')}`);
});

test('fixture: a compliant generated reference page satisfies curated placement', () => {
	const doc = fixtureDoc(validReference(), REFERENCE);
	const findings = [...analyzeCuratedPlacement(doc), ...analyzeGeneratedIntrusion(doc)];
	assert.deepEqual(findings, [], `valid reference fixture must pass; got ${findings.join(', ')}`);
});

test('fixture: a compliant README satisfies link and provenance checks', () => {
	const doc = fixtureDoc(validReadme(), README);
	const findings = [
		...analyzeFleetLink(doc),
		...analyzeProvenance(doc, { nonUpstream: true, credentials: true, configOnlyOk: true }),
	];
	assert.deepEqual(findings, [], `valid README fixture must pass; got ${findings.join(', ')}`);
});

test('mutation: removing the URL from a site page fails', () => {
	const doc = fixtureDoc(validHowTo().replace(FLEET_URL, ''), HOW_TO);
	const findings = analyzeFleetLink(doc);
	assert.ok(findings.includes('missing-fleet-url'), `expected missing URL; got ${findings.join(', ')}`);
});

test('mutation: removing the URL from the README fails', () => {
	const doc = fixtureDoc(validReadme().replace(FLEET_URL, ''), README);
	const findings = analyzeFleetLink(doc);
	assert.ok(findings.includes('missing-fleet-url'), `expected missing URL; got ${findings.join(', ')}`);
});

test('mutation: placing the URL only inside a generated block fails', () => {
	const text = [
		'# Fleet reference',
		'',
		'<!-- generated:reference/fleet:start -->',
		`See [agentic-sdd-fleet](${FLEET_URL}).`,
		'<!-- generated:reference/fleet:end -->',
		'',
		'Curated prose that never links the external repository.',
	].join('\n');
	const findings = analyzeCuratedPlacement(fixtureDoc(text, REFERENCE));
	assert.ok(
		findings.includes('fleet-url-not-in-curated-prose') && findings.includes('missing-curated-fleet-reference'),
		`expected generated-only URL to fail; got ${findings.join(', ')}`,
	);
});

test('mutation: placing the URL only inside a code fence fails', () => {
	const text = [
		'# Set up an agent fleet',
		'',
		'```md',
		`[agentic-sdd-fleet](${FLEET_URL})`,
		'```',
		'',
		'External, personal example.',
	].join('\n');
	const findings = analyzeFleetLink(fixtureDoc(text, HOW_TO));
	assert.ok(findings.includes('missing-fleet-url'), `expected fenced-only URL to fail; got ${findings.join(', ')}`);
});

test('mutation: an unsafe real-home fleet apply --adopt without --home fails', () => {
	const text = [
		validHowTo(),
		'',
		'```sh',
		'asdx-core fleet apply --adopt --plan /path/to/fleet-plan.json',
		'```',
	].join('\n');
	const findings = analyzeCommands(fixtureDoc(text, HOW_TO));
	assert.ok(
		findings.some((finding) => finding.startsWith('unsafe-real-home-adopt:')) &&
			findings.some((finding) => finding.startsWith('fleet-write-without-isolated-home:')),
		`expected real-home adopt to fail; got ${findings.join(', ')}`,
	);
});

test('mutation: dropping the non-upstream disclaimer fails', () => {
	const text = validHowTo().replace(HOW_TO_FRAMING, `[agentic-sdd-fleet](${FLEET_URL}) is a repository observed on 2026-11-16.`);
	const findings = analyzeProvenance(fixtureDoc(text, HOW_TO), { nonUpstream: true, wide: true });
	assert.ok(
		findings.includes('missing-non-upstream-disclaimer'),
		`expected missing disclaimer to fail; got ${findings.join(', ')}`,
	);
});

test('mutation: dropping the no-credentials statement fails', () => {
	const text = validHowTo().replace('it contains configuration only, and it stores no credentials or secrets', 'it contains configuration only');
	const findings = analyzeProvenance(fixtureDoc(text, HOW_TO), { credentials: true, wide: true });
	assert.ok(
		findings.includes('missing-no-credentials-statement'),
		`expected missing credential boundary to fail; got ${findings.join(', ')}`,
	);
});

test('mutation: pasting a full fleet.json copy fails', () => {
	const findings = analyzeFullFleetCopy(fixtureDoc(manyInstanceDoc(20), REFERENCE));
	assert.ok(
		findings.some((finding) => finding.startsWith('full-fleet-json-copy:')),
		`expected a 20-instance copy to fail; got ${findings.join(', ')}`,
	);
});

test('mutation: replacing the repository URL with a deep blob URL fails', () => {
	const doc = fixtureDoc(validReference().replace(FLEET_URL, `${FLEET_URL}/blob/main/fleet.json`), REFERENCE);
	const findings = analyzeFleetLink(doc, { curated: true });
	assert.ok(
		findings.includes('missing-fleet-url') && findings.some((finding) => finding.startsWith('fleet-deep-link:')),
		`expected deep blob URL to fail; got ${findings.join(', ')}`,
	);
});
