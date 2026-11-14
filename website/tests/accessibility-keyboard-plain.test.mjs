// Spec 020 T12 / B5-W1-F — Q1 keyboard/plain-Markdown source checks.
// Source observations only. Actual focus order, focus traps, search Esc,
// horizontal scrolling and screen-reader behavior require a real browser or
// assistive technology and are recorded as not_run below. No browser is started.

import assert from 'node:assert/strict';
import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const websiteDir = path.resolve(testDir, '..');
const repoRoot = path.resolve(websiteDir, '..');
const docsDir = path.resolve(websiteDir, 'src/content/docs');
const manifest = JSON.parse(readFileSync(path.resolve(websiteDir, 'docs-manifest.json'), 'utf8'));

function listMarkdown(dir) {
	const out = [];
	for (const entry of readdirSync(dir, { withFileTypes: true })) {
		const full = path.join(dir, entry.name);
		const stat = statSync(full);
		assert.equal(stat.isSymbolicLink(), false, `unexpected symlink: ${full}`);
		if (stat.isDirectory()) out.push(...listMarkdown(full));
		else if (/\.mdx?$/.test(full) && !/^404\.mdx?$/.test(entry.name)) out.push(full);
	}
	return out;
}

function repoRelative(file) {
	return path.relative(repoRoot, file).split(path.sep).join('/');
}

