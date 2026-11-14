// SPEC020 B4-W1-D — built-output fixtures: rendered fragments, encoded paths
// and canonical/base metadata parity. The fixtures are hand-written static
// HTML, so the checker is proven against actual output shape without invoking
// Astro or any network operation.
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { linkSync, mkdirSync, mkdtempSync, readFileSync, rmSync, unlinkSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { computeReviewDigest } from '../scripts/lib/manifest.mjs';
import { checkBuilt, checkDocumentation } from '../scripts/check.mjs';

const testsDir = path.dirname(fileURLToPath(import.meta.url));
const websiteDir = path.resolve(testsDir, '..');
const checkScript = path.join(websiteDir, 'scripts/check.mjs');
const DOCS = 'website/src/content/docs';
const SITE = 'https://docs.example.test';
const BASE = '/agentic-sdd';

function sha256(value) {
  return createHash('sha256').update(value).digest('hex');
}

function pageEntry(id, source, text) {
  const evidence = [{ path: source, sha256: sha256(text) }];
  return {
    id,
    source,
    type: 'reference',
    audience: ['maintainer'],
    class: 'L',
    sources: evidence,
    reviewDigest: computeReviewDigest(evidence),
    appliesTo: 'unreleased',
    reviewStatus: 'unreviewed',
  };
}

function canonicalFor(id, base = BASE) {
  const prefix = base === '/' ? '' : base;
  return `${SITE}${prefix}${id === 'index' ? '/' : `/${id}/`}`;
}

function html({ canonical, body }) {
  return `<!doctype html>\n<html lang="en">\n<head>\n<link rel="canonical" href="${canonical}" />\n</head>\n<body>${body}</body>\n</html>\n`;
}

function makeFixture(t, { base = BASE, indexHtml, nextHtml, extra = {} } = {}) {
  mkdirSync('/tmp/opencode', { recursive: true });
  const repoRoot = mkdtempSync('/tmp/opencode/spec020-b4-built-');
  t.after(() => rmSync(repoRoot, { recursive: true, force: true }));
  const pages = [
    ['index', `${DOCS}/index.md`, '# Home\n'],
    ['next', `${DOCS}/next.md`, '# Next\n'],
  ].map(([id, source, body]) => [id, source, `---\ntitle: ${id}\n---\n\n${body}`]);
  const manifest = {
    schemaVersion: 1,
    site: SITE,
    base,
    repository: 'https://github.com/example/repo',
    docsVersion: 'dev',
    pages: pages.map(([id, source, text]) => pageEntry(id, source, text)),
  };
  for (const [, source, text] of pages) {
    const absolute = path.join(repoRoot, source);
    mkdirSync(path.dirname(absolute), { recursive: true });
    writeFileSync(absolute, text, 'utf8');
  }
  mkdirSync(path.join(repoRoot, 'website'), { recursive: true });
  writeFileSync(path.join(repoRoot, 'website/docs-manifest.json'), `${JSON.stringify(manifest, null, 2)}\n`, 'utf8');

  const dist = path.join(repoRoot, 'dist');
  const files = {
    'index.html': indexHtml ?? html({ canonical: canonicalFor('index', base), body: '<h1 id="home">Home</h1>' }),
    'next/index.html': nextHtml ?? html({ canonical: canonicalFor('next', base), body: '<h2 id="target">Target</h2>' }),
    ...extra,
  };
  for (const [relative, content] of Object.entries(files)) {
    const absolute = path.join(dist, relative);
    mkdirSync(path.dirname(absolute), { recursive: true });
    writeFileSync(absolute, content, 'utf8');
  }
  return { repoRoot, builtDir: dist };
}

function codes(result) {
  return result.errors.map((entry) => entry.code).sort();
}

function find(result, code) {
  return result.errors.find((entry) => entry.code === code);
}

test('accepts canonical/base parity, rendered fragments and encoded paths', (t) => {
  const indexHtml = html({
    canonical: canonicalFor('index'),
    body:
      '<a href="/agentic-sdd/next/#target">next</a>' +
      '<a href="/agentic-sdd/a%20b/">spaced</a>' +
      '<a href="https://example.test/external">external</a>',
  });
  const { repoRoot, builtDir } = makeFixture(t, {
    indexHtml,
    extra: { 'a b/index.html': html({ canonical: `${SITE}/agentic-sdd/a%20b/`, body: '<h2 id="space">Space</h2>' }) },
  });
  const result = checkBuilt({ repoRoot, builtDir });
  assert.equal(result.ok, true, JSON.stringify(result.errors));
  assert.equal(result.summary.builtPages, 2);
  assert.equal(result.summary.external, 1);
  assert.equal(result.observations.length, 1);
  assert.equal(result.observations[0].code, 'built/external');

  const combined = checkDocumentation({ repoRoot, builtDir });
  assert.equal(combined.ok, true, JSON.stringify(combined.errors));
  assert.equal(combined.summary.builtPages, 2);
  assert.equal(JSON.stringify(result), JSON.stringify(checkBuilt({ repoRoot, builtDir })));
});

test('reports a broken rendered fragment with path, line and fix', (t) => {
  const { repoRoot, builtDir } = makeFixture(t, {
    indexHtml: html({ canonical: canonicalFor('index'), body: '<a href="/agentic-sdd/next/#missing">bad</a>' }),
  });
  const result = checkBuilt({ repoRoot, builtDir });
  const error = find(result, 'built/broken-anchor');
  assert.ok(error, `expected built/broken-anchor, got ${JSON.stringify(codes(result))}`);
  assert.match(error.path, /dist\/index\.html$/);
  assert.ok(error.line > 0);
  assert.match(error.fix, /id|fragment/);
});

test('reports a rendered link whose page was not emitted', (t) => {
  const { repoRoot, builtDir } = makeFixture(t, {
    indexHtml: html({ canonical: canonicalFor('index'), body: '<a href="/agentic-sdd/missing/">gone</a>' }),
  });
  const result = checkBuilt({ repoRoot, builtDir });
  const error = find(result, 'built/broken-link');
  assert.ok(error, `expected built/broken-link, got ${JSON.stringify(codes(result))}`);
  assert.match(error.message, /missing\//);
});

test('reports a missing emitted page for a manifest entry', (t) => {
  const { repoRoot, builtDir } = makeFixture(t, { nextHtml: '' });
  writeFileSync(path.join(builtDir, 'next/index.html'), '', 'utf8');
  const result = checkBuilt({ repoRoot, builtDir });
  // An empty file is still emitted HTML with no canonical/id diagnostics.
  assert.ok(codes(result).includes('built/canonical-missing'), JSON.stringify(codes(result)));

  rmSync(path.join(builtDir, 'next/index.html'));
  const missing = checkBuilt({ repoRoot, builtDir });
  const error = find(missing, 'built/missing-page');
  assert.ok(error, `expected built/missing-page, got ${JSON.stringify(codes(missing))}`);
  assert.match(error.message, /next/);
});

test('reports a hard-linked emitted page and never follows it', (t) => {
  const { repoRoot, builtDir } = makeFixture(t);
  const nextHtml = readFileSync(path.join(builtDir, 'next/index.html'), 'utf8');
  writeFileSync(path.join(builtDir, 'next-copy.html'), nextHtml, 'utf8');
  unlinkSync(path.join(builtDir, 'next/index.html'));
  linkSync(path.join(builtDir, 'next-copy.html'), path.join(builtDir, 'next/index.html'));
  const result = checkBuilt({ repoRoot, builtDir });
  const error = find(result, 'built/hardlink-output');
  assert.ok(error, `expected built/hardlink-output, got ${JSON.stringify(codes(result))}`);
  assert.match(error.path, /dist\/next\/index\.html$/);
  assert.match(error.fix, /hardlinks/);
  assert.equal(result.ok, false);
});

test('reports canonical metadata that diverges from the manifest', (t) => {
  const { repoRoot, builtDir } = makeFixture(t, {
    indexHtml: html({ canonical: `${SITE}/wrong-base/`, body: '<h1 id="home">Home</h1>' }),
  });
  const result = checkBuilt({ repoRoot, builtDir });
  const error = find(result, 'built/canonical-mismatch');
  assert.ok(error, `expected built/canonical-mismatch, got ${JSON.stringify(codes(result))}`);
  assert.equal(error.details.expected, canonicalFor('index'));
  assert.match(error.fix, /manifest\.site|manifest\.base/);
});

test('reports wrong-base internal links', (t) => {
  const { repoRoot, builtDir } = makeFixture(t, {
    indexHtml: html({ canonical: canonicalFor('index'), body: '<a href="/other-base/next/">bad base</a>' }),
  });
  const result = checkBuilt({ repoRoot, builtDir });
  const error = find(result, 'built/wrong-base');
  assert.ok(error, `expected built/wrong-base, got ${JSON.stringify(codes(result))}`);
  assert.match(error.fix, /\/agentic-sdd/);
});

test('manifest base mutation makes built canonical and link base fail', (t) => {
  const { repoRoot, builtDir } = makeFixture(t, {
    base: '/other-base',
    indexHtml: html({
      canonical: canonicalFor('index', BASE),
      body: '<a href="/agentic-sdd/next/">stale base</a>',
    }),
  });
  const result = checkBuilt({ repoRoot, builtDir });
  assert.ok(codes(result).includes('built/canonical-mismatch'), JSON.stringify(codes(result)));
  assert.ok(codes(result).includes('built/wrong-base'), JSON.stringify(codes(result)));
});

test('reports duplicate rendered ids and encoded-path misses', (t) => {
  const { repoRoot, builtDir } = makeFixture(t, {
    indexHtml: html({
      canonical: canonicalFor('index'),
      body:
        '<h1 id="home">Home</h1><span id="home"></span>' +
        '<a href="/agentic-sdd/missing%20dir/">miss</a>',
    }),
  });
  const result = checkBuilt({ repoRoot, builtDir });
  assert.ok(codes(result).includes('built/duplicate-id'), JSON.stringify(codes(result)));
  assert.ok(codes(result).includes('built/broken-link'), JSON.stringify(codes(result)));
});

test('supports a root project base', (t) => {
  const { repoRoot, builtDir } = makeFixture(t, {
    base: '/',
    indexHtml: html({
      canonical: canonicalFor('index', '/'),
      body: '<a href="/next/#target">next</a><link-x>',
    }),
    extra: {
      'next/index.html': html({ canonical: canonicalFor('next', '/'), body: '<h2 id="target">Target</h2>' }),
    },
  });
  const result = checkBuilt({ repoRoot, builtDir });
  assert.equal(result.ok, true, JSON.stringify(result.errors));
});

test('reports a missing built directory actionably', (t) => {
  const { repoRoot } = makeFixture(t);
  const result = checkBuilt({ repoRoot, builtDir: 'not-built' });
  const error = find(result, 'built/missing-directory');
  assert.ok(error, `expected built/missing-directory, got ${JSON.stringify(codes(result))}`);
  assert.match(error.fix, /build/);
});

test('CLI resolves a relative --built path from the repository root, not the caller cwd', (t) => {
  const { repoRoot } = makeFixture(t);
  const run = spawnSync(
    process.execPath,
    [checkScript, '--repo', repoRoot, '--built', 'dist', '--json'],
    { encoding: 'utf8', cwd: websiteDir },
  );
  assert.equal(run.status, 0, run.stderr);
  const report = JSON.parse(run.stdout);
  assert.equal(report.ok, true);
  assert.equal(report.summary.builtPages, 2);
});

test('CLI --built runs built-only mode and reports built pages', (t) => {
  const { repoRoot, builtDir } = makeFixture(t);
  const run = spawnSync(
    process.execPath,
    [checkScript, '--repo', repoRoot, '--built', builtDir, '--json'],
    { encoding: 'utf8' },
  );
  assert.equal(run.status, 0, run.stderr);
  const report = JSON.parse(run.stdout);
  assert.equal(report.ok, true);
  assert.equal(report.summary.builtPages, 2);
  assert.equal(report.summary.links, 0);
});
