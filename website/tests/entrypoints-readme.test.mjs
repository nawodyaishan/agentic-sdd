// B7/T16 README entry-point contract.
//
// The README is the public front door, so these checks are deliberately about
// structure and destinations rather than prose: one visible ordinary Markdown
// Documentation link to the manifest-derived homepage, website-first
// getting-started and Explore routes, a secondary repository-source fallback,
// manifest-backed deep routes, the frozen A-01 anchors and guide prompts, the
// retained historical docs/getting-started.md pointer, current capability
// families and a safe unique temporary-home first success.
//
// Source-level and offline: reads README.md, the C1 manifest and the frozen
// guide prompts; Markdown structure comes from the existing collectBaseline
// helper. It never fetches the site, builds the site, executes a README fence or
// writes outside a unique temporary fixture directory. Passing here does not
// prove rendered visual prominence or accessibility, and it records no
// deployment or runtime observation.

import assert from 'node:assert/strict';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { collectBaseline, loadManifest } from '../scripts/lib/manifest.mjs';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const websiteDir = path.resolve(testDir, '..');
const repoRoot = path.resolve(websiteDir, '..');
const manifestPath = path.join(websiteDir, 'docs-manifest.json');
const readmePath = path.join(repoRoot, 'README.md');
const guidePath = path.join(repoRoot, 'cmd/agentic-sdd/guide.go');

const manifest = loadManifest(manifestPath, { repoRoot });
const SITE_ROOT = `${manifest.site}${manifest.base === '/' ? '/' : `${manifest.base}/`}`;
const SITE_ORIGIN = new URL(manifest.site).origin;
const PAGE_BY_ID = new Map(manifest.pages.map((page) => [page.id, page]));
const PAGE_IDS = new Set(PAGE_BY_ID.keys());

const FALLBACK_PATH = 'website/src/content/docs/index.md';
const RETAINED_POINTER = 'docs/getting-started.md';

// A-01 compatibility anchors: heading edits change GitHub fragments, so these
// exact slugs stay part of the README contract. The OpenCode heading keeps its
// literal double hyphen produced by "(`--with-opencode`)".
const A01_ANCHORS = [
	'quick-start',
	'support-and-release-scope',
	'where-the-skills-go',
	'where-the-files-go',
	'opencode-orchestration-pack---with-opencode',
	'the-workflow',
	'safety-model',
	'explore-further',
];

// FR-13 capability families and the direct canonical site pages that can
// satisfy each one. A relative source link is a fallback, not a family route.
const CAPABILITY_FAMILIES = [
	{ family: 'orientation', ids: ['index', 'introduction/concepts'] },
	{
		family: 'setup-and-first-success',
		ids: ['getting-started/choose-setup', 'getting-started/installation', 'getting-started/first-five-minutes'],
	},
	{ family: 'first-workflow', ids: ['tutorials/direct-fix', 'tutorials/first-feature'] },
	{ family: 'optional-learning', ids: ['tutorials/research-and-debugging', 'tutorials/specialists'] },
	{ family: 'clients-and-destinations', ids: ['introduction/clients'] },
	{ family: 'cli-and-tui', ids: ['reference/cli', 'reference/tui'] },
	{ family: 'migration', ids: ['releases/migration'] },
	{ family: 'skills', ids: ['reference/skills', 'how-to/skills'] },
	{ family: 'profiles-and-orchestration', ids: ['reference/profiles', 'how-to/orchestration'] },
	{ family: 'fleet', ids: ['how-to/fleet', 'reference/fleet'] },
	{ family: 'mcp', ids: ['how-to/mcp', 'reference/mcp'] },
	{ family: 'execution', ids: ['how-to/execution', 'reference/execution'] },
	{ family: 'compatibility', ids: ['releases/compatibility'] },
	{ family: 'limitations', ids: ['operations/known-limitations'] },
	{ family: 'security', ids: ['security/trust'] },
	{ family: 'recovery', ids: ['how-to/updates-and-recovery'] },
	{ family: 'files-and-backups', ids: ['reference/files-and-backups'] },
	{ family: 'architecture', ids: ['architecture/overview'] },
	{ family: 'contributing-and-testing', ids: ['contributing/testing', 'contributing/development'] },
];