function withoutFencedCode(source) {
	const lines = source.split(/\r?\n/);
	let fence = null;
	const kept = [];
	for (const line of lines) {
		const marker = line.match(/^\s{0,3}(`{3,}|~{3,})/);
		if (fence) {
			if (marker && marker[1][0] === fence[0] && marker[1].length >= fence.length) fence = null;
			kept.push('');
			continue;
		}
		if (marker) {
			fence = marker[1];
			kept.push('');
			continue;
		}
		kept.push(line);
	}
	return kept.join('\n');
}

function visibleMarkdown(source) {
	const frontmatter = source.replace(/^---\r?\n[\s\S]*?\r?\n---(?:\r?\n|$)/, '');
	return withoutFencedCode(frontmatter)
		.replace(/<!--[\s\S]*?-->/g, '')
		.split(/\r?\n/)
		.map((line) => line.replace(/(`+)([^`]*?)\1/g, ''))
		.join('\n');
}

const ALLOWED_RAW_TAGS = new Set([
	'a',
	'img',
	'br',
	'hr',
	'kbd',
	'sub',
	'sup',
	'details',
	'summary',
	'code',
	'pre',
	'mark',
]);

// Existing prose uses bare angle-bracket placeholders such as `mcp.<name>` and
// `<home>` path roots. They are not elements and carry no interaction. A tag
// with attributes or a closing form is still treated as raw HTML.
const PLACEHOLDER_TAGS = new Set([
	'name',
	'home',
	'repo',
	'path',
	'task',
	'check-id',
	'pkg-id',
	'rec-id',
	'run-root',
	'feature',
	'profile',
	'server',
]);

const POINTER_HANDLER = /\son(?:click|dblclick|mousedown|mouseup|mouseover|pointerdown|touchstart)\s*=/i;

function rawTagFindings(source) {
	const findings = [];
	for (const match of source.matchAll(/<\/?([A-Za-z][A-Za-z0-9-]*)(?=[\s/>])[^>]*>/g)) {
		const tag = match[1];
		const barePlaceholder = match[0].match(/^<([a-z][a-z0-9-]*)>$/);
		if (barePlaceholder && PLACEHOLDER_TAGS.has(barePlaceholder[1])) continue;
		if (!ALLOWED_RAW_TAGS.has(tag.toLowerCase())) {
			findings.push(`raw <${tag}> is not part of the plain-Markdown contract`);
			continue;
		}
		if (match[0].startsWith('</')) continue;
		if (tag.toLowerCase() === 'a') {
			const anchor = match[0];
			const href = anchor.match(/\bhref\s*=\s*["']([^"']*)["']/i);
			if (!href || !href[1].trim()) findings.push('<a> without a non-empty href');
			else if (/^javascript:/i.test(href[1])) findings.push('<a> uses javascript: URL');
			else if (href[1].trim() === '#') findings.push('<a href="#"> is a placeholder, not a target');
		}
		if (tag.toLowerCase() === 'img' && !/\balt\s*=\s*["'][^"']+["']/i.test(match[0])) {
			findings.push('<img> without non-empty alt');
		}
	}
	return findings;
}

function fenceIsBalanced(source) {
	let fence = null;
	for (const line of source.split(/\r?\n/)) {
		const marker = line.match(/^\s{0,3}(`{3,}|~{3,})/);
		if (!marker) continue;
		if (!fence) fence = marker[1];
		else if (marker[1][0] === fence[0] && marker[1].length >= fence.length) fence = null;
	}
	return fence === null;
}

const docFiles = listMarkdown(docsDir);
const docSources = new Set(docFiles.map(repoRelative));

test('docs content has no pointer-only interaction or non-native interactive markup', () => {
	const problems = [];
	for (const file of docFiles) {
		const source = visibleMarkdown(readFileSync(file, 'utf8'));
		for (const finding of rawTagFindings(source)) {
			problems.push(`${repoRelative(file)}: ${finding}`);
		}
		if (POINTER_HANDLER.test(source)) {
			problems.push(`${repoRelative(file)}: inline pointer-only event handler`);
		}
	}

	// Fixture verification: the detector is not a tautology.
	assert.equal(rawTagFindings('<div onclick="open()">open</div>').length >= 1, true);
	assert.equal(rawTagFindings('<a href="#">placeholder</a>').length, 1);
	assert.equal(rawTagFindings('<a href="/agentic-sdd/reference/cli/">CLI</a>').length, 0);
	assert.equal(rawTagFindings('<kbd>Esc</kbd>').length, 0);
	assert.equal(rawTagFindings('mcp.<name> entries').length, 0);
	assert.equal(rawTagFindings('<name onclick="x()">').length, 1);
	assert.equal(POINTER_HANDLER.test('<button onmouseover="x()">x</button>'), true);
	assert.equal(POINTER_HANDLER.test('<button onkeydown="x()">x</button>'), false);

	assert.deepEqual(problems, []);
});

test('every page remains readable as plain Markdown without the site', () => {
	const problems = [];
	for (const file of docFiles) {
		const source = readFileSync(file, 'utf8');
		const frontmatter = source.match(/^---\r?\n[\s\S]*?\r?\n---(?:\r?\n|$)/);
		if (!frontmatter) {
			problems.push(`${repoRelative(file)}: missing frontmatter`);
			continue;
		}
		const body = source.slice(frontmatter[0].length);
		const text = visibleMarkdown(body).replace(/\s+/g, ' ').trim();
		if (text.length < 40) problems.push(`${repoRelative(file)}: body has no readable text`);
		if (!fenceIsBalanced(source)) problems.push(`${repoRelative(file)}: unclosed code fence`);
		const comments = [...source.matchAll(/<!--/g)].length;
		const closes = [...source.matchAll(/-->/g)].length;
		if (comments !== closes) problems.push(`${repoRelative(file)}: unbalanced HTML comment`);
		if (/<(?:script|style|iframe|object|embed|canvas|video|audio|noscript)\b/i.test(source)) {
			problems.push(`${repoRelative(file)}: embeds a site-only element`);
		}
	}
	assert.deepEqual(problems, []);
});

test('relative Markdown and image targets resolve for a reader without the site', () => {
	const problems = [];
	for (const file of docFiles) {
		const source = withoutFencedCode(readFileSync(file, 'utf8'));
		const targets = [];
		for (const match of source.matchAll(/!?\[[^\]]*\]\(([^)\s]+)(?:\s+["'][^"']*["'])?\)/g)) {
			targets.push(match[1]);
		}
		for (const match of source.matchAll(/^\[[^\]]+\]:\s*(\S+)/gm)) targets.push(match[1]);
		for (const target of targets) {
			if (/^[a-z][a-z0-9+.-]*:/i.test(target)) continue; // external or mailto
			if (target.startsWith('#') || target.startsWith('/')) continue; // page-local or site route
			const relative = target.split('#')[0].split('?')[0];
			if (!relative) continue;
			const resolved = path.resolve(path.dirname(file), relative);
			if (!existsSync(resolved)) {
				problems.push(`${repoRelative(file)}: unresolved relative target ${target}`);
			}
		}
	}
	assert.deepEqual(problems, []);
});

test('every manifest page participates in the plain-Markdown contract', () => {
	for (const page of manifest.pages ?? []) {
		assert.ok(docSources.has(page.source), `manifest page has no content file: ${page.source}`);
		const file = path.resolve(repoRoot, page.source);
		assert.ok(existsSync(file), `missing page source ${page.source}`);
		assert.ok(fenceIsBalanced(readFileSync(file, 'utf8')), `${page.id}: unclosed code fence`);
	}
});

test('keyboard-only traversal and focus order require a real browser', { skip: 'requires a real browser; not_run' }, () => {
	assert.fail('not_run');
});

test('search open/close/Esc, TOC and code-copy keyboard behavior require a real browser', { skip: 'requires a real browser; not_run' }, () => {
	assert.fail('not_run');
});

test('320/1280 CSS px and 100%/200% zoom overflow observation requires a real browser', { skip: 'requires a real browser; not_run' }, () => {
	assert.fail('not_run');
});

test('screen-reader spot check requires assistive technology', { skip: 'requires a real screen reader; not_run' }, () => {
	assert.fail('not_run');
});

test('colour contrast and dark/plain theme legibility require human inspection', { skip: 'requires human visual inspection; not_run' }, () => {
	assert.fail('not_run');
});
