// C1 manifest schema validation: required/optional fields, identity, evidence
// metadata, immutability and unknown-field refusal. Behavioural tests only: the
// module is expected to exist and throw ManifestError for each refusal.

import assert from 'node:assert/strict';
import test from 'node:test';

import { ManifestError, validateManifest } from '../scripts/lib/manifest.mjs';
import { ONE_SOURCE_DIGEST, SHA_A, manifest, page, releaseEvidence } from './manifest-fixtures.mjs';

function expectError(run, code, errorPath) {
  assert.throws(run, (error) => {
    assert.ok(error instanceof ManifestError, `expected ManifestError, got ${error?.constructor?.name}`);
    assert.equal(error.name, 'ManifestError');
    assert.equal(error.code, code, `code ${error.code} !== ${code}: ${error.message}`);
    if (errorPath !== undefined) {
      assert.equal(error.path, errorPath, `path ${error.path} !== ${errorPath}: ${error.message}`);
    }
    assert.ok(typeof error.message === 'string' && error.message.length > 0);
    return true;
  });
}

test('accepts the frozen valid fixture and returns an immutable copy', () => {
  const input = manifest();
  const before = structuredClone(input);
  const result = validateManifest(input);

  assert.deepEqual(result, before);
  assert.notStrictEqual(result, input);
  assert.notStrictEqual(result.pages, input.pages);
  assert.notStrictEqual(result.pages[0], input.pages[0]);
  assert.notStrictEqual(result.pages[0].sources, input.pages[0].sources);
  assert.ok(Object.isFrozen(result));
  assert.ok(Object.isFrozen(result.pages));
  assert.ok(Object.isFrozen(result.pages[0]));
  assert.ok(Object.isFrozen(result.pages[0].sources));
  assert.ok(Object.isFrozen(result.pages[0].sources[0]));
  assert.ok(Object.isFrozen(result.pages[0].audience));
  assert.deepEqual(input, before, 'validateManifest must not mutate its input');
});

test('accepts the root project base and a nested page id', () => {
  const root = validateManifest(manifest({ base: '/' }));
  assert.equal(root.base, '/');
  const nested = validateManifest(manifest({ pages: [page({ id: 'guides/install-tools' })] }));
  assert.equal(nested.pages[0].id, 'guides/install-tools');
});

test('rejects an unknown root field', () => {
  expectError(() => validateManifest(manifest({ publish: true })), 'unknown-field', 'publish');
});

test('rejects missing required root fields', () => {
  for (const field of ['schemaVersion', 'site', 'base', 'repository', 'docsVersion', 'pages']) {
    const input = manifest();
    delete input[field];
    expectError(() => validateManifest(input), 'missing-field', field);
  }
});

test('rejects a non-object manifest and an unsupported schemaVersion', () => {
  for (const value of [null, [], 'manifest', 1]) {
    assert.throws(() => validateManifest(value), (error) => error instanceof ManifestError);
  }
  expectError(() => validateManifest(manifest({ schemaVersion: 2 })), 'invalid-schema-version', 'schemaVersion');
  expectError(() => validateManifest(manifest({ schemaVersion: '1' })), 'invalid-schema-version', 'schemaVersion');
});

test('rejects a non-HTTPS, non-origin or credential-bearing site', () => {
  for (const site of [
    'http://example.github.io',
    'https://example.github.io/',
    'https://example.github.io/project',
    'https://user:pass@example.github.io',
    'https://example.github.io?q=1',
    'example.github.io',
    '',
  ]) {
    expectError(() => validateManifest(manifest({ site })), 'invalid-site', 'site');
  }
});

test('rejects an unnormalised base', () => {
  for (const base of [
    '',
    'project',
    '/project/',
    '//project',
    '/project//nested',
    '/project/../other',
    '/project/./nested',
    '/project?x=1',
    '/project#frag',
    '/pro ject',
  ]) {
    expectError(() => validateManifest(manifest({ base })), 'invalid-base', 'base');
  }
  assert.equal(validateManifest(manifest({ base: '/Project-Docs' })).base, '/Project-Docs');
});

