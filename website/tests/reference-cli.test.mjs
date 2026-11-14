import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import {
  chmodSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { REFERENCE_BINARY_RECIPE } from '../scripts/reference.mjs';

const moduleURL = new URL('../scripts/reference/cli.mjs', import.meta.url);
const referenceScript = fileURLToPath(new URL('../scripts/reference.mjs', import.meta.url));
const worktreeRoot = path.resolve(fileURLToPath(new URL('../..', import.meta.url)));
// Real-binary parity is opt-in only: no hardcoded default, no implicit path.
// Without ASDX_CORE_BINARY the parity test reports `blocked` (skip with the
// exact build recipe) and must never count as a pass.
const candidateBinaryPath = process.env.ASDX_CORE_BINARY?.trim() || '';
const candidateExpectedCommit = process.env.ASDX_CORE_COMMIT?.trim() || '';
const STANDARD_FAKE_COMMIT = '2fc31e9f9c4d019e2b9f3b4f004978028808c74b';

const EXPECTED_SOURCES = [
  'cmd/agentic-sdd/env.go',
  'cmd/agentic-sdd/execution_cli.go',
  'cmd/agentic-sdd/fleet.go',
  'cmd/agentic-sdd/fleet_doctor.go',
  'cmd/agentic-sdd/fleet_plan.go',
  'cmd/agentic-sdd/guide.go',
  'cmd/agentic-sdd/helptext.go',
  'cmd/agentic-sdd/logs.go',
  'cmd/agentic-sdd/main.go',
  'cmd/agentic-sdd/mcp.go',
  'cmd/agentic-sdd/mcp_write.go',
  'cmd/agentic-sdd/opencode_disabled.go',
  'cmd/agentic-sdd/review.go',
  'cmd/agentic-sdd/run.go',
  'cmd/agentic-sdd/task.go',
  'cmd/agentic-sdd/trace.go',
  'cmd/agentic-sdd/tui.go',
  'internal/buildinfo/buildinfo.go',
];

async function loadCollector() {
  try {
    return await import(moduleURL.href);
  } catch (error) {
    if (error.code === 'ERR_MODULE_NOT_FOUND' && error.url === moduleURL.href) return {};
    throw error;
  }
}

function makeTempDir(t, prefix) {
  mkdirSync('/tmp/opencode', { recursive: true });
  const dir = mkdtempSync(`/tmp/opencode/${prefix}`);
  t.after(() => rmSync(dir, { recursive: true, force: true }));
  return dir;
}

function makeSyntheticRepo(t, { omitSource } = {}) {
  const root = makeTempDir(t, 'ref-cli-repo-');
  for (const rel of EXPECTED_SOURCES) {
    if (rel === omitSource) continue;
    const abs = path.join(root, rel);
    mkdirSync(path.dirname(abs), { recursive: true });
    writeFileSync(abs, '// synthetic source fixture\n', 'utf8');
  }
  return root;
}

function writeFakeExecutable(dir, name, scriptBody) {
  const filePath = path.join(dir, name);
  writeFileSync(filePath, `#!${process.execPath}\n${scriptBody}\n`, { mode: 0o755 });
  chmodSync(filePath, 0o755);
  return filePath;
}

const STANDARD_FAKE_CLI_BODY = `
const fs = require('node:fs');
const args = process.argv.slice(2);
const key = args.join(' ');
if (process.env.FAKE_ARGV_LOG) {
  fs.appendFileSync(process.env.FAKE_ARGV_LOG, JSON.stringify(args) + '\\n');
}
const outputs = {
  '--help': [
    'Usage: asdx-core [preview|apply] [--repo PATH] [--home PATH]',
    '       asdx-core mcp status|doctor|preview [--config PATH] [--home PATH] [--json]',
    'Commands:',
    '  mcp            Inspect the default MCP tools read-only (status, doctor, preview).',
    'Flags:',
    '  --repo PATH  Repository containing skills-files',
    '  --home PATH  User home',
    '  --apply      Legacy form of apply',
    '  --version    Legacy form of version',
    '  -h, --help  Show this help.',
  ].join('\\n'),
  'help preview': 'preview — show what apply would change (read-only)\\n\\nUsage: asdx-core [preview|apply] [--repo PATH] [--home PATH]\\n',
  'help apply': 'apply — back up changed skills, then install\\n\\nUsage: asdx-core [preview|apply] [--repo PATH] [--home PATH]\\n',
  'help backups': 'backups — list available backups, newest first\\n\\nUsage: asdx-core backups [--repo PATH] [--home PATH]\\n',
  'help restore': 'restore — preview or apply a backup by id or number\\n\\nUsage: asdx-core restore [ID] [--apply] [--repo PATH] [--home PATH]\\n',
  'help logs': 'logs — show the latest apply or restore log\\n\\nUsage: asdx-core logs [--home PATH]\\n',
  'help guide': 'guide — interactive step-by-step wizard\\n\\nUsage: asdx-core guide [--repo PATH] [--home PATH]\\n',
  'help version': 'version — print version, commit, and build date\\n\\nUsage: asdx-core version\\n',
  'help tui': 'tui — open the terminal UI in a real terminal (compatibility launcher)\\n\\nUsage: asdx-core tui [--repo PATH] [--home PATH]\\n',
  'tui --help': [
    'Usage: asdx-core tui [--repo PATH] [--home PATH] [--fleet-config PATH] [--with-opencode[=false]] [--opencode-free-first]',
    '',
    'Options:',
    '  --home PATH',
    '  --repo PATH',
    '  --fleet-config PATH',
    '  --with-opencode',
    '  --opencode-free-first',
    '  -h, --help',
  ].join('\\n'),
  'help orchestrate': [
    'Usage: asdx-core [preview|apply] --with-opencode [--opencode-free-first] [--repo PATH] [--home PATH]',
    '',
    'OpenCode Orchestration Options:',
    '  --with-opencode',
    '  --opencode-free-first',
  ].join('\\n'),
  'fleet --help': [
    'Usage: asdx-core fleet status --config PATH [--home PATH] [--repo PATH]',
    '       asdx-core fleet plan --config PATH [--output PATH] [--adopt] [--home PATH] [--repo PATH]',
    '       asdx-core fleet apply --plan PATH [--home PATH]',
    '       asdx-core fleet restore --backup ID [--home PATH]',
    '       asdx-core fleet doctor [--opencode-bin PATH] [--config PATH]',
  ].join('\\n'),
  'fleet status --help': 'Usage: asdx-core fleet status --config PATH [--home PATH] [--repo PATH]\\n\\nFlags:\\n  --config PATH\\n  --home PATH\\n  --repo PATH\\n',
  'fleet plan --help': 'Usage: asdx-core fleet plan --config PATH [--output PATH] [--adopt] [--home PATH] [--repo PATH]\\n\\nFlags:\\n  --config PATH\\n  --output PATH\\n  --adopt\\n  --home PATH\\n  --repo PATH\\n',
  'fleet apply --help': 'Usage: asdx-core fleet apply --plan PATH [--home PATH]\\n\\nFlags:\\n  --plan PATH\\n  --home PATH\\n',
  'fleet restore --help': 'Usage: asdx-core fleet restore --backup ID [--home PATH]\\n\\nFlags:\\n  --backup ID\\n  --home PATH\\n',
  'fleet doctor --help': 'Usage: asdx-core fleet doctor [--opencode-bin PATH] [--config PATH]\\n\\nFlags:\\n  --opencode-bin PATH\\n  --config PATH\\n',
  'mcp help': [
    'Usage: asdx-core mcp [status] [--home PATH] [--config PATH] [--project PATH] [--json]',
    '       asdx-core mcp doctor [--home PATH] [--config PATH] [--project PATH] [--opencode-bin PATH] [--codegraph-bin PATH] [--json]',
    '       asdx-core mcp preview [--servers NAME,...] [--auth SERVER=MODE] [--home PATH] [--config PATH] [--json]',
    '       asdx-core mcp apply --expect-config SHA256 [--opencode-bin PATH] [--servers NAME,...] [--auth SERVER=MODE] [--home PATH] [--config PATH] [--json]',
    '       asdx-core mcp restore list [--home PATH] [--json]',
    '       asdx-core mcp restore preview ID [--home PATH] [--config PATH] [--json]',
    '       asdx-core mcp restore apply ID --expect-config SHA256 [--home PATH] [--config PATH] [--json]',
    '',
    'Flags:',
    '  --home PATH',
    '  --config PATH',
    '  --project PATH',
    '  --servers NAME,...',
    '  --auth SERVER=MODE',
    '  --expect-config SHA256',
    '  --opencode-bin PATH',
    '  --codegraph-bin PATH',
    '  --json',
  ].join('\\n'),
  'mcp status --help': 'Usage: asdx-core mcp [status] [--home PATH] [--config PATH] [--project PATH] [--json]\\n\\nFlags:\\n  --home PATH\\n  --config PATH\\n  --project PATH\\n  --json\\n',
  'mcp doctor --help': 'Usage: asdx-core mcp doctor [--home PATH] [--config PATH] [--project PATH] [--opencode-bin PATH] [--codegraph-bin PATH] [--json]\\n\\n--network and --connect are deferred out of MCP-A and are rejected.\\n',
  'mcp preview --help': 'Usage: asdx-core mcp preview [--servers NAME,...] [--auth SERVER=MODE] [--home PATH] [--config PATH] [--json]\\n\\nFlags:\\n  --servers NAME,...\\n  --auth SERVER=MODE\\n  --home PATH\\n  --config PATH\\n  --json\\n',
  'mcp apply --help': 'Usage: asdx-core mcp apply --expect-config SHA256 [--opencode-bin PATH] [--servers NAME,...] [--auth SERVER=MODE] [--home PATH] [--config PATH] [--json]\\n\\nFlags:\\n  --expect-config SHA256\\n  --opencode-bin PATH\\n  --servers NAME,...\\n  --auth SERVER=MODE\\n  --home PATH\\n  --config PATH\\n  --json\\n',
  'mcp restore help': 'Usage: asdx-core mcp restore list [--home PATH] [--json]\\n       asdx-core mcp restore preview ID [--home PATH] [--config PATH] [--json]\\n       asdx-core mcp restore apply ID --expect-config SHA256 [--home PATH] [--config PATH] [--json]\\n\\nFlags:\\n  --expect-config SHA256\\n  --home PATH\\n  --config PATH\\n  --json\\n',
  'help run': 'run — private execution workspace and check evidence\\n',
  'run help': 'Usage: asdx-core run init --feature DIR --batch ID [--repo DIR] [--run-root DIR] [--format text|json]\\n       asdx-core run init --direct [--label TEXT] [--repo DIR] [--run-root DIR] [--format text|json]\\n       asdx-core run status --run ID [--repo DIR] [--run-root DIR] [--format text|json]\\n       asdx-core run record --run ID [--task T01] --result FILE|- [--repo DIR] [--run-root DIR] [--format text|json]\\n       asdx-core run clean --run ID [--yes] [--repo DIR] [--run-root DIR] [--format text|json]\\n       asdx-core run check --run ID [--task T01] [--timeout DURATION] [--max-output-bytes N] [--repo DIR] [--run-root DIR] [--format text|json] -- COMMAND [ARG...]\\n',
  'help task': 'task — extract one authorized task into a run\\n',
  'task help': 'Usage: asdx-core task brief --run ID --task T01 [--force] [--repo DIR] [--run-root DIR] [--format text|json]\\n',
  'help review': 'review — package actual Git changes for a bounded review\\n',
  'review help': 'Usage: asdx-core review package --run ID --base COMMIT [--head COMMIT] [--max-file-bytes N] [--max-total-bytes N] [--repo DIR] [--run-root DIR] [--format text|json]\\n',
  'version': 'asdx-core 0.11.0-dev\\n  commit:     ${STANDARD_FAKE_COMMIT}\\n  date:       2026-11-14T00:00:00Z\\n  go version: go1.27.1',
};

if (!Object.hasOwn(outputs, key)) {
  process.stderr.write('error: unexpected argv: ' + key + '\\n');
  process.exit(2);
}
process.stdout.write(outputs[key] + '\\n');
`;

test('CLI reference collector exports async collectCLI function', async () => {
  const mod = await loadCollector();
  assert.equal(typeof mod.collectCLI, 'function', 'collectCLI export is missing');
});

test('collects deterministic {sections, inventory} for reference/cli from fake executable', async (t) => {
  const mod = await loadCollector();
  assert.equal(typeof mod.collectCLI, 'function', 'collectCLI export is missing');

  const repoRoot = makeSyntheticRepo(t);
  const binDir = makeTempDir(t, 'ref-cli-bin-');
  const fakeBin = writeFakeExecutable(binDir, 'asdx-core', STANDARD_FAKE_CLI_BODY);

  const first = await mod.collectCLI({ repoRoot, binaryPath: fakeBin });
  const second = await mod.collectCLI({ repoRoot, binaryPath: fakeBin });

  assert.deepEqual(first, second);
  assert.equal(first.sections.length, 1);

  const [section] = first.sections;
  assert.equal(section.pageId, 'reference/cli');
  assert.equal(typeof section.title, 'string');
  assert.ok(section.title.trim().length > 0);
  assert.equal(typeof section.markdown, 'string');
  assert.ok(section.markdown.trim().length > 0);
  assert.ok(!section.markdown.includes('<!-- generated:'), 'section markdown must not embed generated markers');
  assert.ok(!section.markdown.includes('\x1b'), 'section markdown must not contain ANSI escapes');
  assert.ok(!section.markdown.includes(repoRoot), 'section markdown must not leak host repoRoot');
  assert.ok(!section.markdown.includes(fakeBin), 'section markdown must not leak host binaryPath');
  assert.deepEqual(section.sources, [...EXPECTED_SOURCES].sort());

  // Verify extracted flags avoid prose false positives (--network/--connect/--direct).
  const byId = new Map(first.inventory.commands.map((cmd) => [cmd.id, cmd]));
  assert.ok(byId.has('top'));
  assert.deepEqual(byId.get('top').flags, ['--apply', '--help', '--home', '--repo', '--version', '-h']);
  assert.deepEqual(byId.get('mcp-doctor').flags, [
    '--codegraph-bin',
    '--config',
    '--home',
    '--json',
    '--opencode-bin',
    '--project',
  ]);
  assert.deepEqual(byId.get('task').flags, ['--force', '--format', '--repo', '--run', '--run-root', '--task']);
  assert.ok(byId.get('mcp-apply').flags.includes('--expect-config'));
  assert.ok(first.inventory.discrepancies.some((item) => item.id === 'mcp-top-level-help'));
  assert.ok(first.inventory.experimentalExclusions.some((item) => item.names.includes('opencode')));
  assert.match(section.markdown, /mcp apply/);
  assert.match(section.markdown, /mcp restore/);
  assert.match(section.markdown, /incomplete|discrepanc|top-level/i);
});

test('returned section integrates cleanly with reference.mjs renderGeneratedPage', async (t) => {
  const mod = await loadCollector();
  assert.equal(typeof mod.collectCLI, 'function', 'collectCLI export is missing');

  const { renderGeneratedPage } = await import('../scripts/reference.mjs');
  const repoRoot = makeSyntheticRepo(t);
  const binDir = makeTempDir(t, 'ref-cli-compose-');
  const fakeBin = writeFakeExecutable(binDir, 'asdx-core', STANDARD_FAKE_CLI_BODY);

  const { sections } = await mod.collectCLI({ repoRoot, binaryPath: fakeBin });
  const rendered = renderGeneratedPage(sections[0]);
  assert.match(rendered, /<!-- generated:reference\/cli:start -->/);
  assert.match(rendered, /<!-- generated:reference\/cli:end -->/);
});

test('sanitizes environment, avoids PATH lookup, and invokes allowlisted help argv only', async (t) => {
  const mod = await loadCollector();
  assert.equal(typeof mod.collectCLI, 'function', 'collectCLI export is missing');

  const repoRoot = makeSyntheticRepo(t);
  const binDir = makeTempDir(t, 'ref-cli-env-');
  const markerPath = path.join(binDir, 'env-audit.jsonl');

  const prevHome = process.env.HOME;
  const prevTrace = process.env.AGENTIC_SDD_TRACE;
  const prevForce = process.env.AGENTIC_SDD_FORCE_INTERACTIVE;
  const prevAnimate = process.env.AGENTIC_SDD_ANIMATE;
  const prevSecret = process.env.OPENAI_API_KEY;
  process.env.HOME = '/tmp/host-home-must-not-leak';
  process.env.AGENTIC_SDD_TRACE = '/tmp/trace-must-not-run.jsonl';
  process.env.AGENTIC_SDD_FORCE_INTERACTIVE = '1';
  process.env.AGENTIC_SDD_ANIMATE = '1';
  process.env.OPENAI_API_KEY = 'sk-secret-must-not-pass';
  t.after(() => {
    if (prevHome === undefined) delete process.env.HOME; else process.env.HOME = prevHome;
    if (prevTrace === undefined) delete process.env.AGENTIC_SDD_TRACE; else process.env.AGENTIC_SDD_TRACE = prevTrace;
    if (prevForce === undefined) delete process.env.AGENTIC_SDD_FORCE_INTERACTIVE; else process.env.AGENTIC_SDD_FORCE_INTERACTIVE = prevForce;
    if (prevAnimate === undefined) delete process.env.AGENTIC_SDD_ANIMATE; else process.env.AGENTIC_SDD_ANIMATE = prevAnimate;
    if (prevSecret === undefined) delete process.env.OPENAI_API_KEY; else process.env.OPENAI_API_KEY = prevSecret;
  });

  const auditBody = `
const fs = require('node:fs');
const args = process.argv.slice(2);
const record = {
  args,
  envKeys: Object.keys(process.env).sort(),
  noColor: process.env.NO_COLOR,
  term: process.env.TERM,
  columns: process.env.COLUMNS,
  hasHome: Object.hasOwn(process.env, 'HOME'),
  hasTrace: Object.hasOwn(process.env, 'AGENTIC_SDD_TRACE'),
  hasForce: Object.hasOwn(process.env, 'AGENTIC_SDD_FORCE_INTERACTIVE'),
  hasAnimate: Object.hasOwn(process.env, 'AGENTIC_SDD_ANIMATE'),
  hasSecret: Object.hasOwn(process.env, 'OPENAI_API_KEY'),
};
fs.appendFileSync(${JSON.stringify(markerPath)}, JSON.stringify(record) + '\\n');
if (args.join(' ') === 'version') {
  process.stdout.write('asdx-core 0.11.0-dev\\n  commit:     ${STANDARD_FAKE_COMMIT}\\n  date:       2026-11-14T00:00:00Z\\n  go version: go1.27.1\\n');
  process.exit(0);
}
process.stdout.write('Usage: asdx-core\\nFlags:\\n  --help\\n');
`;
  const fakeBin = writeFakeExecutable(binDir, 'asdx-core', auditBody);
  const { inventory } = await mod.collectCLI({ repoRoot, binaryPath: fakeBin });

  const lines = readFileSync(markerPath, 'utf8').trim().split('\n').map((line) => JSON.parse(line));
  assert.equal(lines.length, inventory.commands.length + 1, 'help inventory plus one identity probe');
  const probes = lines.filter((entry) => entry.args.join(' ') === 'version');
  assert.equal(probes.length, 1, 'exactly one version identity probe');
  for (const entry of lines) {
    assert.equal(entry.noColor, '1');
    assert.equal(entry.term, 'dumb');
    assert.equal(entry.columns, '80');
    assert.equal(entry.hasHome, false);
    assert.equal(entry.hasTrace, false);
    assert.equal(entry.hasForce, false);
    assert.equal(entry.hasAnimate, false);
    assert.equal(entry.hasSecret, false);
    assert.ok(
      entry.args.includes('--help') || entry.args.includes('help') || entry.args.join(' ') === 'version',
      `non-help argv executed: ${entry.args.join(' ')}`,
    );
    assert.ok(!entry.args.includes('opencode'), 'experimental opencode command must never be invoked');
    assert.ok(!entry.args.includes('opencode-harness'), 'experimental opencode-harness command must never be invoked');
  }
});

test('refuses missing, relative, symlink, directory, non-executable binaries and bad options', async (t) => {
  const mod = await loadCollector();
  assert.equal(typeof mod.collectCLI, 'function', 'collectCLI export is missing');

  const repoRoot = makeSyntheticRepo(t);
  const dir = makeTempDir(t, 'ref-cli-refusal-');
  const realBin = writeFakeExecutable(dir, 'real-asdx-core', STANDARD_FAKE_CLI_BODY);
  const symlinkBin = path.join(dir, 'link-asdx-core');
  symlinkSync(realBin, symlinkBin);

  const nonExecBin = path.join(dir, 'non-exec');
  writeFileSync(nonExecBin, '#!/bin/sh\necho hi\n', { mode: 0o644 });
  chmodSync(nonExecBin, 0o644);

  const subDir = path.join(dir, 'subdir');
  mkdirSync(subDir);

  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: '' }), /binary/i);
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: 'asdx-core' }), /absolute/i);
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: './asdx-core' }), /absolute/i);
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: path.join(dir, 'missing') }), /missing|not found|binary/i);
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: symlinkBin }), /symlink/i);
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: subDir }), /regular/i);
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: nonExecBin }), /executable/i);
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: realBin, argv: ['apply'] }), /unknown|option|argv/i);
  await assert.rejects(() => mod.collectCLI({ repoRoot: path.join(dir, 'missing-repo'), binaryPath: realBin }), /repoRoot|missing/i);

  const symlinkRepo = path.join(dir, 'symlink-repo');
  symlinkSync(repoRoot, symlinkRepo, 'dir');
  await assert.rejects(() => mod.collectCLI({ repoRoot: symlinkRepo, binaryPath: realBin }), /symlink|repoRoot/i);

  const incompleteRepo = makeSyntheticRepo(t, { omitSource: 'cmd/agentic-sdd/mcp_write.go' });
  await assert.rejects(() => mod.collectCLI({ repoRoot: incompleteRepo, binaryPath: realBin }), /missing|source/i);
});

