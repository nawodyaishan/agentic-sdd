// C1 review fingerprint: path-sorted compact-JSON SHA-256, never an automatic
// stamp. Known digests are precomputed from the frozen array shape.

import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import test from 'node:test';

import { ManifestError, computeReviewDigest, validateManifest } from '../scripts/lib/manifest.mjs';
import { EMPTY_DIGEST, ONE_SOURCE_DIGEST, SHA_A, SHA_B, TWO_SOURCE_DIGEST, manifest, page } from './manifest-fixtures.mjs';

test('computes the frozen digest vectors', () => {
  assert.equal(computeReviewDigest([]), EMPTY_DIGEST);
  assert.equal(computeReviewDigest([{ path: 'README.md', sha256: SHA_A }]), ONE_SOURCE_DIGEST);
  assert.equal(
    computeReviewDigest([
      { path: 'README.md', sha256: SHA_A },
      { path: 'docs/getting-started.md', sha256: SHA_B },
    ]),
    TWO_SOURCE_DIGEST,
  );
});

test('sorts by normalized path so input order does not change the digest', () => {
  const forward = [
    { path: 'README.md', sha256: SHA_A },
    { path: 'docs/getting-started.md', sha256: SHA_B },
  ];
  assert.equal(computeReviewDigest(forward), TWO_SOURCE_DIGEST);
  assert.equal(computeReviewDigest([...forward].reverse()), TWO_SOURCE_DIGEST);
});

test('sorts by code units, not locale collation', () => {
  const expected = createHash('sha256')
    .update(
      JSON.stringify([
        { path: 'Z.md', sha256: SHA_A },
        { path: 'a.md', sha256: SHA_A },
      ]),
      'utf8',
    )
    .digest('hex');
  const reverse = [
    { path: 'a.md', sha256: SHA_A },
    { path: 'Z.md', sha256: SHA_A },
  ];
  assert.equal(computeReviewDigest(reverse), expected);
});

test('ignores optional evidence annotations that are not part of the digest', () => {
  const plain = computeReviewDigest([{ path: 'README.md', sha256: SHA_A }]);
  const annotated = computeReviewDigest([{ path: 'README.md', sha256: SHA_A, symbol: 'main' }]);
  assert.equal(annotated, plain);
});

test('does not mutate the supplied source list', () => {
  const sources = [
    { path: 'docs/getting-started.md', sha256: SHA_B },
    { path: 'README.md', sha256: SHA_A },
  ];
  const before = structuredClone(sources);
  computeReviewDigest(sources);
  assert.deepEqual(sources, before);
});

test('refuses malformed digest inputs', () => {
  const cases = [
    ['not-an-array', 'invalid-evidence-list'],
    [[{ path: '../README.md', sha256: SHA_A }], 'invalid-evidence'],
    [[{ path: 'README.md', sha256: SHA_A.toUpperCase() }], 'invalid-evidence'],
    [[{ path: 'README.md', sha256: 'abc' }], 'invalid-evidence'],
    [[{ path: 'README.md' }], 'invalid-evidence'],
    [[{ sha256: SHA_A }], 'invalid-evidence'],
    [[{ path: 'README.md', sha256: SHA_A }, { path: 'README.md', sha256: SHA_B }], 'duplicate-evidence-path'],
  ];
  for (const [value, code] of cases) {
    assert.throws(
      () => computeReviewDigest(value),
      (error) => error instanceof ManifestError && error.code === code,
      `expected ${code} for ${JSON.stringify(value)}`,
    );
  }
});

test('a changed source hash invalidates the stored fingerprint until a human refreshes it', () => {
  const stale = page({ sources: [{ path: 'README.md', sha256: SHA_B }] });
  assert.throws(
    () => validateManifest(manifest({ pages: [stale] })),
    (error) => error instanceof ManifestError && error.code === 'review-digest-mismatch',
  );

  const refreshed = { ...stale, reviewDigest: computeReviewDigest(stale.sources) };
  assert.notEqual(refreshed.reviewDigest, stale.reviewDigest);
  assert.equal(validateManifest(manifest({ pages: [refreshed] })).pages[0].reviewDigest, refreshed.reviewDigest);
});

test('validation never rewrites the supplied fingerprint', () => {
  const input = manifest();
  const before = structuredClone(input);
  const result = validateManifest(input);
  assert.equal(result.pages[0].reviewDigest, before.pages[0].reviewDigest);
  assert.deepEqual(input, before);
});
