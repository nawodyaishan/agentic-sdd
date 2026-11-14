import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(fileURLToPath(new URL('../..', import.meta.url)));
const snapshot = JSON.parse(readFileSync(path.join(repoRoot, 'website/tests/baseline/migration-anchors.json'), 'utf8'));

test('records the pre-migration README and integration anchor inventory', () => {
  assert.equal(snapshot.schemaVersion, 1);
  assert.equal(snapshot.snapshotCommit, '2fc31e9f9c4d019e2b9f3b4f004978028808c74b');
  assert.deepEqual(snapshot.paths, [
    'README.md',
    'CONTRIBUTING.md',
    'integrations/opencode/README.md',
    'integrations/opencode/adoption.md',
    'integrations/opencode/usage.md',
    'integrations/opencode/safety.md',
    'integrations/opencode/verification.md',
    'integrations/opencode/recovery.md',
    'integrations/opencode/supported-version.md',
    'opencode-pack/README.md',
    'opencode-pack/routing.md',
  ]);
  assert.equal(snapshot.files.length, snapshot.paths.length);
  for (const file of snapshot.files) {
    assert.ok(Array.isArray(file.headings) && file.headings.length > 0, `${file.path} headings`);
    assert.ok(Array.isArray(file.links), `${file.path} links`);
  }
  assert.ok(snapshot.files.find((file) => file.path === 'README.md').headings.some((heading) => heading.text === 'Quick start'));
  assert.ok(snapshot.files.find((file) => file.path === 'README.md').headings.some((heading) => heading.text === 'Where the files go'));
});