test('fails clearly on non-zero exit, empty stdout, ANSI leakage, timeout, and maxBuffer overflow', async (t) => {
  const mod = await loadCollector();
  assert.equal(typeof mod.collectCLI, 'function', 'collectCLI export is missing');

  const repoRoot = makeSyntheticRepo(t);
  const dir = makeTempDir(t, 'ref-cli-fail-');

  const exitFailBin = writeFakeExecutable(dir, 'exit-fail', `
if (process.argv.slice(2).join(' ') === 'version') {
  process.stdout.write('asdx-core 0.11.0-dev\\n  commit:     ${STANDARD_FAKE_COMMIT}\\n  date:       2026-11-14T00:00:00Z\\n  go version: go1.27.1\\n');
  process.exit(0);
}
if (process.argv.slice(2).join(' ') === 'fleet status --help') {
  process.stderr.write('error: unknown fleet command\\n');
  process.exit(2);
}
process.stdout.write('Usage: asdx-core\\nFlags:\\n  --help\\n');
`);
  await assert.rejects(
    () => mod.collectCLI({ repoRoot, binaryPath: exitFailBin }),
    /code 2.*"fleet status --help"|"fleet status --help".*(?:exit|code)\s*2/i,
  );

  const emptyBin = writeFakeExecutable(dir, 'empty-out', `
if (process.argv.slice(2).join(' ') === 'version') {
  process.stdout.write('asdx-core 0.11.0-dev\\n  commit:     ${STANDARD_FAKE_COMMIT}\\n  date:       2026-11-14T00:00:00Z\\n  go version: go1.27.1\\n');
  process.exit(0);
}
process.stdout.write('   \\n');
`);
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: emptyBin }), /empty/i);

  const ansiBin = writeFakeExecutable(dir, 'ansi-out', `
if (process.argv.slice(2).join(' ') === 'version') {
  process.stdout.write('asdx-core 0.11.0-dev\\n  commit:     ${STANDARD_FAKE_COMMIT}\\n  date:       2026-11-14T00:00:00Z\\n  go version: go1.27.1\\n');
  process.exit(0);
}
process.stdout.write('\\x1b[31mUsage: asdx-core\\x1b[0m\\n');
`);
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: ansiBin }), /ansi|escape/i);

  const slowBin = writeFakeExecutable(dir, 'slow-out', `
if (process.argv.slice(2).join(' ') === 'version') {
  process.stdout.write('asdx-core 0.11.0-dev\\n  commit:     ${STANDARD_FAKE_COMMIT}\\n  date:       2026-11-14T00:00:00Z\\n  go version: go1.27.1\\n');
  process.exit(0);
}
setTimeout(() => process.stdout.write('late\\n'), 5000);
`);
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: slowBin, timeoutMs: 80 }), /timed out/i);

  const floodBin = writeFakeExecutable(dir, 'flood-out', `
if (process.argv.slice(2).join(' ') === 'version') {
  process.stdout.write('asdx-core 0.11.0-dev\\n  commit:     ${STANDARD_FAKE_COMMIT}\\n  date:       2026-11-14T00:00:00Z\\n  go version: go1.27.1\\n');
  process.exit(0);
}
process.stdout.write('x'.repeat(8192));
`);
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: floodBin, maxBuffer: 512 }), /maxBuffer|exceeded/i);
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: realOrEmpty(dir), timeoutMs: 0 }), /timeoutMs/i);
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: realOrEmpty(dir), maxBuffer: 1.5 }), /maxBuffer/i);
  await assert.rejects(() => mod.collectCLI(null), /plain object/i);
});

