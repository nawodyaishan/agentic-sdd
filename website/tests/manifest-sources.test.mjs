// C1 source containment and fingerprinting: hashing only when requested,
// traversal/symlink/special-file refusal, private-tree refusal and a read-only
// loader. All filesystem work happens in a disposable temporary repository.

import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import {
  lstatSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  realpathSync,
  rmSync,
  statSync,
  symlinkSync,
  writeFileSync,
} from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import {
  ManifestError,
  collectBaseline,
  computeReviewDigest,
  hashSources,
  loadManifest,
  validateManifest,
} from '../scripts/lib/manifest.mjs';
import { ONE_SOURCE_DIGEST, manifest, page } from './manifest-fixtures.mjs';

const WORKTREE_ROOT = fileURLToPath(new URL('../../', import.meta.url));

function sha256(value) {
  return createHash('sha256').update(value).digest('hex');
}

function expectError(run, code, errorPath) {
  assert.throws(run, (error) => {
    assert.ok(error instanceof ManifestError, `expected ManifestError, got ${error?.constructor?.name}`);
    assert.equal(error.code, code, `code ${error.code} !== ${code}: ${error.message}`);
    if (errorPath !== undefined) {
      assert.equal(error.path, errorPath, `path ${error.path} !== ${errorPath}: ${error.message}`);
    }
    return true;
  });
}

function write(root, relative, contents = `# ${relative}\n`) {
  const target = path.join(root, relative);
  mkdirSync(path.dirname(target), { recursive: true });
  writeFileSync(target, contents);
  return target;
}

function makeRepo(t) {
  const root = realpathSync(mkdtempSync(path.join(tmpdir(), 'spec020-manifest-')));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  write(root, 'README.md', '# Fixture\n');
  write(root, 'go.mod', 'module example\n');
  write(root, 'website/src/content/docs/index.md', '# Index\n');
  write(root, 'docs/guide.md', '# Guide\n');
  write(root, 'specs/020-feature/spec.md', '# Internal\n');
  write(root, 'backups/previous.md', '# Backup\n');
  write(root, '.git/config', '[core]\n');
  write(root, 'docs/testing/report.md', '# Report\n');
  write(root, 'docs/handovers/2026-01-01_020.md', '# Handover\n');
  return root;
}

function scaffoldManifest(root, overrides = {}) {
  const goMod = readFileSync(path.join(root, 'go.mod'), 'utf8');
  const sources = overrides.sources ?? [{ path: 'go.mod', sha256: sha256(goMod) }];
  return {
    schemaVersion: 1,
    site: 'https://example.github.io',
    base: '/project',
    repository: 'https://github.com/example/project',
    docsVersion: 'dev',
    pages: [
      {
        id: 'index',
        source: 'website/src/content/docs/index.md',
        type: 'explanation',
        audience: ['maintainer'],
        class: 'L',
        sources,
        reviewDigest: computeReviewDigest(sources),
        reviewedAt: '2026-11-14',
        appliesTo: 'unreleased',
      },
    ],
    ...overrides.root,
  };
}

function writeScaffold(root, overrides = {}) {
  const target = path.join(root, 'website/docs-manifest.json');
  writeFileSync(target, `${JSON.stringify(scaffoldManifest(root, overrides), null, 2)}\n`);
  return target;
}

test('hashes actual regular contained files and sorts the result', (t) => {
  const root = makeRepo(t);
  const goMod = readFileSync(path.join(root, 'go.mod'));

  const single = hashSources(root, [{ path: 'go.mod' }]);
  assert.deepEqual(single, [{ path: 'go.mod', sha256: sha256(goMod) }]);

  const ordered = hashSources(root, [{ path: 'README.md' }, { path: 'go.mod' }]);
  assert.deepEqual(ordered.map((entry) => entry.path), ['README.md', 'go.mod']);

  const reversed = hashSources(root, [{ path: 'go.mod' }, { path: 'README.md' }]);
  assert.deepEqual(reversed, ordered, 'output order must not depend on input order');

  assert.deepEqual(hashSources(root, []), []);
  assert.deepEqual(hashSources(root, ['go.mod'])[0].path, 'go.mod');
});

test('verifies a supplied fingerprint against the file and reports the mismatch', (t) => {
  const root = makeRepo(t);
  const actual = sha256(readFileSync(path.join(root, 'go.mod')));
  assert.deepEqual(hashSources(root, [{ path: 'go.mod', sha256: actual }]), [{ path: 'go.mod', sha256: actual }]);
  expectError(
    () => hashSources(root, [{ path: 'go.mod', sha256: 'f'.repeat(64) }]),
    'source-sha-mismatch',
    'go.mod',
  );
});

test('refuses traversal, absolute, backslash and malformed paths', (t) => {
  const root = makeRepo(t);
  for (const declared of ['../outside.md', 'docs/../README.md', './README.md', '/README.md', 'docs\\guide.md', 'docs//guide.md', '']) {
    assert.throws(
      () => hashSources(root, [declared]),
      (error) => error instanceof ManifestError && error.code === 'path-escape',
      `expected path-escape for ${JSON.stringify(declared)}`,
    );
  }
});

