// SPEC020 B4-W1-F — focused tests for the docs impact report.
//
// The report is an observation only: it reads the C1 manifest, never writes
// it, never calls a model and never touches the network. These tests pin the
// no-impact / tracked-gap taxonomy, the one-observation-per-path rule and the
// deterministic ordering and exit contract.

import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import {
  ImpactError,
  collectGitChanges,
  formatReport,
  impactReport,
  normalizeRepoPath,
  parseChangedPaths,
} from '../scripts/impact.mjs';
import { computeReviewDigest, validateManifest } from '../scripts/lib/manifest.mjs';
import { SHA_A, manifest as baseManifest } from './manifest-fixtures.mjs';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const impactScript = path.resolve(testDir, '../scripts/impact.mjs');
const source = fs.readFileSync(impactScript, 'utf8');
const SOURCE_CHECK = 'npm --prefix website run docs:check';

function fixturePage({ id, impactPaths, sources = [] }) {
  const evidence = sources.map((evidencePath) => ({ path: evidencePath, sha256: SHA_A }));
  const built = {
    id,
    source: `website/src/content/docs/${id}.md`,
    type: 'reference',
    audience: ['maintainer'],
    class: 'C',
    sources: evidence,
    reviewDigest: computeReviewDigest(evidence),
    reviewStatus: 'unreviewed',
    appliesTo: 'unreleased',
  };
  if (impactPaths !== undefined) built.impactPaths = impactPaths;
  return built;
}

function fixtureManifest(pages) {
  return baseManifest({ pages });
}

function makeTempDir(t) {
  fs.mkdirSync('/tmp/opencode', { recursive: true });
  const root = fs.mkdtempSync('/tmp/opencode/impact-w1f-');
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  return root;
}

function expectImpactError(fn, code) {
  assert.throws(fn, (error) => {
    assert.ok(error instanceof ImpactError, `expected ImpactError, got ${error?.name}: ${error?.message}`);
    assert.equal(error.code, code, `expected code ${code}, got ${error.code}: ${error.message}`);
    assert.ok(error.message.length > 0);
    return true;
  });
}

test('classifies engineering records and automation as explicit no-impact', () => {
  const report = impactReport({
    manifest: fixtureManifest([fixturePage({ id: 'reference/cli', sources: ['cmd/agentic-sdd/main.go'] })]),
    changedPaths: [
      'specs/020-documentation-ecosystem/spec.md',
      '.github/workflows/ci.yml',
      'website/package-lock.json',
      'docs/handovers/2026-11-14_020.md',
      'website/tests/impact-report.test.mjs',
    ],
  });
  assert.equal(report.summary.mapped, 0);
  assert.equal(report.summary['no-impact'], 5);
  assert.equal(report.summary['tracked-gap'], 0);
  for (const observation of report.observations) {
    assert.equal(observation.category, 'no-impact');
    assert.equal(observation.check, 'none');
    assert.equal(typeof observation.reason, 'string');
    assert.ok(observation.reason.length > 0);
    assert.equal(observation.pages, undefined);
  }
  assert.match(report.observations.find((item) => item.path.startsWith('specs/')).reason, /engineering specification record/);
  assert.match(report.observations.find((item) => item.path.startsWith('.github/')).reason, /CI and automation/);
});