function realOrEmpty(dir) {
  return path.join(dir, 'empty-out');
}

test('detects flag removal drift via reference.mjs check mode and handles spaces with shell:false', async (t) => {
  const mod = await loadCollector();
  const { applySections } = await import('../scripts/reference.mjs');

  const repoRoot = makeSyntheticRepo(t);
  const spacedDir = makeTempDir(t, 'ref cli $spaced-');
  const fullBin = writeFakeExecutable(spacedDir, 'asdx core', STANDARD_FAKE_CLI_BODY);
  const mutatedBin = writeFakeExecutable(
    spacedDir,
    'asdx-core-no-adopt',
    STANDARD_FAKE_CLI_BODY.replaceAll('  --adopt\\n', '').replaceAll('[--adopt] ', ''),
  );

  const baseline = await mod.collectCLI({ repoRoot, binaryPath: fullBin });
  const generated = applySections({ repoRoot, sections: baseline.sections, mode: 'generate' });
  assert.deepEqual(generated.changed, ['website/src/content/docs/reference/cli.md']);
  assert.deepEqual(applySections({ repoRoot, sections: baseline.sections, mode: 'check' }).changed, []);

  const drifted = await mod.collectCLI({ repoRoot, binaryPath: mutatedBin });
  assert.ok(!drifted.inventory.commands.find((c) => c.id === 'fleet-plan').flags.includes('--adopt'));
  const checkDrift = applySections({ repoRoot, sections: drifted.sections, mode: 'check' });
  assert.deepEqual(checkDrift.changed, ['website/src/content/docs/reference/cli.md']);
});

