// B2 entry and navigation contract, extended at B3: the explicit Starlight
// sidebar presents all 39 registered pages in the frozen groups, every destination is a registered page
// whose source exists, edit links follow the repository's `website/` layout, and
// the landing page is the first destination with a path to the setup chooser.
//
// Source-level and offline: parses astro.config.mjs (or the local sidebar module
// it imports), docs-manifest.json and the Markdown sources. It never runs a site
// build, so it is safe to run before Astro compiles anything.

import assert from 'node:assert/strict';
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { loadManifest } from '../scripts/lib/manifest.mjs';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const websiteDir = path.resolve(testDir, '..');
const repoRoot = path.resolve(websiteDir, '..');
const manifestPath = path.join(websiteDir, 'docs-manifest.json');
const docsRoot = path.join(websiteDir, 'src/content/docs');
const CONTENT_PREFIX = 'website/src/content/docs/';

// Frozen navigation (tasks.md A-04, extended at B3 so every registered page is
// reachable). Group order and membership are part of the contract; order inside
// a group is presentation, so it is compared as a set.
const EXPECTED_NAV = [
	{ label: 'Start here', slugs: ['index'] },
	{
		label: 'Introduction',
		slugs: ['introduction/concepts', 'introduction/clients', 'introduction/related-approaches'],
	},
	{
		label: 'Getting started',
		slugs: ['getting-started/choose-setup', 'getting-started/installation', 'getting-started/first-five-minutes'],
	},
	{
		label: 'Tutorials',
		slugs: [
			'tutorials/direct-fix',
			'tutorials/first-feature',
			'tutorials/specialists',
			'tutorials/research-and-debugging',
		],
	},
	{
		label: 'How-to',
		slugs: [
			'how-to/skills',
			'how-to/updates-and-recovery',
			'how-to/orchestration',
			'how-to/fleet',
			'how-to/mcp',
			'how-to/execution',
			'how-to/customization',
		],
	},
	{
		label: 'Reference',
		slugs: [
			'reference/cli',
			'reference/tui',
			'reference/files-and-backups',
			'reference/skills',
			'reference/profiles',
			'reference/configuration',
			'reference/fleet',
			'reference/mcp',
			'reference/execution',
		],
	},
	{
		label: 'Architecture',
		slugs: [
			'architecture/overview',
			'architecture/interfaces',
			'architecture/workflows',
			'architecture/evidence',
		],
	},
	{
		label: 'Operations',
		slugs: ['operations/troubleshooting', 'operations/known-limitations'],
	},
	{ label: 'Security', slugs: ['security/trust'] },
	{
		label: 'Contributing',
		slugs: ['contributing/development', 'contributing/testing', 'contributing/docs-and-changes'],
	},
	{
		label: 'Releases',
		slugs: ['releases/compatibility', 'releases/migration'],
	},
];

const L_PAGE_IDS = [
	'index',
	'introduction/concepts',
	'introduction/clients',
	'getting-started/choose-setup',
	'getting-started/installation',
	'getting-started/first-five-minutes',
	'tutorials/direct-fix',
	'tutorials/first-feature',
	'how-to/skills',
	'how-to/updates-and-recovery',
	'reference/cli',
	'reference/tui',
	'reference/files-and-backups',
	'reference/skills',
	'architecture/overview',
	'operations/troubleshooting',
	'operations/known-limitations',
	'security/trust',
	'contributing/development',
	'contributing/testing',
	'releases/compatibility',
	'releases/migration',
];
// Class C pages are built for direct links and, since B3, are also navigable.
const C_PAGE_IDS = [
	'introduction/related-approaches',
	'tutorials/specialists',
	'tutorials/research-and-debugging',
	'how-to/orchestration',
	'how-to/fleet',
	'how-to/mcp',
	'how-to/execution',
	'how-to/customization',
	'reference/profiles',
	'reference/configuration',
	'reference/fleet',
	'reference/mcp',
	'reference/execution',
	'architecture/interfaces',
	'architecture/workflows',
	'architecture/evidence',
	'contributing/docs-and-changes',
];
const ALL_PAGE_IDS = [...L_PAGE_IDS, ...C_PAGE_IDS].sort();

function loadDocumentationManifest() {
	return loadManifest(manifestPath, { repoRoot });
}

function pagesById(manifest) {
	return new Map(manifest.pages.map((page) => [page.id, page]));
}

