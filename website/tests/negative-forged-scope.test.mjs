// Spec 020 B4-W1-G — forged review status and fabricated approval records.
//
// Requirement: pages are registered `reviewStatus: unreviewed` with no
// `reviewedAt`, and no page, registry or fixture may claim a human review or
// approval that did not happen (plan C1 review identity; B4 contract "Never
// claim human review or acceptance"). Expectations come from that frozen
// contract, not from the validator source. A schema-valid `reviewed` + date
// pair cannot be distinguished by a registry validator, so the shipped corpus
// is checked directly as well as the rejection cases being mutated. Page front
// matter is also rejected by the documentation check, so a forged claim cannot
// hide in content while the registry stays unreviewed.

import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

import {
  BASELINE_MANIFEST,
  FORGED_CLAIM_KEYS,
  PUBLIC_DOCS,
  errorsWith,
  expectRejected,
  fixture,
  fixtureJson,
  forgedClaims,
  frontmatterKeys,
  loadFrozenApi,
  loadProducer,
  readJson,
  rel,
  repoRoot,
  tempRepo,
  walkFiles,
} from './negative-fixtures/lib/harness.mjs';

const api = await loadFrozenApi('website/scripts/lib/manifest.mjs', ['validateManifest', 'computeReviewDigest']);
const exportApi = await loadProducer('export', ['buildExport']);
const checkApi = await loadProducer('check', ['checkSource']);

const skip = api.skip;
const M = api.mod;
const SHIPPED_MANIFEST = path.join(repoRoot, 'website', 'docs-manifest.json');

// ---------------------------------------------------------------------------
// Shipped corpus: the frozen review vocabulary
// ---------------------------------------------------------------------------

test('the shipped manifest registers every page unreviewed with no human review date', { skip }, () => {
  const manifest = readJson(SHIPPED_MANIFEST);
  assert.ok(Array.isArray(manifest.pages) && manifest.pages.length > 0, 'the shipped manifest must list pages');
  for (const page of manifest.pages) {
    assert.equal(
      page.reviewStatus,
      'unreviewed',
      `${page.id}: B4 registers pages as unreviewed; update this expectation only when an actual human review is recorded in the feature approval record`,
    );
    assert.equal(page.reviewedAt, undefined, `${page.id}: an unreviewed page must not carry a human review date`);
    assert.notEqual(page.reviewStatus, 'reviewed', `${page.id}: no page may claim human review in B4`);
  }
});

test('the shipped manifest is the validated control for these mutations', { skip }, () => {
  const manifest = readJson(SHIPPED_MANIFEST);
  const validated = M.validateManifest(manifest, { repoRoot, verifySources: true });
  assert.equal(validated.pages.length, manifest.pages.length, 'the control manifest must validate without loss');
});

test('no public page front matter carries a forged review or approval claim', {}, () => {
  // Public pages (the site content root) plus the top-level docs guides that
  // T08 may turn into compatibility stubs. Engineering reports and handovers
  // are not public pages and are out of scope for this claim scan.
  const files = [
    ...walkFiles(path.join(repoRoot, PUBLIC_DOCS)).map((entry) => entry.path),
    ...fs.readdirSync(path.join(repoRoot, 'docs'))
      .filter((name) => name.endsWith('.md'))
      .map((name) => path.join(repoRoot, 'docs', name)),
  ];
  let parsedFrontmatter = 0;
  const violations = [];
  for (const file of files) {
    const text = fs.readFileSync(file, 'utf8');
    const keys = frontmatterKeys(text);
    if (keys.length > 0) parsedFrontmatter += 1;
    const forged = forgedClaims(text);
    if (forged.length > 0) violations.push(`${rel(file)}: ${forged.join(', ')}`);
  }
  assert.deepEqual(violations, [], `forged review/approval front matter is not accepted anywhere:\n${violations.join('\n')}`);
  assert.ok(
    parsedFrontmatter > 0,
    'the scanner must actually parse front matter; otherwise this check would pass vacuously',
  );
});

test('the claim scanner separates forged fixtures from compliant ones', {}, () => {
  const forgedReview = forgedClaims(fixture('pages/forged-review.md'));
  assert.deepEqual([...forgedReview].sort(), ['reviewStatus', 'reviewedAt'].sort(), 'a claimed review must be detected');
  const forgedApproval = forgedClaims(fixture('pages/forged-approval.md'));
  assert.deepEqual([...forgedApproval].sort(), ['approval', 'approvedBy'].sort(), 'a fabricated approval record must be detected');
  assert.deepEqual(forgedClaims(fixture('pages/compliant.md')), [], 'a compliant page carries no claim');
  // The scanner vocabulary is data, so a renamed claim field is a deliberate
  // maintainer decision rather than a silent gap.
  assert.ok(FORGED_CLAIM_KEYS.includes('reviewStatus'));
});

