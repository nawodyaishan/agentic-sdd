import assert from 'node:assert/strict';
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, statSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(fileURLToPath(new URL('../..', import.meta.url)));
const moduleURL = new URL('../scripts/reference/configuration.mjs', import.meta.url);

const PAGE_IDS = [
  'reference/configuration',
  'reference/fleet',
  'reference/mcp',
  'reference/execution',
  'reference/files-and-backups',
];
const SYSTEM_IDS = ['configuration', 'fleet', 'mcp', 'execution', 'files-and-backups'];

const { collectConfiguration, ConfigurationError, CONFIGURATION_SOURCE_PATHS } = await import(moduleURL.href);

function fixtureRepo() {
  const root = mkdtempSync(path.join(tmpdir(), 'asdx-config-collect-'));
  for (const rel of CONFIGURATION_SOURCE_PATHS) {
    const target = path.join(root, rel);
    mkdirSync(path.dirname(target), { recursive: true });
    copyFileSync(path.join(repoRoot, rel), target);
  }
  return root;
}

function requireFilePath(rel) {
  return statSync(path.join(repoRoot, rel), { throwIfNoEntry: false })?.isFile() === true;
}

test('collects exactly the five canonical configuration pages from the current checkout', () => {
  const output = collectConfiguration({ repoRoot });
  assert.equal(typeof output, 'object');
  assert.ok(Array.isArray(output.sections), 'sections must be an array');
  assert.deepEqual(output.sections.map((section) => section.pageId), PAGE_IDS);
  for (const section of output.sections) {
    assert.equal(typeof section.title, 'string');
    assert.ok(section.title.trim().length > 0);
    assert.equal(typeof section.markdown, 'string');
    assert.ok(section.markdown.trim().length > 200, `${section.pageId} facts are too thin`);
    assert.ok(Array.isArray(section.sources) && section.sources.length > 0);
    assert.deepEqual(section.sources, [...section.sources].sort(), `${section.pageId} sources are not sorted`);
    assert.equal(new Set(section.sources).size, section.sources.length, `${section.pageId} repeats a source`);
    for (const source of section.sources) {
      assert.ok(!source.startsWith('/') && !source.includes('..'), `unsafe source pointer ${source}`);
      assert.ok(requireFilePath(source), `missing source pointer ${source}`);
    }
    assert.ok(!section.markdown.includes(repoRoot), 'host path leaked into generated facts');
    assert.ok(!/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/.test(section.markdown), 'wall-clock timestamp leaked into generated facts');
  }
});