test('rejects a repository that is not a fixed HTTPS GitHub repository', () => {
  for (const repository of [
    'http://github.com/example/project',
    'https://github.com/example/project/',
    'https://github.com/example/project.git',
    'https://github.com/example/project/tree/main',
    'https://gitlab.com/example/project',
    'https://github.com/example',
    'git@github.com:example/project.git',
  ]) {
    expectError(() => validateManifest(manifest({ repository })), 'invalid-repository', 'repository');
  }
});

test('requires an evidenced release tag when docsVersion is not dev', () => {
  expectError(() => validateManifest(manifest({ docsVersion: 'latest' })), 'invalid-docs-version', 'docsVersion');
  expectError(() => validateManifest(manifest({ docsVersion: 'v0.11.0' })), 'unevidenced-version', 'docsVersion');
  const evidenced = validateManifest(manifest({ docsVersion: 'v0.11.0', releaseEvidence: [releaseEvidence()] }));
  assert.equal(evidenced.docsVersion, 'v0.11.0');
});

test('validates releaseEvidence metadata and its fixed release URL', () => {
  expectError(
    () => validateManifest(manifest({ releaseEvidence: [releaseEvidence({ url: 'https://example.com/releases/tag/v0.11.0' })] })),
    'invalid-release-evidence',
    'releaseEvidence[0].url',
  );
  expectError(
    () => validateManifest(manifest({ releaseEvidence: [releaseEvidence({ commit: 'main' })] })),
    'invalid-release-evidence',
    'releaseEvidence[0].commit',
  );
  expectError(
    () => validateManifest(manifest({ releaseEvidence: [releaseEvidence({ observedAt: '14/11/2026' })] })),
    'invalid-release-evidence',
    'releaseEvidence[0].observedAt',
  );
  expectError(
    () => validateManifest(manifest({ releaseEvidence: [releaseEvidence({ sha256: 'ABC' })] })),
    'invalid-release-evidence',
    'releaseEvidence[0].sha256',
  );
  expectError(
    () => validateManifest(manifest({ releaseEvidence: [releaseEvidence({ upstream: 'latest' })] })),
    'unknown-field',
    'releaseEvidence[0].upstream',
  );
  expectError(
    () => validateManifest(manifest({ releaseEvidence: [releaseEvidence(), releaseEvidence()] })),
    'duplicate-release-evidence',
    'releaseEvidence[1].tag',
  );
});

test('rejects an unknown page field', () => {
  expectError(
    () => validateManifest(manifest({ pages: [page({ navigation: 'top' })] })),
    'unknown-field',
    'pages[0].navigation',
  );
});

