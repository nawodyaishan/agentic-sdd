// Website attribution contract.
//
// Owner decision (2026-11-17): the personal credit appears exactly once, on the
// homepage near the introduction. Deep pages carry no site-wide footer branding,
// so the Starlight default Footer is used unchanged and no Footer override is
// registered. This file checks the authored source and configuration
// statically.
//
// It never imports Astro/Starlight, never builds the site and never marks a
// visual or browser check as performed. Rendered 320/1280-width, zoom,
// light/dark, keyboard-focus and screen-reader observations remain parent-owned
// after a real build; a passing string check here is not proof of visual
// prominence or accessibility.

import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { collectBaseline } from '../scripts/lib/manifest.mjs';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const websiteDir = path.resolve(testDir, '..');
const repoRoot = path.resolve(websiteDir, '..');
const maintainerPath = path.join(websiteDir, 'src/maintainer.mjs');
const footerPath = path.join(websiteDir, 'src/components/Footer.astro');
const configPath = path.join(websiteDir, 'astro.config.mjs');
const homepagePath = path.join(websiteDir, 'src/content/docs/index.md');
const packagePath = path.join(websiteDir, 'package.json');

const MAINTAINER_RELATIVE_PATH = 'website/src/maintainer.mjs';
const FOOTER_RELATIVE_PATH = 'website/src/components/Footer.astro';
const HOMEPAGE_RELATIVE_PATH = 'website/src/content/docs/index.md';

const EXPECTED_MAINTAINER = Object.freeze({
	name: 'Nawodya Ishan',
	github: 'https://github.com/nawodyaishan',
	website: 'https://nawodyaishan.github.io/',
});
const EXPECTED_DEPENDENCIES = ['@astrojs/markdown-remark', '@astrojs/starlight', 'astro', 'yaml'];

function readOptional(filePath) {
	return existsSync(filePath) ? readFileSync(filePath, 'utf8') : null;
}

async function loadMaintainer() {
	if (!existsSync(maintainerPath)) return { exists: false };
	try {
		const module = await import(pathToFileURL(maintainerPath).href);
		return { exists: true, module };
	} catch (error) {
		return { exists: true, error };
	}
}

function analyzeMaintainer(maintainer) {
	const findings = [];
	if (maintainer === null || typeof maintainer !== 'object') return ['missing-maintainer-module'];

	for (const field of ['name', 'github', 'website']) {
		if (maintainer[field] !== EXPECTED_MAINTAINER[field]) findings.push(`maintainer-${field}-mismatch`);
	}
	const keys = Object.keys(maintainer).sort();
	if (keys.length !== 3 || keys[0] !== 'github' || keys[1] !== 'name' || keys[2] !== 'website') {
		findings.push('unexpected-maintainer-fields');
	}
	return findings;
}

// The credit must not be reintroduced site-wide. No custom Footer component and
// no `components.Footer` registration means every deep page falls back to the
// stock Starlight footer.
function analyzeFooterOverride({ footerSource, configSource }) {
	const findings = [];
	if (footerSource !== null) findings.push('site-wide-credit-still-present');
	if (configSource === null) return [...findings, 'missing-astro-config'];

	if (/\bFooter\s*:\s*['"][^'"]*Footer\.astro['"]/.test(configSource)) {
		findings.push('footer-override-registered');
	}

	// Framework utilities stay enabled; with no override, edit links, social
	// navigation and search render from Starlight's defaults.
	if (!/editLink\s*:/.test(configSource)) findings.push('edit-link-removed');
	if (!/social\s*:\s*\[/.test(configSource)) findings.push('social-links-removed');
	if (!/pagefind\s*:\s*true/.test(configSource)) findings.push('search-removed');

	return findings;
}

function analyzeAttribution({ maintainer, footerSource, configSource }) {
	return [...analyzeMaintainer(maintainer), ...analyzeFooterOverride({ footerSource, configSource })];
}

function homepageParts() {
	const text = readFileSync(homepagePath, 'utf8');
	const structure = collectBaseline(repoRoot, [HOMEPAGE_RELATIVE_PATH]).files[0];
	return { text, structure };
}

function analyzeHomepage({ text, structure }, maintainer) {
	const findings = [];
	const firstH2 = structure.headings.find((heading) => heading.level === 2);
	const endLine = firstH2 === undefined ? Number.POSITIVE_INFINITY : firstH2.line;
	const introLinks = structure.links.filter((link) => link.line < endLine);

	const nameLink = introLinks.find(
		(link) => link.text.trim() === maintainer.name && link.href === maintainer.github,
	);
	if (nameLink === undefined) findings.push('homepage-byline-missing');

	const contributorLink = introLinks.find((link) => /(?:^|\/)contribut/i.test(link.href));
	if (contributorLink === undefined) findings.push('homepage-contributor-link-missing');

	if (introLinks.some((link) => link.href === maintainer.website)) {
		findings.push('homepage-personal-website-link');
	}

	const lines = text.replace(/^\uFEFF/, '').split(/\r?\n/);
	const introText = lines.slice(0, Math.min(lines.length, endLine - 1)).join(' ');
	if (!/Created and maintained by/i.test(introText)) findings.push('homepage-credit-wording-missing');
	if (/!\[[^\]]*\]\([^)]*\)/.test(introText) || /<img\b/i.test(introText)) {
		findings.push('homepage-image-only-credit');
	}
	if (/<script\b/i.test(text)) findings.push('homepage-script-added');

	return findings;
}

