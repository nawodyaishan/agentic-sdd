import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { collectBaseline } from '../scripts/lib/manifest.mjs';

const repoRoot = path.resolve(fileURLToPath(new URL('../..', import.meta.url)));
const snapshotPath = path.join(repoRoot, 'website/tests/baseline/real-guides.json');
const snapshot = JSON.parse(readFileSync(snapshotPath, 'utf8'));

test('records the pre-migration old-guide heading and link baseline', () => {
  assert.equal(snapshot.schemaVersion, 1);
  assert.equal(snapshot.snapshotCommit, '2fc31e9f9c4d019e2b9f3b4f004978028808c74b');
  assert.deepEqual(snapshot.paths, [
    'docs/README.md',
    'docs/getting-started.md',
    'docs/installer-reference.md',
    'docs/guide-flows.md',
    'docs/execution-runs.md',
    'docs/orchestration-guide.md',
    'docs/mcp-tools.md',
    'docs/fleet-manager-tui.md',
    'docs/tui-architecture.md',
  ]);
  assert.equal(snapshot.files.length, snapshot.paths.length);
  assert.deepEqual(snapshot.files.map((file) => file.path), snapshot.paths);
  for (const file of snapshot.files) {
    assert.ok(Array.isArray(file.headings) && file.headings.length > 0, `${file.path} has no headings`);
    assert.ok(Array.isArray(file.links), `${file.path} has no link array`);
  }
});

test('current guides still match the recorded baseline before migration', () => {
  const current = collectBaseline(repoRoot, snapshot.paths);
  assert.deepEqual(current.files, snapshot.files);
});
