// Spec 020 T12 / B5-W1-F — Q1 accessible authoring source checks (validation code A).
// These are deterministic source observations only. They do not certify WCAG
// conformance and do not replace a browser, screen reader or human review.

import assert from 'node:assert/strict';
import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const websiteDir = path.resolve(testDir, '..');
const repoRoot = path.resolve(websiteDir, '..');
const docsDir = path.resolve(websiteDir, 'src/content/docs');
const manifestPath = path.resolve(websiteDir, 'docs-manifest.json');
const starlightDir = path.resolve(websiteDir, 'node_modules/@astrojs/starlight');
const astroConfigPath = path.resolve(websiteDir, 'astro.config.mjs');

const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));

function listFiles(dir, predicate) {
	const out = [];
	for (const entry of readdirSync(dir, { withFileTypes: true })) {
		const full = path.join(dir, entry.name);
		const stat = statSync(full);
		assert.equal(stat.isSymbolicLink(), false, `unexpected symlink: ${full}`);
		if (stat.isDirectory()) out.push(...listFiles(full, predicate));
		else if (!predicate || predicate(full)) out.push(full);
	}
	return out;
}

function repoRelative(file) {
	return path.relative(repoRoot, file).split(path.sep).join('/');
}

function readDoc(file) {
	return readFileSync(file, 'utf8');
}

function splitFrontmatter(source) {
	const match = source.match(/^---\r?\n([\s\S]*?)\r?\n---(?:\r?\n|$)/);
	if (!match) return null;
	const data = {};
	for (const line of match[1].split(/\r?\n/)) {
		const entry = line.match(/^([A-Za-z0-9_-]+):\s*(.*)$/);
		if (!entry) continue;
		data[entry[1]] = entry[2].trim().replace(/^["']|["']$/g, '');
	}
	return { data, body: source.slice(match[0].length) };
}

// Remove fenced code before looking for authored markup. A fence inside a fenced
// block is a literal example, not a heading, table or pointer instruction.
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

function stripInlineCode(text) {
	return text
		.split(/\r?\n/)
		.map((line) => line.replace(/(`+)([^`]*?)\1/g, ''))
		.join('\n');
}

function stripComments(text) {
	return text.replace(/<!--[\s\S]*?-->/g, '');
}

function prose(source) {
	return stripInlineCode(stripComments(withoutFencedCode(source)));
}

function tableCells(line) {
	let value = line.trim();
	if (value.startsWith('|')) value = value.slice(1);
	if (value.endsWith('|')) value = value.slice(0, -1);
	return value.split('|').map((cell) => cell.trim());
}

function isTableDelimiter(line) {
	if (!line.includes('|')) return false;
	if (!/^\s*\|?[\s:|-]+\|?\s*$/.test(line)) return false;
	return tableCells(line).every((cell) => /^:?-{3,}:?$/.test(cell));
}

const COLOR_ONLY_PATTERNS = [
	/\b(?:click|tap|press|select|choose|use)\s+(?:the\s+)?(?:red|green|blue|yellow|orange|purple|pink)\b/i,
	/\b(?:red|green|blue|yellow|orange|purple|pink)\s+(?:button|link|text|label|icon|row|rows|cell|badge|highlight|indicator|border)\b/i,
	/\b(?:highlighted|marked|shown|displayed)\s+in\s+(?:red|green|blue|yellow|orange|purple|pink)\b/i,
	/\bcolou?r(?:ed)?[- ](?:only|coded)\b/i,
];

function colorOnlyFindings(text) {
	const findings = [];
	for (const [index, line] of text.split(/\r?\n/).entries()) {
		// A sentence that explicitly says NOT to rely on colour is the opposite
		// of a colour-only instruction.
		if (/\b(?:not|never|without|cannot|can't|does not|do not|no)\b[^.\n]{0,40}\bcolou?r/i.test(line)) continue;
		for (const pattern of COLOR_ONLY_PATTERNS) {
			const match = line.match(pattern);
			if (match) findings.push({ line: index + 1, text: match[0] });
		}
	}
	return findings;
}

const contentDocs = listFiles(docsDir, (file) => /\.mdx?$/.test(file)).filter(
	(file) => !/^404\.mdx?$/.test(path.basename(file)),
);
const manifestSources = new Map(
	(manifest.pages ?? []).map((page) => [page.source, path.resolve(repoRoot, page.source)]),
);

test('every manifest page has a source file with title and description frontmatter', () => {
	assert.ok(manifest.pages?.length > 0, 'manifest has no pages');
	for (const page of manifest.pages) {
		const file = path.resolve(repoRoot, page.source);
		assert.ok(existsSync(file), `${page.id}: missing source ${page.source}`);
		const parsed = splitFrontmatter(readDoc(file));
		assert.ok(parsed, `${page.id}: missing YAML frontmatter`);
		assert.ok(parsed.data.title?.trim(), `${page.id}: frontmatter title is empty`);
		assert.ok(parsed.data.description?.trim(), `${page.id}: frontmatter description is empty`);
		assert.doesNotMatch(
			parsed.data.title,
			/^(?:untitled|todo|tbd)$/i,
			`${page.id}: placeholder title`,
		);
		assert.doesNotMatch(
			parsed.data.description,
			/^(?:todo|tbd)$/i,
			`${page.id}: placeholder description`,
		);
	}
});

test('content docs and manifest page sources stay in one-to-one agreement', () => {
	const contentRelative = new Set(contentDocs.map(repoRelative));
	for (const source of manifestSources.keys()) {
		assert.ok(contentRelative.has(source), `manifest source has no content file: ${source}`);
	}
	for (const source of contentRelative) {
		assert.ok(manifestSources.has(source), `content file is missing from docs-manifest.json: ${source}`);
	}
});

test('markdown and HTML images declare non-empty alt text', () => {
	const problems = [];
	for (const file of contentDocs) {
		const source = prose(readDoc(file));
		for (const match of source.matchAll(/!\[([^\]]*)\](?:\([^)]*\)|\[[^\]]*\])/g)) {
			if (!match[1].trim()) problems.push(`${repoRelative(file)}: markdown image without alt text`);
		}
		for (const match of source.matchAll(/<img\b[^>]*>/gi)) {
			const alt = match[0].match(/\balt\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))/i);
			const value = alt ? alt[1] ?? alt[2] ?? alt[3] ?? '' : '';
			if (!value.trim()) problems.push(`${repoRelative(file)}: <img> without non-empty alt`);
		}
	}
	assert.deepEqual(problems, []);
});

