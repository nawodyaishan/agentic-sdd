// Offline checks for the execution example fixtures (Spec 020 B9-W1-B, T21, FR-16.3).
//
// The fixtures under tests/examples/execution/ are hand-written shape examples.
// This file verifies their documented kinds, IDs, states and reasons, runs the
// inert fake check only inside a disposable copy, and confirms the fixture tree
// is not executable, names no real home or URL and carries no credential-shaped
// text. Nothing here calls a model, an MCP server or the network, and no test
// resolves a command through PATH or writes to a real home.
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { cpSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, statSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const fixtureRoot = path.resolve(fileURLToPath(new URL('./examples/execution/', import.meta.url)));

const RUN_ID = /^run-\d{8}T\d{6}Z-[0-9a-f]{6}$/;
const CHECK_ID = /^chk-\d{8}T\d{6}Z-[0-9a-f]{6}$/;
const PACKAGE_ID = /^pkg-\d{8}T\d{6}Z-[0-9a-f]{6}$/;

// Vocabularies copied from the execution reference and the service constants.
const CHECKPOINT_STATES = new Set([
  'prepared', 'running', 'observed-partial', 'observed-failed',
  'awaiting-verification', 'verified-evidence', 'stale', 'blocked',
]);
const CHECK_RESULTS = new Set(['running', 'passed', 'failed', 'timed_out', 'interrupted', 'output_limit', 'not_started']);
const CLAIM_RESULTS = new Set(['passed', 'failed', 'partial', 'blocked', 'needs_input']);
const INVESTIGATIONS = new Set([
  'none', 'inspect_changed_paths', 'rerun_authorized_check', 'reconcile_source_change', 'resolve_blocker',
]);
const PARTIAL_REASONS = new Set([
  'base_not_ancestor', 'head_not_current', 'index_conflict', 'sensitive_path',
  'max_file_bytes_exceeded', 'max_total_bytes_exceeded', 'unreadable_path',
]);

function readJson(name) {
  return JSON.parse(readFileSync(path.join(fixtureRoot, name), 'utf8'));
}

function walk(dir) {
  const files = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...walk(full));
    } else {
      files.push(full);
    }
  }
  return files;
}

function makeTemp(t) {
  const dir = mkdtempSync(path.join(tmpdir(), 'asdx-examples-execution-'));
  t.after(() => rmSync(dir, { recursive: true, force: true }));
  return dir;
}

test('run identity, status and checkpoint use documented kinds, IDs and states', () => {
  const identity = readJson('run-identity.json');
  assert.equal(identity.schemaVersion, 1);
  assert.equal(identity.kind, 'execution.run.identity');
  assert.match(identity.run_id, RUN_ID);
  assert.equal(identity.scope.kind, 'feature');
  assert.ok(identity.run_dir.startsWith(`${identity.run_root}/`), 'run_dir must sit under run_root');

  const status = readJson('status.json');
  assert.equal(status.schemaVersion, 1);
  assert.equal(status.kind, 'execution.run.status');
  assert.equal(status.run_id, identity.run_id);
  assert.ok(CHECKPOINT_STATES.has(status.state), `unknown state ${status.state}`);
  assert.ok(INVESTIGATIONS.has(status.next_permitted_investigation));
  assert.doesNotMatch(JSON.stringify(status), /approved|resume|start next batch/i);

  const checkpoint = readJson('checkpoint.json');
  assert.equal(checkpoint.schemaVersion, 1);
  assert.equal(checkpoint.run_id, identity.run_id);
  assert.ok(CHECKPOINT_STATES.has(checkpoint.state), `unknown checkpoint state ${checkpoint.state}`);
  for (const check of checkpoint.checks) {
    assert.match(check.check_id, CHECK_ID);
    assert.ok(CHECK_RESULTS.has(check.result), `unknown check result ${check.result}`);
    assert.ok(check.summary.length <= 40, 'summary exceeds the documented 40-line cap');
  }
});

