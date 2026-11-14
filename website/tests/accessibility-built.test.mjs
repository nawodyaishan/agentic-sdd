// Spec 020 T12 / B5-W1-F — Q1 built-artifact accessibility checks.
// Real artifact checks run only when DOCS_BUILD points at a completed static
// build (for example DOCS_BUILD=website/dist). Without that directory the
// observations are explicitly skipped as not_run, never inferred from source.
// No browser is launched here.

import assert from 'node:assert/strict';
import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const websiteDir = path.resolve(testDir, '..');
const repoRoot = path.resolve(websiteDir, '..');
const buildDir = process.env.DOCS_BUILD
	? path.resolve(repoRoot, process.env.DOCS_BUILD)
	: null;
const buildSkip = buildDir
	? false
	: 'DOCS_BUILD not supplied; built-artifact accessibility observation is not_run';

function walk(dir, predicate = () => true) {
	const out = [];
	for (const entry of readdirSync(dir, { withFileTypes: true })) {
		const full = path.join(dir, entry.name);
		const stat = statSync(full);
		assert.equal(stat.isSymbolicLink(), false, `unexpected symlink in build output: ${full}`);
		if (stat.isDirectory()) out.push(...walk(full, predicate));
		else if (predicate(full)) out.push(full);
	}
	return out;
}

