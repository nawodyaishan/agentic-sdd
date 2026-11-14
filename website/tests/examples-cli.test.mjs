import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { chmodSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { EXIT } from '../scripts/examples.mjs';
import { writeStubCli } from './examples/stub-cli.mjs';

const scriptPath = path.resolve(fileURLToPath(new URL('../scripts/examples.mjs', import.meta.url)));

function makeTempDir(t, prefix) {
  mkdirSync('/tmp/opencode', { recursive: true });
  const dir = mkdtempSync(path.join('/tmp/opencode', prefix));
  t.after(() => rmSync(dir, { recursive: true, force: true }));
  return dir;
}

function runCli(args) {
  const result = spawnSync(process.execPath, [scriptPath, ...args], {
    encoding: 'utf8',
    env: { ...process.env, NO_COLOR: '1' },
    timeout: 60_000,
  });
  assert.equal(result.error, undefined, `CLI failed to start: ${result.error?.message}`);
  return result;
}

test('CLI refuses an unknown example id with exit 2 and an actionable message', () => {
  const result = runCli(['--only', 'first-apply']);
  assert.equal(result.status, EXIT.REFUSED);
  assert.match(result.stderr, /Unknown example id "first-apply"/);
  assert.match(result.stderr, /Available ids: first-preview, direct-fix-illustration, feature-approval/);
  assert.equal(result.stdout, '');
});

test('CLI refuses Markdown fences and positional code blocks with exit 2', () => {
  const fence = '```sh\nasdx-core version\nasdx-core preview --home /tmp/example\n```';
  const fencedOnly = runCli(['--only', fence]);
  assert.equal(fencedOnly.status, EXIT.REFUSED);
  assert.match(fencedOnly.stderr, /Markdown fences/);
  assert.match(fencedOnly.stderr, /not executable examples/);

  const fencedPositional = runCli([fence]);
  assert.equal(fencedPositional.status, EXIT.REFUSED);
  assert.match(fencedPositional.stderr, /Markdown fences/);
});

test('CLI reports a missing or non-executable binary as blocked (exit 3)', (t) => {
  const dir = makeTempDir(t, 'examples-cli-blocked-');
  const missing = runCli(['--binary', path.join(dir, 'missing-asdx-core'), '--only', 'first-preview', '--json']);
  assert.equal(missing.status, EXIT.BLOCKED);
  const missingSummary = JSON.parse(missing.stdout);
  assert.equal(missingSummary.status, 'blocked');
  assert.equal(missingSummary.examples[0].status, 'blocked');
  assert.match(missingSummary.examples[0].reason, /No executable CLI/);

  const noBinary = runCli(['--only', 'first-preview']);
  assert.equal(noBinary.status, EXIT.BLOCKED);
  assert.match(noBinary.stdout, /\[blocked\] first-preview/);
  assert.match(noBinary.stdout, /No --binary supplied/);

  const nonExec = path.join(dir, 'not-executable');
  writeFileSync(nonExec, '#!/bin/sh\nexit 0\n', { mode: 0o644 });
  chmodSync(nonExec, 0o644);
  const nonExecResult = runCli(['--binary', nonExec, '--only', 'first-preview']);
  assert.equal(nonExecResult.status, EXIT.BLOCKED);
  assert.match(nonExecResult.stdout, /not executable/);
});

test('CLI refuses a relative --binary instead of searching PATH', () => {
  const result = runCli(['--binary', 'asdx-core', '--only', 'first-preview']);
  assert.equal(result.status, EXIT.REFUSED);
  assert.match(result.stderr, /explicit absolute path/);
  assert.match(result.stderr, /PATH search is never used/);
});

test('CLI reports manual examples without executing anything', () => {
  const result = runCli(['--only', 'feature-approval', '--json']);
  assert.equal(result.status, EXIT.PASS);
  const summary = JSON.parse(result.stdout);
  assert.equal(summary.status, 'pass');
  assert.equal(summary.binaryPath, null);
  assert.equal(summary.examples.length, 1);
  assert.equal(summary.examples[0].status, 'manual');
  assert.match(summary.examples[0].reason, /human approval/);
});

test('CLI runs first-preview against a generated fixture stub and cleans up', (t) => {
  const dir = makeTempDir(t, 'examples-cli-pass-');
  const binary = writeStubCli(dir, { logPath: path.join(dir, 'argv.jsonl') });
  const result = runCli(['--binary', binary, '--only', 'first-preview', '--json']);
  assert.equal(result.status, EXIT.PASS);
  const summary = JSON.parse(result.stdout);
  assert.equal(summary.status, 'pass');
  assert.equal(summary.ok, true);
  assert.equal(summary.tempRoot, null, 'the owned temp root is removed by default');
  assert.deepEqual(summary.examples[0].argv, [['version'], ['preview', '--home', '{home}']]);
  assert.equal(summary.examples[0].steps.length, 2);
  assert.equal(summary.examples[0].home.entries, 0);
});

test('CLI output is deterministic across identical runs and keeps stable ordering', (t) => {
  const dir = makeTempDir(t, 'examples-cli-deterministic-');
  const binary = writeStubCli(dir, { logPath: path.join(dir, 'argv.jsonl') });
  const first = runCli(['--binary', binary, '--json']);
  const second = runCli(['--binary', binary, '--json']);
  assert.equal(first.status, EXIT.PASS);
  assert.equal(second.status, EXIT.PASS);
  assert.equal(first.stdout, second.stdout);
  const summary = JSON.parse(first.stdout);
  assert.deepEqual(summary.examples.map((example) => example.id), [
    'first-preview',
    'direct-fix-illustration',
    'feature-approval',
  ]);
  assert.deepEqual(summary.examples.map((example) => example.status), ['pass', 'manual', 'manual']);
});

test('CLI refuses unknown options and prints help without touching a binary', () => {
  const unknown = runCli(['--apply']);
  assert.equal(unknown.status, EXIT.REFUSED);
  assert.match(unknown.stderr, /Unknown example option: --apply/);

  const help = runCli(['--help']);
  assert.equal(help.status, EXIT.PASS);
  assert.match(help.stdout, /Usage: node scripts\/examples\.mjs --binary <absolute-path>/);
  assert.match(help.stdout, /Exit codes: 0 pass, 1 fail, 2 refused/);
});