test('check metadata, argv and caller claim match the documented shapes', () => {
  const checkpoint = readJson('checkpoint.json');
  const expected = readJson('expected-check.json');
  const meta = readJson('check-meta.json');
  assert.equal(meta.schemaVersion, 1);
  assert.equal(meta.kind, 'execution.run.check.meta');
  assert.equal(meta.finished, true);
  assert.equal(meta.check.check_id, checkpoint.checks[0].check_id);
  assert.ok(CHECK_RESULTS.has(meta.check.result));
  assert.equal(meta.check.stdout_bytes, Buffer.byteLength(expected.stdout));
  assert.equal(meta.check.stderr_bytes, Buffer.byteLength(expected.stderr));

  const argv = readJson('argv.json');
  assert.equal(argv.kind, 'execution.run.check.argv');
  assert.deepEqual(argv.argv, ['node', 'fake-check.mjs']);
  assert.equal(argv.argv[0], meta.check.command_program);
  assert.equal(argv.argv.length - 1, meta.check.command_arg_count);

  const claim = readJson('claim-record.json');
  assert.equal(claim.schemaVersion, 1);
  assert.ok(CLAIM_RESULTS.has(claim.result), `unknown claim result ${claim.result}`);
  for (const check of claim.checks) {
    assert.equal(typeof check.exitCode, 'number');
  }
});

test('partial review package is not complete and names every omission', () => {
  const review = readJson('review-package.json');
  assert.equal(review.schemaVersion, 1);
  assert.equal(review.kind, 'execution.review.package');
  assert.match(review.package_id, PACKAGE_ID);
  assert.ok(review.partial_reasons.length > 0, 'the fixture is a partial package');
  assert.equal(review.complete, false);
  for (const reason of review.partial_reasons) {
    assert.ok(PARTIAL_REASONS.has(reason), `unknown partial reason ${reason}`);
  }
  for (const omission of review.omissions) {
    assert.ok(PARTIAL_REASONS.has(omission.reason), `unknown omission reason ${omission.reason}`);
  }
  assert.equal(review.counts.omitted, review.omissions.length);
  assert.equal(review.counts.unstaged, review.files.filter((file) => file.unstaged).length);
});

test('task brief fixture is labelled synthetic and carries the documented sections', () => {
  const brief = readFileSync(path.join(fixtureRoot, 'task-brief.md'), 'utf8');
  assert.match(brief, /synthetic shape example \(not generated output\)/);
  for (const heading of [
    '## Batch context',
    '## Task definition',
    '## Requirement references',
    '## Authority and stop conditions',
  ]) {
    assert.ok(brief.includes(heading), `missing ${heading}`);
  }
});

test('the inert fake check reproduces the expected output inside a disposable copy only', (t) => {
  const root = makeTemp(t);
  const repo = path.join(root, 'repo');
  const home = path.join(root, 'home');
  mkdirSync(home);
  cpSync(path.join(fixtureRoot, 'repo'), repo, { recursive: true });
  const before = walk(repo).map((file) => path.relative(repo, file)).sort();

  const result = spawnSync(process.execPath, [path.join(fixtureRoot, 'fake-check.mjs')], {
    cwd: repo,
    encoding: 'utf8',
    env: { HOME: home, LC_ALL: 'C' },
    timeout: 30_000,
  });
  assert.equal(result.error, undefined, `fake check failed to start: ${result.error?.message}`);

  const expected = readJson('expected-check.json');
  assert.equal(result.status, expected.exitCode);
  assert.equal(result.stdout, expected.stdout);
  assert.equal(result.stderr, expected.stderr);
  assert.deepEqual(readdirSync(home), [], 'the fake check must not write under HOME');
  assert.deepEqual(walk(repo).map((file) => path.relative(repo, file)).sort(), before);
});

test('fixtures stay inert: no executables, shebangs, real home paths, URLs or credential-shaped text', () => {
  const files = walk(fixtureRoot);
  assert.ok(files.length >= 12, 'the fixture set is unexpectedly small');
  for (const file of files) {
    const name = path.relative(fixtureRoot, file);
    assert.equal(statSync(file).mode & 0o111, 0, `${name} must not be executable`);
    const text = readFileSync(file, 'utf8');
    assert.ok(!text.startsWith('#!'), `${name} must not have a shebang`);
    assert.doesNotMatch(text, /\/home\//, `${name} names a real home path`);
    assert.doesNotMatch(text, /https?:\/\//, `${name} contains a URL`);
    assert.doesNotMatch(text, /\b(sk-|ghp_|xox[bp]-|AKIA)[A-Za-z0-9]/, `${name} looks like a credential`);
  }
});