test('never modifies repoRoot or fake HOME during collection', async (t) => {
  const mod = await loadCollector();
  const repoRoot = makeSyntheticRepo(t);
  const binDir = makeTempDir(t, 'ref-cli-nowrite-');
  const fakeHome = makeTempDir(t, 'ref-cli-home-');
  const sentinel = path.join(fakeHome, 'sentinel.txt');
  writeFileSync(sentinel, 'untouched\n', 'utf8');

  const prevHome = process.env.HOME;
  process.env.HOME = fakeHome;
  t.after(() => {
    if (prevHome === undefined) delete process.env.HOME;
    else process.env.HOME = prevHome;
  });

  const fakeBin = writeFakeExecutable(binDir, 'asdx-core', STANDARD_FAKE_CLI_BODY);
  await mod.collectCLI({ repoRoot, binaryPath: fakeBin });

  assert.equal(readFileSync(sentinel, 'utf8'), 'untouched\n');
  assert.equal(existsSync(path.join(repoRoot, 'website/src/content/docs/reference/cli.md')), false);
});

test('binds generated CLI facts to the candidate binary commit and records provenance', async (t) => {
  const mod = await loadCollector();
  const repoRoot = makeSyntheticRepo(t);
  const binDir = makeTempDir(t, 'ref-cli-provenance-');
  const fakeBin = writeFakeExecutable(binDir, 'asdx-core', STANDARD_FAKE_CLI_BODY);

  const first = await mod.collectCLI({ repoRoot, binaryPath: fakeBin, expectedCommit: STANDARD_FAKE_COMMIT });
  const second = await mod.collectCLI({ repoRoot, binaryPath: fakeBin, expectedCommit: STANDARD_FAKE_COMMIT });
  assert.deepEqual(first, second);

  const [section] = first.sections;
  assert.deepEqual(section.metadata, {
    producer: 'website/scripts/reference/cli.mjs',
    binary: {
      product: 'asdx-core',
      version: '0.11.0-dev',
      commit: STANDARD_FAKE_COMMIT,
      date: '2026-11-14T00:00:00Z',
      goVersion: 'go1.27.1',
    },
    expectedCommit: STANDARD_FAKE_COMMIT,
    commitVerified: true,
  });
  assert.deepEqual(first.inventory.provenance, section.metadata);
  assert.match(section.markdown, /captured by running the candidate/);
  assert.match(section.markdown, /expected inspected-revision commit/);

  const abbreviated = await mod.collectCLI({
    repoRoot,
    binaryPath: fakeBin,
    expectedCommit: STANDARD_FAKE_COMMIT.slice(0, 12),
  });
  assert.equal(abbreviated.inventory.provenance.commitVerified, true);

  const unverified = await mod.collectCLI({ repoRoot, binaryPath: fakeBin });
  assert.equal(unverified.inventory.provenance.commitVerified, false);
  assert.equal(unverified.inventory.provenance.expectedCommit, null);
  assert.doesNotMatch(unverified.sections[0].markdown, /expected inspected-revision commit/);
});

