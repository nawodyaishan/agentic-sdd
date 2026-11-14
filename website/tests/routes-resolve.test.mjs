import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';

import { RouteError, resolveLink } from '../scripts/lib/routes.mjs';

const SOURCE_INDEX = 'website/src/content/docs/index.md';
const SOURCE_INTRO = 'website/src/content/docs/guide/intro.md';

function makeManifest(overrides = {}) {
  return {
    schemaVersion: 1,
    site: 'https://nawodyaishan.github.io',
    base: '/agentic-sdd',
    repository: 'https://github.com/nawodyaishan/agentic-sdd',
    docsVersion: 'dev',
    pages: [
      { id: 'index', source: SOURCE_INDEX },
      { id: 'guide/getting-started', source: 'website/src/content/docs/guide/getting started.md' },
      { id: 'guide/intro', source: SOURCE_INTRO },
      { id: 'reference/cli', source: 'website/src/content/docs/reference/cli.md' },
    ],
    ...overrides,
  };
}

function resolve(overrides = {}) {
  return resolveLink({
    manifest: makeManifest(),
    sourcePath: SOURCE_INDEX,
    href: './index.md',
    ...overrides,
  });
}

function expectRouteError(fn, code) {
  assert.throws(fn, (error) => {
    assert.ok(error instanceof RouteError, `expected RouteError, got ${error?.name}: ${error?.message}`);
    assert.equal(error.code, code, `expected code ${code}, got ${error.code}: ${error.message}`);
    assert.equal(typeof error.message, 'string');
    assert.ok(error.message.length > 0);
    return true;
  });
}

function makeTempRepo(t) {
  fs.mkdirSync('/tmp/opencode', { recursive: true });
  const root = fs.mkdtempSync('/tmp/opencode/routes-w1b-');
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  return root;
}

test('projects the root index route under the project base', () => {
  assert.equal(resolve(), '/agentic-sdd/');
  assert.equal(resolve({ href: './index.md#foundation-validation' }), '/agentic-sdd/#foundation-validation');
});

test('resolves relative page links through manifest source identity', () => {
  assert.equal(resolve({ href: './guide/intro.md' }), '/agentic-sdd/guide/intro/');
  assert.equal(
    resolve({ sourcePath: SOURCE_INTRO, href: '../reference/cli.md#flags' }),
    '/agentic-sdd/reference/cli/#flags',
  );
  assert.equal(resolve({ sourcePath: SOURCE_INTRO, href: 'intro.md' }), '/agentic-sdd/guide/intro/');
});

test('normalizes root and trailing-slash base variants', () => {
  const rootBase = makeManifest({ base: '/' });
  assert.equal(resolve({ manifest: rootBase }), '/');
  assert.equal(resolve({ manifest: rootBase, href: './guide/intro.md' }), '/guide/intro/');
  assert.equal(resolve({ manifest: rootBase, href: './index.md', target: 'markdown' }), '/index.md');

  const slashBase = makeManifest({ base: '/agentic-sdd/' });
  assert.equal(resolve({ manifest: slashBase }), '/agentic-sdd/');
});

test('emits Markdown twin routes for pages, not HTML routes', () => {
  assert.equal(resolve({ href: './index.md#foundation-validation', target: 'markdown' }), '/agentic-sdd/index.md#foundation-validation');
  assert.equal(resolve({ href: './guide/intro.md', target: 'markdown' }), '/agentic-sdd/guide/intro/index.md');
});

test('resolves page and source-escape links to fixed GitHub blob URLs', () => {
  assert.equal(
    resolve({ href: './guide/intro.md#L10', target: 'source' }),
    'https://github.com/nawodyaishan/agentic-sdd/blob/main/website/src/content/docs/guide/intro.md#L10',
  );
  assert.equal(
    resolve({ href: './guide/intro.md', target: 'source', ref: 'v0.11.0' }),
    'https://github.com/nawodyaishan/agentic-sdd/blob/v0.11.0/website/src/content/docs/guide/intro.md',
  );
  assert.equal(
    resolve({ href: '../../../../internal/fleet/fleet.go' }),
    'https://github.com/nawodyaishan/agentic-sdd/blob/main/internal/fleet/fleet.go',
  );
  assert.equal(
    resolve({ href: '../../../../CONTRIBUTING.md', target: 'markdown' }),
    'https://github.com/nawodyaishan/agentic-sdd/blob/main/CONTRIBUTING.md',
  );
});

test('encodes paths once and preserves existing fragment escapes', () => {
  assert.equal(
    resolve({ href: './guide/getting%20started.md#wire%20format' }),
    '/agentic-sdd/guide/getting-started/#wire%20format',
  );
  assert.equal(
    resolve({ href: './guide/getting started.md#wire format' }),
    '/agentic-sdd/guide/getting-started/#wire%20format',
  );
  assert.equal(
    resolve({ href: './guide/getting%20started.md#symbols%2Fpaths' }),
    '/agentic-sdd/guide/getting-started/#symbols%2Fpaths',
  );
  assert.equal(
    resolve({ href: './guide/getting%20started.md', target: 'source' }),
    'https://github.com/nawodyaishan/agentic-sdd/blob/main/website/src/content/docs/guide/getting%20started.md',
  );
});

