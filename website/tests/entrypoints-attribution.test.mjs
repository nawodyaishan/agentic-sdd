// B7/T16 website attribution contract.
//
// Frozen C7 seam: one shared `website/src/maintainer.mjs` identity owner, a
// homepage Markdown byline near the introduction, and one footer credit that
// composes the pinned Starlight default Footer instead of replacing its
// utilities. This file checks the authored source and configuration statically.
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

function analyzeFooterSource(source) {
	const findings = [];
	if (source === null) return ['missing-footer-component'];
	const compact = source.replace(/\s+/g, ' ');

	const defaultImport =
		/import\s+([A-Za-z_$][\w$]*)\s+from\s+['"]@astrojs\/starlight\/components\/Footer\.astro['"]/.exec(source);
	if (defaultImport === null) findings.push('missing-default-footer-import');
	else {
		const componentName = defaultImport[1];
		const renders = [...source.matchAll(new RegExp(`<${componentName}\\b`, 'g'))];
		if (renders.length === 0) findings.push('default-footer-not-rendered');
		if (renders.length > 1) findings.push('default-footer-rendered-multiple-times');
	}

	// The default Footer already owns the footer landmark and utilities. A raw
	// <footer> here would replace or nest that landmark instead of composing it.
	if (/<footer\b/i.test(source)) findings.push('nested-footer-element');

	const maintainerImport =
		/import\s+\{[^}]*\bmaintainer\b[^}]*\}\s+from\s+['"](?:\.\.?\/)+maintainer\.mjs['"]/.test(source);
	if (!maintainerImport) findings.push('missing-maintainer-import');

	if (!/Created and maintained by/i.test(compact)) findings.push('credit-wording-missing');

	const anchors = [...source.matchAll(/<a\b([^>]*)>([\s\S]*?)<\/a>/g)].map((match) => ({
		attrs: match[1],
		body: match[2],
	}));
	if (
		!anchors.some(
			(anchor) => /\bhref=\{maintainer\.github\}/.test(anchor.attrs) && /\{maintainer\.name\}/.test(anchor.body),
		)
	) {
		findings.push('credit-name-not-linked');
	}
	if (
		!anchors.some(
			(anchor) =>
				/\bhref=\{maintainer\.website\}/.test(anchor.attrs) && /personal website/i.test(anchor.body),
		)
	) {
		findings.push('personal-website-link-missing');
	}

	if (
		!/<(?:section|aside)\b/i.test(source) &&
		!/<(?:section|aside|div)\b[^>]*(?:aria-label|aria-labelledby|role)\s*=/i.test(source)
	) {
		findings.push('credit-region-not-semantic');
	}

	if (/<script\b|<img\b|<iframe\b/i.test(source)) findings.push('remote-or-script-asset-added');
	if (/\bsrc\s*=\s*['"]https?:/i.test(source)) findings.push('remote-or-script-asset-added');
	if (
		/fonts\.googleapis|fonts\.gstatic|@import\s+url\(\s*['"]?https?:|url\(\s*['"]?https?:/i.test(source)
	) {
		findings.push('remote-or-script-asset-added');
	}
	if (/(?:google-analytics|googletagmanager|plausible\.io|matomo|segment\.io|posthog)/i.test(source)) {
		findings.push('tracking-added');
	}

	return findings;
}

function analyzeConfigSource(source) {
	const findings = [];
	if (source === null) return ['missing-astro-config'];

	const components = /components\s*:\s*\{([\s\S]*?)\}/.exec(source);
	if (components === null) findings.push('missing-components-registration');
	else {
		const block = components[1];
		const footer = /Footer\s*:\s*['"]([^'"]+)['"]/.exec(block);
		if (footer === null) findings.push('missing-footer-registration');
		else {
			const registered = footer[1].replace(/^\.\//, '');
			if (registered !== 'src/components/Footer.astro') findings.push('footer-registration-wrong-path');
		}
		const otherKeys = [...block.matchAll(/([A-Za-z][A-Za-z0-9]*)\s*:/g)]
			.map((match) => match[1])
			.filter((key) => key !== 'Footer');
		if (otherKeys.length > 0) findings.push(`unexpected-component-override:${otherKeys.join(',')}`);
	}

	// Framework utilities stay enabled; the override adds credit instead of
	// replacing edit links, social navigation or search.
	if (!/editLink\s*:/.test(source)) findings.push('edit-link-removed');
	if (!/social\s*:\s*\[/.test(source)) findings.push('social-links-removed');
	if (!/pagefind\s*:\s*true/.test(source)) findings.push('search-removed');

	return findings;
}

function analyzeAttribution({ maintainer, footerSource, configSource }) {
	return [
		...analyzeMaintainer(maintainer),
		...analyzeFooterSource(footerSource),
		...analyzeConfigSource(configSource),
	];
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

	const websiteLink = introLinks.find(
		(link) => /^personal website$/i.test(link.text.trim()) && link.href === maintainer.website,
	);
	if (websiteLink === undefined) findings.push('homepage-personal-website-missing');

	const contributorLink = introLinks.find((link) => /(?:^|\/)contribut/i.test(link.href));
	if (contributorLink === undefined) findings.push('homepage-contributor-link-missing');

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

const VALID_FOOTER = [
	'---',
	"import DefaultFooter from '@astrojs/starlight/components/Footer.astro';",
	"import { maintainer } from '../maintainer.mjs';",
	'---',
	'',
	'<div class="maintainer-credit" role="note">',
	'  <p>',
	'    Created and maintained by',
	'    <a href={maintainer.github}>{maintainer.name}</a>.',
	'    <a href={maintainer.website}>Personal website</a>',
	'  </p>',
	'</div>',
	'<DefaultFooter />',
	'',
].join('\n');

const VALID_CONFIG = [
	"import { defineConfig } from 'astro/config';",
	'',
	'export default defineConfig({',
	'  integrations: [',
	'    starlight({',
	'      pagefind: true,',
	'      components: {',
	"        Footer: './src/components/Footer.astro',",
	'      },',
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
		`missing ${MAINTAINER_RELATIVE_PATH}; T15 must add the single shared maintainer identity owner`,
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

test('homepage shows an ordinary Markdown byline near the introduction with the shared identity values', async () => {
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
		`homepage must show a visible Markdown byline before the first H2 with the linked name, Personal website and contributor route; got ${findings.join(', ')}`,
	);
});

test('new Footer composes the pinned Starlight default footer and renders one visible shared credit', () => {
	const footer = readOptional(footerPath);
	assert.ok(footer !== null, `missing ${FOOTER_RELATIVE_PATH}; T15 must compose the default Starlight Footer`);
	const findings = analyzeFooterSource(footer);
	assert.deepEqual(
		findings,
		[],
		`Footer.astro must preserve the default Footer utilities and add one semantic shared credit; got ${findings.join(', ')}`,
	);
});

test('astro.config registers components.Footer without replacing other Starlight utilities', () => {
	const config = readOptional(configPath);
	assert.ok(config !== null, 'missing website/astro.config.mjs');
	const findings = analyzeConfigSource(config);
	assert.deepEqual(
		findings,
		[],
		`astro config must register only the Footer component override and keep edit/search/social utilities; got ${findings.join(', ')}`,
	);
});

test('attribution adds no new runtime package, tracking, avatar or remote asset dependency', () => {
	const pkg = JSON.parse(readFileSync(packagePath, 'utf8'));
	assert.deepEqual(
		Object.keys(pkg.dependencies).sort(),
		EXPECTED_DEPENDENCIES,
		'website dependencies must stay at the pinned build set; attribution adds no runtime package',
	);

	const footer = readOptional(footerPath);
	assert.ok(footer !== null, `missing ${FOOTER_RELATIVE_PATH}; cannot check attribution assets`);
	assert.doesNotMatch(footer, /<script\b|<img\b|<iframe\b|google-analytics|googletagmanager|plausible\.io|matomo/i);

	const homepage = readFileSync(homepagePath, 'utf8');
	assert.doesNotMatch(homepage, /<script\b|!\[[^\]]*\]\([^)]*\)|<img\b/i);
});

test('attribution mutation fixture satisfies the static contract', () => {
	const findings = analyzeAttribution({
		maintainer: VALID_MAINTAINER,
		footerSource: VALID_FOOTER,
		configSource: VALID_CONFIG,
	});
	assert.deepEqual(findings, [], `valid attribution fixture must pass; got ${findings.join(', ')}`);
});

test('mutation: a footer without visible credit fails', () => {
	const footer = VALID_FOOTER.replace(/<div class="maintainer-credit"[\s\S]*?<\/div>/, '');
	const findings = analyzeAttribution({
		maintainer: VALID_MAINTAINER,
		footerSource: footer,
		configSource: VALID_CONFIG,
	});
	assert.ok(findings.includes('credit-wording-missing'), `expected missing credit failure; got ${findings.join(', ')}`);
});

test('mutation: a wrong maintainer identity fails', () => {
	const findings = analyzeAttribution({
		maintainer: { ...VALID_MAINTAINER, name: 'Someone Else' },
		footerSource: VALID_FOOTER,
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
		footerSource: VALID_FOOTER,
		configSource: VALID_CONFIG,
	});
	assert.ok(
		findings.includes('maintainer-github-mismatch'),
		`expected wrong-URL failure; got ${findings.join(', ')}`,
	);
});

test('mutation: replacing the default footer utilities fails', () => {
	const footer = VALID_FOOTER.replace(/import DefaultFooter[^\n]*\n/, '').replace(
		'<DefaultFooter />',
		'<footer>default utilities removed</footer>',
	);
	const findings = analyzeAttribution({
		maintainer: VALID_MAINTAINER,
		footerSource: footer,
		configSource: VALID_CONFIG,
	});
	assert.ok(
		findings.includes('missing-default-footer-import') && findings.includes('nested-footer-element'),
		`expected missing-default-utilities failure; got ${findings.join(', ')}`,
	);
});
