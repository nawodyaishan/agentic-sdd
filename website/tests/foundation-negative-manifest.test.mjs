// Spec 020 B1-W1-D — independent counterexamples for the frozen C1 manifest API
// declared in specs/020-documentation-ecosystem/plan.md C1 and the B1 contract.
//
// These tests read the contract's declared behaviour, not the production source:
// required/optional fields, identity, containment, digest and read-only rules.
// They do not re-implement schema validation or hashing as an oracle; the only
// crypto used here is the standard library digest of real fixture bytes, which
// the contract itself defines.

import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

import {
  INDEX_SOURCE,
  PUBLIC_DOCS,
  VALID_MANIFEST,
  assertApiError,
  capture,
  clone,
  deepFreeze,
  fileDigest,
  fixtureTree,
  isErrorCode,
  isNonEmptyString,
  loadFrozenApi,
  readJson,
  reportPending,
  scratchRoot,
  snapshot,
} from './foundation-negative-support/support.mjs';

const api = await loadFrozenApi('website/scripts/lib/manifest.mjs', [
  'validateManifest',
  'loadManifest',
  'computeReviewDigest',
  'hashSources',
  'collectBaseline',
]);
reportPending('foundation-negative-manifest', api);

const skip = api.skip;
const M = api.mod;
const valid = readJson(VALID_MANIFEST);
const options = { repoRoot: fixtureTree };

const errorFields = {
  code: isErrorCode,
  path: (value) => typeof value === 'string',
};

/** Validate a mutated copy and require a ManifestError carrying code + path. */
function expectInvalid(label, mutate, mention) {
  test(`rejects ${label}`, { skip }, () => {
    const manifest = clone(valid);
    mutate(manifest);
    const { threw, error } = capture(() => M.validateManifest(manifest, options));
    assert.ok(threw, `${label}: validateManifest accepted an invalid manifest`);
    assertApiError(error, { name: 'ManifestError', fields: errorFields });
    if (mention) {
      const diagnostic = `${error.path} ${error.message}`;
      assert.match(diagnostic, mention, `${label}: diagnostic should identify the offending field (${diagnostic})`);
    }
  });
}

function expectedReviewDigest(manifest, pageIndex) {
  manifest.pages[pageIndex].reviewDigest = M.computeReviewDigest(manifest.pages[pageIndex].sources);
}

/**
 * A complete copy of the fixture tree in owned scratch space, so a rejection in
 * a symlink/directory case can only be caused by the injected condition rather
 * than by an unrelated missing page.
 */
function scratchTree() {
  const scratch = scratchRoot();
  fs.cpSync(fixtureTree, scratch, { recursive: true });
  return scratch;
}

const diagnosticText = (error) => `${error.path} ${error.message}`;

// ---------------------------------------------------------------------------
// Valid fixture, identity and immutability
// ---------------------------------------------------------------------------

test('accepts the frozen valid manifest fixture', { skip }, () => {
  const result = M.validateManifest(clone(valid), options);
  assert.equal(typeof result, 'object');
  assert.deepEqual(result, valid);
});

test('documented default options validate a contained manifest', { skip }, () => {
  // Contract signature: validateManifest(manifest, {repoRoot, verifySources=false}={}).
  assert.doesNotThrow(() => M.validateManifest(clone(valid)));
});

test('does not mutate its input document', { skip }, () => {
  const manifest = clone(valid);
  const before = snapshot(manifest);
  M.validateManifest(manifest, options);
  assert.equal(snapshot(manifest), before, 'validateManifest must not mutate the caller document');
});

test('accepts a deeply frozen input document', { skip }, () => {
  assert.doesNotThrow(() => M.validateManifest(deepFreeze(clone(valid)), options));
});

test('is deterministic for the same document', { skip }, () => {
  const manifest = clone(valid);
  assert.deepEqual(M.validateManifest(manifest, options), M.validateManifest(manifest, options));
});

test('rejects an undefined document', { skip }, () => {
  const { threw, error } = capture(() => M.validateManifest(undefined, options));
  assert.ok(threw, 'validateManifest(undefined) must fail');
  assertApiError(error, { name: 'ManifestError', fields: errorFields });
});

// ---------------------------------------------------------------------------
// Root schema
// ---------------------------------------------------------------------------

