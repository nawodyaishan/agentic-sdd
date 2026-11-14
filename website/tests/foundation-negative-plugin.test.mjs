// Spec 020 B1-W1-D — independent counterexamples for the frozen remark plugin
// contract in plan C1: only Markdown `link` and `definition` nodes are visited,
// the current source comes from the vfile path, raw HTML navigation links and
// duplicate heading slugs are refused, and images/code/prose are untouched.
//
// The plugin factory and its sibling manifest/routes modules are not present in
// this isolated wave; a missing module or dependency is integration-pending,
// never a behavioural pass or red proof.

import assert from 'node:assert/strict';
import path from 'node:path';
import test from 'node:test';

import {
  INDEX_SOURCE,
  VALID_MANIFEST,
  captureAsync,
  fixtureTree,
  loadFrozenApi,
  reportPending,
} from './foundation-negative-support/support.mjs';

const api = await loadFrozenApi('website/plugins/remark-link-routes.mjs', ['default']);
reportPending('foundation-negative-plugin', api);

const skip = api.skip;

// The plugin normally receives a real vfile from unified. Use it when the
// workspace provides it, otherwise a minimal stub with the same documented
// surface (path, messages, message(), fail()).
let VFile = null;
try {
  ({ VFile } = await import('vfile'));
} catch {
  VFile = null;
}

const text = (value) => ({ type: 'text', value });
const paragraph = (...children) => ({ type: 'paragraph', children });
const link = (url) => ({ type: 'link', url, children: [text('link')] });
const root = (...children) => ({ type: 'root', children });
const heading = (value, depth = 2) => ({ type: 'heading', depth, children: [text(value)] });

const sourcePath = path.join(fixtureTree, INDEX_SOURCE);

function makeFile(filePath = sourcePath) {
  if (VFile) return new VFile({ path: filePath, value: '# fixture\n' });
  return {
    path: filePath,
    value: '# fixture\n',
    messages: [],
    history: filePath ? [filePath] : [],
    cwd: process.cwd(),
    data: {},
    message(reason, place, origin) {
      const diagnostic = { reason, place, origin, line: place?.start?.line, column: place?.start?.column };
      this.messages.push(diagnostic);
      return diagnostic;
    },
    fail(reason, place) {
      const error = new Error(reason);
      error.name = 'VFileMessage';
      error.place = place;
      throw error;
    },
    toString() {
      return this.value;
    },
  };
}

async function apply(tree, file = makeFile()) {
  const transform = api.mod.default({ manifestPath: VALID_MANIFEST, repoRoot: fixtureTree });
  assert.equal(typeof transform, 'function', 'remarkLinkRoutes(...) must return a transformer');
  const { threw, error } = await captureAsync(() => transform(tree, file));
  return { threw, error, tree, file };
}

const messagesOf = (file) => (Array.isArray(file.messages) ? file.messages : []);

/** The contract fixes rejection, not the diagnostic channel (throw vs vfile). */
function assertDiagnosed(result, label) {
  const messages = messagesOf(result.file);
  assert.ok(result.threw || messages.length > 0,
    `${label}: plugin must reject the input with a file/position-aware diagnostic`);
  return result.threw ? result.error : messages[0];
}

const linksIn = (tree) => tree.children.flatMap((node) => node.children ?? []);

// ---------------------------------------------------------------------------
// Positive rewriting through the shared resolver
// ---------------------------------------------------------------------------

test('rewrites a Markdown link to the canonical base route', { skip }, async () => {
  const tree = root(paragraph(link('./index.md#foundation-validation')));
  const result = await apply(tree);
  assert.equal(result.threw, false, `unexpected failure: ${result.error?.message}`);
  assert.equal(linksIn(tree)[0].url, '/agentic-sdd/#foundation-validation');
});