test('the fixture registries carry no forged review claim of their own', {}, () => {
  const names = fs.readdirSync(path.dirname(BASELINE_MANIFEST)).filter((name) => name.endsWith('.json'));
  assert.ok(names.length > 0, 'at least one fixture registry is required');
  for (const name of names) {
    const manifest = fixtureJson(path.join('manifests', name));
    for (const page of manifest.pages ?? []) {
      assert.notEqual(page.reviewStatus, 'reviewed', `${name}/${page.id}: fixture registries must stay unreviewed`);
      assert.equal(page.reviewedAt, undefined, `${name}/${page.id}: fixture registries must not claim a review date`);
    }
    for (const key of ['approval', 'approvals', 'approvedBy']) {
      assert.equal(manifest[key], undefined, `${name}: fixtures must not embed an approval record`);
    }
  }
});

// ---------------------------------------------------------------------------
// Mutations: schema-level forgeries must fail closed
// ---------------------------------------------------------------------------

expectRejected(api, 'a forged reviewed status without a review date', (m) => {
  delete m.pages[0].reviewedAt;
  m.pages[0].reviewStatus = 'reviewed';
}, { pathMentions: /reviewedAt|review/i });

expectRejected(api, 'an unreviewed page carrying a human review date', (m) => {
  m.pages[0].reviewStatus = 'unreviewed';
  m.pages[0].reviewedAt = '2026-11-14';
}, { pathMentions: /reviewedAt|unreviewed/i });

expectRejected(api, 'a fabricated root-level approval record', (m) => {
  m.approval = { by: 'repository owner', at: '2026-11-14', status: 'accepted' };
}, { pathMentions: /approval/i });

expectRejected(api, 'a fabricated page-level approver field', (m) => {
  m.pages[0].approvedBy = 'repository owner';
}, { pathMentions: /approvedBy/i });

expectRejected(api, 'a fabricated page-level reviewer field', (m) => {
  m.pages[0].reviewedBy = 'repository owner';
}, { pathMentions: /reviewedBy/i });

// ---------------------------------------------------------------------------
// Producer-dependent enforcement
// ---------------------------------------------------------------------------

test('a page body presenting a fabricated review claim is rejected by the documentation check', { skip: checkApi.skip }, () => {
  const forgedReview = fixture('pages/forged-review.md');
  const forgedApproval = fixture('pages/forged-approval.md');
  const compliant = fixture('pages/compliant.md');
  const quotedClaim = '---\ntitle: Quoted claim\n"humanReview": "accepted"\n---\n\nQuoted claim body.\n';
  const forgedSource = `${PUBLIC_DOCS}/forged-review.md`;
  const approvalSource = `${PUBLIC_DOCS}/forged-approval.md`;
  const compliantSource = `${PUBLIC_DOCS}/compliant.md`;
  const quotedSource = `${PUBLIC_DOCS}/quoted-claim.md`;
  const repo = tempRepo({
    [forgedSource]: forgedReview,
    [approvalSource]: forgedApproval,
    [compliantSource]: compliant,
    [quotedSource]: quotedClaim,
  });
  try {
    const specs = [
      { id: 'forged-review', source: forgedSource, text: forgedReview },
      { id: 'forged-approval', source: approvalSource, text: forgedApproval },
      { id: 'compliant', source: compliantSource, text: compliant },
      { id: 'quoted-claim', source: quotedSource, text: quotedClaim },
    ];
    const pages = specs.map(({ id, source, text }) => {
      const sources = [{ path: source, sha256: createHash('sha256').update(text).digest('hex') }];
      return {
        id,
        source,
        type: 'how-to',
        audience: ['maintainer'],
        class: 'C',
        sources,
        reviewDigest: api.mod.computeReviewDigest(sources),
        appliesTo: 'unreleased',
        reviewStatus: 'unreviewed',
      };
    });
    const manifest = {
      schemaVersion: 1,
      site: 'https://docs.example.test',
      base: '/agentic-sdd',
      repository: 'https://github.com/example/repo',
      docsVersion: 'dev',
      pages,
    };
    fs.mkdirSync(path.join(repo.root, 'website'), { recursive: true });
    fs.writeFileSync(path.join(repo.root, 'website', 'docs-manifest.json'), `${JSON.stringify(manifest, null, 2)}\n`);

    const result = checkApi.mod.checkSource({ repoRoot: repo.root });
    const claims = errorsWith(result, 'frontmatter/forged-claim', forgedSource);
    assert.deepEqual(
      claims.map((entry) => entry.details.key).sort(),
      ['reviewStatus', 'reviewedAt'],
      'both forged review keys must be reported against the page',
    );
    const approvals = errorsWith(result, 'frontmatter/forged-claim', approvalSource);
    assert.deepEqual(
      approvals.map((entry) => entry.details.key).sort(),
      ['approval', 'approvedBy'],
      'a fabricated approval record must be reported against the page',
    );
    for (const claim of [...claims, ...approvals]) {
      assert.equal(claim.severity, 'error');
      assert.ok(claim.line > 1, 'the diagnostic must point inside the front matter');
      assert.match(claim.message, new RegExp(claim.details.key));
      assert.match(claim.fix, /docs-manifest\.json|approval record/);
    }
    assert.equal(result.ok, false);

    // A quoted top-level key is still a claim; quoting must not bypass the scan.
    const quotedClaims = errorsWith(result, 'frontmatter/forged-claim', quotedSource);
    assert.deepEqual(quotedClaims.map((entry) => entry.details.key), ['humanReview']);

    // Control: an otherwise identical page with compliant front matter is clean,
    // so the rejection is the claim and not the page shape.
    const compliantClaims = errorsWith(result, 'frontmatter/forged-claim', compliantSource);
    assert.deepEqual(compliantClaims, []);
    assert.deepEqual(errorsWith(result, 'source/unreadable'), []);
  } finally {
    repo.cleanup();
  }
});