test('fails closed on commit mismatch, malformed expectation and unidentifiable binaries', async (t) => {
  const mod = await loadCollector();
  const repoRoot = makeSyntheticRepo(t);
  const binDir = makeTempDir(t, 'ref-cli-provenance-fail-');
  const fakeBin = writeFakeExecutable(binDir, 'asdx-core', STANDARD_FAKE_CLI_BODY);

  await assert.rejects(
    () => mod.collectCLI({ repoRoot, binaryPath: fakeBin, expectedCommit: 'a'.repeat(40) }),
    /mismatch|expected commit/i,
  );
  await assert.rejects(
    () => mod.collectCLI({ repoRoot, binaryPath: fakeBin, expectedCommit: 'not-a-commit' }),
    /--commit|hex/i,
  );

  const noCommitBin = writeFakeExecutable(
    binDir,
    'no-commit',
    `process.stdout.write('asdx-core dev\\n  date:       2026-11-14T00:00:00Z\\n  go version: go1.27.1\\n');`,
  );
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: noCommitBin }), /commit|rebuild/i);

  const legacyBin = writeFakeExecutable(
    binDir,
    'legacy-name',
    `process.stdout.write('agentic-sdd dev\\n  commit:     ${STANDARD_FAKE_COMMIT}\\n  date:       2026-11-14T00:00:00Z\\n  go version: go1.27.1\\n');`,
  );
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: legacyBin }), /canonical|asdx-core/i);

  const failingProbeBin = writeFakeExecutable(
    binDir,
    'failing-probe',
    `if (process.argv.slice(2).join(' ') === 'version') process.exit(2);\nprocess.stdout.write('Usage: asdx-core\\n');`,
  );
  await assert.rejects(() => mod.collectCLI({ repoRoot, binaryPath: failingProbeBin }), /code 2/i);
});

