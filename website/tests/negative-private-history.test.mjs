// Spec 020 B4-W1-G — private and historical paths must never become public
// pages, evidence inputs or published artifacts.
//
// Frozen sources: plan C1 (`sources[]` are contained public evidence inputs),
// C4 ("Repository history, docs/testing reports, specs, handovers, runs,
// backups ... never recursively copied") and the page contract's public root.
// The private-tree names are taken from that contract, not from the validator.
// The export producer and the source link classifier are exercised directly;
// the built-artifact leak scan remains blocked with its owning task.

import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

import { computeReviewDigest } from '../scripts/lib/manifest.mjs';
import {
  BASELINE_MANIFEST,
  PUBLIC_DOCS,
  blockedOptions,
  errorsWith,
  expectRejected,
  loadFrozenApi,
  loadProducer,
  readJson,
  repoRoot,
  tempRepo,
  walkFiles,
} from './negative-fixtures/lib/harness.mjs';

const api = await loadFrozenApi('website/scripts/lib/manifest.mjs', ['validateManifest']);
const exportApi = await loadProducer('export', ['buildExport']);
const checkApi = await loadProducer('check', ['checkSource']);
const skip = api.skip;
const SHIPPED_MANIFEST = path.join(repoRoot, 'website', 'docs-manifest.json');

const blocked = {};
function blockedCase(key, producer, absent, unadapted) {
  blocked[key] = blockedOptions(producer, absent, unadapted);
  return blocked[key];
}

// Paths the contract names as private/history, plus credential-shaped files.
const PRIVATE_PAGE_SOURCES = [
  ['a feature handover', 'docs/handovers/2026-11-14_020.md'],
  ['execution run scratch', '.agentic-sdd/runs/020-b4-w1-g/checkpoint.json'],
  ['an installer backup tree', 'backups/2026-11-14/skills/agentic-sdd-implement/SKILL.md'],
  ['a credential directory', 'credentials/provider-token.json'],
  ['a traversal escape out of the public root', `${PUBLIC_DOCS}/../../../docs/handovers/2026-11-14_020.md`],
];

const PRIVATE_EVIDENCE = [
  ['a feature handover', 'docs/handovers/2026-11-14_020.md'],
  ['execution run scratch', '.agentic-sdd/runs/020-b4-w1-g/checkpoint.json'],
  ['an installer backup tree', 'backups/2026-11-14/skills/agentic-sdd-implement/SKILL.md'],
  ['a built site artifact', 'website/dist/index.html'],
  ['an Astro state directory', 'website/.astro/content-assets.mjs'],
  ['an environment file', '.env'],
  ['a private key file', 'ops/deploy.pem'],
  ['an SSH private key', 'keys/id_ed25519'],
  ['a secrets directory', 'secrets/provider.json'],
];

// ---------------------------------------------------------------------------
// A private/history path cannot become a public page
// ---------------------------------------------------------------------------

for (const [label, privatePath] of PRIVATE_PAGE_SOURCES) {
  expectRejected(api, `${label} as a public page source`, (m) => {
    m.pages[0].source = privatePath;
  }, { pathMentions: /source/i });
}

// ---------------------------------------------------------------------------
// A private/history path cannot become a page evidence input
// ---------------------------------------------------------------------------

for (const [label, privatePath] of PRIVATE_EVIDENCE) {
  expectRejected(api, `${label} as an evidence input`, (m) => {
    m.pages[0].sources[0].path = privatePath;
  }, { pathMentions: /sources\[0\]\.path/ });
}

test('private evidence is rejected for a private-tree reason, not an incidental digest mismatch', { skip }, () => {
  const manifest = readJson(BASELINE_MANIFEST);
  manifest.pages[0].sources[0].path = 'docs/handovers/2026-11-14_020.md';
  let error;
  try {
    api.mod.validateManifest(manifest, {});
  } catch (caught) {
    error = caught;
  }
  assert.ok(error, 'a handover evidence path must be rejected');
  assert.match(
    `${error.path} ${error.message}`,
    /private|history/i,
    'the diagnostic must explain that the path is private/history rather than merely stale',
  );
});