function viewportTags(html) {
	return [...html.matchAll(/<meta\b[^>]*>/gi)]
		.map((match) => match[0])
		.filter((tag) => /\bname\s*=\s*["']viewport["']/i.test(tag));
}

function viewportProblems(content) {
	const tokens = new Map();
	for (const token of content.split(',')) {
		const [rawName, ...rest] = token.split('=');
		if (!rawName || rest.length === 0) continue;
		tokens.set(rawName.trim().toLowerCase(), rest.join('=').trim().toLowerCase());
	}
	const problems = [];
	if (!tokens.has('width')) {
		problems.push('missing width');
	} else if (tokens.get('width') !== 'device-width') {
		problems.push(`fixed width ${tokens.get('width')}`);
	}
	const scalable = tokens.get('user-scalable');
	if (scalable === 'no' || scalable === '0') problems.push('user-scalable=no');
	if (tokens.has('maximum-scale')) {
		const maximum = Number.parseFloat(tokens.get('maximum-scale'));
		if (!Number.isFinite(maximum) || maximum < 2) {
			problems.push(`maximum-scale=${tokens.get('maximum-scale')} blocks 200% zoom`);
		}
	}
	return problems;
}

function stripCssComments(css) {
	return css.replace(/\/\*[\s\S]*?\*\//g, '');
}

// Parse top-level (non-@media/@container) CSS rules and return absolute
// min-width declarations wider than a 320 CSS-pixel viewport. Conditional
// widths inside media queries cannot force horizontal scroll at 320.
function minWidthFindings(css) {
	const clean = stripCssComments(css);
	const stack = [];
	const findings = [];
	let buffer = '';
	let index = 0;

	function openBlock(head) {
		const parent = stack[stack.length - 1];
		const name = head.trim();
		stack.push({
			head: name,
			conditional:
				Boolean(parent?.conditional) || /^@(?:media|container)\b/i.test(name),
			skipped:
				Boolean(parent?.skipped) || /^@(?:keyframes|-webkit-keyframes|font-face)\b/i.test(name),
		});
	}

	function recordDeclaration(declaration, block) {
		const selector = block && !block.head.startsWith('@') ? block.head : null;
		if (!block || !selector || block.conditional || block.skipped) return;
		const match = declaration.match(
			/^(min-width|min-inline-size)\s*:\s*([0-9.]+)(px|rem|em)\b/i,
		);
		if (!match) return;
		const px = Number.parseFloat(match[2]) * (match[3].toLowerCase() === 'px' ? 1 : 16);
		if (px > 320) findings.push({ selector, declaration, px });
	}

	while (index < clean.length) {
		const char = clean[index];
		if (char === '{') {
			openBlock(buffer);
			buffer = '';
			index += 1;
			continue;
		}
		if (char === '}') {
			recordDeclaration(buffer.trim(), stack[stack.length - 1]);
			stack.pop();
			buffer = '';
			index += 1;
			continue;
		}
		if (char === ';') {
			recordDeclaration(buffer.trim(), stack[stack.length - 1]);
			buffer = '';
			index += 1;
			continue;
		}
		buffer += char;
		index += 1;
	}
	return findings;
}

function inlineMinWidthProblems(html) {
	const problems = [];
	for (const match of html.matchAll(/<([a-z][a-z0-9-]*)\b[^>]*\bstyle\s*=\s*["']([^"']*)["'][^>]*>/gi)) {
		for (const width of match[2].matchAll(/(?:min-width|min-inline-size)\s*:\s*([0-9.]+)(px|rem|em)\b/gi)) {
			const px = Number.parseFloat(width[1]) * (width[2].toLowerCase() === 'px' ? 1 : 16);
			if (px > 320) problems.push(`<${match[1].toLowerCase()}> inline min-width ${width[0]}`);
		}
	}
	return problems;
}

function pageProblems(html, { requireMainText = true } = {}) {
	const problems = [];

	const htmlTag = html.match(/<html\b([^>]*)>/i);
	if (!htmlTag) {
		problems.push('missing <html> element');
	} else {
		const lang = htmlTag[1].match(/\blang\s*=\s*["']([^"']*)["']/i);
		if (!lang || !lang[1].trim()) problems.push('missing non-empty <html lang>');
		else if (!/^[A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*$/.test(lang[1])) {
			problems.push(`invalid language tag ${lang[1]}`);
		}
	}

	const viewports = viewportTags(html);
	if (viewports.length === 0) problems.push('missing viewport meta');
	for (const tag of viewports) {
		const content = tag.match(/\bcontent\s*=\s*["']([^"']*)["']/i);
		if (!content) problems.push('viewport meta without content');
		else for (const issue of viewportProblems(content[1])) problems.push(`viewport: ${issue}`);
	}

	const skip = html.match(/<a\b[^>]*class\s*=\s*["'][^"']*\bsl-skip-link\b[^"']*["'][^>]*>/i);
	if (!skip) {
		problems.push('missing theme skip link');
	} else {
		const href = skip[0].match(/\bhref\s*=\s*["']#([\w-]+)["']/i);
		if (!href) problems.push('skip link has no in-page target');
		const main = html.search(/<main\b/i);
		if (main !== -1 && html.indexOf(skip[0]) > main) problems.push('skip link appears after <main>');
	}

	const main = html.match(/<main\b[^>]*>([\s\S]*?)<\/main>/i);
	if (!main) {
		problems.push('missing <main> content region');
	} else {
		const text = main[1]
			.replace(/<script\b[\s\S]*?<\/script>/gi, ' ')
			.replace(/<[^>]+>/g, ' ')
			.replace(/&[a-z]+;/gi, ' ')
			.replace(/\s+/g, ' ')
			.trim();
		if (requireMainText && text.length < 100) {
			problems.push('main content is not server-rendered as readable text');
		}	}

	for (const match of html.matchAll(/<img\b[^>]*>/gi)) {
		const alt = match[0].match(/\balt\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))/i);
		const value = alt ? alt[1] ?? alt[2] ?? alt[3] ?? '' : '';
		if (!value.trim()) problems.push('<img> without non-empty alt');
	}

	for (const match of html.matchAll(/<table\b[\s\S]*?<\/table>/gi)) {
		if (!/<thead\b/i.test(match[0])) problems.push('<table> without <thead> header row');
		if (!/<th\b/i.test(match[0])) problems.push('<table> without header cells');
	}

	for (const match of html.matchAll(/<a\b[^>]*>/gi)) {
		if (!/\bhref\s*=\s*["'][^"']+["']/i.test(match[0])) {
			problems.push('<a> without a non-empty href');
		}
	}

	if (/\btabindex\s*=\s*["']?\s*[1-9]\d*/i.test(html)) problems.push('positive tabindex breaks focus order');
	if (/\son(?:click|mousedown|mouseup|mouseover|pointerdown|touchstart)\s*=/i.test(html)) {
		problems.push('inline pointer-only event handler');
	}

	return problems;
}

function focusCssProblems(css) {
	const problems = [];
	const clean = stripCssComments(css);
	for (const match of clean.matchAll(/([^{}]*:focus[^{}]*)\{([^}]*)\}/gi)) {
		if (!/outline\s*:\s*(?:none|0)\b/.test(match[2])) continue;
		if (/box-shadow\s*:|border(?:-[a-z]+)?\s*:|background(?:-color)?\s*:/.test(match[2])) continue;
		problems.push(`focus outline removed without replacement: ${match[1].trim()}`);
	}
	return problems;
}

test('viewport fixture checks reject fixed width, blocked zoom and unreadable scale caps', () => {
	assert.deepEqual(viewportProblems('width=device-width, initial-scale=1'), []);
	assert.deepEqual(viewportProblems('width=device-width, initial-scale=1, maximum-scale=2'), []);
	assert.match(viewportProblems('width=1024').join('; '), /fixed width/);
	assert.match(viewportProblems('width=device-width, user-scalable=no').join('; '), /user-scalable/);
	assert.match(viewportProblems('width=device-width, maximum-scale=1').join('; '), /200% zoom/);
	assert.match(viewportProblems('initial-scale=1').join('; '), /missing width/);
});

test('minimum-width fixture checks ignore breakpoints and flag absolute page widths', () => {
	assert.equal(minWidthFindings('html { min-width: 320px; }').length, 0);
	assert.equal(minWidthFindings('@media (min-width: 72rem) { .x { color: red; } }').length, 0);
	assert.equal(minWidthFindings('.tree { min-width: 12rem; }').length, 0);
	const fixed = minWidthFindings('body { min-width: 1024px; }');
	assert.equal(fixed.length, 1);
	assert.equal(fixed[0].selector, 'body');
	assert.equal(minWidthFindings('body{min-width:1024px}').length, 1);
	assert.equal(minWidthFindings('@supports (display: grid) { .page { min-inline-size: 40rem; } }')[0].px, 640);
});

test('every built page has a non-empty lang and a zoom-friendly viewport', { skip: buildSkip }, () => {
	assert.ok(buildDir && existsSync(buildDir), `DOCS_BUILD does not exist: ${buildDir}`);
	const pages = walk(buildDir, (file) => file.endsWith('.html'));
	assert.ok(pages.length > 0, 'build contains no HTML pages');
	for (const page of pages) {
		const html = readFileSync(page, 'utf8');
		const issues = pageProblems(html);
		assert.deepEqual(
			issues.filter((issue) => /lang|viewport|<html>/.test(issue)),
			[],
			`${path.relative(buildDir, page)}`,
		);
	}
});

test('every built page keeps the theme skip link before server-rendered main content', { skip: buildSkip }, () => {
	assert.ok(buildDir && existsSync(buildDir), `DOCS_BUILD does not exist: ${buildDir}`);
	const pages = walk(buildDir, (file) => file.endsWith('.html'));
	assert.ok(pages.length > 0, 'build contains no HTML pages');
	for (const page of pages) {
		const html = readFileSync(page, 'utf8');
		const issues = pageProblems(html, { requireMainText: !/\/404\.html$/.test(page) }).filter((issue) =>
			/skip link|<main>|server-rendered/.test(issue),
		);
		assert.deepEqual(issues, [], `${path.relative(buildDir, page)}`);
	}
});

test('every built page has image alternatives, table headers and native keyboard controls', { skip: buildSkip }, () => {
	assert.ok(buildDir && existsSync(buildDir), `DOCS_BUILD does not exist: ${buildDir}`);
	const pages = walk(buildDir, (file) => file.endsWith('.html'));
	assert.ok(pages.length > 0, 'build contains no HTML pages');
	for (const page of pages) {
		const html = readFileSync(page, 'utf8');
		const issues = pageProblems(html).filter((issue) =>
			/<img>|<table>|<a>|tabindex|pointer-only/.test(issue),
		);
		assert.deepEqual(issues, [], `${path.relative(buildDir, page)}`);
	}
});

test('built CSS has no unconditional minimum width beyond 320 CSS pixels', { skip: buildSkip }, () => {
	assert.ok(buildDir && existsSync(buildDir), `DOCS_BUILD does not exist: ${buildDir}`);
	const cssFiles = walk(buildDir, (file) => file.endsWith('.css'));
	assert.ok(cssFiles.length > 0, 'build contains no CSS assets');
	for (const file of cssFiles) {
		const findings = minWidthFindings(readFileSync(file, 'utf8'));
		assert.deepEqual(
			findings.map((finding) => `${finding.selector} -> ${finding.declaration}`),
			[],
			path.relative(buildDir, file),
		);
	}
	for (const page of walk(buildDir, (file) => file.endsWith('.html'))) {
		const html = readFileSync(page, 'utf8');
		assert.deepEqual(inlineMinWidthProblems(html), [], path.relative(buildDir, page));
	}
});

test('built CSS keeps a visible focus indicator', { skip: buildSkip }, () => {
	assert.ok(buildDir && existsSync(buildDir), `DOCS_BUILD does not exist: ${buildDir}`);
	const styles = walk(buildDir, (file) => file.endsWith('.css')).map((file) => readFileSync(file, 'utf8'));
	for (const page of walk(buildDir, (file) => file.endsWith('.html'))) {
		const html = readFileSync(page, 'utf8');
		for (const match of html.matchAll(/<style\b[^>]*>([\s\S]*?)<\/style>/gi)) styles.push(match[1]);
	}
	const css = styles.join('\n');
	assert.match(css, /:focus(?:-visible|-within)?\b/, 'built CSS has no focus styles');
	assert.deepEqual(focusCssProblems(css), []);
});