test('rewrites a reference definition used by reference links', { skip }, async () => {
  const linkReference = { type: 'linkReference', identifier: 'cli', referenceType: 'full', children: [text('cli')] };
  const definition = { type: 'definition', identifier: 'cli', url: './reference/cli.md' };
  const tree = root(paragraph(linkReference), definition);
  const result = await apply(tree);
  assert.equal(result.threw, false, `unexpected failure: ${result.error?.message}`);
  assert.equal(definition.url, '/agentic-sdd/reference/cli/');
  // A reference link keeps its identifier; the definition owns the URL.
  assert.equal(linkReference.url, undefined);
});

test('leaves external links unchanged', { skip }, async () => {
  const tree = root(paragraph(link('https://example.com/guide?q=1#frag')));
  await apply(tree);
  assert.equal(linksIn(tree)[0].url, 'https://example.com/guide?q=1#frag');
});

// ---------------------------------------------------------------------------
// Non-link nodes are not mutated
// ---------------------------------------------------------------------------

test('does not touch image nodes or image references', { skip }, async () => {
  const image = { type: 'image', url: './assets/diagram.png', alt: 'diagram' };
  const imageRef = { type: 'imageReference', identifier: 'd', referenceType: 'full', alt: 'd' };
  const tree = root(paragraph(image, imageRef));
  await apply(tree);
  assert.equal(image.url, './assets/diagram.png');
  assert.equal(imageRef.url, undefined);
});

test('does not touch code, inline code or prose', { skip }, async () => {
  const ghostLink = { type: 'link', url: './reference/cli.md', children: [text('ghost')] };
  const tree = root(
    { type: 'code', value: '[x](./reference/cli.md)', lang: 'md' },
    paragraph({ type: 'inlineCode', value: '[x](./reference/cli.md)' }),
    paragraph(text('Literal prose mentioning ./reference/cli.md stays as written.')),
  );
  await apply(tree);
  assert.equal(tree.children[0].value, '[x](./reference/cli.md)');
  assert.equal(tree.children[1].children[0].value, '[x](./reference/cli.md)');
  assert.match(tree.children[2].children[0].value, /\.\/reference\/cli\.md/);
  assert.equal(ghostLink.url, './reference/cli.md', 'a detached node must not be visited');
});

test('does not rewrite benign raw HTML', { skip }, async () => {
  const html = { type: 'html', value: '<div class="note">plain note</div>' };
  const tree = root(html);
  await apply(tree);
  assert.equal(html.value, '<div class="note">plain note</div>');
});

// ---------------------------------------------------------------------------
// Refusals
// ---------------------------------------------------------------------------

test('rejects a raw HTML navigation link', { skip }, async () => {
  const tree = root({ type: 'html', value: '<a href="./reference/cli.md">CLI</a>' });
  const result = await apply(tree);
  const diagnostic = assertDiagnosed(result, 'raw HTML navigation anchor');
  assert.match(JSON.stringify(diagnostic), /html|anchor|navigation/i);
});

test('rejects duplicate heading slugs in the current source', { skip }, async () => {
  const tree = root(heading('Same title'), heading('Same title'));
  const result = await apply(tree);
  assertDiagnosed(result, 'duplicate heading slug');
});

test('rejects a missing source-file context', { skip }, async () => {
  const file = makeFile();
  delete file.path;
  if (Array.isArray(file.history)) file.history = [];
  const tree = root(paragraph(link('./index.md')));
  const result = await apply(tree, file);
  assertDiagnosed(result, 'missing vfile path');
});

test('rejects an unknown public Markdown target with file context', { skip }, async () => {
  const tree = root(paragraph(link('./missing.md')));
  const result = await apply(tree);
  const diagnostic = assertDiagnosed(result, 'unknown target');
  const shaped = JSON.stringify({ reason: diagnostic.reason ?? diagnostic.message, place: diagnostic.place });
  assert.match(shaped, /index\.md|missing\.md|missing/i);
});

test('rejects an unsafe scheme in page content', { skip }, async () => {
  const tree = root(paragraph(link('javascript:alert(1)')));
  const result = await apply(tree);
  assertDiagnosed(result, 'unsafe scheme');
});
