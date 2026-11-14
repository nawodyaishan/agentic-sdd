import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, readFileSync, rmSync, symlinkSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';

const moduleURL = new URL('../scripts/reference.mjs', import.meta.url);

async function implementation() {
  try {
    return await import(moduleURL.href);
  } catch (error) {
    if (error.code === 'ERR_MODULE_NOT_FOUND' && error.url === moduleURL.href) return {};
    throw error;
  }
}

test('reference composer exposes the bounded section writer', async () => {
  const mod = await implementation();
  assert.equal(typeof mod.renderGeneratedPage, 'function', 'generated-section rendering is missing');
  assert.equal(typeof mod.applySections, 'function', 'check/generate composition is missing');
});

test('generated rendering preserves authored bytes outside exact markers', async () => {
  const mod = await implementation();
  assert.equal(typeof mod.renderGeneratedPage, 'function');
  const existing = '---\ntitle: CLI\n---\n\nAuthored introduction.\n\n<!-- generated:reference/cli:start -->\nold\n<!-- generated:reference/cli:end -->\n\nAuthored footer.\n';
  const rendered = mod.renderGeneratedPage({
    pageId: 'reference/cli',
    title: 'CLI reference',
    markdown: 'new facts\n',
    existing,
  });
  assert.equal(rendered, existing.replace('\nold\n', '\nnew facts\n'));
  assert.equal(mod.renderGeneratedPage({ pageId: 'reference/cli', title: 'CLI reference', markdown: 'new facts\n', existing: rendered }), rendered);
});

test('malformed duplicate and source-traversal sections fail closed', async () => {
  const mod = await implementation();
  assert.equal(typeof mod.renderGeneratedPage, 'function');
  const section = { pageId: 'reference/cli', title: 'CLI', markdown: 'facts' };
  assert.throws(() => mod.renderGeneratedPage({ ...section, pageId: '../README' }), /page|route|unsafe/i);
  assert.throws(() => mod.renderGeneratedPage({ ...section, existing: '<!-- generated:reference/cli:start -->\nold\n' }), /marker|section/i);
  assert.throws(() => mod.renderGeneratedPage({ ...section, existing: '<!-- generated:reference/cli:start --><!-- generated:reference/cli:start --><!-- generated:reference/cli:end -->' }), /marker|section/i);
  assert.throws(() => mod.renderGeneratedPage({ ...section, markdown: '<!-- generated:other:start -->' }), /marker|section/i);
});

test('check mode reports drift without writes and generation becomes a real no-op', async () => {
  const mod = await implementation();
  assert.equal(typeof mod.applySections, 'function');
  const root = mkdtempSync(path.join(tmpdir(), 'asdx-reference-test-'));
  try {
    mkdirSync(path.join(root, 'website/src/content/docs/reference'), { recursive: true });
    const target = path.join(root, 'website/src/content/docs/reference/cli.md');
    writeFileSync(target, '---\ntitle: CLI\n---\n\nAuthored only.\n');
    const before = readFileSync(target, 'utf8');
    const sections = [{ pageId: 'reference/cli', title: 'CLI reference', markdown: 'facts\n', sources: [] }];
    const check = mod.applySections({ repoRoot: root, sections, mode: 'check' });
    assert.deepEqual(check.changed, ['website/src/content/docs/reference/cli.md']);
    assert.equal(readFileSync(target, 'utf8'), before);
    mod.applySections({ repoRoot: root, sections, mode: 'generate' });
    assert.match(readFileSync(target, 'utf8'), /Authored only/);
    assert.deepEqual(mod.applySections({ repoRoot: root, sections, mode: 'check' }).changed, []);
    assert.throws(() => mod.applySections({ repoRoot: root, sections, mode: 'write-whatever' }), /mode/i);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test('a symlink repository root is refused instead of reading outside the declared root', async () => {
  const mod = await implementation();
  const fixture = mkdtempSync(path.join(tmpdir(), 'asdx-reference-root-'));
  try {
    const actual = path.join(fixture, 'actual');
    const alias = path.join(fixture, 'alias');
    mkdirSync(actual);
    symlinkSync(actual, alias, 'dir');
    assert.throws(() => mod.applySections({
      repoRoot: alias,
      sections: [{ pageId: 'reference/cli', title: 'CLI', markdown: 'facts' }],
      mode: 'check',
    }), /symlink|root|unsafe/i);
  } finally {
    rmSync(fixture, { recursive: true, force: true });
  }
});