test('refuses private repository trees as evidence inputs', (t) => {
  const root = makeRepo(t);
  write(root, '.env', 'TOKEN=secret\n');
  write(root, '.npmrc', 'registry=https://example.com\n');
  write(root, 'certs/server.pem', 'not really a key\n');
  for (const declared of [
    'backups/previous.md',
    '.git/config',
    'docs/handovers/2026-01-01_020.md',
    'credentials/token.md',
    '.env',
    '.npmrc',
    'certs/server.pem',
  ]) {
    expectError(() => hashSources(root, [declared]), 'private-source-path', declared);
  }
});

test('engineering specifications and test inventories are hashable evidence, not public page bodies', (t) => {
  const root = makeRepo(t);
  const sources = hashSources(root, [
    'specs/020-feature/spec.md',
    'docs/testing/report.md',
  ]);
  const registry = scaffoldManifest(root, { sources });
  assert.doesNotThrow(() => validateManifest(registry, { repoRoot: root, verifySources: true }));
  registry.pages[0].source = 'docs/testing/report.md';
  assert.throws(() => validateManifest(registry), /source|public/i);
});

test('deeply nested manifest input produces a bounded typed diagnostic, not stack exhaustion', (t) => {
  const registry = scaffoldManifest(makeRepo(t));
  let nested = { value: 'fixture' };
  for (let index = 0; index < 5000; index += 1) nested = { nested };
  registry.diagrams = [nested];
  assert.throws(() => validateManifest(registry), (error) =>
    error instanceof ManifestError && /depth|nest/i.test(error.message));
});

test('refuses missing paths, directories and symlinks', (t) => {
  const root = makeRepo(t);
  expectError(() => hashSources(root, ['missing.md']), 'missing-source-file', 'missing.md');
  expectError(() => hashSources(root, ['docs']), 'not-a-regular-file', 'docs');

  const outside = realpathSync(mkdtempSync(path.join(tmpdir(), 'spec020-outside-')));
  t.after(() => rmSync(outside, { recursive: true, force: true }));
  write(outside, 'secret.md', '# Outside\n');

  symlinkSync(path.join(outside, 'secret.md'), path.join(root, 'linked.md'));
  expectError(() => hashSources(root, ['linked.md']), 'symlink-path', 'linked.md');

  symlinkSync(outside, path.join(root, 'linked-dir'));
  expectError(() => hashSources(root, ['linked-dir/secret.md']), 'symlink-path', 'linked-dir/secret.md');
});

test('refuses special files', (t) => {
  const root = makeRepo(t);
  const fifo = path.join(root, 'pipe.md');
  try {
    execFileSync('mkfifo', [fifo]);
  } catch {
    t.skip('mkfifo unavailable on this platform');
    return;
  }
  expectError(() => hashSources(root, ['pipe.md']), 'not-a-regular-file', 'pipe.md');
});

test('refuses missing, malformed and duplicate input lists', (t) => {
  const root = makeRepo(t);
  expectError(() => hashSources(root, 'go.mod'), 'invalid-evidence-list');
  expectError(() => hashSources(root, [42]), 'invalid-evidence', '[0]');
  expectError(
    () => hashSources(root, [{ path: 'go.mod' }, { path: 'go.mod' }]),
    'duplicate-evidence-path',
    'go.mod',
  );
  expectError(() => hashSources(path.join(root, 'nope'), ['go.mod']), 'invalid-repo-root');
});

test('read-only helpers neither create nor modify files', (t) => {
  const root = makeRepo(t);
  write(root, 'docs/guide.md', '# Guide\n\n[back](../README.md)\n');
  const manifestPath = writeScaffold(root);

  const snapshot = () =>
    readdirSync(root, { recursive: true })
      .sort()
      .map((relative) => {
        const stats = lstatSync(path.join(root, relative));
        return [relative, stats.isFile() ? stats.mtimeMs : 'directory'];
      });
  const before = snapshot();

  hashSources(root, ['go.mod', 'README.md']);
  collectBaseline(root, ['README.md', 'docs/guide.md']);
  loadManifest(manifestPath, { verifySources: true });

  assert.deepEqual(snapshot(), before, 'read-only helpers must not create or rewrite files');
});

test('verifySources validates declared files and hashes only when requested', (t) => {
  const root = makeRepo(t);
  const valid = scaffoldManifest(root);
  const validated = validateManifest(valid, { repoRoot: root, verifySources: true });
  assert.ok(Object.isFrozen(validated.pages[0].sources[0]));
  assert.equal(validated.pages[0].sources[0].sha256, sha256(readFileSync(path.join(root, 'go.mod'))));
});