test('collection is deterministic and does not mutate the source tree', () => {
  const root = fixtureRepo();
  try {
    const before = new Map(CONFIGURATION_SOURCE_PATHS.map((rel) => [rel, readFileSync(path.join(root, rel))]));
    const first = collectConfiguration({ repoRoot: root });
    const second = collectConfiguration({ repoRoot: root });
    assert.equal(JSON.stringify(first), JSON.stringify(second));
    for (const [rel, bytes] of before) {
      assert.deepEqual(readFileSync(path.join(root, rel)), bytes, `collector wrote ${rel}`);
    }
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test('inventory separates the fleet, skill-manifest, MCP and execution contracts', () => {
  const { inventory } = collectConfiguration({ repoRoot });
  assert.ok(Array.isArray(inventory.systems));
  assert.deepEqual(inventory.systems.map((system) => system.id), SYSTEM_IDS);
  const systems = new Map(inventory.systems.map((system) => [system.id, system.facts]));

  const fleet = systems.get('fleet');
  assert.equal(fleet.schemaVersion, 1);
  assert.equal(fleet.configFile, 'fleet.json');
  assert.equal(fleet.configRequired, true);
  assert.equal(fleet.maxConfigBytes, 1048576);
  assert.deepEqual(fleet.rootFields, ['schemaVersion', 'fleet', 'preset', 'instances']);
  assert.deepEqual(fleet.instanceFields, ['source', 'model', 'enabled']);
  assert.equal(fleet.instanceIdPattern, '^[a-z0-9][a-z0-9-]{0,63}$');
  assert.equal(fleet.sourceRefPattern, '^(agent|preset):([a-z0-9][a-z0-9-]*)$');
  assert.deepEqual(fleet.sourceRoots, ['agents', 'adapters', 'presets/free-first']);
  assert.match(fleet.modelPattern, /#variant|#\[A-Za-z0-9\]/);
  assert.equal(fleet.enabledDefault, true);
  assert.deepEqual(fleet.reservedNames, ['spec', 'plan', 'vision']);
  assert.equal(fleet.manifest.file, 'fleet-manifest.json');
  assert.equal(fleet.manifest.format, 4);
  assert.deepEqual(fleet.manifest.states, ['applying', 'applied', 'failed']);
  assert.deepEqual(fleet.target, { scope: 'global', root: '<home>/.config/opencode/agents' });
  assert.equal(fleet.tui.default, 'preview');
  assert.deepEqual(fleet.tui.markers, ['preview', 'live']);
  assert.equal(fleet.tui.override, 'none');

  const mcp = systems.get('mcp');
  assert.equal(mcp.maxConfigBytes, 1048576);
  assert.deepEqual(mcp.formats, ['json', 'jsonc']);
  assert.deepEqual(mcp.schemas, ['none', 'v2']);
  assert.deepEqual(mcp.servers.map((server) => server.name), ['codegraph', 'context7', 'exa']);
  assert.deepEqual(mcp.servers.map((server) => server.kind), ['local', 'remote', 'remote']);
  assert.equal(mcp.plan.version, 'mcp-plan-v1');
  assert.equal(mcp.ownership.path, '<home>/.agentic-sdd/mcp/manifest.json');
  assert.equal(mcp.ownership.format, 1);
  assert.equal(mcp.ownership.manifestVersion, 1);
  assert.equal(mcp.ownership.snapshotFormat, 1);
  assert.deepEqual(mcp.ownership.intentPhases, [
    'prepared',
    'renamed',
    'manifest-written',
    'complete',
    'rolled-back',
    'needs-recovery',
  ]);

  const execution = systems.get('execution');
  assert.equal(execution.schemaVersion, 1);
  assert.equal(execution.runRoot.default, '<repo>/.agentic-sdd/runs');
  assert.deepEqual(execution.scopeKinds, ['feature', 'direct']);
  assert.equal(execution.kinds.run, 'execution.run.identity');
  assert.equal(execution.kinds.brief, 'execution.task.brief');
  assert.equal(execution.kinds.review, 'execution.review.package');
  assert.deepEqual(execution.artifacts.checkFiles, ['meta.json', 'argv.json', 'stdout.log', 'stderr.log']);
  assert.equal(execution.limits.tasksBytes, 4 * 1024 * 1024);
  assert.equal(execution.limits.specBytes, 8 * 1024 * 1024);
  assert.equal(execution.limits.checkTimeoutSeconds, 900);
  assert.deepEqual(execution.cli.commonFlags, ['--repo', '--run-root', '--format']);
  assert.equal(execution.cli.formatDefault, 'text');

  const files = systems.get('files-and-backups');
  assert.equal(files.managedPrefix, 'agentic-sdd-');
  assert.deepEqual(files.clients.map((client) => client.id), ['agents', 'codex', 'claude', 'agy']);
  assert.deepEqual(files.clients.map((client) => client.path), [
    '<home>/.agents/skills',
    '<home>/.codex/skills',
    '<home>/.claude/skills',
    '<home>/.gemini/antigravity-cli/skills',
  ]);
  assert.equal(files.openCodeAgents.path, '<home>/.config/opencode/agents');
  assert.equal(files.openCodeAgents.manifestKind, 'opencode-agent');
  assert.equal(files.backupIdPattern, '20060102T150405.000000000Z');
  assert.deepEqual(files.formats.one, ['created', 'source', 'targets', 'skills']);
  assert.equal(files.fleetSidecar.file, 'fleet-manifest.json');
  assert.equal(files.fleetSidecar.format, 4);
  assert.equal(files.mcpOwnership.format, 1);

  const configuration = systems.get('configuration');
  assert.deepEqual(configuration.environment.map((entry) => entry.name).sort(), [
    'AGENTIC_SDD_ANIMATE',
    'AGENTIC_SDD_ASCII',
    'AGENTIC_SDD_FORCE_INTERACTIVE',
    'AGENTIC_SDD_REDUCED_MOTION',
    'AGENTIC_SDD_TRACE',
    'COLUMNS',
    'NO_COLOR',
    'TERM',
  ]);
  assert.equal(configuration.pathOverrides.productEnvironmentVariables, false);
  assert.deepEqual(configuration.outputs.execution.format, ['text', 'json']);
  assert.equal(configuration.outputs.execution.schemaVersion, 1);
  assert.equal(configuration.outputs.mcp.jsonFlag, '--json');
});

test('generated facts keep the subsystem ownership boundaries visible', () => {
  const { sections } = collectConfiguration({ repoRoot });
  const byId = new Map(sections.map((section) => [section.pageId, section.markdown]));
  assert.match(byId.get('reference/fleet'), /schemaVersion 1/i);
  assert.match(byId.get('reference/fleet'), /fleet-manifest\.json/);
  assert.match(byId.get('reference/fleet'), /format 4/);
  assert.match(byId.get('reference/fleet'), /preview/);
  assert.match(byId.get('reference/mcp'), /mcp\.servers/);
  assert.match(byId.get('reference/mcp'), /never migrat/i);
  assert.match(byId.get('reference/execution'), /execution\.run\.identity/);
  assert.match(byId.get('reference/execution'), /\.agentic-sdd\/runs/);
  assert.match(byId.get('reference/files-and-backups'), /manifest\.json/);
  assert.match(byId.get('reference/files-and-backups'), /Format 1|format 1/);
  assert.match(byId.get('reference/configuration'), /AGENTIC_SDD_TRACE/);
  assert.match(byId.get('reference/configuration'), /no product-specific environment variable/i);
  assert.ok(ConfigurationError.prototype instanceof Error);
});

test('invalid roots and sources fail with typed, actionable errors', () => {
  const root = fixtureRepo();
  try {
    assert.throws(() => collectConfiguration({}), (error) => error instanceof ConfigurationError && error.code === 'invalid_repo_root');
    assert.throws(
      () => collectConfiguration({ repoRoot: path.join(root, 'missing') }),
      (error) => error instanceof ConfigurationError && error.code === 'invalid_repo_root',
    );
    const notDirectory = path.join(root, 'file.txt');
    mkdirSync(path.dirname(notDirectory), { recursive: true });
    copyFileSync(path.join(repoRoot, 'go.mod'), notDirectory);
    assert.throws(
      () => collectConfiguration({ repoRoot: notDirectory }),
      (error) => error instanceof ConfigurationError && error.code === 'invalid_repo_root',
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});
