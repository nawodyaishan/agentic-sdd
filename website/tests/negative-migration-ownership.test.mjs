// Spec 020 B4-W1-G — migration stub shape, duplicate ownership and aliases.
//
// Frozen sources: plan C1 (`aliases[]` = old file/URL plus supported fragment
// mapping, "no conflicting destination"; page id/source unique), T08
// verification (stub shape prohibits procedures and code fences; duplicate
// source/canonical/body ownership fails; deleting a retained anchor fails) and
// T09 verification (duplicate URL/source/body). Expectations are written from
// that contract. `checkSource` enforces stub shape, retained anchors, duplicate
// body/source ownership and front-matter review claims; alias destination
// semantics and duplicate canonical URL remain blocked with their owning task.

import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

import {
  blockedOptions,
  errorsWith,
  expectRejected,
  fixture,
  fixtureJson,
  forgedClaims,
  frontmatterKeys,
  loadFrozenApi,
  loadProducer,
  readJson,
  repoRoot,
  requireError,
  tempRepo,
} from './negative-fixtures/lib/harness.mjs';

const api = await loadFrozenApi('website/scripts/lib/manifest.mjs', ['validateManifest', 'computeReviewDigest']);
const checkApi = await loadProducer('check', ['checkSource']);
const skip = api.skip;
const SHIPPED_MANIFEST = path.join(repoRoot, 'website', 'docs-manifest.json');
const STUBS = 'website/tests/negative-fixtures/stubs';

const blocked = {};
function blockedCase(key, producer, absent, unadapted) {
  blocked[key] = blockedOptions(producer, absent, unadapted);
  return blocked[key];
}

/** Minimal schema-valid C1 page for a producer fixture repository. */
function pageEntry(id, source, type) {
  const sources = [{ path: source, sha256: 'a'.repeat(64) }];
  return {
    id,
    source,
    type,
    audience: ['maintainer'],
    class: 'C',
    sources,
    reviewDigest: api.mod.computeReviewDigest(sources),
    appliesTo: 'unreleased',
    reviewStatus: 'unreviewed',
  };
}

/** Minimal schema-valid C1 manifest for a producer fixture repository. */
function devManifest(pages) {
  return {
    schemaVersion: 1,
    site: 'https://example.github.io',
    base: '/agentic-sdd',
    repository: 'https://github.com/example/repo',
    docsVersion: 'dev',
    pages,
  };
}

const sourceCheck = (stubs) => checkApi.mod.checkSource({ repoRoot, stubs });

/**
 * Independent stub-shape probe used only to prove the stub fixtures are not
 * vacuous: a compliant stub has no fence and no procedure, each violating
 * fixture trips exactly the rule its name claims.
 */