// ---------------------------------------------------------------------------
// The shipped registry and public tree carry no private or credential path
// ---------------------------------------------------------------------------

test('the history tree this feature must not publish actually exists', () => {
  assert.ok(
    fs.existsSync(path.join(repoRoot, 'docs', 'handovers')),
    'docs/handovers must exist; otherwise the leakage checks below are vacuous',
  );
});

test('no shipped page source or evidence input lies in a private tree', { skip }, () => {
  const manifest = readJson(SHIPPED_MANIFEST);
  const privatePrefixes = ['docs/handovers/', '.agentic-sdd/', 'backups/', 'website/dist/', 'website/.astro/'];
  const offenders = [];
  for (const page of manifest.pages) {
    const declared = [page.source, ...page.sources.map((entry) => entry.path)];
    for (const value of declared) {
      if (privatePrefixes.some((prefix) => value.startsWith(prefix))) {
        offenders.push(`${page.id}: ${value}`);
      }
    }
  }
  assert.deepEqual(offenders, [], `private/history paths are not public inputs:\n${offenders.join('\n')}`);
});

test('the public content tree contains no symlink and no credential-shaped file', () => {
  const contentRoot = path.join(repoRoot, PUBLIC_DOCS);
  const credential = /^(?:\.env(?:\..+)?|\.npmrc|\.netrc|id_(?:rsa|ed25519|ecdsa|dsa)(?:\.pub)?)$/i;
  const credentialExtension = /\.(?:pem|key|p12|pfx|keystore|jks)$/i;
  const entries = walkFiles(contentRoot);
  assert.ok(entries.length > 0, 'the public content tree must not be empty');
  const offenders = [];
  for (const entry of entries) {
    const name = path.basename(entry.path);
    if (entry.symlink) offenders.push(`${entry.path}: symlink`);
    if (credential.test(name) || credentialExtension.test(name)) offenders.push(`${entry.path}: credential-shaped name`);
  }
  assert.deepEqual(offenders, [], `the public content tree must not carry symlinks or credentials:\n${offenders.join('\n')}`);
});

test('the public content tree holds only Markdown and reviewed public assets', () => {
  const allowed = new Set(['.md', '.png', '.jpg', '.jpeg', '.svg', '.webp', '.avif', '.gif', '.ico']);
  const offenders = [];
  for (const entry of walkFiles(path.join(repoRoot, PUBLIC_DOCS))) {
    if (entry.symlink) continue;
    const extension = path.extname(entry.path).toLowerCase();
    if (!allowed.has(extension)) offenders.push(entry.path);
  }
  assert.deepEqual(offenders, [], `unexpected public content file type:\n${offenders.join('\n')}`);
});

// ---------------------------------------------------------------------------
// Producer-dependent enforcement
// ---------------------------------------------------------------------------