test('check mode emits a bounded deterministic first-difference diff per drifted page', async (t) => {
  const { applySections } = await import('../scripts/reference.mjs');
  const root = makeTempDir(t, 'ref-check-diff-');
  const docs = path.join(root, 'website/src/content/docs/reference');
  mkdirSync(docs, { recursive: true });
  const longLine = 'x'.repeat(400);
  const sections = [];
  for (let index = 1; index <= 7; index += 1) {
    sections.push({
      pageId: `reference/page-${index}`,
      title: `Page ${index}`,
      markdown: index === 1
        ? `${longLine}\nsecond expected line\nthird expected line`
        : `expected line ${index}`,
      sources: [],
    });
  }

  applySections({ repoRoot: root, sections: [sections[0]], mode: 'generate' });
  const firstPage = path.join(docs, 'page-1.md');
  writeFileSync(firstPage, readFileSync(firstPage, 'utf8').replace(longLine, 'y'.repeat(400)), 'utf8');
  for (const section of sections.slice(1)) {
    writeFileSync(path.join(docs, `${section.pageId.split('/')[1]}.md`), '---\ntitle: Authored\n---\n\nAuthored body.\n', 'utf8');
  }

  const first = applySections({ repoRoot: root, sections, mode: 'check' });
  const second = applySections({ repoRoot: root, sections, mode: 'check' });
  assert.deepEqual(first, second, 'bounded check diff is deterministic');
  assert.equal(first.changed.length, 7);
  assert.equal(first.differences.length, 5, 'bounded page count');
  assert.equal(first.diffOmitted, 2);
  for (const difference of first.differences) {
    assert.match(difference.path, /^website\/src\/content\/docs\/reference\/page-\d\.md$/);
    assert.equal(typeof difference.line, 'number');
    assert.ok(difference.line >= 1);
    assert.ok(difference.expected.length > 0);
    assert.ok(difference.expected.length <= 3);
    assert.ok(difference.actual.length <= 3);
    for (const line of [...difference.expected, ...difference.actual]) assert.ok(line.length <= 160);
  }
  assert.equal(first.differences[0].missing, false);
  assert.ok(first.differences[0].expected[0].endsWith('...'), 'long expected line is truncated');
  assert.ok(first.differences[0].actual[0].endsWith('...'), 'long actual line is truncated');
  assert.equal(first.differences[0].line, 9, 'first difference points at the generated block line');

  const missingRoot = makeTempDir(t, 'ref-check-missing-');
  const missing = applySections({ repoRoot: missingRoot, sections: [sections[0]], mode: 'check' });
  assert.equal(missing.differences.length, 1);
  assert.equal(missing.differences[0].missing, true);
  assert.deepEqual(missing.differences[0].actual, []);
  assert.match(missing.changed[0], /page-1\.md$/);

  const generated = applySections({ repoRoot: makeTempDir(t, 'ref-check-generate-'), sections: [sections[0]], mode: 'generate' });
  assert.deepEqual(generated.differences, []);
  assert.equal(generated.diffOmitted, 0);
});