const VALID_MAINTAINER = Object.freeze({
	name: 'Nawodya Ishan',
	github: 'https://github.com/nawodyaishan',
	website: 'https://nawodyaishan.github.io/',
});

const VALID_HOMEPAGE = [
	'---',
	'title: Agentic-SDD',
	'---',
	'',
	'Intro paragraph.',
	'',
	'Created and maintained by [Nawodya Ishan](https://github.com/nawodyaishan) · [Contributing](contributing/development.md).',
	'',
	'## Who Agentic-SDD is for',
	'',
].join('\n');

const VALID_CONFIG = [
	"import { defineConfig } from 'astro/config';",
	'',
	'export default defineConfig({',
	'  integrations: [',
	'    starlight({',
	'      pagefind: true,',
	"      editLink: { baseUrl: 'https://github.com/example/repo/edit/main/website/' },",
	"      social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/example/repo' }],",
	'    }),',
	'  ],',
	'});',
	'',
].join('\n');

test('shared maintainer identity module exists and exports the frozen identity', async () => {
	const loaded = await loadMaintainer();
	assert.equal(
		loaded.exists,
		true,
		`missing ${MAINTAINER_RELATIVE_PATH}; the homepage byline must keep one shared identity owner`,
	);
	assert.equal(
		loaded.error,
		undefined,
		`cannot import ${MAINTAINER_RELATIVE_PATH}: ${loaded.error instanceof Error ? loaded.error.message : loaded.error}`,
	);
	assert.ok(loaded.module.maintainer !== undefined, 'maintainer.mjs must export the named `maintainer` object');
	assert.deepEqual({ ...loaded.module.maintainer }, EXPECTED_MAINTAINER);
	assert.equal(Object.isFrozen(loaded.module.maintainer), true, 'maintainer identity must be Object.freeze(...)');
	assert.deepEqual(Object.keys(loaded.module.maintainer).sort(), ['github', 'name', 'website']);
});

test('homepage shows the only visible credit as an ordinary Markdown byline near the introduction', async () => {
	const loaded = await loadMaintainer();
	assert.equal(loaded.exists, true, `missing ${MAINTAINER_RELATIVE_PATH}; the homepage byline cannot be parity-checked`);
	assert.equal(
		loaded.error,
		undefined,
		`cannot import ${MAINTAINER_RELATIVE_PATH}: ${loaded.error instanceof Error ? loaded.error.message : loaded.error}`,
	);
	const homepage = homepageParts();
	const findings = analyzeHomepage(homepage, loaded.module.maintainer);
	assert.deepEqual(
		findings,
		[],
		`homepage must show one visible Markdown byline before the first H2 with the linked name and contributor route; got ${findings.join(', ')}`,
	);
});

test('no site-wide footer credit is rendered on deep pages', () => {
	const findings = analyzeFooterOverride({
		footerSource: readOptional(footerPath),
		configSource: readOptional(configPath),
	});
	assert.deepEqual(
		findings,
		[],
		`${FOOTER_RELATIVE_PATH} must stay absent and astro.config must not register a Footer override; got ${findings.join(', ')}`,
	);
});