test('every markdown table starts with a non-empty header row and matching delimiter', () => {
	const problems = [];
	for (const file of contentDocs) {
		const lines = prose(readDoc(file)).split(/\r?\n/);
		for (let index = 0; index < lines.length; index += 1) {
			if (!isTableDelimiter(lines[index])) continue;
			const header = lines[index - 1];
			if (header === undefined || !header.includes('|')) {
				problems.push(`${repoRelative(file)}:${index + 1}: table delimiter without a header row`);
				continue;
			}
			const headerCells = tableCells(header);
			const delimiterCells = tableCells(lines[index]);
			if (headerCells.length !== delimiterCells.length) {
				problems.push(`${repoRelative(file)}:${index + 1}: header/delimiter column count differs`);
			}
			if (headerCells.some((cell) => cell.length === 0)) {
				problems.push(`${repoRelative(file)}:${index}: table header has an empty cell`);
			}
		}
	}
	assert.deepEqual(problems, []);
});

test('authored headings do not skip levels', () => {
	const problems = [];
	for (const file of contentDocs) {
		const lines = prose(readDoc(file)).split(/\r?\n/);
		let previous = 1; // frontmatter title renders as the page H1
		for (const [index, line] of lines.entries()) {
			const heading = line.match(/^(#{1,6})\s*(.*)$/);
			if (!heading) continue;
			if (!heading[2].trim()) {
				problems.push(`${repoRelative(file)}:${index + 1}: empty heading`);
				continue;
			}
			const level = heading[1].length;
			if (level > previous + 1) {
				problems.push(
					`${repoRelative(file)}:${index + 1}: heading level jumps H${previous} -> H${level}`,
				);
			}
			previous = level;
		}
	}
	assert.deepEqual(problems, []);
});

test('authored instructions do not rely on colour alone', () => {
	const problems = [];
	for (const file of contentDocs) {
		for (const finding of colorOnlyFindings(prose(readDoc(file)))) {
			problems.push(`${repoRelative(file)}:${finding.line}: colour-only wording "${finding.text}"`);
		}
	}
	assert.deepEqual(problems, []);

	// The detector itself is fixture-verified so a green run is meaningful.
	assert.ok(colorOnlyFindings('Click the red button to continue.').length >= 1);
	assert.ok(colorOnlyFindings('Green rows mean success.').length >= 1);
	assert.equal(colorOnlyFindings('Do not rely on colour alone.').length, 0);
	assert.equal(colorOnlyFindings('The table shows colour values for the theme.').length, 0);
});

function themeFile(relative) {
	const file = path.resolve(starlightDir, relative);
	assert.ok(existsSync(file), `missing theme file: ${relative}`);
	return readFileSync(file, 'utf8');
}

test('the configured theme supplies a non-empty document language', () => {
	const astroConfig = readFileSync(astroConfigPath, 'utf8');
	assert.match(astroConfig, /\bstarlight\s*\(/, 'Starlight integration is not configured');
	assert.doesNotMatch(astroConfig, /\bdefaultLocale\s*:\s*['"]\s*['"]/, 'empty defaultLocale');
	assert.doesNotMatch(astroConfig, /\blang\s*:\s*['"]\s*['"]/, 'empty locale lang');

	// The project relies on Starlight's built-in default locale. Verify the
	// installed theme actually defines a non-empty language instead of assuming it.
	const i18n = themeFile('dist/utils/i18n.js');
	assert.match(i18n, /\blang:\s*["']en["']/, 'theme default language is missing or empty');
});

test('the theme supplies a skip link and focus indicator styles', () => {
	const astroConfig = readFileSync(astroConfigPath, 'utf8');
	const skipLink = themeFile('dist/components/SkipLink.astro');
	assert.match(skipLink, /<a\b/, 'theme skip link is not an anchor');
	assert.match(skipLink, /href=/, 'theme skip link has no target');
	assert.match(skipLink, /:focus\b/, 'theme skip link has no visible focus style');

	// A project override must keep an anchor with a target and a focus style.
	const overrides = astroConfig.match(/\bcomponents\s*:\s*\{([\s\S]*?)\}/);
	if (overrides) {
		const skipOverride = overrides[1].match(/\bSkipLink\s*:\s*['"]([^'"]+)['"]/);
		if (skipOverride) {
			const custom = path.resolve(websiteDir, skipOverride[1]);
			assert.ok(existsSync(custom), `custom SkipLink does not exist: ${skipOverride[1]}`);
			const source = readFileSync(custom, 'utf8');
			assert.match(source, /<a\b/, 'custom SkipLink is not an anchor');
			assert.match(source, /href=/, 'custom SkipLink has no target');
			assert.match(source, /:focus\b/, 'custom SkipLink has no focus style');
		}
	}

	// Starlight ships focus rules; make sure the installed theme does not erase
	// outlines globally or on focus.
	const themeStyles = listFiles(path.resolve(starlightDir, 'dist/style'), (file) => file.endsWith('.css'));
	for (const file of themeStyles) {
		const css = readFileSync(file, 'utf8').replace(/\/\*[\s\S]*?\*\//g, '');
		assert.doesNotMatch(
			css,
			/(?:^|})\s*(?:html|body)\s*(?:,[^{}]*)?\{[^}]*outline\s*:\s*(?:none|0)\b/,
			`theme removes the document focus outline: ${path.basename(file)}`,
		);
	}
});

test('project styles do not remove focus outlines without a replacement', () => {
	const astroConfig = readFileSync(astroConfigPath, 'utf8');
	const customCss = [];
	const configured = astroConfig.match(/\bcustomCss\s*:\s*\[([\s\S]*?)\]/);
	if (configured) {
		for (const match of configured[1].matchAll(/['"]([^'"]+)['"]/g)) {
			customCss.push(path.resolve(websiteDir, match[1]));
		}
	}
	const srcStyles = listFiles(path.resolve(websiteDir, 'src'), (file) => file.endsWith('.css'));
	const files = [...new Set([...customCss, ...srcStyles])];
	for (const file of files) {
		assert.ok(existsSync(file), `configured stylesheet does not exist: ${repoRelative(file)}`);
		const css = readFileSync(file, 'utf8').replace(/\/\*[\s\S]*?\*\//g, '');
		const rules = [...css.matchAll(/([^{}]+)\{([^}]*)\}/g)];
		for (const [, selector, body] of rules) {
			if (!/outline\s*:\s*(?:none|0)\b/.test(body)) continue;
			const replaced = /box-shadow\s*:|border(?:-[a-z]+)?\s*:|background(?:-color)?\s*:/.test(body);
			assert.ok(
				replaced,
				`${repoRelative(file)}: focus outline removed without replacement in "${selector.trim()}"`,
			);
		}
	}
});

test('the theme viewport contract permits zoom', () => {
	const head = themeFile('dist/utils/head.js');
	const viewport = head.match(/name:\s*["']viewport["'][\s\S]{0,120}?content:\s*["']([^"']+)["']/);
	assert.ok(viewport, 'theme head does not define a viewport meta tag');
	assert.match(viewport[1], /\bwidth=device-width\b/, 'theme viewport must use device-width');
	assert.match(viewport[1], /\binitial-scale=1\b/, 'theme viewport must start at scale 1');
	assert.doesNotMatch(viewport[1], /\buser-scalable\s*=\s*no\b/i, 'theme blocks user zoom');
	assert.doesNotMatch(viewport[1], /\bmaximum-scale\s*=\s*(?:0|1)(?:\.0+)?\b/i, 'theme caps zoom');

	// A project-level viewport override must be zoom-friendly too.
	const astroConfig = readFileSync(astroConfigPath, 'utf8');
	assert.doesNotMatch(astroConfig, /user-scalable\s*=\s*no/i, 'astro config blocks user zoom');
	assert.doesNotMatch(astroConfig, /maximum-scale\s*=\s*(?:0|1)(?:\.0+)?\b/i, 'astro config caps zoom');
});