function siteHref(id = '') {
	return id === '' ? SITE_ROOT : `${SITE_ROOT}${id}/`;
}

function actualReadme() {
	return {
		text: readFileSync(readmePath, 'utf8'),
		structure: collectBaseline(repoRoot, ['README.md']).files[0],
	};
}

// Mutation fixtures are authored synthetic READMEs, parsed with the same
// collectBaseline helper as the real file. The unique temporary directory keeps
// the fixture isolated and cleanup scoped.
function parseFixture(text) {
	const root = mkdtempSync(path.join(os.tmpdir(), 'agentic-sdd-b7-readme-'));
	try {
		writeFileSync(path.join(root, 'README.md'), text);
		return collectBaseline(root, ['README.md']).files[0];
	} finally {
		rmSync(root, { recursive: true, force: true });
	}
}

function siteUrl(href) {
	try {
		const url = new URL(href);
		return url.origin === SITE_ORIGIN ? url : null;
	} catch {
		return null;
	}
}

function isSiteLink(link) {
	return siteUrl(link.href) !== null;
}

function siteRouteId(href) {
	const url = siteUrl(href);
	if (url === null) return null;
	const prefix = manifest.base === '/' ? '/' : `${manifest.base}/`;
	if (!url.pathname.startsWith(prefix)) return null;
	const rest = url.pathname.slice(prefix.length).replace(/\/$/, '');
	return rest === '' ? 'index' : rest;
}

function linksInSection(structure, slug) {
	const index = structure.headings.findIndex((heading) => heading.slug === slug);
	if (index === -1) return null;
	const heading = structure.headings[index];
	const next = structure.headings.slice(index + 1).find((candidate) => candidate.level <= heading.level);
	const endLine = next === undefined ? Number.POSITIVE_INFINITY : next.line;
	return structure.links.filter((link) => link.line > heading.line && link.line < endLine);
}