function stubShape(text) {
  return {
    fence: /^```/m.test(text),
    procedure: /^\s{0,3}\d+[.)]\s+\S/m.test(text),
  };
}

// ---------------------------------------------------------------------------
// Stub fixtures: the shape rule is data, and the fixtures exercise it
// ---------------------------------------------------------------------------

test('the stub fixtures exercise the shape rule and the compliant stub is clean', {}, () => {
  const compliant = fixture('stubs/compliant.md');
  const compliantShape = stubShape(compliant);
  assert.deepEqual(compliantShape, { fence: false, procedure: false }, 'the compliant stub must be clean');
  assert.ok(frontmatterKeys(compliant).includes('status'), 'a stub declares its status');
  assert.match(compliant, /Canonical page:/, 'a stub keeps a canonical link');
  assert.match(compliant, /Retained anchors:/, 'a stub keeps its retained anchors');

  assert.equal(stubShape(fixture('stubs/code-fence.md')).fence, true, 'the code-fence stub must trip the fence rule');
  assert.equal(stubShape(fixture('stubs/procedure.md')).procedure, true, 'the procedure stub must trip the step rule');
  assert.deepEqual(
    forgedClaims(fixture('stubs/forged-status.md')).sort(),
    ['reviewStatus', 'reviewedAt'],
    'the forged-status stub must carry a detectable review claim',
  );
});

// ---------------------------------------------------------------------------
// Duplicate ownership inside the frozen registry
// ---------------------------------------------------------------------------

test('the fixture registries declare unique page ids and unique canonical sources', {}, () => {
  // Control: the mutation cases below must fail for the injected duplicate,
  // not because the baseline registry was already inconsistent.
  const baseline = fixtureJson(path.join('manifests', 'baseline.json'));
  assert.equal(new Set(baseline.pages.map((page) => page.id)).size, baseline.pages.length);
  assert.equal(new Set(baseline.pages.map((page) => page.source)).size, baseline.pages.length);
});

expectRejected(api, 'a duplicate page id', (m) => {
  m.pages[2].id = m.pages[1].id;
}, { pathMentions: /\bid\b/i });

expectRejected(api, 'a duplicate canonical page source', (m) => {
  m.pages[2].source = m.pages[1].source;
}, { pathMentions: /source/i });

expectRejected(api, 'a duplicate evidence path inside one page', (m) => {
  m.pages[0].sources.push({ ...m.pages[0].sources[0] });
}, { pathMentions: /sources\[1\]\.path/ });

test('the duplicate-ownership fixtures name the duplicate they claim', {}, () => {
  const url = fixtureJson('ownership/duplicate-url.json');
  const urls = url.pages.map((page) => page.canonicalUrl);
  assert.equal(new Set(urls).size, 1, 'duplicate-url.json must actually duplicate a canonical URL');
  assert.equal(new Set(url.pages.map((page) => page.source)).size, url.pages.length, 'the sources stay distinct');

  const body = fixtureJson('ownership/duplicate-body.json');
  const bodies = body.pages.map((page) => page.bodySha256);
  assert.equal(new Set(bodies).size, 1, 'duplicate-body.json must actually duplicate a body');
  assert.notEqual(body.pages[0].id, body.pages[1].id, 'the duplicate body must be owned by two distinct pages');
});

// ---------------------------------------------------------------------------
// Aliases
// ---------------------------------------------------------------------------

test('the alias fixtures cover a resolved, a dangling and a conflicting alias', {}, () => {
  const valid = fixtureJson('aliases/valid.json');
  const canonical = new Set(valid.canonicalPageIds);
  assert.ok(valid.aliases.every((alias) => canonical.has(alias.to)), 'valid.json must resolve');

  const dangling = fixtureJson('aliases/dangling.json');
  assert.ok(
    dangling.aliases.some((alias) => !new Set(dangling.canonicalPageIds).has(alias.to)),
    'dangling.json must contain an alias whose target is not a canonical page',
  );

  const conflict = fixtureJson('aliases/conflict.json');
  const from = conflict.aliases.map((alias) => alias.from);
  assert.notEqual(new Set(from).size, from.length, 'conflict.json must reuse one old path for two destinations');
});

test('every declared alias resolves to a declared canonical page id (none declared yet)', { skip }, () => {
  const manifest = readJson(SHIPPED_MANIFEST);
  const ids = new Set(manifest.pages.map((page) => page.id));
  assert.ok(Array.isArray(manifest.pages) && manifest.pages.length > 0, 'the shipped manifest must list pages');
  const aliases = manifest.aliases ?? [];
  assert.ok(Array.isArray(aliases), 'aliases must be an array when present');
  // T08 owns the alias table; zero declared aliases means this guard has
  // nothing to resolve yet. It becomes real the moment an alias ships.
  for (const alias of aliases) {
    assert.ok(ids.has(alias.to), `${alias.from} -> ${alias.to} must name a declared canonical page id`);
  }
});

// ---------------------------------------------------------------------------
// Producer-dependent enforcement
// ---------------------------------------------------------------------------

test('a migration stub containing a code fence is rejected', { skip: checkApi.skip }, () => {
  const stubPath = `${STUBS}/code-fence.md`;
  const result = sourceCheck([`${stubPath}=reference/cli`]);
  const [error] = requireError(result, 'stub/code-fence', stubPath);
  assert.equal(error.severity, 'error');
  assert.equal(error.line, 8, 'the fence line must be identified');
  assert.match(error.message, /code fence/i);
  assert.match(error.fix, /remove the code fence/i);
  assert.equal(result.ok, false);

  const control = sourceCheck([`${STUBS}/compliant.md=reference/cli`]);
  assert.deepEqual(errorsWith(control, 'stub/code-fence'), [], 'the compliant stub must not trip the fence rule');
  assert.deepEqual(errorsWith(control, 'stub/procedure'), [], 'the compliant stub must not trip the procedure rule');
});

test('a migration stub containing a procedure is rejected', { skip: checkApi.skip }, () => {
  const stubPath = `${STUBS}/procedure.md`;
  const result = sourceCheck([`${stubPath}=getting-started/installation`]);
  const steps = requireError(result, 'stub/procedure', stubPath);
  assert.deepEqual(
    steps.map((entry) => entry.line).sort((left, right) => left - right),
    [8, 9, 10],
    'every numbered procedure step must be reported',
  );
  for (const entry of steps) assert.match(entry.message, /numbered procedure step/i);
  assert.equal(result.ok, false);
});

test('a migration stub claiming a human review is rejected', { skip: checkApi.skip }, () => {
  const stubPath = `${STUBS}/forged-status.md`;
  const result = sourceCheck([`${stubPath}=reference/cli`]);
  const claims = errorsWith(result, 'frontmatter/forged-claim', stubPath);
  assert.deepEqual(
    claims.map((entry) => entry.details.key).sort(),
    ['reviewStatus', 'reviewedAt'],
    'both forged stub review keys must be reported',
  );
  for (const claim of claims) {
    assert.equal(claim.severity, 'error');
    assert.ok(claim.line > 1, 'the diagnostic must point inside the stub front matter');
    assert.match(claim.fix, /docs-manifest\.json|approval record/);
  }
  assert.equal(result.ok, false);

  const control = sourceCheck([`${STUBS}/compliant.md=reference/cli`]);
  assert.deepEqual(
    errorsWith(control, 'frontmatter/forged-claim'),
    [],
    'the compliant stub must not carry a review or approval claim',
  );
});

test('an alias pointing at a non-existent page is rejected', blockedCase(
  'alias-dangling',
  'check',
  'alias resolution is not present in this worktree',
  'check.mjs compares alias entries only for exact duplication and alias/source or alias/URL collisions; C1 alias destination resolution (dangling-target detection) is not implemented (owner: T08 alias table/C1)',
), () => {
  // Fixture: aliases/dangling.json.
});

test('two aliases sending one old path to different pages are rejected', blockedCase(
  'alias-conflict',
  'check',
  'alias conflict detection is not present in this worktree',
  'check.mjs does not compare alias destinations: two entries with the same "from" and different "to" are not detected, and the alias schema itself is owned by T08/C1',
), () => {
  // Fixture: aliases/conflict.json.
});

test('two pages owning one canonical URL are rejected', blockedCase(
  'duplicate-url',
  'check',
  'canonical URL ownership checking is not present in this worktree',
  'canonical URLs are derived from unique page ids; validateManifest rejects duplicate ids and the page schema has no canonicalUrl field, so the fixture cannot express two pages owning one URL (owner: T08 URL/alias ownership; the check.mjs ownership/duplicate-url branch is unreachable for a valid manifest)',
), () => {
  // Fixture: ownership/duplicate-url.json.
});

test('two pages owning one body are rejected', { skip: checkApi.skip }, () => {
  // The fixture is a digest sketch, not a manifest: the producer compares the
  // parsed bodies of real contained sources, so the two fixture pages are
  // materialised with one identical body in an owned temporary repository.
  const [first, second] = fixtureJson('ownership/duplicate-body.json').pages;
  const sharedBody = 'Duplicate installation body.\n\n## Steps\n\n1. One shared procedure.\n';
  const repo = tempRepo({
    [first.source]: sharedBody,
    [second.source]: sharedBody,
  });
  try {
    const manifestPath = path.join(repo.root, 'website', 'docs-manifest.json');
    const manifest = devManifest([
      pageEntry(first.id, first.source, 'how-to'),
      pageEntry(second.id, second.source, 'reference'),
    ]);
    fs.writeFileSync(manifestPath, JSON.stringify(manifest, null, 2));

    const result = checkApi.mod.checkSource({ repoRoot: repo.root });
    assert.deepEqual(
      errorsWith(result, 'source/unreadable'),
      [],
      'the fixture pages must be readable, otherwise the body check would be vacuous',
    );
    const duplicates = requireError(result, 'ownership/duplicate-body');
    assert.equal(duplicates.length, 1, 'one duplicated body must fail once');
    const [duplicate] = duplicates;
    assert.ok(
      [first.source, second.source].includes(duplicate.path),
      `the duplicate must be reported against one of the two page sources, not ${duplicate.path}`,
    );
    const other = duplicate.path === first.source ? second.source : first.source;
    assert.ok(duplicate.message.includes(other), 'the diagnostic must name the page that already owns the body');
    assert.equal(result.ok, false);
  } finally {
    repo.cleanup();
  }
});

test('deleting a retained alias anchor fails with the path and slug', { skip: checkApi.skip }, () => {
  const stubPath = `${STUBS}/compliant.md`;
  const result = sourceCheck([`${stubPath}=reference/cli:install-linux`]);
  const [missing] = requireError(result, 'stub/missing-anchor', stubPath);
  assert.equal(missing.details.anchor, 'install-linux', 'the diagnostic must name the deleted slug');
  assert.match(missing.message, /#install-linux/);
  assert.match(missing.fix, /restore a heading/i);
  assert.equal(result.ok, false);

  const control = sourceCheck([`${stubPath}=reference/cli`]);
  assert.deepEqual(
    errorsWith(control, 'stub/missing-anchor'),
    [],
    'a stub with no retained-anchor declaration has nothing to fail',
  );
});