expectInvalid('an unknown root field', (m) => { m.unexpectedRoot = true; }, /unexpectedRoot|unknown/i);
expectInvalid('schemaVersion 2', (m) => { m.schemaVersion = 2; }, /schemaVersion/i);
expectInvalid('a missing schemaVersion', (m) => { delete m.schemaVersion; }, /schemaVersion/i);
expectInvalid('a non-HTTPS site origin', (m) => { m.site = 'http://example.github.io'; }, /site/i);
expectInvalid('a site origin carrying a path', (m) => { m.site = 'https://example.github.io/docs'; }, /site/i);
expectInvalid('a base without a leading slash', (m) => { m.base = 'agentic-sdd'; }, /base/i);
expectInvalid('a base containing traversal', (m) => { m.base = '/docs/../etc'; }, /base/i);
expectInvalid('a non-GitHub repository', (m) => { m.repository = 'https://gitlab.com/example/repo'; }, /repository/i);
expectInvalid('a missing repository', (m) => { delete m.repository; }, /repository/i);
expectInvalid('a missing docsVersion', (m) => { delete m.docsVersion; }, /docsVersion/i);
expectInvalid('a non-array pages value', (m) => { m.pages = {}; }, /pages/i);

// ---------------------------------------------------------------------------
// Page identity and source containment
// ---------------------------------------------------------------------------

expectInvalid('duplicate page IDs', (m) => { m.pages[1].id = m.pages[0].id; }, /id/i);
expectInvalid('duplicate page sources', (m) => { m.pages[1].source = m.pages[0].source; }, /source/i);
expectInvalid('an unknown page field', (m) => { m.pages[0].unexpectedPageField = true; }, /unexpectedPageField|unknown/i);
expectInvalid('a traversing page ID', (m) => { m.pages[0].id = '../outside'; }, /id/i);
expectInvalid('an uppercase page ID', (m) => { m.pages[0].id = 'Reference/CLI'; }, /id/i);
expectInvalid('a page ID with an extension', (m) => { m.pages[0].id = 'reference/cli.md'; }, /id/i);
expectInvalid('a page ID with spaces', (m) => { m.pages[0].id = 'reference/cli reference'; }, /id/i);
expectInvalid('a source outside the public docs root', (m) => { m.pages[0].source = 'docs/getting-started.md'; }, /source/i);
expectInvalid('a traversing source path', (m) => { m.pages[0].source = `${PUBLIC_DOCS}/../secret.md`; }, /source/i);
expectInvalid('a non-Markdown source path', (m) => { m.pages[0].source = `${PUBLIC_DOCS}/index.txt`; }, /source/i);
expectInvalid('an absolute source path', (m) => { m.pages[0].source = '/etc/passwd.md'; }, /source/i);
expectInvalid('a missing page source', (m) => { delete m.pages[0].source; }, /source/i);
expectInvalid('a missing page ID', (m) => { delete m.pages[0].id; }, /id/i);

// ---------------------------------------------------------------------------
// Page classification
// ---------------------------------------------------------------------------

expectInvalid('an unknown page type', (m) => { m.pages[0].type = 'tutorial2'; }, /type/i);
expectInvalid('a missing page class', (m) => { delete m.pages[0].class; }, /class/i);
expectInvalid('an unknown page class', (m) => { m.pages[0].class = 'X'; }, /class/i);
expectInvalid('a lowercase page class', (m) => { m.pages[0].class = 'l'; }, /class/i);
expectInvalid('an empty audience entry', (m) => { m.pages[0].audience = ['maintainer', '']; }, /audience/i);
expectInvalid('a non-array audience', (m) => { m.pages[0].audience = 'maintainer'; }, /audience/i);

// ---------------------------------------------------------------------------
// Evidence hashes and review identity
// ---------------------------------------------------------------------------