test('keeps pure fragments, HTTPS, mailto, and image references unchanged', () => {
  assert.equal(resolve({ href: '#foundation-validation' }), '#foundation-validation');
  assert.equal(resolve({ href: 'https://example.com/docs?x=1#frag' }), 'https://example.com/docs?x=1#frag');
  assert.equal(resolve({ href: 'mailto:docs@example.com?subject=Hi' }), 'mailto:docs@example.com?subject=Hi');
  assert.equal(resolve({ sourcePath: SOURCE_INTRO, href: './images/architecture.png' }), './images/architecture.png');
  assert.equal(resolve({ sourcePath: SOURCE_INTRO, href: '../assets/diagram.svg#view' }), '../assets/diagram.svg#view');
  assert.equal(resolve({ href: 'https://example.com/diagram.png' }), 'https://example.com/diagram.png');
});

test('rejects unsafe schemes and protocol-relative references', () => {
  for (const href of ['javascript:alert(1)', 'data:text/html,<a>', 'http://example.com', '//example.com/x', 'file:///etc/passwd']) {
    expectRouteError(() => resolve({ href }), 'unsafe-scheme');
  }
});

test('rejects unknown public Markdown targets and non-page relative links', () => {
  expectRouteError(() => resolve({ href: './missing.md' }), 'unknown-target');
  expectRouteError(() => resolve({ sourcePath: SOURCE_INTRO, href: './notes.txt' }), 'unsupported-relative-link');
  expectRouteError(() => resolve({ href: '/etc/passwd' }), 'path-escape');
});

test('rejects repository path escapes but accepts contained source escapes', (t) => {
  expectRouteError(() => resolve({ href: '../../../../../outside.md' }), 'path-escape');
  expectRouteError(() => resolve({ href: '../../../../../../outside.md' }), 'path-escape');
  expectRouteError(() => resolve({ sourcePath: '../outside.md', href: './index.md' }), 'path-escape');

  const root = makeTempRepo(t);
  const outside = path.join(root, '..', 'outside-index.md');
  expectRouteError(
    () => resolve({ repoRoot: root, sourcePath: outside, href: './index.md' }),
    'path-escape',
  );
});

test('accepts absolute source paths contained by repoRoot', (t) => {
  const root = makeTempRepo(t);
  const sourcePath = path.join(root, SOURCE_INDEX);
  assert.equal(resolve({ repoRoot: root, sourcePath, href: './guide/intro.md' }), '/agentic-sdd/guide/intro/');
});

test('rejects symlinked repository path components', (t) => {
  const root = makeTempRepo(t);
  const repoRoot = path.join(root, 'repo');
  const outside = path.join(root, 'outside');
  fs.mkdirSync(repoRoot, { recursive: true });
  fs.mkdirSync(outside, { recursive: true });
  fs.symlinkSync(outside, path.join(repoRoot, 'website'), 'dir');

  expectRouteError(
    () => resolve({ repoRoot, sourcePath: SOURCE_INDEX, href: './index.md' }),
    'symlink-path',
  );
});

test('does not mutate the supplied manifest and rejects duplicate identities', () => {
  const manifest = makeManifest();
  const snapshot = JSON.stringify(manifest);
  resolve({ manifest, href: './guide/intro.md' });
  assert.equal(JSON.stringify(manifest), snapshot);

  const duplicateSource = makeManifest({
    pages: [
      { id: 'index', source: SOURCE_INDEX },
      { id: 'guide/intro', source: SOURCE_INDEX },
    ],
  });
  expectRouteError(() => resolve({ manifest: duplicateSource }), 'duplicate-source');

  const duplicateId = makeManifest({
    pages: [
      { id: 'index', source: SOURCE_INDEX },
      { id: 'index', source: SOURCE_INTRO },
    ],
  });
  expectRouteError(() => resolve({ manifest: duplicateId }), 'duplicate-id');
});

test('validates resolver inputs without relying on filesystem existence', () => {
  expectRouteError(() => resolve({ sourcePath: undefined }), 'missing-source-context');
  expectRouteError(() => resolve({ href: undefined }), 'invalid-href');
  expectRouteError(() => resolve({ target: 'html5' }), 'invalid-target');
  expectRouteError(() => resolve({ manifest: null }), 'invalid-manifest');
  expectRouteError(() => resolve({ manifest: makeManifest({ base: 'agentic-sdd' }) }), 'invalid-manifest');
  expectRouteError(() => resolve({ manifest: makeManifest({ base: '/../escape' }) }), 'invalid-manifest');
  expectRouteError(() => resolve({ manifest: makeManifest({ repository: 'http://github.com/a/b' }) }), 'invalid-manifest');
  expectRouteError(() => resolve({ manifest: makeManifest({ repository: 'https://gitlab.com/a/b' }) }), 'invalid-manifest');
});