function markdownFiles(dir, prefix = '') {
	const files = [];
	for (const entry of readdirSync(dir, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
		const relative = prefix === '' ? entry.name : `${prefix}/${entry.name}`;
		if (entry.isDirectory()) files.push(...markdownFiles(path.join(dir, entry.name), relative));
		else if (entry.isFile() && entry.name.endsWith('.md')) files.push(relative);
	}
	return files;
}

function skipStringLiteral(source, start) {
	const quote = source[start];
	for (let index = start + 1; index < source.length; index += 1) {
		if (source[index] === '\\') {
			index += 1;
			continue;
		}
		if (source[index] === quote) return index;
	}
	return source.length - 1;
}

function skipComment(source, start) {
	if (source[start + 1] === '/') {
		const newline = source.indexOf('\n', start);
		return newline === -1 ? source.length - 1 : newline;
	}
	const end = source.indexOf('*/', start + 2);
	return end === -1 ? source.length - 1 : end + 1;
}

function extractBalancedLiteral(source, openIndex) {
	let depth = 0;
	for (let index = openIndex; index < source.length; index += 1) {
		const char = source[index];
		if (char === "'" || char === '"' || char === '`') {
			index = skipStringLiteral(source, index);
			continue;
		}
		if (char === '/' && (source[index + 1] === '/' || source[index + 1] === '*')) {
			index = skipComment(source, index);
			continue;
		}
		if (char === '[' || char === '{') depth += 1;
		else if (char === ']' || char === '}') {
			depth -= 1;
			if (depth === 0) return source.slice(openIndex, index + 1);
		}
	}
	return null;
}

function extractSidebarArray(source) {
	const key = /(?:^|[\s,{])sidebar\s*:\s*/m.exec(source);
	if (key === null) return null;
	const openIndex = source.indexOf('[', key.index + key[0].length);
	if (openIndex === -1) return null;
	return extractBalancedLiteral(source, openIndex);
}

function readSidebar() {
	const inline = extractSidebarArray(readFileSync(path.join(websiteDir, 'astro.config.mjs'), 'utf8'));
	if (inline !== null) return { sourcePath: 'website/astro.config.mjs', literal: inline };

	// A local module is acceptable when astro.config.mjs imports the sidebar
	// rather than inlining it; the same literal parsing applies.
	for (const candidate of ['website/sidebar.mjs', 'website/config/sidebar.mjs', 'website/src/sidebar.mjs']) {
		const absolute = path.join(repoRoot, candidate);
		if (!existsSync(absolute)) continue;
		const moduleSource = readFileSync(absolute, 'utf8');
		const declaration = /(?:sidebar\s*=\s*|export\s+default\s+)\[/.exec(moduleSource);
		if (declaration === null) continue;
		const literal = extractBalancedLiteral(moduleSource, moduleSource.indexOf('[', declaration.index));
		if (literal !== null) return { sourcePath: candidate, literal };
	}
	return null;
}

function sidebarTokens(literal) {
	const tokens = [];
	let depth = 0;
	for (let index = 0; index < literal.length; index += 1) {
		const char = literal[index];
		if (char === '/' && (literal[index + 1] === '/' || literal[index + 1] === '*')) {
			index = skipComment(literal, index);
			continue;
		}
		if (char === "'" || char === '"' || char === '`') {
			const end = skipStringLiteral(literal, index);
			const head = literal.slice(Math.max(0, index - 24), index);
			const value = literal.slice(index + 1, end);
			if (/(?:^|[\s,{])label\s*:\s*$/.test(head)) tokens.push({ kind: 'label', value, depth });
			else if (/(?:^|[\s,{])slug\s*:\s*$/.test(head)) tokens.push({ kind: 'slug', value, depth });
			index = end;
			continue;
		}
		if (char === '[' || char === '{') depth += 1;
		else if (char === ']' || char === '}') depth -= 1;
	}
	return tokens;
}

function readSidebarGroups() {
	const sidebar = readSidebar();
	assert.ok(
		sidebar !== null,
		'tasks.md A-04 requires an explicit Starlight sidebar for the 39 registered pages in website/astro.config.mjs (or a local sidebar module); none was found',
	);
	const tokens = sidebarTokens(sidebar.literal);
	const labelDepths = tokens.filter((token) => token.kind === 'label').map((token) => token.depth);
	assert.ok(
		labelDepths.length > 0,
		`the sidebar in ${sidebar.sourcePath} has no labelled groups; expected the frozen grouped navigation`,
	);
	const groupDepth = Math.min(...labelDepths);
	const groups = [];
	for (const token of tokens) {
		if (token.kind === 'label') {
			if (token.depth === groupDepth) groups.push({ label: token.value, slugs: [] });
			continue;
		}
		if (groups.length > 0) groups[groups.length - 1].slugs.push(token.value);
	}
	return { sourcePath: sidebar.sourcePath, groups };
}

function maskMarkdown(body) {
	const lines = body.split('\n');
	let fence = null;
	for (let index = 0; index < lines.length; index += 1) {
		const line = lines[index];
		const open = /^[ \t]{0,3}(`{3,}|~{3,})/.exec(line);
		if (fence === null && open !== null) {
			fence = open[1];
			lines[index] = '';
			continue;
		}
		if (fence !== null) {
			if (line.trim().startsWith(fence)) fence = null;
			lines[index] = '';
			continue;
		}
		lines[index] = line.replace(/(`+)[^`]*?\1/g, '');
	}
	return lines;
}

// Normalised page IDs targeted by relative links in the given Markdown source.
function contentPageTargets(text, sourcePath) {
	const body = text.replace(/^---\r?\n[\s\S]*?\r?\n---[ \t]*\r?\n?/, '').replace(/<!--[\s\S]*?-->/g, '');
	const destinations = [];
	for (const line of maskMarkdown(body)) {
		for (const match of line.matchAll(/(?<!!)\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g)) destinations.push(match[2]);
		const definition = /^\[[^\]]+\]:\s*(\S+)/.exec(line);
		if (definition !== null) destinations.push(definition[1]);
	}
	const targets = [];
	for (const destination of destinations) {
		const clean = destination.split('#')[0].split('?')[0];
		if (clean === '' || /^[a-z][a-z0-9+.-]*:/i.test(clean) || clean.startsWith('/')) continue;
		const target = path.posix.normalize(path.posix.join(path.posix.dirname(sourcePath), clean));
		if (!target.startsWith(CONTENT_PREFIX)) continue;
		targets.push(target.slice(CONTENT_PREFIX.length).replace(/\.md$/, '').replace(/\/$/, ''));
	}
	return targets;
}

test('the explicit sidebar groups exactly the 39 registered pages', () => {
	const { sourcePath, groups } = readSidebarGroups();
	const actual = groups.map((group) => ({ label: group.label, slugs: [...group.slugs].sort() }));
	const expected = EXPECTED_NAV.map((group) => ({ label: group.label, slugs: [...group.slugs].sort() }));
	assert.deepEqual(actual, expected, `sidebar groups in ${sourcePath} must match the frozen navigation; found ${JSON.stringify(actual)}`);
});

test('every sidebar slug is a registered page whose source file exists', () => {
	const { groups } = readSidebarGroups();
	const byId = pagesById(loadDocumentationManifest());
	for (const slug of groups.flatMap((group) => group.slugs)) {
		const page = byId.get(slug);
		assert.ok(page, `sidebar slug "${slug}" does not resolve to a registered manifest page`);
		assert.equal(page.source, `${CONTENT_PREFIX}${slug}.md`, `sidebar slug "${slug}" must map to its conventional source path`);
		assert.equal(existsSync(path.join(repoRoot, page.source)), true, `sidebar slug "${slug}" points at missing source ${page.source}`);
	}
});

test('every registered page is reachable from the sidebar exactly once', () => {
	const { groups } = readSidebarGroups();
	const slugs = groups.flatMap((group) => group.slugs);
	assert.equal(slugs.length, ALL_PAGE_IDS.length, `sidebar must list ${ALL_PAGE_IDS.length} pages, found ${slugs.length}: ${slugs.join(', ')}`);
	assert.equal(new Set(slugs).size, slugs.length, 'sidebar lists a page more than once');
	assert.deepEqual([...slugs].sort(), [...ALL_PAGE_IDS].sort(), 'sidebar must present exactly the 39 registered page IDs');
});

test('registered pages keep their review classes while all appear in navigation', () => {
	const { groups } = readSidebarGroups();
	const slugs = new Set(groups.flatMap((group) => group.slugs));
	const byId = pagesById(loadDocumentationManifest());
	for (const id of L_PAGE_IDS) {
		assert.equal(byId.get(id)?.class, 'L', `B2 page ${id} must stay class L`);
	}
	for (const id of C_PAGE_IDS) {
		assert.equal(byId.get(id)?.class, 'C', `page ${id} must stay class C`);
		assert.equal(slugs.has(id), true, `class C page ${id} must appear in the sidebar exactly once since B3`);
	}
});

test('the manifest registers exactly the 39 expected pages with frozen review state', () => {
	const manifest = loadDocumentationManifest();
	const ids = manifest.pages.map((page) => page.id);
	assert.equal(ids.length, ALL_PAGE_IDS.length, `manifest must register ${ALL_PAGE_IDS.length} pages, found ${ids.length}`);
	assert.deepEqual([...ids].sort(), ALL_PAGE_IDS);
	for (const page of manifest.pages) {
		assert.equal(page.reviewStatus, 'unreviewed', `${page.id} reviewStatus`);
		assert.equal(page.appliesTo, 'unreleased', `${page.id} appliesTo`);
	}
});

test('the content tree and the manifest registry agree with no gaps', () => {
	const manifest = loadDocumentationManifest();
	const registered = manifest.pages.map((page) => {
		assert.ok(page.source.startsWith(CONTENT_PREFIX), `${page.id} source ${page.source} must live under ${CONTENT_PREFIX}`);
		return page.source.slice(CONTENT_PREFIX.length);
	});
	assert.equal(new Set(registered).size, registered.length, 'two pages share one source file');
	assert.deepEqual(
		[...registered].sort(),
		markdownFiles(docsRoot),
		'every Markdown page under website/src/content/docs must be registered, and every registered page must exist on disk',
	);
});

function readEditLinkBaseUrl() {
	const config = readFileSync(path.join(websiteDir, 'astro.config.mjs'), 'utf8');
	const key = /editLink\s*:\s*\{/.exec(config);
	assert.ok(key, 'Starlight editLink must be configured in website/astro.config.mjs');
	const literal = extractBalancedLiteral(config, config.indexOf('{', key.index));
	assert.ok(literal !== null, 'editLink configuration must be an object literal');
	const baseUrl = /baseUrl\s*:\s*(?:`([^`]*)`|'([^']*)'|"([^"]*)")/.exec(literal);
	assert.ok(baseUrl, 'editLink.baseUrl must be a string literal');
	return baseUrl[1] ?? baseUrl[2] ?? baseUrl[3];
}

test('editLink.baseUrl is the repository edit URL for website/ and derives every page URL', () => {
	const manifest = loadDocumentationManifest();
	const template = readEditLinkBaseUrl();
	const unresolved = template.replace(/\$\{repoUrl\}/g, manifest.repository);
	assert.doesNotMatch(unresolved, /\$\{/, 'editLink.baseUrl may interpolate only ${repoUrl}');
	const expectedBase = `${manifest.repository}/edit/main/website/`;
	assert.equal(unresolved, expectedBase, 'editLink.baseUrl must be the GitHub edit URL for the website/ directory');
	assert.match(unresolved, /^https:\/\/github\.com\/[^/]+\/[^/]+\/edit\/main\/website\/$/, 'editLink.baseUrl must be a fixed repository edit destination');

	for (const page of manifest.pages) {
		assert.ok(page.source.startsWith('website/'), `${page.id} source ${page.source} must live under website/`);
		const editUrl = `${unresolved}${page.source.slice('website/'.length)}`;
		assert.equal(editUrl, `${manifest.repository}/edit/main/${page.source}`, `${page.id} edit URL must resolve from its source path`);
		assert.equal(existsSync(path.join(repoRoot, page.source)), true, `${page.id} edit URL points at missing source ${page.source}`);
	}
});

test('the landing page is the first sidebar destination and links to the setup chooser', () => {
	const { groups } = readSidebarGroups();
	const slugs = groups.flatMap((group) => group.slugs);
	assert.equal(groups[0]?.label, 'Start here', 'the first sidebar group must be "Start here"');
	assert.equal(slugs[0], 'index', 'the landing index page must be the first sidebar destination');

	const byId = pagesById(loadDocumentationManifest());
	const indexPage = byId.get('index');
	assert.ok(indexPage, 'the landing index page must be registered');
	const targets = contentPageTargets(readFileSync(path.join(repoRoot, indexPage.source), 'utf8'), indexPage.source);
	assert.ok(
		targets.includes('getting-started/choose-setup'),
		`the landing page must link to the setup chooser; links found: ${targets.join(', ') || '(none)'}`,
	);
});
