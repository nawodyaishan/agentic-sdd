// C1 baseline inventory: deterministic headings/links for explicitly supplied
// public Markdown only. Fenced code and inline code are not structure; no
// recursive scan and no private-tree input.

import assert from 'node:assert/strict';
import {
  mkdirSync,
  mkdtempSync,
  realpathSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { ManifestError, collectBaseline } from '../scripts/lib/manifest.mjs';

const WORKTREE_ROOT = fileURLToPath(new URL('../../', import.meta.url));
const BASELINE_README = 'website/tests/baseline/README.md';
const BASELINE_GUIDE = 'website/tests/baseline/docs/getting-started.md';

function expectError(run, code, errorPath) {
  assert.throws(run, (error) => {
    assert.ok(error instanceof ManifestError, `expected ManifestError, got ${error?.constructor?.name}`);
    assert.equal(error.code, code, `code ${error.code} !== ${code}: ${error.message}`);
    if (errorPath !== undefined) {
      assert.equal(error.path, errorPath, `path ${error.path} !== ${errorPath}: ${error.message}`);
    }
    return true;
  });
}

function makeRepo(t) {
  const root = realpathSync(mkdtempSync(path.join(tmpdir(), 'spec020-baseline-')));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  mkdirSync(path.join(root, 'docs'), { recursive: true });
  mkdirSync(path.join(root, 'specs'), { recursive: true });
  writeFileSync(path.join(root, 'docs/old-guide.md'), '# Old guide\n\n[link](../README.md)\n');
  writeFileSync(path.join(root, 'README.md'), '# Root\n');
  writeFileSync(path.join(root, 'specs/internal.md'), '# Internal\n');
  return root;
}

test('inventories the README fixture headings and links in document order', () => {
  const result = collectBaseline(WORKTREE_ROOT, [BASELINE_README]);
  assert.deepEqual(result, {
    files: [
      {
        path: BASELINE_README,
        headings: [
          { line: 6, level: 1, text: 'Baseline fixture', slug: 'baseline-fixture' },
          { line: 11, level: 2, text: 'Reusable section', slug: 'reusable-section' },
          { line: 15, level: 2, text: 'Reusable section', slug: 'reusable-section-1' },
          { line: 26, level: 2, text: 'Setext heading', slug: 'setext-heading' },
        ],
        links: [
          { line: 8, href: 'docs/getting-started.md', text: 'relative guide' },
          { line: 9, href: '#fenced-code-is-not-an-index', text: 'anchor' },
          { line: 17, href: 'https://example.com/a(b)', text: 'external site' },
          { line: 31, href: 'https://example.com/spec', text: 'spec' },
        ],
      },
    ],
  });
});

test('inventories the nested guide fixture with encoded and query links', () => {
  const result = collectBaseline(WORKTREE_ROOT, [BASELINE_GUIDE]);
  assert.deepEqual(result.files, [
    {
      path: BASELINE_GUIDE,
      headings: [
        { line: 1, level: 1, text: 'Getting started', slug: 'getting-started' },
        { line: 6, level: 2, text: 'Paths and spaces', slug: 'paths-and-spaces' },
      ],
      links: [
        { line: 3, href: '../README.md#baseline-fixture', text: 'back link' },
        { line: 4, href: 'https://example.com/docs?q=1#frag', text: 'HTTPS reference' },
        { line: 8, href: '../baseline%20notes.md', text: 'Encoded space' },
        { line: 8, href: '../README.md?view=1', text: 'query' },
      ],
    },
  ]);
});

test('preserves explicit input order, deduplicates and stays deterministic', () => {
  const guideFirst = collectBaseline(WORKTREE_ROOT, [BASELINE_GUIDE, BASELINE_README]);
  assert.deepEqual(guideFirst.files.map((file) => file.path), [BASELINE_GUIDE, BASELINE_README]);

  const repeated = collectBaseline(WORKTREE_ROOT, [BASELINE_README, BASELINE_GUIDE, BASELINE_README]);
  assert.deepEqual(repeated.files.map((file) => file.path), [BASELINE_README, BASELINE_GUIDE]);
  assert.deepEqual(collectBaseline(WORKTREE_ROOT, [BASELINE_README, BASELINE_GUIDE, BASELINE_README]), repeated);
  assert.deepEqual(collectBaseline(WORKTREE_ROOT, []), { files: [] });
});

test('inventories explicitly supplied public guide paths outside the site root', (t) => {
  const root = makeRepo(t);
  writeFileSync(path.join(root, 'README.md'), '# Project\n\n[recipe](docs/getting-started.md)\n');
  writeFileSync(path.join(root, 'CONTRIBUTING.md'), '# Contributing\n\n[setup](docs/getting-started.md)\n');
  writeFileSync(path.join(root, 'docs/getting-started.md'), '## Install\n\n[back](../README.md#project)\n');

  const result = collectBaseline(root, ['README.md', 'CONTRIBUTING.md', 'docs/getting-started.md']);
  assert.deepEqual(result.files.map((file) => file.path), ['README.md', 'CONTRIBUTING.md', 'docs/getting-started.md']);
  assert.deepEqual(result.files[0].headings, [{ line: 1, level: 1, text: 'Project', slug: 'project' }]);
  assert.deepEqual(result.files[0].links, [{ line: 3, href: 'docs/getting-started.md', text: 'recipe' }]);
  assert.deepEqual(result.files[2].headings, [{ line: 1, level: 2, text: 'Install', slug: 'install' }]);
});

test('ignores fenced code, inline code, frontmatter, comments and images', () => {
  const readme = collectBaseline(WORKTREE_ROOT, [BASELINE_README]).files[0];
  assert.equal(readme.headings.some((heading) => heading.text === 'Not a heading'), false);
  assert.equal(readme.links.some((link) => link.href === 'fake.md'), false);
  assert.equal(readme.links.some((link) => link.href === 'inline.md'), false);
  assert.equal(readme.links.some((link) => link.href === 'commented.md'), false);
  assert.equal(readme.links.some((link) => link.href === 'assets/diagram.png'), false);
  assert.equal(readme.headings.some((heading) => heading.slug === 'title'), false, 'frontmatter is not content');
});

test('refuses directories, non-Markdown paths and private trees', (t) => {
  expectError(() => collectBaseline(WORKTREE_ROOT, ['website/tests/baseline']), 'invalid-baseline-path', 'website/tests/baseline');
  expectError(() => collectBaseline(WORKTREE_ROOT, ['README']), 'invalid-baseline-path', 'README');
  expectError(() => collectBaseline(WORKTREE_ROOT, ['README.mdx']), 'invalid-baseline-path', 'README.mdx');

  const root = makeRepo(t);
  mkdirSync(path.join(root, 'docs/directory.md'));
  expectError(() => collectBaseline(root, ['docs/directory.md']), 'not-a-regular-file', 'docs/directory.md');
  expectError(() => collectBaseline(root, ['specs/internal.md']), 'private-source-path', 'specs/internal.md');
  expectError(() => collectBaseline(root, ['docs/testing/report.md']), 'private-source-path', 'docs/testing/report.md');
  expectError(() => collectBaseline(root, ['backups/previous.md']), 'private-source-path', 'backups/previous.md');
});

test('refuses traversal, absolute paths, symlinks and missing files', (t) => {
  const root = makeRepo(t);
  for (const declared of ['../outside.md', 'docs/../README.md', '/README.md', 'docs\\old-guide.md', '']) {
    expectError(() => collectBaseline(root, [declared]), 'path-escape', declared);
  }
  expectError(() => collectBaseline(root, ['docs/missing.md']), 'missing-source-file', 'docs/missing.md');

  const target = path.join(root, 'docs/old-guide.md');
  symlinkSync(target, path.join(root, 'linked.md'));
  expectError(() => collectBaseline(root, ['linked.md']), 'symlink-path', 'linked.md');
});

test('refuses a malformed path list', (t) => {
  const root = makeRepo(t);
  expectError(() => collectBaseline(root, 'README.md'), 'invalid-baseline-list');
  expectError(() => collectBaseline(root, [42]), 'invalid-baseline-path', '[0]');
});