test('curated CLI facts record preview exits, root --version, experimental topics and execution defaults', async (t) => {
  const mod = await loadCollector();
  const repoRoot = makeSyntheticRepo(t);
  const binDir = makeTempDir(t, 'ref-cli-curated-');
  const fakeBin = writeFakeExecutable(binDir, 'asdx-core', STANDARD_FAKE_CLI_BODY);
  const { sections } = await mod.collectCLI({ repoRoot, binaryPath: fakeBin });
  const markdown = sections[0].markdown;

  assert.match(markdown, /`--version` \(root flag\)/);
  assert.match(markdown, /not bare-only/);
  assert.match(markdown, /Exits `0`, `1` \(runtime\/I\/O error\), or `2` \(usage\/flag error\)/);
  assert.match(markdown, /`help opencode`\/`help opencode-harness`/);
  assert.match(markdown, /Execution flag defaults/);
  assert.match(markdown, /run check --timeout 15m/);
  assert.match(markdown, /67108864/);
  assert.match(markdown, /2097152/);
  assert.match(markdown, /33554432/);
});

test(
  'collects accurate CLI inventory and parity from the configured ASDX_CORE_BINARY',
  {
    skip: candidateBinaryPath
      ? false
      : `blocked: ASDX_CORE_BINARY is unset, so real-binary reference parity was not evaluated.\nBuild the candidate binary from the inspected revision, then set ASDX_CORE_BINARY to it:\n${REFERENCE_BINARY_RECIPE}`,
  },
  async () => {
    const mod = await loadCollector();
    assert.equal(typeof mod.collectCLI, 'function', 'collectCLI export is missing');

    const options = candidateExpectedCommit
      ? { repoRoot: worktreeRoot, binaryPath: candidateBinaryPath, expectedCommit: candidateExpectedCommit }
      : { repoRoot: worktreeRoot, binaryPath: candidateBinaryPath };
    const first = await mod.collectCLI(options);
    const second = await mod.collectCLI(options);
    assert.deepEqual(first, second);

    const [section] = first.sections;
    assert.equal(section.pageId, 'reference/cli');
    assert.equal(section.metadata.binary.product, 'asdx-core');
    assert.match(section.metadata.binary.commit, /^[0-9a-f]{40}$/);
    assert.equal(first.inventory.provenance.binary.commit, section.metadata.binary.commit);
    assert.ok(section.markdown.includes('commit `' + section.metadata.binary.commit + '`'));
    assert.ok(!section.markdown.includes('\x1b'));
    assert.ok(!section.markdown.includes(worktreeRoot));
    assert.ok(!section.markdown.includes(candidateBinaryPath));

    const byId = new Map(first.inventory.commands.map((cmd) => [cmd.id, cmd]));
    assert.equal(byId.size, 29);
    for (const cmd of first.inventory.commands) {
      assert.equal(cmd.exit, 0);
      assert.equal(cmd.stderr, '');
      assert.ok(cmd.stdout.length > 0);
    }

    assert.deepEqual(byId.get('top').flags, ['--apply', '--help', '--home', '--repo', '--version', '-h']);
    assert.deepEqual(byId.get('tui').flags, [
      '--fleet-config',
      '--help',
      '--home',
      '--opencode-free-first',
      '--repo',
      '--with-opencode',
      '-h',
    ]);
    assert.deepEqual(byId.get('fleet-plan').flags, ['--adopt', '--config', '--home', '--output', '--repo']);
    assert.deepEqual(byId.get('mcp-apply').flags, [
      '--auth',
      '--config',
      '--expect-config',
      '--home',
      '--json',
      '--opencode-bin',
      '--servers',
    ]);
    assert.deepEqual(byId.get('mcp-doctor').flags, [
      '--codegraph-bin',
      '--config',
      '--home',
      '--json',
      '--opencode-bin',
      '--project',
    ]);
    assert.deepEqual(byId.get('run').flags, [
      '--batch',
      '--direct',
      '--feature',
      '--format',
      '--label',
      '--max-output-bytes',
      '--repo',
      '--result',
      '--run',
      '--run-root',
      '--task',
      '--timeout',
      '--yes',
    ]);
    assert.deepEqual(byId.get('task').flags, ['--force', '--format', '--repo', '--run', '--run-root', '--task']);
    assert.deepEqual(byId.get('review').flags, [
      '--base',
      '--format',
      '--head',
      '--max-file-bytes',
      '--max-total-bytes',
      '--repo',
      '--run',
      '--run-root',
    ]);
  },
);

