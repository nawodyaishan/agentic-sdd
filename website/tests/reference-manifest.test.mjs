import assert from 'node:assert/strict';
import { readdirSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { computeReviewDigest, loadManifest } from '../scripts/lib/manifest.mjs';

const repoRoot = path.resolve(fileURLToPath(new URL('../..', import.meta.url)));
const manifestPath = path.join(repoRoot, 'website/docs-manifest.json');
const docsRoot = path.join(repoRoot, 'website/src/content/docs');

function markdownFiles(dir, prefix = '') {
  const files = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const relative = prefix ? `${prefix}/${entry.name}` : entry.name;
    if (entry.isDirectory()) files.push(...markdownFiles(path.join(dir, entry.name), relative));
    else if (entry.isFile() && entry.name.endsWith('.md')) files.push(relative);
  }
  return files.sort();
}

test('the live manifest registers every built Markdown page exactly once', () => {
  const manifest = loadManifest(manifestPath, { repoRoot, verifySources: true });
  const ids = manifest.pages.map((page) => page.id);
  assert.equal(new Set(ids).size, ids.length, 'manifest page IDs must be unique');
  assert.deepEqual(ids.slice().sort(), [
    'index',
    'reference/cli',
    'reference/configuration',
    'reference/execution',
    'reference/files-and-backups',
    'reference/fleet',
    'reference/mcp',
    'reference/profiles',
    'reference/skills',
  ]);
  const sources = manifest.pages.map((page) => page.source);
  assert.equal(new Set(sources).size, sources.length, 'manifest sources must be unique');
  assert.deepEqual(
    manifest.pages.map((page) => page.source.replace('website/src/content/docs/', '')).sort(),
    markdownFiles(docsRoot),
    'every public Markdown page must be registered',
  );
});

test('reference pages declare complete current-feature coverage and valid evidence', () => {
  const manifest = loadManifest(manifestPath, { repoRoot, verifySources: true });
  for (const page of manifest.pages) {
    assert.equal(page.reviewDigest, computeReviewDigest(page.sources), `${page.id} review digest`);
    assert.ok(page.sources.length > 0, `${page.id} needs evidence inputs`);
    assert.equal(page.reviewStatus, 'unreviewed');
    assert.equal(page.appliesTo, 'unreleased');
    if (page.id.startsWith('reference/')) {
      assert.equal(page.type, 'reference', `${page.id} type`);
      const expectedClass = ['reference/cli', 'reference/files-and-backups', 'reference/skills'].includes(page.id) ? 'L' : 'C';
      assert.equal(page.class, expectedClass, `${page.id} class`);
      assert.ok(page.generatedSections?.length === 1, `${page.id} generated-section provenance`);
      const section = page.generatedSections[0];
      assert.equal(section.id, page.id);
      assert.match(section.producer, /^website\/scripts\/reference\/.+\.mjs$/);
      assert.equal(section.markerStart, `<!-- generated:${page.id}:start -->`);
      assert.equal(section.markerEnd, `<!-- generated:${page.id}:end -->`);
      assert.ok(section.inputs.length > 0, `${page.id} generated inputs`);
      for (const input of section.inputs) {
        assert.ok(page.sources.some((source) => source.path === input), `${page.id} must fingerprint generated input ${input}`);
      }
      assert.ok(page.sources.some((source) => source.path === page.source), `${page.id} must fingerprint its source`);
    }
  }
});
