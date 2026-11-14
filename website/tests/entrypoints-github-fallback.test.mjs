// B2 GitHub-readable fallback and link fence: each page carries its own title and
// description, headings keep unique anchors, no section is images-only, and B2
// pages never link to B3-owned pages or the legacy docs/ tree.
//
// Source-level and offline: parses the Markdown that the manifest registers. It
// never runs a site build and never fetches anything.

import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { loadManifest } from '../scripts/lib/manifest.mjs';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const websiteDir = path.resolve(testDir, '..');
const repoRoot = path.resolve(websiteDir, '..');
const manifestPath = path.join(websiteDir, 'docs-manifest.json');
const CONTENT_PREFIX = 'website/src/content/docs/';

const B2_L_PAGE_IDS = [
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
];

// The B2 amendment's A-02 fence ("no relative links to B3-owned pages") was a
// staging guard: B3 pages did not exist when B2 shipped. B3 is now integrated,
// so the fence is retired and replaced with the stronger post-integration
// invariant below: every relative link resolves to a registered page or an
// existing repository file, and no page links the legacy docs/ tree.

const MARKDOWN_IMAGE = /!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g;
const HTML_IMAGE = /<img\b[^>]*>/gi;

function loadDocumentationManifest() {
	return loadManifest(manifestPath, { repoRoot });
}

function pagesById(manifest) {
	return new Map(manifest.pages.map((page) => [page.id, page]));
}

function pageText(page) {
	return readFileSync(path.join(repoRoot, page.source), 'utf8');
}

function splitFrontmatter(text) {
	const match = /^---\r?\n([\s\S]*?)\r?\n---[ \t]*\r?\n?/.exec(text);
	if (match === null) return { raw: '', body: text };
	return { raw: match[1], body: text.slice(match[0].length) };
}