test('rejects missing or malformed page required fields', () => {
  for (const field of ['id', 'source', 'type', 'audience', 'class', 'sources', 'reviewDigest', 'appliesTo']) {
    const bad = page();
    delete bad[field];
    expectError(() => validateManifest(manifest({ pages: [bad] })), 'missing-field', `pages[0].${field}`);
  }

  const unreviewedWithDate = page({ reviewStatus: 'unreviewed' });
  expectError(
    () => validateManifest(manifest({ pages: [unreviewedWithDate] })),
    'invalid-review-status',
    'pages[0].reviewedAt',
  );

  const reviewedWithoutDate = page({ reviewStatus: 'reviewed' });
  delete reviewedWithoutDate.reviewedAt;
  expectError(
    () => validateManifest(manifest({ pages: [reviewedWithoutDate] })),
    'invalid-review-status',
    'pages[0].reviewedAt',
  );

  for (const id of ['', 'Index', 'guides/Install', 'guides/install.md', 'guides//install', '/guides', 'guides/', '..', 'guides/../x', 'guides/index', 'a--b', 'a_b']) {
    expectError(() => validateManifest(manifest({ pages: [page({ id })] })), 'invalid-id', 'pages[0].id');
  }

  for (const type of ['guide', 'Tutorial', '']) {
    expectError(() => validateManifest(manifest({ pages: [page({ type })] })), 'invalid-type', 'pages[0].type');
  }

  for (const audience of [[], [''], [1], 'maintainer']) {
    assert.throws(
      () => validateManifest(manifest({ pages: [page({ audience })] })),
      (error) => error instanceof ManifestError && error.path.startsWith('pages[0].audience'),
    );
  }

  for (const audience of [['maintainer', 'maintainer']]) {
    assert.throws(
      () => validateManifest(manifest({ pages: [page({ audience })] })),
      (error) => error instanceof ManifestError && error.path.startsWith('pages[0].audience'),
    );
  }

  for (const klass of ['l', 'X', '']) {
    expectError(() => validateManifest(manifest({ pages: [page({ class: klass })] })), 'invalid-class', 'pages[0].class');
  }

  for (const appliesTo of ['', 'latest', 'main', 'v0.11.0']) {
    expectError(
      () => validateManifest(manifest({ pages: [page({ appliesTo })] })),
      'invalid-applies-to',
      'pages[0].appliesTo',
    );
  }

  for (const reviewedAt of ['', '2026-13-01', '2026-02-30', '14-11-2026', '2026-11-14T00:00:00Z']) {
    expectError(
      () => validateManifest(manifest({ pages: [page({ reviewedAt })] })),
      'invalid-reviewed-at',
      'pages[0].reviewedAt',
    );
  }
});

test('accepts appliesTo only for unreleased or an evidenced tag', () => {
  const evidenced = validateManifest(
    manifest({
      releaseEvidence: [releaseEvidence()],
      pages: [page({ appliesTo: 'v0.11.0' })],
    }),
  );
  assert.equal(evidenced.pages[0].appliesTo, 'v0.11.0');
});

test('accepts optional reviewedCommit and rejects a malformed one', () => {
  assert.equal(
    validateManifest(manifest({ pages: [page({ reviewedCommit: '2fc31e9' })] })).pages[0].reviewedCommit,
    '2fc31e9',
  );
  for (const reviewedCommit of ['', 'xyz1234', '2FC31E9', '2fc31e9f9c4d019e2b9f3b4f004978028808c74b00', 1234567]) {
    expectError(
      () => validateManifest(manifest({ pages: [page({ reviewedCommit })] })),
      'invalid-reviewed-commit',
      'pages[0].reviewedCommit',
    );
  }
});

test('rejects page sources outside the public content root or with unsafe paths', () => {
  for (const source of [
    'README.md',
    'website/src/content/docs/',
    'website/src/content/docs/page.mdx',
    'website/src/content/docs/../docs/page.md',
    '/website/src/content/docs/page.md',
    'website\\src\\content\\docs\\page.md',
    'website/src/content/docs//page.md',
    './website/src/content/docs/page.md',
  ]) {
    expectError(
      () => validateManifest(manifest({ pages: [page({ source })] })),
      'invalid-source-path',
      'pages[0].source',
    );
  }
});

test('rejects malformed evidence records', () => {
  for (const sources of [
    [{ path: '../README.md', sha256: SHA_A }],
    [{ path: '/README.md', sha256: SHA_A }],
    [{ path: 'docs\\README.md', sha256: SHA_A }],
    [{ path: 'README.md', sha256: SHA_A.toUpperCase() }],
    [{ path: 'README.md', sha256: SHA_A.slice(0, 32) }],
    [{ path: 'README.md' }],
    [{ sha256: SHA_A }],
    [{ path: 'README.md', sha256: SHA_A, revision: 'main' }],
    ['README.md'],
    'README.md',
  ]) {
    assert.throws(
      () => validateManifest(manifest({ pages: [page({ sources })] })),
      (error) => error instanceof ManifestError && error.path.startsWith('pages[0].sources'),
      `expected refusal for ${JSON.stringify(sources)}`,
    );
  }
});