test('verifySources refuses missing files, stale hashes, symlinks and a missing repoRoot', (t) => {
  const root = makeRepo(t);
  const valid = scaffoldManifest(root);

  expectError(() => validateManifest(valid, { verifySources: true }), 'missing-repo-root');

  const missingPage = structuredClone(valid);
  missingPage.pages[0].source = 'website/src/content/docs/missing.md';
  expectError(
    () => validateManifest(missingPage, { repoRoot: root, verifySources: true }),
    'missing-source-file',
    'website/src/content/docs/missing.md',
  );

  const missingEvidence = structuredClone(valid);
  missingEvidence.pages[0].sources = [{ path: 'missing.md', sha256: 'a'.repeat(64) }];
  missingEvidence.pages[0].reviewDigest = computeReviewDigest(missingEvidence.pages[0].sources);
  expectError(
    () => validateManifest(missingEvidence, { repoRoot: root, verifySources: true }),
    'missing-source-file',
    'missing.md',
  );

  const stale = structuredClone(valid);
  stale.pages[0].sources = [{ path: 'go.mod', sha256: 'b'.repeat(64) }];
  stale.pages[0].reviewDigest = computeReviewDigest(stale.pages[0].sources);
  expectError(
    () => validateManifest(stale, { repoRoot: root, verifySources: true }),
    'source-sha-mismatch',
    'go.mod',
  );

  const outside = realpathSync(mkdtempSync(path.join(tmpdir(), 'spec020-outside-')));
  t.after(() => rmSync(outside, { recursive: true, force: true }));
  write(outside, 'page.md', '# Outside\n');
  symlinkSync(path.join(outside, 'page.md'), path.join(root, 'website/src/content/docs/linked.md'));
  const symlinked = structuredClone(valid);
  symlinked.pages[0].source = 'website/src/content/docs/linked.md';
  expectError(
    () => validateManifest(symlinked, { repoRoot: root, verifySources: true }),
    'symlink-path',
    'website/src/content/docs/linked.md',
  );
});

test('schema-only validation never reads declared files', (t) => {
  const root = makeRepo(t);
  const input = manifest({
    pages: [
      page({
        source: 'website/src/content/docs/missing.md',
        sources: [{ path: 'missing.md', sha256: 'c'.repeat(64) }],
        reviewDigest: computeReviewDigest([{ path: 'missing.md', sha256: 'c'.repeat(64) }]),
      }),
    ],
  });
  const validated = validateManifest(input, { repoRoot: root });
  assert.equal(validated.pages[0].source, 'website/src/content/docs/missing.md');
});

test('loadManifest parses, validates and derives the repository root without writing', (t) => {
  const root = makeRepo(t);
  const manifestPath = writeScaffold(root);
  const listingBefore = readdirSync(root, { recursive: true }).sort();
  const manifestBefore = statSync(manifestPath).mtimeMs;
  const evidenceBefore = statSync(path.join(root, 'go.mod')).mtimeMs;

  const loaded = loadManifest(manifestPath);
  assert.equal(loaded.pages[0].source, 'website/src/content/docs/index.md');
  assert.ok(Object.isFrozen(loaded));

  assert.deepEqual(readdirSync(root, { recursive: true }).sort(), listingBefore, 'loadManifest must not create files');
  assert.equal(statSync(manifestPath).mtimeMs, manifestBefore);
  assert.equal(statSync(path.join(root, 'go.mod')).mtimeMs, evidenceBefore);

  const verified = loadManifest(manifestPath, { verifySources: true });
  assert.equal(verified.pages[0].sources[0].sha256, sha256(readFileSync(path.join(root, 'go.mod'))));
});

test('loadManifest refuses unknown options, bad JSON, missing files, directories and symlinks', (t) => {
  const root = makeRepo(t);
  const manifestPath = writeScaffold(root);

  expectError(() => loadManifest(manifestPath, { follow: true }), 'invalid-options', 'follow');
  expectError(() => loadManifest(manifestPath, { verifySources: 'yes' }), 'invalid-options', 'verifySources');

  expectError(() => loadManifest(path.join(root, 'missing.json')), 'manifest-missing');
  expectError(() => loadManifest(root), 'manifest-not-regular-file');

  writeFileSync(manifestPath, '{ "schemaVersion": 1, ');
  expectError(() => loadManifest(manifestPath), 'invalid-json');

  const target = path.join(root, 'website/other-manifest.json');
  writeFileSync(target, `${JSON.stringify(scaffoldManifest(root), null, 2)}\n`);
  const link = path.join(root, 'website/linked-manifest.json');
  symlinkSync(target, link);
  expectError(() => loadManifest(link), 'manifest-symlink');
});

test('loadManifest validates the live B1 scaffold with source verification', () => {
  const loaded = loadManifest(path.join(WORKTREE_ROOT, 'website/docs-manifest.json'), {
    repoRoot: WORKTREE_ROOT,
    verifySources: true,
  });
  assert.equal(loaded.schemaVersion, 1);
  assert.ok(loaded.pages.length >= 1);
  for (const entry of loaded.pages) {
    assert.equal(entry.reviewDigest, computeReviewDigest(entry.sources));
  }
  assert.ok(ONE_SOURCE_DIGEST.length === 64);
});