test('the export allowlist never includes a private or historical path', { skip: exportApi.skip }, () => {
  const manifest = readJson(SHIPPED_MANIFEST);
  const plan = exportApi.mod.buildExport({ repoRoot });
  assert.ok(plan.pages.length > 0, 'the export must produce at least one page, or this check is vacuous');

  // The shipped plan itself: every exported source is a public-root page and
  // no structured output field names a private/history path.
  const structured = JSON.stringify({
    sources: plan.pages.map((page) => page.source),
    sourceUrls: plan.pages.map((page) => page.sourceUrl),
    markdownUrls: plan.pages.map((page) => page.markdownUrl),
    files: plan.files.map((file) => file.path),
    excluded: plan.excluded,
  });
  for (const [label, privatePath] of PRIVATE_EVIDENCE) {
    assert.ok(!structured.includes(privatePath), `${label} must not appear in the export`);
  }
  const privatePageSources = /^(?:docs\/handovers|docs\/testing|specs|backups|\.agentic-sdd|website\/(?:dist|\.astro))\//;
  for (const page of plan.pages) {
    assert.ok(page.source.startsWith(`${PUBLIC_DOCS}/`), `${page.id}: ${page.source} is outside the public root`);
    assert.doesNotMatch(page.source, privatePageSources, `${page.id}: a private/history source entered the export`);
  }
  for (const file of plan.files) {
    assert.match(file.path, /^[a-z0-9][a-z0-9/-]*\.(?:md|json|txt)$/, `${file.path} is not an allowlisted export name`);
  }

  // The producer refuses a manifest that declares one of those paths instead.
  for (const [label, privatePath] of PRIVATE_EVIDENCE) {
    const mutated = readJson(SHIPPED_MANIFEST);
    mutated.pages[0].source = privatePath;
    const repo = tempRepo({ 'manifest.json': JSON.stringify(mutated, null, 2) });
    try {
      assert.throws(
        () => exportApi.mod.buildExport({ repoRoot, manifestPath: path.join(repo.root, 'manifest.json') }),
        (error) => {
          assert.match(
            `${error.path ?? ''} ${error.message}`,
            /source/i,
            `${label}: the refusal must name the page source field`,
          );
          return true;
        },
        `${label} must not be exportable`,
      );
    } finally {
      repo.cleanup();
    }
  }
  assert.ok(manifest.pages.length > 0);
});

test('the built-artifact check rejects leaked private or historical content', blockedCase(
  'built-leak',
  'check',
  'the built-artifact leak scan is not present in this worktree',
  'checkBuilt parses rendered links, ids and canonical URLs only; it never scans emitted text for private/history paths or credential-shaped content, so there is no built-artifact leak scan (owner: T09/B4-W1-D built checks)',
), () => {
  // T09 verification: "leaked history/private path".
});

test('a private link target is reported by the source link/fragment check', { skip: checkApi.skip }, () => {
  // Historical findings must be enumerated, not suppressed with a broad ignore.
  const index = `${PUBLIC_DOCS}/index.md`;
  const handover = 'docs/handovers/2026-11-14_020.md';
  const spec = 'specs/020-documentation-ecosystem/spec.md';
  const indexText = `---\ntitle: index\n---\n\n[handover](../../../../${handover})\n\n[spec](../../../../${spec})\n`;
  const repo = tempRepo({
    [index]: indexText,
    [handover]: '# Handover\n',
    [spec]: '# Spec\n',
  });
  try {
    const sources = [
      { path: index, sha256: createHash('sha256').update(indexText).digest('hex') },
    ];
    fs.writeFileSync(
      path.join(repo.root, 'website/docs-manifest.json'),
      `${JSON.stringify({
        schemaVersion: 1,
        site: 'https://docs.example.test',
        base: '/agentic-sdd',
        repository: 'https://github.com/example/repo',
        docsVersion: 'dev',
        pages: [
          {
            id: 'index',
            source: index,
            type: 'reference',
            audience: ['maintainer'],
            class: 'L',
            sources,
            reviewDigest: computeReviewDigest(sources),
            appliesTo: 'unreleased',
            reviewStatus: 'unreviewed',
          },
        ],
      }, null, 2)}\n`,
    );
    const result = checkApi.mod.checkSource({ repoRoot: repo.root });
    const errors = errorsWith(result, 'links/private-target', index);
    assert.deepEqual(
      errors.map((entry) => entry.details.target).sort(),
      [handover, spec],
      'both private/history targets must be named, not only a missing-file symptom',
    );
    for (const error of errors) {
      assert.equal(error.severity, 'error');
      assert.match(error.fix, /public page|private\/history/);
    }
    assert.equal(result.ok, false);
  } finally {
    repo.cleanup();
  }
});