test('maps a changed evidence or impactPaths surface to its page IDs and named check', () => {
  const report = impactReport({
    manifest: fixtureManifest([
      fixturePage({ id: 'reference/cli', sources: ['cmd/agentic-sdd/main.go', 'cmd/agentic-sdd/helptext.go'] }),
      fixturePage({ id: 'reference/fleet', impactPaths: ['internal/fleet/**', 'website/scripts/reference.mjs'] }),
      fixturePage({ id: 'reference/execution', impactPaths: ['internal/execution/'] }),
    ]),
    changedPaths: ['cmd/agentic-sdd/helptext.go', 'internal/fleet/parse.go', 'website/scripts/reference.mjs'],
  });
  assert.equal(report.summary.mapped, 3);
  assert.deepEqual(
    report.observations.map((observation) => [observation.path, observation.pages, observation.check]),
    [
      ['cmd/agentic-sdd/helptext.go', ['reference/cli'], SOURCE_CHECK],
      ['internal/fleet/parse.go', ['reference/fleet'], SOURCE_CHECK],
      ['website/scripts/reference.mjs', ['reference/fleet'], SOURCE_CHECK],
    ],
  );
  // A trailing "/" is an explicit prefix; "**" spans nested segments.
  assert.deepEqual(
    impactReport({
      manifest: fixtureManifest([
        fixturePage({ id: 'reference/execution', impactPaths: ['internal/execution/'] }),
        fixturePage({ id: 'reference/fleet', impactPaths: ['internal/fleet/**'] }),
      ]),
      changedPaths: ['internal/execution/deep/nested/checks.go', 'internal/fleet/a/b/c.go'],
    }).observations.map((observation) => [observation.path, observation.pages]),
    [
      ['internal/execution/deep/nested/checks.go', ['reference/execution']],
      ['internal/fleet/a/b/c.go', ['reference/fleet']],
    ],
  );
});

test('a changed surface with no mapped page is a tracked gap, never ignored', () => {
  const report = impactReport({
    manifest: fixtureManifest([fixturePage({ id: 'reference/cli', sources: ['cmd/agentic-sdd/main.go'] })]),
    changedPaths: ['README.md', 'Taskfile.yml', 'internal/newpkg/thing.go', 'CHANGELOG.md'],
  });
  assert.equal(report.summary['tracked-gap'], 4);
  assert.equal(report.summary.mapped, 0);
  const byPath = new Map(report.observations.map((observation) => [observation.path, observation]));
  assert.equal(byPath.get('Taskfile.yml').category, 'tracked-gap');
  assert.equal(byPath.get('Taskfile.yml').surface, 'developer commands');
  assert.equal(byPath.get('Taskfile.yml').owner, 'docs-maintainers');
  assert.equal(byPath.get('README.md').surface, 'public entry point');
  assert.equal(byPath.get('internal/newpkg/thing.go').surface, 'core service');
  assert.equal(byPath.get('CHANGELOG.md').surface, 'repository surface');
  for (const observation of report.observations) {
    assert.equal(observation.check, 'none');
    assert.match(observation.nextAction, /impactPaths/);
  }
});

test('classifies a changed Taskfile or .github file exactly once instead of skipping it', () => {
  const report = impactReport({
    manifest: fixtureManifest([fixturePage({ id: 'reference/cli', sources: ['cmd/agentic-sdd/main.go'] })]),
    changedPaths: ['Taskfile.yml', '.github/workflows/ci.yml', 'Taskfile.yml'],
  });
  assert.equal(report.changed, 2, 'duplicate input paths collapse to one observation');
  assert.deepEqual(
    report.observations.map((observation) => [observation.path, observation.category]),
    [
      ['.github/workflows/ci.yml', 'no-impact'],
      ['Taskfile.yml', 'tracked-gap'],
    ],
  );
});

test('orders observations by path and page IDs by id regardless of input order', () => {
  const manifest = fixtureManifest([
    fixturePage({ id: 'reference/mcp', sources: ['internal/skillsync/mcp.go'] }),
    fixturePage({ id: 'reference/cli', sources: ['internal/skillsync/mcp.go'] }),
    fixturePage({ id: 'reference/fleet', impactPaths: ['internal/fleet/**'] }),
  ]);
  const forward = impactReport({
    manifest,
    changedPaths: ['internal/skillsync/mcp.go', 'internal/fleet/fleet.go', 'README.md', 'audits/x.md'],
  });
  const reversed = impactReport({
    manifest,
    changedPaths: ['audits/x.md', 'README.md', 'internal/fleet/fleet.go', 'internal/skillsync/mcp.go'],
  });
  assert.deepEqual(forward, reversed);
  assert.deepEqual(
    forward.observations.map((observation) => observation.path),
    ['README.md', 'audits/x.md', 'internal/fleet/fleet.go', 'internal/skillsync/mcp.go'].sort(),
  );
  assert.deepEqual(
    forward.observations.find((observation) => observation.path === 'internal/skillsync/mcp.go').pages,
    ['reference/cli', 'reference/mcp'],
  );
  assert.equal(formatReport(forward), formatReport(reversed));
  assert.equal(formatReport(forward), formatReport(impactReport({ manifest, changedPaths: forward.observations.map((item) => item.path) })));
});

