import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { collectBaseline } from '../scripts/lib/manifest.mjs';

const repoRoot = path.resolve(fileURLToPath(new URL('../..', import.meta.url)));
const snapshotPath = path.join(repoRoot, 'website/tests/baseline/real-guides.json');
const snapshot = JSON.parse(readFileSync(snapshotPath, 'utf8'));

// Frozen integration base and the commit that recorded the pre-migration
// inventory. The migrated working tree is captured, not a later commit.
const BASE_COMMIT = '449245a990225f2234685dee8314dff1c3b1d3b2';
const PRE_MIGRATION_COMMIT = '2fc31e9f9c4d019e2b9f3b4f004978028808c74b';

const GUIDE_PATHS = [
  'docs/README.md',
  'docs/getting-started.md',
  'docs/installer-reference.md',
  'docs/guide-flows.md',
  'docs/execution-runs.md',
  'docs/orchestration-guide.md',
  'docs/mcp-tools.md',
  'docs/fleet-manager-tui.md',
  'docs/tui-architecture.md',
];

// Reviewed T08 dispositions. A changed disposition is a reviewed migration
// decision, not a routine snapshot refresh.
const DISPOSITIONS = {
  'docs/README.md': 'retained-engineering-index',
  'docs/getting-started.md': 'retained-engineering-trace',
  'docs/installer-reference.md': 'migration-stub',
  'docs/guide-flows.md': 'retained-engineering-trace',
  'docs/execution-runs.md': 'retained-engineering-trace',
  'docs/orchestration-guide.md': 'migration-stub',
  'docs/mcp-tools.md': 'migration-stub',
  'docs/fleet-manager-tui.md': 'migration-stub',
  'docs/tui-architecture.md': 'retained-engineering-history',
};

test('records the post-migration guide snapshot and reviewed dispositions', () => {
  assert.equal(snapshot.schemaVersion, 2);
  assert.equal(snapshot.baseCommit, BASE_COMMIT);
  assert.equal(snapshot.preMigrationSnapshotCommit, PRE_MIGRATION_COMMIT);
  assert.deepEqual(snapshot.paths, GUIDE_PATHS);
  assert.equal(snapshot.files.length, snapshot.paths.length);
  assert.deepEqual(snapshot.files.map((file) => file.path), snapshot.paths);
  for (const file of snapshot.files) {
    assert.ok(Array.isArray(file.headings) && file.headings.length > 0, `${file.path} has no headings`);
    assert.ok(Array.isArray(file.links), `${file.path} has no link array`);
  }
  assert.deepEqual(snapshot.classification, DISPOSITIONS);
});

test('retained getting-started trace keeps the three bounded worked paths', () => {
  const guide = snapshot.files.find((file) => file.path === 'docs/getting-started.md');
  assert.ok(guide, 'docs/getting-started.md is missing from the snapshot');
  const headings = guide.headings.map((heading) => heading.text);
  for (const expected of [
    'Direct fix: no feature paperwork',
    'Feature: one combined approval and one batch',
    'Review and resume: acceptance is not continuation',
  ]) {
    assert.ok(headings.includes(expected), `docs/getting-started.md lost the worked path ${JSON.stringify(expected)}`);
  }
});

test('migration stubs keep no code fences or numbered procedures', () => {
  for (const [file, disposition] of Object.entries(DISPOSITIONS)) {
    if (disposition !== 'migration-stub') continue;
    const text = readFileSync(path.join(repoRoot, file), 'utf8');
    assert.equal(text.includes('```'), false, `${file} is a stub but contains a code fence`);
    assert.equal(/(^|\n) {0,7}\d+[.)][ \t]+\S/.test(text), false, `${file} is a stub but contains a numbered procedure step`);
  }
});

test('post-migration guide snapshot matches the current guides', () => {
  const current = collectBaseline(repoRoot, snapshot.paths);
  assert.deepEqual(current.files, snapshot.files);
});
