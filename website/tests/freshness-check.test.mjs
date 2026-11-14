// SPEC020 B4-W1-D — reviewed-fingerprint freshness. A changed reviewed input
// fails its affected page until a human updates content or intentionally
// refreshes the fingerprint; unrelated pre-existing drift is exactly one
// observation, never a cascade, and the checker never writes the manifest.
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdtempSync, mkdirSync, readFileSync, rmSync, unlinkSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';

import { computeReviewDigest } from '../scripts/lib/manifest.mjs';
import { checkDocumentation, checkFreshness } from '../scripts/check.mjs';

const DOCS = 'website/src/content/docs';

function sha256(value) {
  return createHash('sha256').update(value).digest('hex');
}

function makeRepo(t, pages) {
  mkdirSync('/tmp/opencode', { recursive: true });
  const repoRoot = mkdtempSync('/tmp/opencode/spec020-b4-fresh-');
  t.after(() => rmSync(repoRoot, { recursive: true, force: true }));
  const manifest = {
    schemaVersion: 1,
    site: 'https://docs.example.test',
    base: '/agentic-sdd',
    repository: 'https://github.com/example/repo',
    docsVersion: 'dev',
    pages: [],
  };
  for (const page of pages) {
    const absolute = path.join(repoRoot, page.source);
    mkdirSync(path.dirname(absolute), { recursive: true });
    writeFileSync(absolute, page.text, 'utf8');
    const evidence = page.sources ?? [{ path: page.source, sha256: sha256(page.text) }];
    manifest.pages.push({
      id: page.id,
      source: page.source,
      type: 'reference',
      audience: ['maintainer'],
      class: 'L',
      sources: evidence,
      reviewDigest: computeReviewDigest(evidence),
      appliesTo: 'unreleased',
      reviewStatus: page.reviewStatus ?? 'unreviewed',
      ...(page.reviewedAt === undefined ? {} : { reviewedAt: page.reviewedAt }),
      ...(page.reviewedCommit === undefined ? {} : { reviewedCommit: page.reviewedCommit }),
    });
  }
  const manifestPath = path.join(repoRoot, 'website/docs-manifest.json');
  mkdirSync(path.dirname(manifestPath), { recursive: true });
  writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`, 'utf8');
  return { repoRoot, manifestPath };
}

function page(id, text = `# ${id}\n\nBody for ${id}.\n`) {
  return { id, source: `${DOCS}/${id}.md`, text };
}

function codes(result) {
  return result.errors.map((entry) => entry.code).sort();
}

test('fails an affected page when a reviewed fingerprint changes', (t) => {
  const { repoRoot, manifestPath } = makeRepo(t, [page('a')]);
  const before = readFileSync(manifestPath, 'utf8');
  const sourcePath = path.join(repoRoot, `${DOCS}/a.md`);
  writeFileSync(sourcePath, `${readFileSync(sourcePath, 'utf8')}Changed after review.\n`, 'utf8');

  const result = checkFreshness({ repoRoot });
  assert.equal(result.ok, false);
  const error = result.errors.find((entry) => entry.code === 'freshness/needs-review');
  assert.ok(error, `expected freshness/needs-review, got ${JSON.stringify(codes(result))}`);
  assert.equal(error.path, `${DOCS}/a.md`);
  assert.equal(error.line, 1);
  assert.match(error.message, /a\.md/);
  assert.match(error.fix, /reviewDigest/);
  assert.match(error.fix, /never refreshes|Never refresh/i);
  assert.match(error.details.recommendedReviewDigest, /^[0-9a-f]{64}$/);
  assert.equal(readFileSync(manifestPath, 'utf8'), before, 'freshness must never rewrite the manifest');
});

