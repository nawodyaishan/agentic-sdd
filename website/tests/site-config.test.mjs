import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const testDir = dirname(fileURLToPath(import.meta.url));
const websiteDir = resolve(testDir, '..');

test('runtime pin .node-version matches Node 24 LTS and package.json engines', () => {
	const nodeVersion = readFileSync(resolve(websiteDir, '.node-version'), 'utf8').trim();
	assert.match(nodeVersion, /^24\.\d+\.\d+$/);

	const pkg = JSON.parse(readFileSync(resolve(websiteDir, 'package.json'), 'utf8'));
	assert.equal(pkg.type, 'module');
	assert.equal(pkg.private, true);
	assert.match(pkg.engines?.node ?? '', />=24\.\d+\.\d+\s+<25\.0\.0/);
});

test('package.json uses exact pins including yaml and minimal scripts', () => {
	const pkg = JSON.parse(readFileSync(resolve(websiteDir, 'package.json'), 'utf8'));
	assert.deepEqual(pkg.scripts, {
		build: 'astro build',
		'build:site': 'astro build && node scripts/export.mjs --out dist',
		'docs:check': 'node scripts/check.mjs',
		'docs:check-built': 'node scripts/check.mjs --built website/dist',
		'docs:reference': 'node scripts/reference.mjs --generate',
		'docs:reference:check': 'node scripts/reference.mjs --check',
		'docs:examples': 'node scripts/examples.mjs',
		'docs:impact': 'node scripts/impact.mjs',
		test: 'node --test tests/*.test.mjs',
	});

	const requiredDeps = ['@astrojs/markdown-remark', '@astrojs/starlight', 'astro', 'yaml'];
	assert.deepEqual(Object.keys(pkg.dependencies).sort(), requiredDeps);
	assert.equal(pkg.dependencies['js-yaml'], undefined);

	for (const [name, spec] of Object.entries(pkg.dependencies)) {
		assert.match(spec, /^\d+\.\d+\.\d+$/, `dependency ${name} must use an exact version pin, got ${spec}`);
	}
});

test('package-lock.json uses registry HTTPS URLs, integrity hashes, and no active Linux install scripts', () => {
	const lockPath = resolve(websiteDir, 'package-lock.json');
	assert.equal(existsSync(lockPath), true);
	const lock = JSON.parse(readFileSync(lockPath, 'utf8'));
	assert.equal(lock.lockfileVersion, 3);

	const installScriptPackages = [];
	for (const [pkgPath, meta] of Object.entries(lock.packages ?? {})) {
		if (!pkgPath) continue;
		assert.match(
			meta.resolved ?? '',
			/^https:\/\/registry\.npmjs\.org\//,
			`unexpected resolved URL for ${pkgPath}: ${meta.resolved}`
		);
		assert.match(
			meta.integrity ?? '',
			/^sha(256|384|512)-/,
			`missing SRI integrity for ${pkgPath}`
		);
		if (meta.hasInstallScript) {
			installScriptPackages.push(pkgPath);
		}
	}

	assert.deepEqual(
		installScriptPackages,
		['node_modules/fsevents'],
		'only the macOS-only optional fsevents package may declare an install script; Linux provisioning uses --ignore-scripts',
	);
});

test('astro.config.mjs and content.config.ts follow standard Starlight + manifest C1 boundaries', () => {
	const astroConfig = readFileSync(resolve(websiteDir, 'astro.config.mjs'), 'utf8');
	assert.match(astroConfig, /docs-manifest\.json/);
	assert.match(astroConfig, /loadManifest/);
	assert.match(astroConfig, /unified\(/);
	assert.match(astroConfig, /@astrojs\/markdown-remark/);
	assert.match(astroConfig, /pagefind:\s*true/);
	assert.match(astroConfig, /trailingSlash:\s*'always'/);
	assert.match(astroConfig, /cacheDir:/);
	assert.match(astroConfig, /remark-link-routes/);
	assert.match(astroConfig, /remarkPlugins:\s*\[\[remarkLinkRoutes/);
	assert.doesNotMatch(astroConfig, /JSON\.parse/);

	const contentConfig = readFileSync(resolve(websiteDir, 'src/content.config.ts'), 'utf8');
	assert.match(contentConfig, /docsLoader\(/);
	assert.match(contentConfig, /docsSchema\(/);
});

test('deployable build chains the export into the Astro output directory', () => {
	const pkg = JSON.parse(readFileSync(resolve(websiteDir, 'package.json'), 'utf8'));
	assert.equal(pkg.scripts.build, 'astro build', 'the pure site build must stay Astro-only');
	assert.equal(
		pkg.scripts['build:site'],
		'astro build && node scripts/export.mjs --out dist',
		'the deployable artifact must build the site and then export into dist'
	);

	const astroConfig = readFileSync(resolve(websiteDir, 'astro.config.mjs'), 'utf8');
	assert.match(
		astroConfig,
		/const outDir = process\.env\.ASTRO_OUT_DIR \|\| '\.\/dist';/,
		'export --out dist must target the Astro output directory'
	);

	const taskfile = readFileSync(resolve(websiteDir, '..', 'Taskfile.yml'), 'utf8');
	assert.match(taskfile, /'docs:site':/);
	assert.match(taskfile, /npm --prefix website run build:site/);
});

test('SEO assets are authored SVG first, real raster PNGs second, and wired through the base', () => {
	const svg = readFileSync(resolve(websiteDir, 'public/favicon.svg'), 'utf8');
	assert.match(svg, /^<svg[^>]*viewBox="0 0 64 64"[^>]*>/);
	assert.match(svg, /<rect width="64" height="64" fill="#0f172a"\/>/);
	assert.doesNotMatch(svg, /<image\b/);
	assert.doesNotMatch(svg, /(?:href|src)="(?:https?:)?\/\//);

	function assertPng(name, width, height) {
		const png = readFileSync(resolve(websiteDir, 'public', name));
		assert.deepEqual(
			[...png.subarray(0, 8)],
			[0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a],
			`${name} must be a real PNG, not an SVG renamed`,
		);
		assert.equal(png.subarray(12, 16).toString('ascii'), 'IHDR');
		assert.equal(png.readUInt32BE(16), width, `${name} width`);
		assert.equal(png.readUInt32BE(20), height, `${name} height`);
	}

	assertPng('favicon-48.png', 48, 48);
	assertPng('favicon-192.png', 192, 192);
	assertPng('og-card.png', 1200, 630);

	const astroConfig = readFileSync(resolve(websiteDir, 'astro.config.mjs'), 'utf8');
	assert.match(astroConfig, /routeMiddleware: '\.\/src\/starlightRouteData\.ts'/);
	assert.match(astroConfig, /description:\s*\n\s*'Spec-driven engineering skills/);
	assert.doesNotMatch(astroConfig, /'Documentation for Agentic-SDD\.'/);
	assert.match(astroConfig, /const assetBase = manifest\.base/);
	assert.match(astroConfig, /`\$\{assetBase\}favicon-48\.png`/);
	assert.match(astroConfig, /`\$\{assetBase\}favicon-192\.png`/);
});