test('exported Markdown twins keep the unreviewed vocabulary instead of emitting a claim', { skip: exportApi.skip }, () => {
  const manifest = readJson(SHIPPED_MANIFEST);
  const plan = exportApi.mod.buildExport({ repoRoot });
  const allowlisted = manifest.pages.filter((page) => page.class === 'L' || page.class === 'C');
  assert.ok(allowlisted.length > 0, 'the shipped corpus must contain at least one allowlisted page');
  assert.equal(plan.pages.length, allowlisted.length, 'every L/C page must have one export entry');
  assert.deepEqual(
    plan.pages.map((entry) => entry.id).sort(),
    allowlisted.map((page) => page.id).sort(),
    'the export entries must be exactly the allowlisted pages',
  );

  const byId = new Map(manifest.pages.map((page) => [page.id, page]));
  for (const entry of plan.pages) {
    const page = byId.get(entry.id);
    assert.equal(
      entry.reviewStatus,
      page.reviewStatus ?? null,
      `${entry.id}: the export must not introduce a review status the registry lacks`,
    );
    assert.equal(entry.reviewStatus, 'unreviewed', `${entry.id}: B4 exports stay unreviewed`);
    assert.ok(!Object.hasOwn(entry, 'reviewedAt'), `${entry.id}: an exported entry must not fabricate a review date`);

    const twinPath = entry.id === 'index' ? 'index.md' : `${entry.id}/index.md`;
    const twin = plan.files.find((file) => file.path === twinPath);
    assert.ok(twin, `${entry.id}: the Markdown twin ${twinPath} must exist`);
    assert.match(
      twin.content,
      new RegExp(`Review status: ${entry.reviewStatus}\\b`),
      `${entry.id}: the twin must carry the registry's review vocabulary`,
    );
    assert.doesNotMatch(twin.content, /Review status: reviewed\b/, `${entry.id}: the twin must not upgrade the registry status`);
    assert.doesNotMatch(
      twin.content,
      /^\s*(?:reviewStatus|reviewedAt|reviewedBy|approvedBy|approval)\s*:/im,
      `${entry.id}: the twin must not emit a forged claim field`,
    );
  }

  const index = JSON.parse(plan.files.find((file) => file.path === 'index.json').content);
  assert.ok(Array.isArray(index.pages) && index.pages.length === plan.pages.length, 'index.json must list every export entry');
  for (const page of index.pages) {
    assert.equal(page.reviewStatus, 'unreviewed', `${page.id}: index.json must stay unreviewed`);
    assert.ok(!Object.hasOwn(page, 'reviewedAt'), `${page.id}: index.json must not fabricate a review date`);
  }
});

