import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import remarkLinkRoutes from '../plugins/remark-link-routes.mjs';
import { RouteError } from '../scripts/lib/routes.mjs';

const here = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(here, '../..');
const sourcePath = path.join(repoRoot, 'website/src/content/docs/index.md');
const manifestPath = path.join(repoRoot, 'website/docs-manifest.json');

const position = (line, column = 1) => ({ start: { line, column }, end: { line, column: column + 1 } });
const root = (...children) => ({ type: 'root', children });
const text = (value) => ({ type: 'text', value });
const link = (url, line = 1) => ({ type: 'link', url, title: null, children: [text('link')], position: position(line) });
const image = (url, line = 1) => ({ type: 'image', url, alt: 'image', title: null, position: position(line) });
const heading = (value, line) => ({ type: 'heading', depth: 2, children: [text(value)], position: position(line) });
const definition = (url, line = 1) => ({ type: 'definition', identifier: 'ref', label: 'ref', url, title: null, position: position(line) });

async function run(tree, file = { path: sourcePath }, options = {}) {
  const transformer = remarkLinkRoutes({ manifestPath, repoRoot, ...options });
  return transformer(tree, file);
}

function expectRouteError(fn, code) {
  return assert.rejects(fn, (error) => {
    assert.ok(error instanceof RouteError, `expected RouteError, got ${error?.name}: ${error?.message}`);
    assert.equal(error.code, code, `expected code ${code}, got ${error.code}: ${error.message}`);
    return true;
  });
}

test('exports a remark transformer and validates required options', () => {
  assert.equal(typeof remarkLinkRoutes({ manifestPath, repoRoot }), 'function');
  assert.throws(() => remarkLinkRoutes({}), { name: 'RouteError', code: 'plugin-options' });
  assert.throws(() => remarkLinkRoutes({ manifestPath }), { name: 'RouteError', code: 'plugin-options' });
});

test('leaves external links, definitions, images, code, and prose unchanged', async () => {
  const tree = root(
    { type: 'html', value: '<div class="callout">Prose wrapper</div>', position: position(1) },
    { type: 'paragraph', children: [text('See <a href="./guide/intro.md">prose text</a>.'), link('https://example.com/x?y=1#z')] },
    definition('mailto:docs@example.com'),
    image('./images/architecture.png'),
    { type: 'code', lang: 'html', value: '<a href="./guide/intro.md">code</a>' },
    { type: 'inlineCode', value: '<a href="./guide/intro.md">' },
    { type: 'linkReference', identifier: 'ref', label: 'ref', referenceType: 'full', children: [text('ref')] },
  );
  const snapshot = structuredClone(tree);
  await run(tree);
  assert.deepEqual(tree, snapshot);
});

test('refuses raw HTML navigation links with file and position', async () => {
  const tree = root({ type: 'html', value: '<a\n  class="nav" href="./guide/intro.md">Intro</a>', position: position(4, 3) });
  await assert.rejects(() => run(tree), (error) => {
    assert.ok(error instanceof RouteError);
    assert.equal(error.code, 'raw-html-navigation');
    assert.equal(error.line, 4);
    assert.equal(error.column, 3);
    assert.equal(error.source, sourcePath);
    return true;
  });
});

test('refuses duplicate heading slugs in one file', async () => {
  const tree = root(
    heading('Same heading', 2),
    heading('Other heading', 4),
    heading('Same heading', 6),
  );
  await assert.rejects(() => run(tree), (error) => {
    assert.ok(error instanceof RouteError);
    assert.equal(error.code, 'duplicate-heading');
    assert.equal(error.line, 6);
    assert.equal(error.source, sourcePath);
    assert.match(error.message, /same-heading/);
    return true;
  });
});

test('allows unique headings and non-navigation raw HTML', async () => {
  const tree = root(
    heading('First heading', 2),
    heading('Second heading', 4),
    { type: 'html', value: '<div><strong>Styled prose</strong></div>', position: position(6) },
    link('https://example.com'),
  );
  await run(tree);
});

test('requires source-file context', async () => {
  await expectRouteError(() => run(root(), {}), 'missing-source-context');
});

test('rewrites internal links and definitions through the manifest module', async (t) => {
  const manifestModule = path.join(repoRoot, 'website/scripts/lib/manifest.mjs');
  if (!fs.existsSync(manifestModule)) {
    t.skip('integration-pending: sibling manifest module is not present in this worktree');
    return;
  }

  const tree = root(
    link('./index.md#foundation-validation', 2),
    definition('./index.md', 3),
  );
  await run(tree);
  assert.equal(tree.children[0].url, '/agentic-sdd/#foundation-validation');
  assert.equal(tree.children[1].url, '/agentic-sdd/');
});
