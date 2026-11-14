// Spec 020 B1/T02 lane B1-W2-B: negative catalog fixtures. Each test copies the
// real canonical trees into an owned temporary directory and applies exactly
// one mutation, so a failure proves the specific guard rather than a count
// mismatch or a missing sibling tree.

import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

import {
  catalogsModulePath,
  copyCanonicalTree,
  profileFile,
  replaceWithOutsideSymlink,
  skillFile,
  writeTreeFile,
} from './reference-catalogs-support.mjs';

const { collectCatalogs, CatalogError } = await import(catalogsModulePath);

function expectCatalogError(run, code, rel) {
  assert.throws(run, (error) => {
    assert.ok(error instanceof CatalogError, `expected CatalogError, got ${error?.constructor?.name}: ${error?.message}`);
    assert.equal(error.code, code, `unexpected code: ${error.message}`);
    if (rel !== undefined) {
      assert.equal(error.path, rel, `unexpected error path: ${error.path}`);
      assert.match(error.message, new RegExp(rel.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
    }
    return true;
  });
}

test('a symlinked repository root is refused', (t) => {
  const root = copyCanonicalTree(t);
  const alias = path.join(path.dirname(root), `${path.basename(root)}-alias`);
  fs.symlinkSync(root, alias, 'dir');
  t.after(() => fs.rmSync(alias, { force: true }));
  expectCatalogError(() => collectCatalogs({ repoRoot: alias }), 'unsafe-path');
});

test('missing canonical source directories fail closed', (t) => {
  const noSkills = copyCanonicalTree(t);
  fs.rmSync(path.join(noSkills, 'skills-files'), { recursive: true, force: true });
  expectCatalogError(() => collectCatalogs({ repoRoot: noSkills }), 'missing-source', 'skills-files');

  const noPack = copyCanonicalTree(t);
  fs.rmSync(path.join(noPack, 'opencode-pack'), { recursive: true, force: true });
  expectCatalogError(() => collectCatalogs({ repoRoot: noPack }), 'missing-source', 'opencode-pack');
});

test('malformed YAML frontmatter is rejected through the pinned parser', (t) => {
  const root = copyCanonicalTree(t);
  const rel = 'skills-files/agentic-sdd-router/SKILL.md';
  writeTreeFile(root, rel, '---\nname: agentic-sdd-router\ndescription: [unterminated\n---\n\n# body\n');
  expectCatalogError(() => collectCatalogs({ repoRoot: root }), 'invalid-frontmatter', rel);
});

test('a file without frontmatter delimiters is rejected', (t) => {
  const root = copyCanonicalTree(t);
  const rel = 'opencode-pack/agents/sdd-test-design.md';
  writeTreeFile(root, rel, '# no frontmatter here\n');
  expectCatalogError(() => collectCatalogs({ repoRoot: root }), 'invalid-frontmatter', rel);
});

test('missing skill description and name/directory mismatch are rejected', (t) => {
  const missing = copyCanonicalTree(t);
  const rel = 'skills-files/agentic-sdd-plan/SKILL.md';
  writeTreeFile(missing, rel, '---\nname: agentic-sdd-plan\n---\n\n# body\n');
  expectCatalogError(() => collectCatalogs({ repoRoot: missing }), 'missing-metadata', rel);

  const mismatch = copyCanonicalTree(t);
  const other = 'skills-files/agentic-sdd-plan/SKILL.md';
  writeTreeFile(mismatch, other, skillFile({ name: 'agentic-sdd-not-the-directory' }));
  expectCatalogError(() => collectCatalogs({ repoRoot: mismatch }), 'invalid-metadata', other);
});

test('an unpinned model is only allowed for the orchestrator', (t) => {
  const root = copyCanonicalTree(t);
  const rel = 'opencode-pack/agents/sdd-debug-diagnose.md';
  writeTreeFile(root, rel, profileFile({}));
  expectCatalogError(() => collectCatalogs({ repoRoot: root }), 'missing-metadata', rel);
});

test('unknown mode and malformed permission rules are rejected', (t) => {
  const badMode = copyCanonicalTree(t);
  const modeRel = 'opencode-pack/agents/sdd-test-design.md';
  writeTreeFile(badMode, modeRel, profileFile({ mode: 'writer', model: 'opencode/deepseek-v4.1-flash-free' }));
  expectCatalogError(() => collectCatalogs({ repoRoot: badMode }), 'invalid-metadata', modeRel);

  const noRules = copyCanonicalTree(t);
  const rulesRel = 'opencode-pack/agents/sdd-test-design.md';
  writeTreeFile(noRules, rulesRel, profileFile({ model: 'opencode/deepseek-v4.1-flash-free', permissions: false }));
  expectCatalogError(() => collectCatalogs({ repoRoot: noRules }), 'missing-metadata', rulesRel);

  const badEffect = copyCanonicalTree(t);
  const effectRel = 'opencode-pack/agents/sdd-test-design.md';
  writeTreeFile(
    badEffect,
    effectRel,
    '---\ndescription: "Fixture."\nmode: subagent\nmodel: opencode/deepseek-v4.1-flash-free\npermissions:\n  - action: read\n    resource: "*"\n    effect: maybe\n---\n\nbody\n',
  );
  expectCatalogError(() => collectCatalogs({ repoRoot: badEffect }), 'invalid-metadata', effectRel);
});

test('a profile ID duplicated between adapters and agents is rejected', (t) => {
  const root = copyCanonicalTree(t);
  fs.copyFileSync(
    path.join(root, 'opencode-pack/adapters/sdd-spec-9r.md'),
    path.join(root, 'opencode-pack/agents/sdd-spec-9r.md'),
  );
  expectCatalogError(() => collectCatalogs({ repoRoot: root }), 'duplicate-id');
});

test('a filename that is not a safe catalog ID is rejected', (t) => {
  const root = copyCanonicalTree(t);
  writeTreeFile(
    root,
    'opencode-pack/agents/bad name.md',
    profileFile({ model: 'opencode/deepseek-v4.1-flash-free' }),
  );
  expectCatalogError(() => collectCatalogs({ repoRoot: root }), 'unsafe-path');
});

test('symlinked profile and skill files are refused instead of followed', (t) => {
  const profileRoot = copyCanonicalTree(t);
  replaceWithOutsideSymlink(
    profileRoot,
    'opencode-pack/agents/sdd-test-design.md',
    profileFile({ model: 'opencode/deepseek-v4.1-flash-free' }),
  );
  expectCatalogError(
    () => collectCatalogs({ repoRoot: profileRoot }),
    'unsafe-path',
    'opencode-pack/agents/sdd-test-design.md',
  );

  const skillRoot = copyCanonicalTree(t);
  replaceWithOutsideSymlink(
    skillRoot,
    'skills-files/agentic-sdd-plan/SKILL.md',
    skillFile({ name: 'agentic-sdd-plan' }),
  );
  expectCatalogError(() => collectCatalogs({ repoRoot: skillRoot }), 'unsafe-path', 'skills-files/agentic-sdd-plan/SKILL.md');
});

test('catalog count drift fails with an actionable acceptance message', (t) => {
  const root = copyCanonicalTree(t);
  fs.rmSync(path.join(root, 'opencode-pack/agents/sdd-lightweight.md'));
  expectCatalogError(() => collectCatalogs({ repoRoot: root }), 'count-mismatch');
  try {
    collectCatalogs({ repoRoot: root });
  } catch (error) {
    assert.match(error.message, /14/);
    assert.match(error.message, /13/);
    assert.match(error.message, /B1\/T02/);
  }
});

test('collection never mutates the fixture tree', (t) => {
  const root = copyCanonicalTree(t);
  const snapshot = (dir) => {
    const entries = [];
    const walk = (current) => {
      for (const entry of fs.readdirSync(current, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
        const absolute = path.join(current, entry.name);
        if (entry.isDirectory()) walk(absolute);
        else entries.push(`${path.relative(root, absolute)}:${fs.statSync(absolute).mtimeMs}`);
      }
    };
    walk(dir);
    return entries;
  };
  const before = snapshot(root);
  collectCatalogs({ repoRoot: root });
  assert.deepEqual(snapshot(root), before);
});
