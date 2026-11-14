// Shared fixture glue for the Spec 020 B1/T02 catalog collector tests.
//
// Scope: build disposable copies of the real canonical source trees and apply
// one mutation per test. It never re-implements frontmatter parsing or catalog
// facts; expectations come from the frozen worker contract, plan C2 and the
// existing Go distribution contracts in internal/skillsync.
//
// This file contains no tests. When the `reference-catalogs-*` glob is passed
// to `node --test`, it is loaded as an empty test file, which is harmless.

import fs from 'node:fs';
import { mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const testsDir = path.dirname(fileURLToPath(import.meta.url));

export const repoRoot = path.resolve(testsDir, '..', '..');
export const catalogsModulePath = path.join(repoRoot, 'website', 'scripts', 'reference', 'catalogs.mjs');

/** Create a disposable tree with the real skills-files/ and opencode-pack/. */
export function copyCanonicalTree(t) {
  const root = mkdtempSync(path.join(tmpdir(), 'asdx-catalogs-'));
  fs.cpSync(path.join(repoRoot, 'skills-files'), path.join(root, 'skills-files'), {
    recursive: true,
    dereference: false,
  });
  fs.cpSync(path.join(repoRoot, 'opencode-pack'), path.join(root, 'opencode-pack'), {
    recursive: true,
    dereference: false,
  });
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  return root;
}

/** Rewrite one file inside a fixture tree, creating parents when needed. */
export function writeTreeFile(root, rel, contents) {
  const target = path.join(root, rel);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, contents, 'utf8');
}

/** Replace a fixture file with a symlink that leaves the declared tree. */
export function replaceWithOutsideSymlink(root, rel, contents) {
  const target = path.join(root, rel);
  const outside = path.join(root, '..', `${path.basename(root)}-outside-${path.basename(rel)}`);
  fs.writeFileSync(outside, contents, 'utf8');
  fs.rmSync(target, { force: true });
  fs.symlinkSync(outside, target);
  return outside;
}

/** A minimal, valid skill file body with caller-supplied frontmatter fields. */
export function skillFile({ name, description = 'Fixture skill description.', extra = '' }) {
  return `---\nname: ${name}\ndescription: ${JSON.stringify(description)}${extra}\n---\n\n# Fixture skill\n`;
}

/** A minimal, valid pack profile file with caller-supplied frontmatter. */
export function profileFile({
  description = 'Fixture profile description.',
  mode = 'subagent',
  model,
  permissions = true,
  extra = '',
}) {
  const lines = [`description: ${JSON.stringify(description)}`, `mode: ${mode}`];
  if (model !== undefined) lines.push(`model: ${model}`);
  if (extra) lines.push(extra.replace(/^\n+/, '').replace(/\n+$/, ''));
  if (permissions) {
    lines.push('permissions:\n  - action: "*"\n    resource: "*"\n    effect: deny\n  - action: read\n    resource: "*"\n    effect: allow');
  }
  return `---\n${lines.join('\n')}\n---\n\nFixture profile body.\n`;
}