test('fails a page whose sources[] omits its own canonical source', (t) => {
  const bText = '# b\n\nBody for b.\n';
  const { repoRoot, manifestPath } = makeRepo(t, [page('a'), page('b')]);
  const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
  const entryA = manifest.pages.find((entry) => entry.id === 'a');
  entryA.sources = [{ path: `${DOCS}/b.md`, sha256: sha256(bText) }];
  entryA.reviewDigest = computeReviewDigest(entryA.sources);
  writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`, 'utf8');

  const unmodified = checkFreshness({ repoRoot });
  const error = unmodified.errors.find((entry) => entry.code === 'freshness/unregistered-source');
  assert.ok(error, `expected freshness/unregistered-source, got ${JSON.stringify(codes(unmodified))}`);
  assert.equal(error.path, `${DOCS}/a.md`);
  assert.equal(error.details.pageId, 'a');
  assert.match(error.message, /sources\[\]/);
  assert.match(error.fix, /reviewDigest/);
  assert.equal(
    unmodified.errors.some((entry) => entry.path === `${DOCS}/a.md` && entry.code === 'freshness/needs-review'),
    false,
    'an unregistered page must fail for the missing registration, not as ordinary drift',
  );

  // Regression: the body change used to pass because the body was never hashed.
  writeFileSync(path.join(repoRoot, `${DOCS}/a.md`), '# a\n\nChanged body after the omitted registration.\n', 'utf8');
  const changed = checkFreshness({ repoRoot, changedPaths: [`${DOCS}/a.md`] });
  assert.equal(changed.ok, false);
  assert.ok(codes(changed).includes('freshness/unregistered-source'), JSON.stringify(codes(changed)));
  assert.equal(
    changed.errors.some((entry) => entry.code === 'freshness/needs-review'),
    false,
    'the omitted self source must not be reported as a reviewable fingerprint change',
  );
});

test('treats stale input outside the change set as exactly one observation', (t) => {
  const { repoRoot } = makeRepo(t, [page('a'), page('b')]);
  const bPath = path.join(repoRoot, `${DOCS}/b.md`);
  writeFileSync(bPath, `${readFileSync(bPath, 'utf8')}Unrelated stale edit.\n`, 'utf8');

  const result = checkFreshness({ repoRoot, changedPaths: [`${DOCS}/a.md`] });
  assert.equal(result.ok, true, JSON.stringify(result.errors));
  assert.equal(result.errors.length, 0);
  assert.equal(result.observations.length, 1);
  const observation = result.observations[0];
  assert.equal(observation.code, 'freshness/unrelated-stale');
  assert.equal(observation.details.pages.length, 1);
  assert.equal(observation.details.pages[0].pageId, 'b');
  assert.match(observation.fix, /one consolidated observation/);
});

test('consolidates multiple unrelated stale pages into one observation', (t) => {
  const { repoRoot } = makeRepo(t, [page('a'), page('b'), page('c')]);
  for (const id of ['b', 'c']) {
    const sourcePath = path.join(repoRoot, `${DOCS}/${id}.md`);
    writeFileSync(sourcePath, `${readFileSync(sourcePath, 'utf8')}Drift.\n`, 'utf8');
  }
  const result = checkFreshness({ repoRoot, changedPaths: [`${DOCS}/a.md`] });
  assert.equal(result.ok, true);
  assert.equal(result.observations.length, 1);
  assert.deepEqual(
    result.observations[0].details.pages.map((entry) => entry.pageId).sort(),
    ['b', 'c'],
  );
});

test('fails the page when the changed input is in the current change set', (t) => {
  const { repoRoot } = makeRepo(t, [page('a'), page('b')]);
  const bPath = path.join(repoRoot, `${DOCS}/b.md`);
  writeFileSync(bPath, `${readFileSync(bPath, 'utf8')}Affected edit.\n`, 'utf8');

  const result = checkFreshness({ repoRoot, changedPaths: [`${DOCS}/b.md`] });
  assert.equal(result.ok, false);
  assert.ok(codes(result).includes('freshness/needs-review'), JSON.stringify(codes(result)));
  assert.equal(result.observations.length, 0);
});

test('a changed directory prefix marks contained pages affected', (t) => {
  const { repoRoot } = makeRepo(t, [page('a'), page('nested/b', '# nested\n\nBody.\n')]);
  const bPath = path.join(repoRoot, `${DOCS}/nested/b.md`);
  writeFileSync(bPath, `${readFileSync(bPath, 'utf8')}Affected by directory.\n`, 'utf8');

  const result = checkFreshness({ repoRoot, changedPaths: [`${DOCS}/nested/`] });
  const error = result.errors.find((entry) => entry.code === 'freshness/needs-review');
  assert.ok(error, `expected freshness/needs-review, got ${JSON.stringify(codes(result))}`);
  assert.match(error.message, /nested\/b/);
});

test('a manifest change set fails every stale page', (t) => {
  const { repoRoot } = makeRepo(t, [page('a')]);
  const sourcePath = path.join(repoRoot, `${DOCS}/a.md`);
  writeFileSync(sourcePath, `${readFileSync(sourcePath, 'utf8')}Drift.\n`, 'utf8');
  const result = checkFreshness({ repoRoot, changedPaths: ['website/docs-manifest.json'] });
  assert.equal(result.ok, false);
  assert.equal(result.observations.length, 0);
});

test('names the reviewed status in the diagnostic', (t) => {
  const { repoRoot } = makeRepo(t, [
    {
      ...page('a'),
      reviewStatus: 'reviewed',
      reviewedAt: '2026-11-14',
      reviewedCommit: '449245a990225f2234685dee8314dff1c3b1d3b2',
    },
  ]);
  const sourcePath = path.join(repoRoot, `${DOCS}/a.md`);
  writeFileSync(sourcePath, `${readFileSync(sourcePath, 'utf8')}Post-review edit.\n`, 'utf8');
  const result = checkFreshness({ repoRoot });
  const error = result.errors.find((entry) => entry.code === 'freshness/needs-review');
  assert.ok(error);
  assert.match(error.message, /reviewed page/);
  assert.equal(error.details.reviewStatus, 'reviewed');
});

test('reports a missing reviewed input at the manifest with a fix', (t) => {
  const { repoRoot } = makeRepo(t, [page('a')]);
  unlinkSync(path.join(repoRoot, `${DOCS}/a.md`));
  const result = checkFreshness({ repoRoot });
  const error = result.errors.find((entry) => entry.code === 'freshness/missing-input');
  assert.ok(error, `expected freshness/missing-input, got ${JSON.stringify(codes(result))}`);
  assert.equal(error.path, 'website/docs-manifest.json');
  assert.ok(error.line > 0);
  assert.match(error.fix, /Restore/);
});

test('is deterministic and read-only across repeated runs', (t) => {
  const { repoRoot, manifestPath } = makeRepo(t, [page('a'), page('b')]);
  const bPath = path.join(repoRoot, `${DOCS}/b.md`);
  writeFileSync(bPath, `${readFileSync(bPath, 'utf8')}Drift.\n`, 'utf8');
  const before = readFileSync(manifestPath, 'utf8');
  const first = checkFreshness({ repoRoot, changedPaths: [`${DOCS}/a.md`] });
  const second = checkFreshness({ repoRoot, changedPaths: [`${DOCS}/a.md`] });
  assert.equal(JSON.stringify(first), JSON.stringify(second));
  assert.equal(readFileSync(manifestPath, 'utf8'), before);
});

test('checkDocumentation combines freshness with link checks', (t) => {
  const { repoRoot } = makeRepo(t, [page('a')]);
  const sourcePath = path.join(repoRoot, `${DOCS}/a.md`);
  writeFileSync(sourcePath, `${readFileSync(sourcePath, 'utf8')}\n[broken](./missing.md)\n`, 'utf8');
  const result = checkDocumentation({ repoRoot });
  assert.ok(codes(result).includes('freshness/needs-review'), JSON.stringify(codes(result)));
  assert.ok(codes(result).includes('links/unknown-target'), JSON.stringify(codes(result)));
});