test('reference generate and check report blocked with the exact build recipe when no binary is configured', (t) => {
  const env = { ...process.env };
  delete env.ASDX_CORE_BINARY;
  for (const mode of ['--generate', '--check']) {
    const result = spawnSync(process.execPath, [referenceScript, mode], { encoding: 'utf8', env });
    assert.equal(result.status, 2, `${mode} must exit 2 (blocked), stderr: ${result.stderr}`);
    assert.match(result.stderr, /reference: blocked:/);
    assert.match(result.stderr, /ASDX_CORE_BINARY is unset/);
    assert.ok(
      result.stderr.includes(REFERENCE_BINARY_RECIPE),
      `${mode} must print the exact reproducible build recipe`,
    );
  }
});

test('reference CLI prints the reproducible build recipe without a binary', () => {
  const result = spawnSync(process.execPath, [referenceScript, '--recipe'], { encoding: 'utf8' });
  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.stdout.trim(), REFERENCE_BINARY_RECIPE);
  assert.match(result.stdout, /-X agentic-sdd\/internal\/version\.Commit=\$\(git rev-parse HEAD\)/);
});

test('reference CLI reads ASDX_CORE_BINARY when --binary is omitted', (t) => {
  const env = { ...process.env, ASDX_CORE_BINARY: path.join(makeTempDir(t, 'ref-env-binary-'), 'absent') };
  const result = spawnSync(process.execPath, [referenceScript, '--check'], { encoding: 'utf8', env });
  assert.equal(result.status, 1, `the configured path must be inspected, stderr: ${result.stderr}`);
  assert.match(result.stderr, /Missing CLI binary/);
  assert.doesNotMatch(result.stderr, /blocked/);
});

test('real-binary parity has no hardcoded fallback path', () => {
  const source = readFileSync(fileURLToPath(import.meta.url), 'utf8');
  assert.match(source, /process\.env\.ASDX_CORE_BINARY/);
  assert.doesNotMatch(source, /\/tmp\/opencode\/spec020/);
});