function fencedLines(text) {
	const lines = text.replace(/^\uFEFF/, '').split(/\r?\n/);
	const fenced = [];
	let fence = null;
	for (let index = 0; index < lines.length; index += 1) {
		const line = lines[index];
		if (fence === null) {
			const opening = /^ {0,3}(`{3,}|~{3,})/.exec(line);
			if (opening !== null) fence = { character: opening[1][0], length: opening[1].length };
			continue;
		}
		const closing = /^ {0,3}(`{3,}|~{3,})[ \t]*$/.exec(line);
		if (closing !== null && closing[1][0] === fence.character && closing[1].length >= fence.length) {
			fence = null;
			continue;
		}
		fenced.push({ line: index + 1, text: line });
	}
	return fenced;
}

const fragmentCache = new Map();

function pageHasFragment(id, fragment) {
	if (!PAGE_BY_ID.has(id)) return false;
	if (!fragmentCache.has(id)) {
		const source = PAGE_BY_ID.get(id).source;
		const structure = collectBaseline(repoRoot, [source]).files[0];
		fragmentCache.set(id, new Set(structure.headings.map((heading) => heading.slug)));
	}
	return fragmentCache.get(id).has(fragment);
}

function hasIntroduction(text, h1Line, ctaLine) {
	const lines = text.replace(/^\uFEFF/, '').split(/\r?\n/);
	for (let index = h1Line; index < ctaLine - 1; index += 1) {
		const line = lines[index].trim();
		if (line === '') continue;
		if (/^#{1,6}\s/.test(line)) continue;
		if (/^[-*_]{3,}$/.test(line)) continue;
		if (/^(?:\[!\[|!\[|<img)/i.test(line)) continue;
		if (/[A-Za-z]{3,}/.test(line)) return true;
	}
	return false;
}

const INSTALL_COMMAND =
	/(?:\bbrew\s+install\b|\bgo\s+(?:build|install)\b[^\n]*\bcmd\/agentic-sdd\b|\bgit\s+clone\b[^\n]*\bagentic-sdd\b)/;

function firstInstallationLine(text) {
	const matches = fencedLines(text).filter((line) => INSTALL_COMMAND.test(line.text));
	return matches.length === 0 ? null : matches[0].line;
}

function analyzeCta({ text, structure }) {
	const findings = [];
	const labelled = structure.links.filter((link) => link.text.trim() === 'Documentation');
	if (labelled.length === 0) findings.push('missing-documentation-cta');
	else {
		if (!labelled.some((link) => link.href === SITE_ROOT)) findings.push('documentation-cta-wrong-url');
		if (labelled.some((link) => link.href !== SITE_ROOT)) findings.push('documentation-label-with-wrong-url');
	}

	const cta = labelled.find((link) => link.href === SITE_ROOT);
	if (cta === undefined) return findings;

	const h1 = structure.headings.find((heading) => heading.level === 1);
	const firstH2 = structure.headings.find((heading) => heading.level === 2);
	if (h1 === undefined) findings.push('missing-readme-h1');
	else if (!hasIntroduction(text, h1.line, cta.line)) findings.push('documentation-cta-before-introduction');

	if (firstH2 === undefined) findings.push('missing-first-h2');
	else if (cta.line > firstH2.line) findings.push('documentation-cta-after-first-h2');

	const installLine = firstInstallationLine(text);
	if (installLine !== null && cta.line > installLine) findings.push('documentation-cta-after-install-command');

	return findings;
}

function analyzeSiteLinks({ structure }) {
	const findings = [];
	const siteLinks = structure.links.filter(isSiteLink);
	if (siteLinks.length === 0) return ['missing-site-links'];

	for (const link of siteLinks) {
		const url = siteUrl(link.href);
		const prefix = manifest.base === '/' ? '/' : `${manifest.base}/`;

		if (url.search !== '') findings.push(`site-link-query:${link.href}`);
		if (url.pathname === manifest.base) {
			findings.push(`site-link-missing-trailing-slash:${link.href}`);
			continue;
		}
		if (!url.pathname.startsWith(prefix)) {
			findings.push(`site-link-wrong-base:${link.href}`);
			continue;
		}

		let rest = url.pathname.slice(prefix.length);
		if (rest !== '') {
			if (!rest.endsWith('/')) {
				findings.push(`site-link-missing-trailing-slash:${link.href}`);
				continue;
			}
			rest = rest.slice(0, -1);
			if (rest.split('/').some((segment) => segment === '' || segment === '.' || segment === '..')) {
				findings.push(`site-link-invalid-route:${link.href}`);
				continue;
			}
			if (!PAGE_IDS.has(rest) || rest === 'index') {
				findings.push(`site-link-unregistered-route:${link.href}`);
				continue;
			}
		}

		const id = rest === '' ? 'index' : rest;
		if (url.hash !== '') {
			let fragment;
			try {
				fragment = decodeURIComponent(url.hash.slice(1));
			} catch {
				fragment = url.hash.slice(1);
			}
			if (fragment === '') findings.push(`site-link-empty-fragment:${link.href}`);
			else if (!pageHasFragment(id, fragment)) findings.push(`site-link-unknown-fragment:${link.href}`);
		}
	}

	return findings;
}

function analyzeStartPriority({ structure }) {
	const findings = [];
	const siteStarts = structure.links.filter((link) => {
		const id = siteRouteId(link.href);
		return id !== null && id.startsWith('getting-started/');
	});
	if (siteStarts.length === 0) findings.push('missing-site-getting-started-link');

	const quickStart = linksInSection(structure, 'quick-start');
	if (quickStart === null) findings.push('missing-quick-start-section');
	else {
		const quickSiteStarts = quickStart.filter((link) => {
			const id = siteRouteId(link.href);
			return id !== null && id.startsWith('getting-started/');
		});
		if (quickSiteStarts.length === 0) findings.push('quick-start-not-website-first');
	}

	const legacyGuides = structure.links.filter((link) => /^docs\/[^/]+\.md(?:#.*)?$/.test(link.href));
	const firstSiteStart = siteStarts.length === 0 ? Number.POSITIVE_INFINITY : Math.min(...siteStarts.map((link) => link.line));
	const firstLegacyGuide =
		legacyGuides.length === 0 ? Number.POSITIVE_INFINITY : Math.min(...legacyGuides.map((link) => link.line));
	if (firstLegacyGuide < firstSiteStart) findings.push('legacy-guide-before-site-start');

	return findings;
}

function analyzeExplore({ structure }) {
	const links = linksInSection(structure, 'explore-further');
	if (links === null) return ['missing-explore-further-section'];
	if (links.length === 0) return ['missing-explore-further-links'];
	if (!isSiteLink(links[0])) return ['explore-further-not-site-first'];
	return [];
}

function analyzeFallback({ structure }) {
	const fallbacks = structure.links.filter((link) => link.href.replace(/^\.\//, '').split('#')[0] === FALLBACK_PATH);
	if (fallbacks.length === 0) return ['missing-source-fallback'];
	const cta = structure.links.find((link) => link.text.trim() === 'Documentation' && link.href === SITE_ROOT);
	if (cta !== undefined && fallbacks.some((link) => link.line < cta.line)) return ['source-fallback-before-cta'];
	return [];
}

function analyzeAnchors({ structure }) {
	const slugs = new Set(structure.headings.map((heading) => heading.slug));
	return A01_ANCHORS.filter((anchor) => !slugs.has(anchor)).map((anchor) => `missing-a01-anchor:${anchor}`);
}

function loadGuidePrompts() {
	const source = readFileSync(guidePath, 'utf8');
	const block = /var guideExamplePrompts = \[\]string\{([\s\S]*?)\n\}/.exec(source);
	assert.ok(block !== null, 'cmd/agentic-sdd/guide.go must keep guideExamplePrompts parseable as the prompt owner');
	const prompts = [...block[1].matchAll(/"(?:\\.|[^"\\])*"/g)].map((match) => JSON.parse(match[0]));
	assert.equal(prompts.length, 2, 'guideExamplePrompts must remain exactly the two everyday prompts');
	return prompts;
}

function analyzePrompts({ text }, prompts) {
	const lines = text.split(/\r?\n/);
	return prompts
		.filter((prompt) => !lines.some((line) => line.includes(prompt)))
		.map((_, index) => `guide-prompt-not-retained:${index + 1}`);
}

function analyzeRetainedPointer({ text, structure }) {
	if (!text.includes(RETAINED_POINTER)) return ['missing-retained-getting-started-pointer'];
	const links = structure.links.filter(
		(link) => link.href === RETAINED_POINTER || link.href.startsWith(`${RETAINED_POINTER}#`),
	);
	if (links.length === 0) return ['missing-retained-getting-started-pointer'];

	const lines = text.split(/\r?\n/);
	const findings = [];
	for (const link of links) {
		const context = [lines[link.line - 2], lines[link.line - 1], lines[link.line]].filter(Boolean).join(' ');
		if (!/retained|historical|legacy|worked example|worked path|original/i.test(context)) {
			findings.push('retained-pointer-not-labelled');
		}
	}
	return findings;
}

function analyzeCapabilities({ structure }) {
	const linked = new Set(structure.links.map((link) => siteRouteId(link.href)).filter((id) => id !== null));
	return CAPABILITY_FAMILIES.filter((entry) => !entry.ids.some((id) => linked.has(id))).map(
		(entry) => `missing-capability-family:${entry.family}`,
	);
}

function analyzeCommands({ text }) {
	const findings = [];
	const fenced = fencedLines(text);
	if (fenced.some((line) => /\/tmp\/test-home\b/.test(line.text))) findings.push('fixed-shared-temp-home');

	const previews = fenced.filter((line) => /\basdx(?:-core)?\s+preview\b/.test(line.text));
	// Only the install/apply subcommand is scoped by --home. `restore apply` is
	// a separate backup operation, not an unqualified write to the real home.
	const applies = fenced.filter((line) => /\basdx(?:-core)?\s+apply\b/.test(line.text));

	if (previews.length === 0) findings.push('missing-preview-command');
	if (previews.length > 0 && !previews.some((line) => /--home\s+\S+/.test(line.text))) {
		findings.push('preview-without-temp-home');
	}

	for (const line of applies) {
		const match = /--home\s+(?:"([^"]+)"|'([^']+)'|(\S+))/.exec(line.text);
		if (match === null) {
			findings.push(`apply-without-home:line-${line.line}`);
			continue;
		}
		const value = match[1] ?? match[2] ?? match[3];
		if (/^(?:\$HOME|~|\/home\/)/.test(value)) findings.push(`apply-real-home:line-${line.line}`);
		if (/^\/tmp\/test-home(?:\/|$)/.test(value)) findings.push('fixed-shared-temp-home');
	}

	if (applies.length > 0) {
		const firstPreview = previews.length === 0 ? Number.POSITIVE_INFINITY : previews[0].line;
		if (!(firstPreview < applies[0].line)) findings.push('apply-before-preview');
	}

	const joined = fenced.map((line) => line.text).join('\n');
	const uniqueMechanism = /mktemp\s+-d|\$TMPDIR|\$\{TMPDIR\}|\$RANDOM|\$\{RANDOM\}/i.test(joined);
	if (previews.length > 0) {
		const firstHome = /--home\s+(?:"([^"]+)"|'([^']+)'|(\S+))/.exec(previews[0].text);
		const firstValue = firstHome === null ? '' : (firstHome[1] ?? firstHome[2] ?? firstHome[3]);
		if (!uniqueMechanism || !/[\$<]/.test(firstValue)) findings.push('temp-home-not-unique');
	}

	return [...new Set(findings)];
}

function analyzeReadme({ text, structure }, prompts) {
	return [
		...analyzeCta({ text, structure }),
		...analyzeSiteLinks({ structure }),
		...analyzeStartPriority({ structure }),
		...analyzeExplore({ structure }),
		...analyzeFallback({ structure }),
		...analyzeAnchors({ structure }),
		...analyzePrompts({ text }, prompts),
		...analyzeRetainedPointer({ text, structure }),
		...analyzeCapabilities({ structure }),
		...analyzeCommands({ text }),
	];
}

// Independently authored compliant fixture. Mutation cases below change exactly
// one contract at a time; this fixture proves the detector accepts the intended
// shape rather than only rejecting real-file omissions.
function validReadme(prompts) {
	return [
		'# Agentic SDD',
		'',
		'Build software with AI. Stay the engineer.',
		'',
		'Agentic SDD is an open-source toolkit for scoped, reviewable AI-assisted changes.',
		'',
		`[Documentation](${SITE_ROOT})`,
		'',
		`Start with [Core concepts](${siteHref('introduction/concepts')}) or read the [website index](${FALLBACK_PATH}) source fallback.`,
		'',
		'## Quick start',
		'',
		`Choose a setup in [Choose your setup](${siteHref('getting-started/choose-setup')}), then follow [Installation](${siteHref('getting-started/installation')}) and [First five minutes](${siteHref('getting-started/first-five-minutes')}#step-1-create-a-disposable-home).`,
		'',
		'```sh',
		'preview_home=$(mktemp -d)',
		'asdx-core preview --home "$preview_home"',
		'asdx-core apply --home "$preview_home"   # optional fixture write; review first',
		'```',
		'',
		'## Support and release scope',
		'',
		`Read [Compatibility](${siteHref('releases/compatibility')}), [Migration](${siteHref('releases/migration')}) and [Known limitations](${siteHref('operations/known-limitations')}).`,
		'',
		'## Where the skills go',
		'',
		`See [Clients and destinations](${siteHref('introduction/clients')}) and the [Skills reference](${siteHref('reference/skills')}).`,
		'',
		'## Where the files go',
		'',
		`See [Files and backups](${siteHref('reference/files-and-backups')}).`,
		'',
		'## OpenCode Orchestration Pack (`--with-opencode`)',
		'',
		`Optional: [Orchestration](${siteHref('how-to/orchestration')}), [Profiles](${siteHref('reference/profiles')}) and [Fleet](${siteHref('how-to/fleet')}).`,
		'',
		'## The workflow',
		'',
		`Follow [Direct fix](${siteHref('tutorials/direct-fix')}) or [First feature](${siteHref('tutorials/first-feature')}); advanced readers can use [Research and debugging](${siteHref('tutorials/research-and-debugging')}).`,
		'',
		'### Everyday prompts in your project',
		'',
		`1. **Start or plan a feature:** ${prompts[0]}`,
		`2. **Fix a small bug:** ${prompts[1]}`,
		'',
		'## Safety model',
		'',
		`Read [Trust model](${siteHref('security/trust')}), [MCP](${siteHref('how-to/mcp')}) and [Updates and recovery](${siteHref('how-to/updates-and-recovery')}).`,
		'',
		'## Explore further',
		'',
		`- [Website](${SITE_ROOT}) — primary documentation.`,
		`- [CLI](${siteHref('reference/cli')}) and [TUI](${siteHref('reference/tui')}).`,
		`- [Execution](${siteHref('how-to/execution')}) and [Architecture](${siteHref('architecture/overview')}).`,
		`- [Testing](${siteHref('contributing/testing')}), [Contributing](CONTRIBUTING.md) and the [MIT License](LICENSE).`,
		`- Retained historical worked example: [old getting-started guide](${RETAINED_POINTER}).`,
		'',
	].join('\n');
}

function fixtureFindings(text, prompts) {
	return analyzeReadme({ text, structure: parseFixture(text) }, prompts);
}

test('README shows the exact Documentation call to action before the first H2 and install command', () => {
	const readme = actualReadme();
	const findings = analyzeCta(readme);
	assert.deepEqual(
		findings,
		[],
		`README must show a normal Markdown [Documentation](${SITE_ROOT}) link after the introduction and before the first H2/install command; got ${findings.join(', ')}`,
	);
});

test('README is website-first for getting started and does not lead onboarding through legacy docs', () => {
	const readme = actualReadme();
	const findings = analyzeStartPriority(readme);
	assert.deepEqual(
		findings,
		[],
		`README quick start must lead with registered getting-started site pages before any docs/ guide; got ${findings.join(', ')}`,
	);
});

test('README Explore further starts with the documentation website', () => {
	const readme = actualReadme();
	const findings = analyzeExplore(readme);
	assert.deepEqual(findings, [], `README Explore further must lead with a site page; got ${findings.join(', ')}`);
});

test('README keeps a secondary repository-relative source fallback after the call to action', () => {
	const readme = actualReadme();
	const findings = analyzeFallback(readme);
	assert.deepEqual(
		findings,
		[],
		`README must link the secondary source fallback ${FALLBACK_PATH} after the site CTA; got ${findings.join(', ')}`,
	);
});

test('every README site link uses the manifest site/base, a registered route, a trailing slash and a real fragment', () => {
	const readme = actualReadme();
	const findings = analyzeSiteLinks(readme);
	assert.deepEqual(
		findings,
		[],
		`README site links must stay under ${SITE_ROOT} and map to C1 page ids; got ${findings.join(', ')}`,
	);
});

test('README preserves the eight A-01 compatibility anchors', () => {
	const readme = actualReadme();
	const findings = analyzeAnchors(readme);
	assert.deepEqual(findings, [], `README must retain A-01 anchors; got ${findings.join(', ')}`);
});

test('README preserves both guideExamplePrompts as exact single-line prompts', () => {
	const prompts = loadGuidePrompts();
	const readme = actualReadme();
	const findings = analyzePrompts(readme, prompts);
	assert.deepEqual(
		findings,
		[],
		`README must retain both prompts verbatim from cmd/agentic-sdd/guide.go; got ${findings.join(', ')}`,
	);
});

test('README retains docs/getting-started.md as a clearly labelled historical worked-example pointer', () => {
	const readme = actualReadme();
	const findings = analyzeRetainedPointer(readme);
	assert.deepEqual(
		findings,
		[],
		`README must retain the literal ${RETAINED_POINTER} pointer with a retained/historical label; got ${findings.join(', ')}`,
	);
});

test('README gives every FR-13 capability family a direct documented site route', () => {
	const readme = actualReadme();
	const findings = analyzeCapabilities(readme);
	assert.deepEqual(
		findings,
		[],
		`README must route each capability family to a registered site page; got ${findings.join(', ')}`,
	);
});

test('README quick start previews in a unique temporary home and never defaults apply to the real home', () => {
	const readme = actualReadme();
	const findings = analyzeCommands(readme);
	assert.deepEqual(
		findings,
		[],
		`README must show a unique temporary home preview first, no shared /tmp/test-home and no bare apply; got ${findings.join(', ')}`,
	);
});

test('mutation fixture baseline satisfies the README contract', () => {
	const prompts = loadGuidePrompts();
	const text = validReadme(prompts);
	const findings = fixtureFindings(text, prompts);
	assert.deepEqual(findings, [], `valid fixture must pass; got ${findings.join(', ')}`);
});

test('mutation: an absent Documentation call to action fails', () => {
	const prompts = loadGuidePrompts();
	const text = validReadme(prompts).replace(`[Documentation](${SITE_ROOT})`, '');
	const findings = fixtureFindings(text, prompts);
	assert.ok(findings.includes('missing-documentation-cta'), `expected missing CTA; got ${findings.join(', ')}`);
});

test('mutation: moving the only Documentation link to the footer fails', () => {
	const prompts = loadGuidePrompts();
	const text = `${validReadme(prompts).replace(`[Documentation](${SITE_ROOT})`, '')}\n\n[Documentation](${SITE_ROOT})\n`;
	const findings = fixtureFindings(text, prompts);
	assert.ok(
		findings.includes('documentation-cta-after-first-h2'),
		`expected footer placement to fail; got ${findings.join(', ')}`,
	);
});

test('mutation: a badge or image-only Documentation link fails', () => {
	const prompts = loadGuidePrompts();
	const text = validReadme(prompts).replace(
		`[Documentation](${SITE_ROOT})`,
		`[![Documentation](https://img.shields.io/badge/docs-website-blue)](${SITE_ROOT})`,
	);
	const findings = fixtureFindings(text, prompts);
	assert.ok(
		findings.includes('missing-documentation-cta'),
		`expected image-only CTA to fail; got ${findings.join(', ')}`,
	);
});

test('mutation: a Documentation link inside a fenced block fails', () => {
	const prompts = loadGuidePrompts();
	const text = validReadme(prompts).replace(
		`[Documentation](${SITE_ROOT})`,
		['```md', `[Documentation](${SITE_ROOT})`, '```'].join('\n'),
	);
	const findings = fixtureFindings(text, prompts);
	assert.ok(
		findings.includes('missing-documentation-cta'),
		`expected fenced CTA to fail; got ${findings.join(', ')}`,
	);
});

test('mutation: a Documentation link hidden in an HTML comment fails', () => {
	const prompts = loadGuidePrompts();
	const text = validReadme(prompts).replace(
		`[Documentation](${SITE_ROOT})`,
		`<!-- [Documentation](${SITE_ROOT}) -->`,
	);
	const findings = fixtureFindings(text, prompts);
	assert.ok(
		findings.includes('missing-documentation-cta'),
		`expected commented CTA to fail; got ${findings.join(', ')}`,
	);
});

test('mutation: a site link under the wrong project base fails', () => {
	const prompts = loadGuidePrompts();
	const text = validReadme(prompts).replace(
		`[Documentation](${SITE_ROOT})`,
		'[Documentation](https://nawodyaishan.github.io/agentic-sdd-typo/)',
	);
	const findings = fixtureFindings(text, prompts);
	assert.ok(
		findings.includes('documentation-cta-wrong-url') &&
			findings.some((finding) => finding.startsWith('site-link-wrong-base')),
		`expected wrong-base failure; got ${findings.join(', ')}`,
	);
});

test('mutation: a nonexistent deep site route fails', () => {
	const prompts = loadGuidePrompts();
	const text = validReadme(prompts).replace(siteHref('how-to/fleet'), siteHref('how-to/nonexistent-route'));
	const findings = fixtureFindings(text, prompts);
	assert.ok(
		findings.includes(`site-link-unregistered-route:${siteHref('how-to/nonexistent-route')}`),
		`expected unregistered route failure; got ${findings.join(', ')}`,
	);
});

test('mutation: a missing repository source fallback fails', () => {
	const prompts = loadGuidePrompts();
	const text = validReadme(prompts).replace(
		` or read the [website index](${FALLBACK_PATH}) source fallback`,
		'',
	);
	const findings = fixtureFindings(text, prompts);
	assert.ok(findings.includes('missing-source-fallback'), `expected missing fallback; got ${findings.join(', ')}`);
});

test('mutation: removing one A-01 anchor fails', () => {
	const prompts = loadGuidePrompts();
	const text = validReadme(prompts).replace('## Safety model\n', '');
	const findings = fixtureFindings(text, prompts);
	assert.ok(
		findings.includes('missing-a01-anchor:safety-model'),
		`expected removed anchor failure; got ${findings.join(', ')}`,
	);
});

test('mutation: changing a guide prompt spelling fails', () => {
	const prompts = loadGuidePrompts();
	const text = validReadme(prompts).replace(prompts[0], prompts[0].replace('Stop before implementing.', 'Implement immediately.'));
	const findings = fixtureFindings(text, prompts);
	assert.ok(
		findings.includes('guide-prompt-not-retained:1'),
		`expected prompt drift failure; got ${findings.join(', ')}`,
	);
});

test('mutation: a fixed shared /tmp/test-home fails', () => {
	const prompts = loadGuidePrompts();
	const text = validReadme(prompts)
		.replace('preview_home=$(mktemp -d)', 'preview_home=/tmp/test-home')
		.replace('optional fixture write; review first', 'optional fixture write');
	const findings = fixtureFindings(text, prompts);
	assert.ok(findings.includes('fixed-shared-temp-home'), `expected fixed-temp failure; got ${findings.join(', ')}`);
});

test('mutation: an apply command without an explicit home fails', () => {
	const prompts = loadGuidePrompts();
	const text = validReadme(prompts).replace(
		'asdx-core apply --home "$preview_home"   # optional fixture write; review first',
		'asdx-core apply --home "$preview_home"\nasdx-core apply',
	);
	const findings = fixtureFindings(text, prompts);
	assert.ok(
		findings.some((finding) => finding.startsWith('apply-without-home:')),
		`expected bare-apply failure; got ${findings.join(', ')}`,
	);
});

test('mutation: dropping a capability family route fails', () => {
	const prompts = loadGuidePrompts();
	const text = validReadme(prompts).replace(
		`[Orchestration](${siteHref('how-to/orchestration')}), [Profiles](${siteHref('reference/profiles')}) and [Fleet](${siteHref('how-to/fleet')}).`,
		`[Orchestration](${siteHref('how-to/orchestration')}) and [Profiles](${siteHref('reference/profiles')}).`,
	);
	const findings = fixtureFindings(text, prompts);
	assert.ok(
		findings.includes('missing-capability-family:fleet'),
		`expected missing family failure; got ${findings.join(', ')}`,
	);
});

test('mutation: a legacy guide leading Explore further fails', () => {
	const prompts = loadGuidePrompts();
	const text = validReadme(prompts).replace(
		`- [Website](${SITE_ROOT}) — primary documentation.`,
		`- [Old guide](${RETAINED_POINTER})\n- [Website](${SITE_ROOT}) — primary documentation.`,
	);
	const findings = fixtureFindings(text, prompts);
	assert.ok(
		findings.includes('explore-further-not-site-first'),
		`expected legacy-first Explore failure; got ${findings.join(', ')}`,
	);
});