function frontmatterValue(raw, field) {
	const lines = raw.split('\n');
	const pattern = new RegExp(`^${field}\\s*:`);
	const index = lines.findIndex((line) => pattern.test(line));
	if (index === -1) return null;
	const value = lines[index].replace(pattern, '').trim();
	if (/^[>|][-+]?$/.test(value)) {
		const block = [];
		for (let next = index + 1; next < lines.length; next += 1) {
			if (lines[next].trim() === '' || /^\S/.test(lines[next])) break;
			block.push(lines[next].trim());
		}
		return block.join(' ').trim();
	}
	return value.replace(/^(['"])([\s\S]*)\1$/, '$2').trim();
}

// Line views with fenced code and inline code blanked, so code samples cannot
// masquerade as headings, links or images.
function markdownLines(body) {
	const lines = body.replace(/<!--[\s\S]*?-->/g, '').split('\n');
	const view = [];
	let fence = null;
	for (const line of lines) {
		const open = /^[ \t]{0,3}(`{3,}|~{3,})/.exec(line);
		if (fence === null && open !== null) {
			fence = open[1];
			view.push({ text: line, code: true });
			continue;
		}
		if (fence !== null) {
			if (line.trim().startsWith(fence)) fence = null;
			view.push({ text: line, code: true });
			continue;
		}
		view.push({ text: line.replace(/(`+)[^`]*?\1/g, ''), code: false });
	}
	return view;
}

function headingText(source) {
	return source
		.replace(/!\[([^\]]*)\]\([^)]*\)/g, '$1')
		.replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
		.replace(/<[^>]*>/g, '')
		.trim();
}

// GitHub-style heading anchor, matching how GitHub renders `# Heading`.
function githubSlug(text) {
	return text
		.trim()
		.toLowerCase()
		.replace(/[^\p{L}\p{N}\p{M}\p{Pc}\- ]/gu, '')
		.replace(/ /g, '-');
}

function parsePage(text) {
	const { raw, body } = splitFrontmatter(text);
	const view = markdownLines(body);
	const headings = [];
	const sections = [];
	let current = { heading: null, slug: null, lines: [], hasCode: false };

	for (const line of view) {
		const match = line.code ? null : /^(#{1,6})[ \t]+(.*?)[ \t]*#*[ \t]*$/.exec(line.text);
		if (match !== null) {
			sections.push(current);
			const value = headingText(match[2]);
			current = { heading: value, slug: githubSlug(value), lines: [], hasCode: false };
			headings.push({ level: match[1].length, text: value, slug: githubSlug(value) });
			continue;
		}
		if (line.code) {
			if (line.text.trim().length > 0) current.hasCode = true;
			continue;
		}
		current.lines.push(line.text);
	}
	sections.push(current);

	return {
		frontmatter: raw,
		headings,
		sections: sections.map((section) => ({ ...section, text: section.lines.join('\n') })),
	};
}

// Relative link destinations resolved against the page's repository path.
function relativeLinks(text, sourcePath) {
	const { body } = splitFrontmatter(text);
	const destinations = [];
	for (const line of markdownLines(body)) {
		if (line.code) continue;
		for (const match of line.text.matchAll(/(?<!!)\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g)) {
			destinations.push(match[2]);
		}
		const definition = /^\[[^\]]+\]:\s*(\S+)/.exec(line.text);
		if (definition !== null) destinations.push(definition[1]);
	}

	const links = [];
	for (const destination of destinations) {
		const clean = destination.split('#')[0].split('?')[0];
		if (clean === '' || /^[a-z][a-z0-9+.-]*:/i.test(clean) || clean.startsWith('/')) continue;
		const target = path.posix.normalize(path.posix.join(path.posix.dirname(sourcePath), clean));
		links.push({ href: destination, target });
	}
	return links;
}

test('every registered page declares title and description frontmatter', () => {
	const manifest = loadDocumentationManifest();
	const missing = [];
	for (const page of manifest.pages) {
		const { frontmatter } = parsePage(pageText(page));
		for (const field of ['title', 'description']) {
			const value = frontmatterValue(frontmatter, field);
			if (value === null || value === '') missing.push(`${page.id}: ${field}`);
		}
	}
	assert.deepEqual(missing, [], 'each page needs non-empty title and description so GitHub and search views stay readable');
});

test('every registered page has unique heading anchors', () => {
	const manifest = loadDocumentationManifest();
	const duplicates = [];
	for (const page of manifest.pages) {
		const { headings } = parsePage(pageText(page));
		const seen = new Map();
		for (const heading of headings) {
			const text = heading.text === '' ? '(empty heading)' : heading.text;
			if (seen.has(heading.slug)) {
				duplicates.push(`${page.id}: "${text}" duplicates "${seen.get(heading.slug)}" as #${heading.slug}`);
				continue;
			}
			seen.set(heading.slug, text);
		}
	}
	assert.deepEqual(duplicates, [], 'duplicate heading anchors break GitHub and in-page navigation');
});

test('no page section is images-only and every image carries alternative text', () => {
	const manifest = loadDocumentationManifest();
	const violations = [];
	for (const page of manifest.pages) {
		const { sections } = parsePage(pageText(page));
		for (const section of sections) {
			const markdownImages = [...section.text.matchAll(MARKDOWN_IMAGE)];
			const htmlImages = [...section.text.matchAll(HTML_IMAGE)];
			const where = `${page.id}: section "${section.heading ?? '(top)'}"`;

			for (const image of markdownImages) {
				if (image[1].trim() === '') violations.push(`${where} image ${image[2]} has empty alt text`);
			}
			for (const image of htmlImages) {
				const alt = /\balt\s*=\s*(?:"([^"]*)"|'([^']*)')/i.exec(image[0]);
				if (alt === null || (alt[1] ?? alt[2]).trim() === '') violations.push(`${where} <img> has no alternative text`);
			}

			if (markdownImages.length + htmlImages.length === 0 || section.hasCode) continue;
			const residual = section.text.replace(MARKDOWN_IMAGE, '').replace(HTML_IMAGE, '').replace(/\s+/g, '');
			if (residual === '') violations.push(`${where} contains only images; keep essential information in text`);
		}
	}
	assert.deepEqual(violations, [], 'GitHub and no-image readers must still get each section\'s meaning');
});

test('every published page resolves its relative links and never links the legacy docs tree', () => {
	const manifest = loadDocumentationManifest();
	const bySource = new Map(manifest.pages.map((page) => [page.source, page]));
	const violations = [];
	const unregistered = [];

	for (const page of manifest.pages) {
		if (!bySource.has(page.source)) {
			unregistered.push(page.id);
			continue;
		}
		for (const link of relativeLinks(pageText(page), page.source)) {
			if (link.target.startsWith('docs/')) {
				violations.push(`${page.id} -> ${link.href} targets the legacy docs/ tree`);
				continue;
			}
			if (link.target.startsWith(CONTENT_PREFIX)) {
				if (!bySource.has(link.target)) violations.push(`${page.id} -> ${link.href} targets an unregistered page ${link.target}`);
				continue;
			}
			if (!existsSync(path.join(repoRoot, link.target))) {
				violations.push(`${page.id} -> ${link.href} does not exist at ${link.target}`);
			}
		}
	}

	assert.deepEqual(unregistered, [], 'every registered page must have a resolvable source before its links are verified');
	assert.deepEqual(violations, [], 'every relative link must resolve to a registered page or an existing repository file');
});