test('attribution adds no new runtime package, tracking, avatar or remote asset dependency', () => {
	const pkg = JSON.parse(readFileSync(packagePath, 'utf8'));
	assert.deepEqual(
		Object.keys(pkg.dependencies).sort(),
		EXPECTED_DEPENDENCIES,
		'website dependencies must stay at the pinned build set; attribution adds no runtime package',
	);

	const homepage = readFileSync(homepagePath, 'utf8');
	assert.doesNotMatch(homepage, /<script\b|!\[[^\]]*\]\([^)]*\)|<img\b/i);
});

test('attribution mutation fixture satisfies the static contract', () => {
	const findings = [
		...analyzeAttribution({
			maintainer: VALID_MAINTAINER,
			footerSource: null,
			configSource: VALID_CONFIG,
		}),
		...analyzeHomepage(
			{
				text: VALID_HOMEPAGE,
				structure: {
					headings: [{ level: 2, line: 9 }],
					links: [
						{ text: 'Nawodya Ishan', href: VALID_MAINTAINER.github, line: 7 },
						{ text: 'Contributing', href: 'contributing/development.md', line: 7 },
					],
				},
			},
			VALID_MAINTAINER,
		),
	];
	assert.deepEqual(findings, [], `valid attribution fixture must pass; got ${findings.join(', ')}`);
});

test('mutation: a reintroduced footer credit fails', () => {
	const findings = analyzeFooterOverride({
		footerSource: '<section>Created and maintained by someone</section>',
		configSource: VALID_CONFIG.replace(
			'pagefind: true,',
			"pagefind: true,\n      components: { Footer: './src/components/Footer.astro' },",
		),
	});
	assert.ok(
		findings.includes('site-wide-credit-still-present') && findings.includes('footer-override-registered'),
		`expected site-wide credit failure; got ${findings.join(', ')}`,
	);
});

test('mutation: a homepage without the byline fails', () => {
	const homepage = VALID_HOMEPAGE.replace(/^Created and maintained by.*$/m, '');
	const findings = analyzeHomepage(
		{
			text: homepage,
			structure: { headings: [{ level: 2, line: 7 }], links: [] },
		},
		VALID_MAINTAINER,
	);
	assert.ok(findings.includes('homepage-byline-missing'), `expected missing byline failure; got ${findings.join(', ')}`);
});

test('mutation: a personal website link in the byline fails', () => {
	const findings = analyzeHomepage(
		{
			text: VALID_HOMEPAGE,
			structure: {
				headings: [{ level: 2, line: 9 }],
				links: [
					{ text: 'Nawodya Ishan', href: VALID_MAINTAINER.github, line: 7 },
					{ text: 'Personal website', href: VALID_MAINTAINER.website, line: 7 },
					{ text: 'Contributing', href: 'contributing/development.md', line: 7 },
				],
			},
		},
		VALID_MAINTAINER,
	);
	assert.ok(
		findings.includes('homepage-personal-website-link'),
		`expected personal-website rejection; got ${findings.join(', ')}`,
	);
});

test('mutation: a wrong maintainer identity fails', () => {
	const findings = analyzeAttribution({
		maintainer: { ...VALID_MAINTAINER, name: 'Someone Else' },
		footerSource: null,
		configSource: VALID_CONFIG,
	});
	assert.ok(
		findings.includes('maintainer-name-mismatch'),
		`expected wrong-identity failure; got ${findings.join(', ')}`,
	);
});

test('mutation: a wrong profile URL fails', () => {
	const findings = analyzeAttribution({
		maintainer: { ...VALID_MAINTAINER, github: 'https://github.com/example' },
		footerSource: null,
		configSource: VALID_CONFIG,
	});
	assert.ok(
		findings.includes('maintainer-github-mismatch'),
		`expected wrong-URL failure; got ${findings.join(', ')}`,
	);
});

test('mutation: removing the default footer utilities fails', () => {
	const findings = analyzeFooterOverride({
		footerSource: null,
		configSource: VALID_CONFIG.replace(/editLink\s*:.*\n/, ''),
	});
	assert.ok(findings.includes('edit-link-removed'), `expected missing-utilities failure; got ${findings.join(', ')}`);
});