expectInvalid('a 63-character evidence SHA-256', (m) => {
  m.pages[0].sources[0].sha256 = 'a'.repeat(63);
}, /sha256|sources/i);
expectInvalid('an uppercase evidence SHA-256', (m) => {
  m.pages[0].sources[0].sha256 = m.pages[0].sources[0].sha256.toUpperCase();
}, /sha256|sources/i);
expectInvalid('a traversing evidence path', (m) => {
  m.pages[0].sources[0].path = '../outside/go.mod';
}, /sources|path/i);
expectInvalid('an evidence entry without a path', (m) => {
  delete m.pages[0].sources[0].path;
}, /sources|path/i);
expectInvalid('a stale reviewDigest', (m) => {
  m.pages[0].sources[0].sha256 = 'b'.repeat(64);
}, /reviewDigest|review/i);
expectInvalid('a malformed reviewDigest', (m) => { m.pages[0].reviewDigest = 'not-a-digest'; }, /reviewDigest|review/i);
expectInvalid('a missing reviewDigest', (m) => { delete m.pages[0].reviewDigest; }, /reviewDigest|review/i);
expectInvalid('an unreviewed page with a human review date', (m) => { m.pages[0].reviewStatus = 'unreviewed'; }, /reviewedAt|unreviewed/i);
expectInvalid('a reviewed page without a human review date', (m) => { m.pages[0].reviewStatus = 'reviewed'; delete m.pages[0].reviewedAt; }, /reviewedAt/i);
expectInvalid('a non-ISO reviewedAt', (m) => { m.pages[0].reviewedAt = '14-11-2026'; }, /reviewedAt/i);
expectInvalid('a missing appliesTo', (m) => { delete m.pages[0].appliesTo; }, /appliesTo/i);
expectInvalid('the unqualified appliesTo "latest"', (m) => { m.pages[0].appliesTo = 'latest'; }, /appliesTo/i);

test('does not auto-stamp a refreshed reviewDigest', { skip }, () => {
  const manifest = clone(valid);
  manifest.pages[0].sources[0].sha256 = 'b'.repeat(64);
  const staleDigest = manifest.pages[0].reviewDigest;
  const { threw } = capture(() => M.validateManifest(manifest, options));
  assert.ok(threw, 'a stale fingerprint must be needs-review, not silently stamped');
  assert.equal(manifest.pages[0].reviewDigest, staleDigest, 'reviewDigest must not be rewritten by validation');
});

test('computeReviewDigest matches the reviewed fixture fingerprint', { skip }, () => {
  // The fixture ships a digest recorded by the parent scaffold; the same sources
  // must reproduce it. This checks the manifest identity contract, not the helper.
  for (const page of valid.pages) {
    assert.equal(M.computeReviewDigest(page.sources), page.reviewDigest, `page ${page.id}`);
  }
});

// ---------------------------------------------------------------------------
// verifySources: real files, regular files only, no symlink resolution
// ---------------------------------------------------------------------------

test('verifySources accepts contained regular fixture sources', { skip }, () => {
  assert.doesNotThrow(() => M.validateManifest(clone(valid), { repoRoot: fixtureTree, verifySources: true }));
});

test('verifySources rejects a digest that does not match the file bytes', { skip }, () => {
  const manifest = clone(valid);
  manifest.pages[0].sources[0].sha256 = 'c'.repeat(64);
  expectedReviewDigest(manifest, 0); // keep review identity coherent; the hash must still fail
  const { threw, error } = capture(() =>
    M.validateManifest(manifest, { repoRoot: fixtureTree, verifySources: true }));
  assert.ok(threw, 'verifySources must compare declared digests with real file bytes');
  assertApiError(error, { name: 'ManifestError', fields: errorFields });
});

test('verifySources rejects a missing source file', { skip }, () => {
  const manifest = clone(valid);
  manifest.pages[0].source = `${PUBLIC_DOCS}/missing.md`;
  const { threw, error } = capture(() =>
    M.validateManifest(manifest, { repoRoot: fixtureTree, verifySources: true }));
  assert.ok(threw, 'a missing page source must fail verification');
  assertApiError(error, { name: 'ManifestError', fields: errorFields });
  assert.match(diagnosticText(error), /missing\.md/i, 'the diagnostic must name the missing source');
});

test('verifySources rejects a symlinked source file', { skip }, () => {
  const scratch = scratchTree();
  try {
    fs.mkdirSync(path.join(scratch, 'outside'), { recursive: true });
    fs.writeFileSync(path.join(scratch, 'outside', 'secret.md'), '# secret\n');
    fs.symlinkSync(path.join(scratch, 'outside', 'secret.md'), path.join(scratch, PUBLIC_DOCS, 'linked.md'));

    const manifest = clone(valid);
    manifest.pages[0].source = `${PUBLIC_DOCS}/linked.md`;
    const { threw, error } = capture(() =>
      M.validateManifest(manifest, { repoRoot: scratch, verifySources: true }));
    assert.ok(threw, 'a symlinked public source must not resolve');
    assertApiError(error, { name: 'ManifestError', fields: errorFields });
    assert.match(diagnosticText(error), /linked\.md|symlink/i, 'the diagnostic must name the symlinked source');
  } finally {
    fs.rmSync(scratch, { recursive: true, force: true });
  }
});

