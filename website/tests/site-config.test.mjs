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
		'docs:reference': 'node scripts/reference.mjs',
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