test('one observation per changed surface, no cascade through shared or page sources', () => {
  const report = impactReport({
    manifest: fixtureManifest([
      fixturePage({ id: 'reference/cli', sources: ['shared/facts.md', 'reference/cli.md'] }),
      fixturePage({ id: 'reference/fleet', sources: ['reference/cli.md', 'reference/fleet.md'] }),
    ]),
    changedPaths: ['shared/facts.md'],
  });
  assert.equal(report.changed, 1);
  assert.deepEqual(report.observations[0].pages, ['reference/cli']);
});

test('a registry edit is observed against every registered page', () => {
  const report = impactReport({
    manifest: fixtureManifest([
      fixturePage({ id: 'reference/cli', sources: ['cmd/agentic-sdd/main.go'] }),
      fixturePage({ id: 'reference/fleet', sources: ['internal/fleet/fleet.go'] }),
    ]),
    changedPaths: ['website/docs-manifest.json'],
  });
  assert.equal(report.summary.mapped, 1);
  assert.deepEqual(report.observations[0].pages, ['reference/cli', 'reference/fleet']);
});

test('reports zero changes as a clean empty report', () => {
  const report = impactReport({ manifest: fixtureManifest([fixturePage({ id: 'index', sources: ['README.md'] })]), changedPaths: [] });
  assert.equal(report.changed, 0);
  assert.deepEqual(report.summary, { mapped: 0, 'no-impact': 0, 'tracked-gap': 0 });
  assert.deepEqual(report.observations, []);
  assert.match(formatReport(report), /nothing to report/);
});

test('fails closed on unsafe changed paths and unknown manifest fields', () => {
  const manifest = fixtureManifest([fixturePage({ id: 'reference/cli', sources: ['cmd/agentic-sdd/main.go'] })]);
  for (const changedPath of ['/etc/passwd', '../outside.md', 'cmd\\agentic-sdd\\main.go', 'cmd//main.go']) {
    expectImpactError(() => impactReport({ manifest, changedPaths: [changedPath] }), 'invalid-path');
  }
  expectImpactError(() => impactReport({ manifest }), 'invalid-changed-paths');
  expectImpactError(
    () => impactReport({ manifest: fixtureManifest([fixturePage({ id: 'reference/cli', sources: ['a.md'], impactPaths: ['specs/{a,b}/**'] })]), changedPaths: [] }),
    'unsupported-impact-pattern',
  );
  expectImpactError(
    () => impactReport({ manifest: fixtureManifest([fixturePage({ id: 'reference/cli', sources: ['a.md'], impactPaths: [{ pattern: 'a/**' }] })]), changedPaths: [] }),
    'invalid-impact-path',
  );
  assert.throws(
    () =>
      validateManifest(
        fixtureManifest([{ ...fixturePage({ id: 'reference/cli', sources: ['a.md'] }), impactPath: ['a/**'] }]),
      ),
    (error) => error.name === 'ManifestError' && error.code === 'unknown-field',
  );
});

test('normalizes changed-path input and ignores comments and blank lines', () => {
  assert.equal(normalizeRepoPath('./cmd/agentic-sdd/main.go'), 'cmd/agentic-sdd/main.go');
  assert.deepEqual(parseChangedPaths('# comment\r\n\r\ncmd/a.go\r\nspecs/b.md\n'), ['cmd/a.go', 'specs/b.md']);
});