test('verifySources rejects a source below a symlinked component', { skip }, () => {
  const scratch = scratchTree();
  try {
    fs.mkdirSync(path.join(scratch, 'outside'), { recursive: true });
    fs.writeFileSync(path.join(scratch, 'outside', 'secret.md'), '# secret\n');
    fs.symlinkSync(path.join(scratch, 'outside'), path.join(scratch, PUBLIC_DOCS, 'linked-dir'));

    const manifest = clone(valid);
    manifest.pages[0].source = `${PUBLIC_DOCS}/linked-dir/secret.md`;
    const { threw, error } = capture(() =>
      M.validateManifest(manifest, { repoRoot: scratch, verifySources: true }));
    assert.ok(threw, 'a source reached through a symlinked directory must not resolve');
    assertApiError(error, { name: 'ManifestError', fields: errorFields });
    assert.match(diagnosticText(error), /linked-dir|symlink/i, 'the diagnostic must name the symlinked component');
  } finally {
    fs.rmSync(scratch, { recursive: true, force: true });
  }
});

test('verifySources rejects a directory used as a source file', { skip }, () => {
  const scratch = scratchTree();
  try {
    fs.mkdirSync(path.join(scratch, PUBLIC_DOCS, 'directory.md'), { recursive: true });

    const manifest = clone(valid);
    manifest.pages[0].source = `${PUBLIC_DOCS}/directory.md`;
    const { threw, error } = capture(() =>
      M.validateManifest(manifest, { repoRoot: scratch, verifySources: true }));
    assert.ok(threw, 'only regular files may be public sources');
    assertApiError(error, { name: 'ManifestError', fields: errorFields });
    assert.match(diagnosticText(error), /directory\.md|regular/i, 'the diagnostic must name the rejected source');
  } finally {
    fs.rmSync(scratch, { recursive: true, force: true });
  }
});

// ---------------------------------------------------------------------------
// loadManifest: parse + validate, no writes
// ---------------------------------------------------------------------------

test('loadManifest returns the validated document without writing', { skip }, () => {
  const digestBefore = fileDigest(VALID_MANIFEST);
  const entriesBefore = fs.readdirSync(path.dirname(VALID_MANIFEST)).sort();
  const loaded = M.loadManifest(VALID_MANIFEST, options);
  assert.deepEqual(loaded, valid);
  assert.equal(fileDigest(VALID_MANIFEST), digestBefore, 'loadManifest must not rewrite its input');
  assert.deepEqual(fs.readdirSync(path.dirname(VALID_MANIFEST)).sort(), entriesBefore);
});

test('loadManifest surfaces malformed JSON as an error', { skip }, () => {
  const { threw, error } = capture(() =>
    M.loadManifest(path.join(path.dirname(VALID_MANIFEST), 'malformed.json'), options));
  assert.ok(threw, 'malformed JSON must not load');
  assert.ok(error instanceof Error);
  assert.equal(typeof error.message, 'string');
});

test('loadManifest fails for a missing manifest file', { skip }, () => {
  const { threw } = capture(() =>
    M.loadManifest(path.join(fixtureTree, 'no-such-manifest.json'), options));
  assert.ok(threw, 'a missing manifest path must fail rather than invent a document');
});

// ---------------------------------------------------------------------------
// computeReviewDigest: sorted, compact, deterministic, read-only
// ---------------------------------------------------------------------------

test('computeReviewDigest is independent of source order', { skip }, () => {
  const page = valid.pages[1];
  const sources = clone(page.sources);
  const reversed = [...sources].reverse();
  assert.equal(M.computeReviewDigest(sources), M.computeReviewDigest(reversed));
  assert.equal(M.computeReviewDigest(sources), M.computeReviewDigest(sources));
  assert.match(M.computeReviewDigest(sources), /^[0-9a-f]{64}$/);
});

test('computeReviewDigest changes with path or digest content', { skip }, () => {
  const sources = clone(valid.pages[0].sources);
  const base = M.computeReviewDigest(sources);
  const otherPath = clone(sources);
  otherPath[0].path = 'go.sum';
  const otherHash = clone(sources);
  otherHash[0].sha256 = 'd'.repeat(64);
  assert.notEqual(M.computeReviewDigest(otherPath), base);
  assert.notEqual(M.computeReviewDigest(otherHash), base);
});