test('accepts optional symbol and heading evidence annotations but duplicates paths fail', () => {
  const annotated = validateManifest(
    manifest({
      pages: [page({ sources: [{ path: 'README.md', sha256: SHA_A, symbol: 'main', heading: 'Usage' }] })],
    }),
  );
  assert.deepEqual(annotated.pages[0].sources[0], { path: 'README.md', sha256: SHA_A, symbol: 'main', heading: 'Usage' });

  expectError(
    () =>
      validateManifest(
        manifest({ pages: [page({ sources: [{ path: 'README.md', sha256: SHA_A }, { path: 'README.md', sha256: SHA_A }] })] }),
      ),
    'duplicate-evidence-path',
    'pages[0].sources[1].path',
  );
});

test('rejects a stale reviewDigest and a non-SHA digest', () => {
  expectError(
    () => validateManifest(manifest({ pages: [page({ reviewDigest: '0'.repeat(64) })] })),
    'review-digest-mismatch',
    'pages[0].reviewDigest',
  );
  expectError(
    () => validateManifest(manifest({ pages: [page({ reviewDigest: 'not-a-digest' })] })),
    'invalid-review-digest',
    'pages[0].reviewDigest',
  );
});

test('rejects duplicate page ids and duplicate canonical sources', () => {
  expectError(
    () => validateManifest(manifest({ pages: [page(), page({ source: 'website/src/content/docs/other.md' })] })),
    'duplicate-page-id',
    'pages[1].id',
  );
  expectError(
    () => validateManifest(manifest({ pages: [page(), page({ id: 'other' })] })),
    'duplicate-page-source',
    'pages[1].source',
  );
});

test('rejects a malformed pages container', () => {
  for (const pages of [undefined, null, 'index', {}, []]) {
    const input = manifest();
    if (pages === undefined) delete input.pages;
    else input.pages = pages;
    assert.throws(
      () => validateManifest(input),
      (error) => error instanceof ManifestError && error.path === 'pages',
    );
  }
});

test('validates optional root and page list shapes without implementing their semantics', () => {
  const valid = validateManifest(
    manifest({
      aliases: [{ from: '/old', to: '/guides/install' }],
      examples: ['help-version'],
      diagrams: ['D01'],
      pages: [
        page({
          aliases: ['/legacy/page'],
          generatedSections: [{ name: 'cli', producer: 'T02' }],
          examples: ['help-version'],
          impactPaths: ['cmd/'],
          diagrams: ['D01'],
        }),
      ],
    }),
  );
  assert.equal(valid.pages[0].aliases[0], '/legacy/page');

  for (const field of ['aliases', 'examples', 'diagrams']) {
    expectError(() => validateManifest(manifest({ [field]: 'not-a-list' })), 'invalid-list', field);
    expectError(() => validateManifest(manifest({ [field]: [42] })), 'invalid-list', `${field}[0]`);
  }
  for (const field of ['aliases', 'generatedSections', 'examples', 'impactPaths', 'diagrams']) {
    expectError(
      () => validateManifest(manifest({ pages: [page({ [field]: [null] })] })),
      'invalid-list',
      `pages[0].${field}[0]`,
    );
  }
});

test('refuses reserved object keys without polluting prototypes', () => {
  const item = JSON.parse('{"__proto__":{"polluted":true}}');
  expectError(
    () => validateManifest(manifest({ pages: [page({ aliases: [item] })] })),
    'unknown-field',
    'pages[0].aliases[0].__proto__',
  );
  assert.equal({}.polluted, undefined);
});

test('accepts an empty evidence list with the canonical empty digest', () => {
  const validated = validateManifest(
    manifest({ pages: [page({ sources: [], reviewDigest: '4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945' })] }),
  );
  assert.deepEqual(validated.pages[0].sources, []);
});

test('reports reviewDigest arithmetic for the one-source fixture', () => {
  assert.equal(ONE_SOURCE_DIGEST.length, 64);
  assert.ok(validateManifest(manifest()).pages[0].reviewDigest === ONE_SOURCE_DIGEST);
});
