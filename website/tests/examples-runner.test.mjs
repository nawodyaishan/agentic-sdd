import assert from 'node:assert/strict';
import {
  chmodSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import {
  EXAMPLES,
  EXAMPLE_IDS,
  EXIT,
  ExamplesBlocked,
  ExamplesRefusal,
  exitCodeForStatus,
  inspectBinary,
  parseArgs,
  runExamples,
  selectExamples,
} from '../scripts/examples.mjs';
import { STUB_VERSION_LINE, writeStubCli } from './examples/stub-cli.mjs';

function makeTempDir(t, prefix) {
  mkdirSync('/tmp/opencode', { recursive: true });
  const dir = mkdtempSync(path.join('/tmp/opencode', prefix));
  t.after(() => rmSync(dir, { recursive: true, force: true }));
  return dir;
}

function readLog(logPath) {
  return readFileSync(logPath, 'utf8')
    .trim()
    .split('\n')
    .filter(Boolean)
    .map((line) => JSON.parse(line));
}

test('frozen example table keeps the documented ids, order, effects and argv', () => {
  assert.deepEqual(EXAMPLE_IDS, ['first-preview', 'direct-fix-illustration', 'feature-approval']);
  assert.equal(EXAMPLES.length, 3);
  for (const example of EXAMPLES) {
    for (const key of ['id', 'effect', 'argv', 'expected']) {
      assert.ok(Object.hasOwn(example, key), `example ${example.id} is missing ${key}`);
    }
    assert.ok(['read-only', 'manual'].includes(example.effect));
    assert.ok(!JSON.stringify(example.argv).includes('apply'), 'no frozen argv may run apply');
  }
  const first = EXAMPLES[0];
  assert.equal(first.effect, 'read-only');
  assert.deepEqual(first.argv, [['version'], ['preview', '--home', '{home}']]);
  for (const example of EXAMPLES.slice(1)) {
    assert.equal(example.effect, 'manual');
    assert.deepEqual(example.argv, []);
    assert.equal(typeof example.reason, 'string');
    assert.ok(example.reason.length > 0);
  }
});

test('refuses unknown ids and never treats arbitrary input as an example', () => {
  assert.throws(
    () => selectExamples({ only: 'first-apply' }),
    (error) =>
      error instanceof ExamplesRefusal &&
      error.exitCode === EXIT.REFUSED &&
      /Unknown example id/.test(error.message) &&
      /first-preview/.test(error.message),
  );
  assert.throws(() => selectExamples({ only: '' }), /requires a value/);
});

test('refuses Markdown fences, inline code and multi-line blocks', () => {
  const fence = '```sh\nasdx-core version\nasdx-core preview --home /tmp/example\n```';
  assert.throws(
    () => selectExamples({ only: fence }),
    (error) =>
      error instanceof ExamplesRefusal &&
      error.exitCode === EXIT.REFUSED &&
      /Markdown fences/.test(error.message) &&
      /documentation, not executable/.test(error.message),
  );
  assert.throws(() => parseArgs(['--only', fence]), /Markdown fences/);
  assert.throws(() => parseArgs([fence]), /Markdown fences/);
  assert.throws(() => parseArgs(['--only', '`asdx-core version`']), /Markdown fences/);
  assert.throws(() => parseArgs(['asdx-core version']), /Unexpected argument/);
});

test('reports a missing or non-executable binary as blocked, not pass or fail', async (t) => {
  const dir = makeTempDir(t, 'examples-blocked-');
  const missing = path.join(dir, 'not-built');
  const missingSummary = await runExamples({ binaryPath: missing, only: 'first-preview' });
  assert.equal(missingSummary.status, 'blocked');
  assert.equal(missingSummary.ok, false);
  assert.equal(missingSummary.examples[0].status, 'blocked');
  assert.match(missingSummary.examples[0].reason, /No executable CLI/);
  assert.match(missingSummary.examples[0].reason, /PATH search is never used/);
  assert.equal(exitCodeForStatus('blocked'), EXIT.BLOCKED);

  const noBinary = await runExamples({});
  assert.equal(noBinary.status, 'blocked');
  assert.match(noBinary.examples[0].reason, /No --binary supplied/);
  assert.equal(noBinary.examples[1].status, 'manual');
  assert.equal(noBinary.examples[2].status, 'manual');

  const nonExec = path.join(dir, 'not-executable');
  writeFileSync(nonExec, '#!/bin/sh\nexit 0\n', { mode: 0o644 });
  chmodSync(nonExec, 0o644);
  const nonExecSummary = await runExamples({ binaryPath: nonExec, only: 'first-preview' });
  assert.equal(nonExecSummary.status, 'blocked');
  assert.match(nonExecSummary.examples[0].reason, /not executable/);
  assert.throws(() => inspectBinary(nonExec), (error) => error instanceof ExamplesBlocked);
});

test('refuses relative, symlink and non-file binary paths instead of searching PATH', async (t) => {
  const dir = makeTempDir(t, 'examples-refuse-bin-');
  const realBin = writeStubCli(dir, { logPath: path.join(dir, 'log.jsonl') });
  const link = path.join(dir, 'link-asdx-core');
  symlinkSync(realBin, link);

  await assert.rejects(() => runExamples({ binaryPath: 'asdx-core', only: 'first-preview' }), /absolute path/);
  await assert.rejects(() => runExamples({ binaryPath: './asdx-core', only: 'first-preview' }), /absolute path/);
  await assert.rejects(() => runExamples({ binaryPath: link, only: 'first-preview' }), /symlink/);
  await assert.rejects(() => runExamples({ binaryPath: dir, only: 'first-preview' }), /not a regular file/);
});

test('executes only version and preview against the fixture, with a sanitized environment', async (t) => {
  const dir = makeTempDir(t, 'examples-pass-');
  const logPath = path.join(dir, 'argv.jsonl');
  const binary = writeStubCli(dir, { logPath });
  const fakeHome = makeTempDir(t, 'examples-real-home-guard-');
  const sentinel = path.join(fakeHome, 'sentinel.txt');
  writeFileSync(sentinel, 'untouched\n');

  const prevHome = process.env.HOME;
  const prevSecret = process.env.OPENAI_API_KEY;
  process.env.HOME = fakeHome;
  process.env.OPENAI_API_KEY = 'sk-fixture-must-not-leak';
  t.after(() => {
    if (prevHome === undefined) delete process.env.HOME;
    else process.env.HOME = prevHome;
    if (prevSecret === undefined) delete process.env.OPENAI_API_KEY;
    else process.env.OPENAI_API_KEY = prevSecret;
  });

  const summary = await runExamples({ binaryPath: binary, only: 'first-preview' });
  assert.equal(summary.status, 'pass');
  assert.equal(summary.binaryPath, binary);
  assert.equal(summary.tempRoot, null);

  const [result] = summary.examples;
  assert.equal(result.status, 'pass');
  assert.equal(result.home.entries, 0, 'the fixture home must stay empty');
  assert.match(result.home.digest, /^[0-9a-f]{64}$/);
  assert.deepEqual(result.argv, [['version'], ['preview', '--home', '{home}']]);
  assert.equal(result.steps.length, 2);
  assert.deepEqual(result.steps.map((step) => step.argv[0]), ['version', 'preview']);
  assert.ok(result.steps[0].stdout.includes(STUB_VERSION_LINE));
  assert.match(result.steps[1].stdout, /\{owned-temp\}/, 'owned paths are placeholdered in reported output');
  assert.ok(!/asdx-examples-/.test(result.steps[1].stdout), 'reported output must not leak the random temp root');
  assert.ok(!result.steps[1].stdout.includes(fakeHome));

  assert.equal(readFileSync(sentinel, 'utf8'), 'untouched\n');
  const records = readLog(logPath);
  assert.equal(records.length, 2);
  assert.deepEqual(records[0].argv, ['version']);
  assert.deepEqual(records[1].argv.slice(0, 2), ['preview', '--home']);
  for (const record of records) {
    assert.equal(record.path, '/usr/bin:/bin');
    assert.ok(record.home && /asdx-examples-[^/]+\/home$/.test(record.home), 'HOME must be the owned fixture home');
    assert.notEqual(record.home, fakeHome);
    assert.ok(!record.envKeys.includes('OPENAI_API_KEY'));
    assert.ok(!record.envKeys.includes('AGENTIC_SDD_TRACE'));
    assert.ok(!record.envKeys.includes('AGENTIC_SDD_FORCE_INTERACTIVE'));
    assert.equal(record.cwd, path.dirname(record.home));
    assert.ok(path.basename(record.cwd).startsWith('asdx-examples-'));
  }
});

test('fails when preview writes into the owned fixture home', async (t) => {
  const dir = makeTempDir(t, 'examples-mutate-');
  const binary = writeStubCli(dir, { logPath: path.join(dir, 'argv.jsonl'), mutateHome: true });
  const summary = await runExamples({ binaryPath: binary, only: 'first-preview' });
  assert.equal(summary.status, 'fail');
  assert.equal(summary.examples[0].status, 'fail');
  assert.match(summary.examples[0].reason, /preview modified the owned fixture home/);
  assert.match(summary.examples[0].reason, /never point --home at a real user home/);
  assert.equal(exitCodeForStatus('fail'), EXIT.FAIL);
});

test('fails clearly on a non-zero step or empty reported output', async (t) => {
  const dir = makeTempDir(t, 'examples-step-fail-');
  const failing = writeStubCli(dir, {
    name: 'version-fails',
    logPath: path.join(dir, 'version-fails.log'),
    versionExit: 3,
  });
  const versionFail = await runExamples({ binaryPath: failing, only: 'first-preview' });
  assert.equal(versionFail.status, 'fail');
  assert.match(versionFail.examples[0].reason, /Step "version" exited with code 3/);

  const silent = writeStubCli(dir, {
    name: 'preview-silent',
    logPath: path.join(dir, 'preview-silent.log'),
    silentPreview: true,
  });
  const silentFail = await runExamples({ binaryPath: silent, only: 'first-preview' });
  assert.equal(silentFail.status, 'fail');
  assert.match(silentFail.examples[0].reason, /produced no stdout/);
});

test('cleans the owned temp root by default and only keeps it on request', async (t) => {
  const tmpBase = makeTempDir(t, 'examples-cleanup-');
  const dir = makeTempDir(t, 'examples-cleanup-bin-');
  const binary = writeStubCli(dir, { logPath: path.join(dir, 'argv.jsonl') });

  const cleaned = await runExamples({ binaryPath: binary, only: 'first-preview', tmpBase });
  assert.equal(cleaned.status, 'pass');
  assert.equal(cleaned.tempRoot, null);
  assert.deepEqual(readdirSync(tmpBase), [], 'default runs leave no owned root behind');

  const kept = await runExamples({ binaryPath: binary, only: 'first-preview', tmpBase, keep: true });
  assert.equal(kept.status, 'pass');
  assert.ok(kept.tempRoot && kept.tempRoot.startsWith(path.join(tmpBase, 'asdx-examples-')));
  assert.ok(existsSync(path.join(kept.tempRoot, 'home')));
  assert.deepEqual(readdirSync(tmpBase), [path.basename(kept.tempRoot)]);

  const blocked = await runExamples({ binaryPath: path.join(dir, 'missing'), only: 'first-preview', tmpBase });
  assert.equal(blocked.status, 'blocked');
  assert.deepEqual(readdirSync(tmpBase), [path.basename(kept.tempRoot)], 'blocked runs create no temp root');

  rmSync(kept.tempRoot, { recursive: true, force: true });
  assert.deepEqual(readdirSync(tmpBase), []);
});

test('produces a deterministic summary and stable example ordering across runs', async (t) => {
  const tmpBase = makeTempDir(t, 'examples-deterministic-');
  const dir = makeTempDir(t, 'examples-deterministic-bin-');
  const binary = writeStubCli(dir, { logPath: path.join(dir, 'argv.jsonl') });

  const first = await runExamples({ binaryPath: binary, tmpBase });
  const second = await runExamples({ binaryPath: binary, tmpBase });
  assert.deepEqual(first, second);
  assert.equal(JSON.stringify(first), JSON.stringify(second));
  assert.deepEqual(first.examples.map((result) => result.id), EXAMPLE_IDS);
  assert.deepEqual(
    first.examples.map((result) => result.status),
    ['pass', 'manual', 'manual'],
  );

  const only = await runExamples({ binaryPath: binary, only: 'first-preview', tmpBase });
  assert.deepEqual(only.examples.map((result) => result.id), ['first-preview']);
  const manualOnly = await runExamples({ only: 'feature-approval', tmpBase });
  assert.equal(manualOnly.status, 'pass');
  assert.equal(manualOnly.binaryPath, null);
  assert.deepEqual(manualOnly.examples.map((result) => result.id), ['feature-approval']);
});

test('parseArgs accepts only the documented flag surface', () => {
  assert.deepEqual(parseArgs(['--binary', '/abs/path', '--only', 'first-preview', '--json', '--keep']), {
    binaryPath: '/abs/path',
    only: 'first-preview',
    json: true,
    keep: true,
    help: false,
  });
  assert.equal(parseArgs(['--help']).help, true);
  assert.throws(() => parseArgs(['--apply']), /Unknown example option/);
  assert.throws(() => parseArgs(['--binary']), /Missing value for --binary/);
  assert.throws(() => parseArgs(['--only']), /Missing value for --only/);
  assert.throws(() => parseArgs(['extra']), /Unexpected argument/);
});