test('reads a git diff range without a network call', (t) => {
  if (spawnSync('git', ['--version']).status !== 0) {
    t.skip('git is unavailable in this environment');
    return;
  }
  const root = makeTempDir(t);
  const git = (...args) =>
    spawnSync('git', ['-C', root, '-c', 'user.name=Impact Test', '-c', 'user.email=impact@example.invalid', '-c', 'commit.gpgsign=false', ...args], {
      encoding: 'utf8',
    });
  assert.equal(git('init', '-q').status, 0);
  fs.writeFileSync(path.join(root, 'README.md'), 'first\n');
  assert.equal(git('add', 'README.md').status, 0);
  assert.equal(git('commit', '-q', '-m', 'first').status, 0);
  fs.writeFileSync(path.join(root, 'README.md'), 'second\n');
  fs.writeFileSync(path.join(root, 'new file.go'), 'package main\n');
  assert.equal(git('add', 'new file.go').status, 0);
  assert.deepEqual(collectGitChanges({ repoRoot: root, base: 'HEAD' }).sort(), ['README.md', 'new file.go']);
  expectImpactError(() => collectGitChanges({ repoRoot: root, base: '--cached' }), 'invalid-base');
  expectImpactError(() => collectGitChanges({ repoRoot: root, base: 'no-such-revision' }), 'git-diff-failed');
});

test('CLI prints a deterministic report and exits 0, reserving non-zero for usage errors', (t) => {
  const root = makeTempDir(t);
  fs.mkdirSync(path.join(root, 'website'), { recursive: true });
  fs.writeFileSync(
    path.join(root, 'website/docs-manifest.json'),
    `${JSON.stringify(fixtureManifest([fixturePage({ id: 'reference/cli', sources: ['cmd/agentic-sdd/main.go'] })]), null, 2)}\n`,
  );
  const pathsFile = path.join(root, 'changed-paths.txt');
  fs.writeFileSync(pathsFile, '# changed paths\ncmd/agentic-sdd/main.go\nTaskfile.yml\n.github/workflows/ci.yml\n');

  const run = (...args) => spawnSync(process.execPath, [impactScript, '--repo', root, ...args], { encoding: 'utf8' });
  const first = run('--paths', pathsFile, '--json');
  const second = run('--paths', pathsFile, '--json');
  assert.equal(first.status, 0, first.stderr);
  assert.equal(first.stdout, second.stdout);
  const report = JSON.parse(first.stdout);
  assert.deepEqual(report.summary, { mapped: 1, 'no-impact': 1, 'tracked-gap': 1 });
  assert.deepEqual(
    report.observations.map((observation) => [observation.path, observation.category]),
    [
      ['.github/workflows/ci.yml', 'no-impact'],
      ['Taskfile.yml', 'tracked-gap'],
      ['cmd/agentic-sdd/main.go', 'mapped'],
    ],
  );

  const text = run('--paths', pathsFile);
  assert.equal(text.status, 0);
  assert.match(text.stdout, /^docs impact: 3 changed path\(s\)/);
  assert.match(text.stdout, /check: npm --prefix website run docs:check/);

  for (const failure of [run(), run('--paths'), run('--unknown')]) {
    assert.equal(failure.status, 1);
    assert.match(failure.stderr, /^impact: /);
  }
  assert.equal(run('--help').status, 0);
});

test('the script stays offline, read-only and free of model calls', () => {
  for (const forbidden of ['node:http', 'node:https', 'node:net', 'node:dns', 'fetch(', 'writeFile', 'appendFile', 'rmSync', 'unlinkSync', 'process.env']) {
    assert.equal(source.includes(forbidden), false, `impact.mjs must not use ${forbidden}`);
  }
  assert.ok(source.includes("'git'") && source.includes("'--name-only'"), 'git diff is the only child process');
});