test('computeReviewDigest does not mutate its argument', { skip }, () => {
  const sources = clone(valid.pages[0].sources);
  const before = snapshot(sources);
  M.computeReviewDigest(sources);
  assert.equal(snapshot(sources), before);
});

// ---------------------------------------------------------------------------
// hashSources: actual regular contained files
// ---------------------------------------------------------------------------

test('hashSources hashes real fixture bytes in input order', { skip }, () => {
  const inputs = [
    { path: 'go.mod' },
    { path: `${PUBLIC_DOCS}/reference/cli.md` },
  ];
  const before = snapshot(inputs);
  const result = M.hashSources(fixtureTree, inputs);
  assert.deepEqual(inputs.map((entry) => entry.path), ['go.mod', `${PUBLIC_DOCS}/reference/cli.md`]);
  assert.equal(result.length, 2);
  assert.equal(result[0].path, 'go.mod');
  assert.equal(result[1].path, `${PUBLIC_DOCS}/reference/cli.md`);
  assert.equal(result[0].sha256, fileDigest(path.join(fixtureTree, 'go.mod')));
  assert.equal(result[1].sha256, fileDigest(path.join(fixtureTree, PUBLIC_DOCS, 'reference', 'cli.md')));
  assert.equal(snapshot(inputs), before, 'hashSources must not mutate its argument');
});

test('hashSources rejects a missing file', { skip }, () => {
  assert.throws(() => M.hashSources(fixtureTree, [{ path: 'go.sum' }]));
});

test('hashSources rejects traversal outside the repository root', { skip }, () => {
  assert.throws(() => M.hashSources(fixtureTree, [{ path: '../outside.md' }]));
  assert.throws(() => M.hashSources(fixtureTree, [{ path: '/etc/hosts' }]));
});

test('hashSources rejects a directory', { skip }, () => {
  assert.throws(() => M.hashSources(fixtureTree, [{ path: 'website' }]));
});

test('hashSources rejects a symlinked file', { skip }, () => {
  const scratch = scratchRoot();
  try {
    fs.writeFileSync(path.join(scratch, 'real.md'), '# real\n');
    fs.symlinkSync(path.join(scratch, 'real.md'), path.join(scratch, 'linked.md'));
    assert.throws(() => M.hashSources(scratch, [{ path: 'linked.md' }]));
  } finally {
    fs.rmSync(scratch, { recursive: true, force: true });
  }
});

// ---------------------------------------------------------------------------
// collectBaseline: explicit public paths only, fenced code ignored
// ---------------------------------------------------------------------------

test('collectBaseline reports only the supplied public paths', { skip }, () => {
  const paths = [INDEX_SOURCE, `${PUBLIC_DOCS}/reference/cli.md`];
  const first = M.collectBaseline(fixtureTree, paths);
  const second = M.collectBaseline(fixtureTree, [...paths]);
  assert.ok(Array.isArray(first.files));
  assert.equal(first.files.length, 2, 'no recursive scan beyond the explicit paths');
  assert.deepEqual(first.files.map((file) => file.path), paths);
  assert.deepEqual(first, second, 'baseline output must be deterministic');
});

test('collectBaseline ignores fenced code for headings and links', { skip }, () => {
  const baseline = M.collectBaseline(fixtureTree, [INDEX_SOURCE]);
  const page = baseline.files[0];
  assert.ok(Array.isArray(page.headings));
  assert.ok(Array.isArray(page.links));
  const headings = JSON.stringify(page.headings);
  const links = JSON.stringify(page.links);
  assert.match(headings, /Real heading/);
  assert.doesNotMatch(headings, /not a heading/, 'fenced pseudo-heading must be ignored');
  assert.doesNotMatch(links, /fenced-target\.md/, 'fenced pseudo-link must be ignored');
  assert.match(links, /reference\/cli\.md/);
  assert.match(links, /guides\/other%20page\.md/);
});

test('collectBaseline inventories explicitly selected existing public guides', { skip }, () => {
  const result = M.collectBaseline(fixtureTree, ['docs/getting-started.md']);
  assert.deepEqual(result.files.map((file) => file.path), ['docs/getting-started.md']);
});

test('collectBaseline refuses private execution scratch rather than treating it as a guide', { skip }, () => {
  assert.throws(() => M.collectBaseline(fixtureTree, ['.agentic-sdd/runs/private.md']));
});

test('collectBaseline rejects a missing explicit path', { skip }, () => {
  assert.throws(() => M.collectBaseline(fixtureTree, [`${PUBLIC_DOCS}/missing.md`]));
});
