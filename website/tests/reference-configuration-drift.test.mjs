import assert from 'node:assert/strict';
import { copyFileSync, lstatSync, mkdirSync, mkdtempSync, readFileSync, rmSync, symlinkSync, unlinkSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(fileURLToPath(new URL('../..', import.meta.url)));
const moduleURL = new URL('../scripts/reference/configuration.mjs', import.meta.url);
const { collectConfiguration, ConfigurationError, CONFIGURATION_SOURCE_PATHS } = await import(moduleURL.href);

function fixtureRepo() {
  const root = mkdtempSync(path.join(tmpdir(), 'asdx-config-drift-'));
  for (const rel of CONFIGURATION_SOURCE_PATHS) {
    const target = path.join(root, rel);
    mkdirSync(path.dirname(target), { recursive: true });
    copyFileSync(path.join(repoRoot, rel), target);
  }
  return root;
}

function mutate(root, rel, from, to) {
  const target = path.join(root, rel);
  const before = readFileSync(target, 'utf8');
  assert.ok(before.includes(from), `fixture mutation target not found in ${rel}: ${from}`);
  writeFileSync(target, before.replace(from, to));
}

function expectDrift(root, factId, sourcePath) {
  assert.throws(
    () => collectConfiguration({ repoRoot: root }),
    (error) => {
      assert.ok(error instanceof ConfigurationError, 'expected a ConfigurationError');
      assert.equal(error.code, 'configuration_drift');
      assert.equal(error.path, sourcePath);
      assert.match(error.message, new RegExp(factId.replaceAll('.', '\\.')));
      return true;
    },
  );
}

test('fleet schema and parser contract drift is rejected', () => {
  const root = fixtureRepo();
  try {
    mutate(root, 'internal/fleet/fleet.go', 'SchemaVersion = 1', 'SchemaVersion = 2');
    expectDrift(root, 'fleet.schemaVersion', 'internal/fleet/fleet.go');
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test('fleet CLI flag registration drift is rejected', () => {
  const root = fixtureRepo();
  try {
    mutate(root, 'cmd/agentic-sdd/fleet.go', 'fs.StringVar(&repo, "repo", "", "skills repository path")', 'fs.StringVar(&repo, "repository", "", "skills repository path")');
    expectDrift(root, 'fleet.cli', 'cmd/agentic-sdd/fleet.go');
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test('fleet model grammar drift is rejected', () => {
  const root = fixtureRepo();
  try {
    mutate(root, 'internal/fleet/fleet.go', '(#[A-Za-z0-9][A-Za-z0-9._-]{0,31})?', '(#[A-Za-z0-9]+)?');
    expectDrift(root, 'fleet.modelPattern', 'internal/fleet/fleet.go');
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test('fleet sidecar format and field drift is rejected', () => {
  const root = fixtureRepo();
  try {
    mutate(root, 'internal/fleet/manifest.go', 'ManifestFormat = 4', 'ManifestFormat = 5');
    expectDrift(root, 'fleet.manifestFormat', 'internal/fleet/manifest.go');
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
  const second = fixtureRepo();
  try {
    mutate(second, 'internal/fleet/manifest.go', 'json:"rendererVersion,omitempty"', 'json:"rendererVersionRenamed,omitempty"');
    expectDrift(second, 'fleet.manifestFileFields', 'internal/fleet/manifest.go');
  } finally {
    rmSync(second, { recursive: true, force: true });
  }
});

test('skill manifest format and entry-action drift is rejected', () => {
  const root = fixtureRepo();
  try {
    mutate(root, 'internal/skillsync/manifest.go', 'actionInstalled = "installed"', 'actionInstalled = "install"');
    expectDrift(root, 'skillsync.actions', 'internal/skillsync/manifest.go');
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
  const second = fixtureRepo();
  try {
    mutate(second, 'internal/skillsync/sync.go', 'format = 3', 'format = 2');
    expectDrift(second, 'skillsync.formatThree', 'internal/skillsync/sync.go');
  } finally {
    rmSync(second, { recursive: true, force: true });
  }
});

test('MCP managed-server, state and ownership drift is rejected', () => {
  const root = fixtureRepo();
  try {
    mutate(root, 'internal/skillsync/mcp_config.go', '"codegraph", "context7", "exa"', '"codegraph", "context7"');
    expectDrift(root, 'mcp.managedServers', 'internal/skillsync/mcp_config.go');
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
  const second = fixtureRepo();
  try {
    mutate(second, 'internal/skillsync/mcp.go', 'MCPStateConfigured          MCPState = "configured"', 'MCPStateConfigured          MCPState = "ready"');
    expectDrift(second, 'mcp.states', 'internal/skillsync/mcp.go');
  } finally {
    rmSync(second, { recursive: true, force: true });
  }
  const third = fixtureRepo();
  try {
    mutate(third, 'internal/skillsync/mcp_transaction.go', 'mcpManifestFormat  = 1', 'mcpManifestFormat  = 2');
    expectDrift(third, 'mcp.ownershipFormat', 'internal/skillsync/mcp_transaction.go');
  } finally {
    rmSync(third, { recursive: true, force: true });
  }
});

test('execution schema, workspace and environment drift is rejected', () => {
  const root = fixtureRepo();
  try {
    mutate(root, 'internal/execution/types.go', 'SchemaVersion = 1', 'SchemaVersion = 2');
    expectDrift(root, 'execution.schemaVersion', 'internal/execution/types.go');
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
  const second = fixtureRepo();
  try {
    mutate(second, 'internal/execution/workspace.go', 'runRootDirName   = "runs"', 'runRootDirName   = "run"');
    expectDrift(second, 'execution.runRootDir', 'internal/execution/workspace.go');
  } finally {
    rmSync(second, { recursive: true, force: true });
  }
  const third = fixtureRepo();
  try {
    mutate(third, 'cmd/agentic-sdd/main.go', 'os.Getenv("AGENTIC_SDD_TRACE")', 'os.Getenv("ASDX_TRACE")');
    expectDrift(third, 'configuration.traceVariable', 'cmd/agentic-sdd/main.go');
  } finally {
    rmSync(third, { recursive: true, force: true });
  }
});

test('client destination and TUI build-capability drift is rejected', () => {
  const root = fixtureRepo();
  try {
    mutate(root, 'internal/skillsync/sync.go', 'filepath.Join(home, ".claude", "skills")', 'filepath.Join(home, ".claude-next", "skills")');
    expectDrift(root, 'files.clients', 'internal/skillsync/sync.go');
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
  const second = fixtureRepo();
  try {
    mutate(second, 'internal/buildinfo/buildinfo.go', 'FleetCapabilityPreview FleetCapability = "preview"', 'FleetCapabilityPreview FleetCapability = "stable"');
    expectDrift(second, 'fleet.tuiCapability', 'internal/buildinfo/buildinfo.go');
  } finally {
    rmSync(second, { recursive: true, force: true });
  }
});

test('missing or replaced sources fail closed and never return partial facts', () => {
  const root = fixtureRepo();
  try {
    unlinkSync(path.join(root, 'internal/execution/types.go'));
    assert.throws(
      () => collectConfiguration({ repoRoot: root }),
      (error) => error instanceof ConfigurationError
        && error.code === 'missing_source'
        && error.path === 'internal/execution/types.go',
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }

  const linked = fixtureRepo();
  try {
    const rel = 'internal/fleet/parse.go';
    unlinkSync(path.join(linked, rel));
    symlinkSync(path.join(linked, 'internal/fleet/fleet.go'), path.join(linked, rel));
    assert.equal(lstatSync(path.join(linked, rel)).isSymbolicLink(), true);
    assert.throws(
      () => collectConfiguration({ repoRoot: linked }),
      (error) => error instanceof ConfigurationError && error.code === 'unsafe_source' && error.path === rel,
    );
  } finally {
    rmSync(linked, { recursive: true, force: true });
  }
});

test('a rejected collection leaves every fixture source byte-identical', () => {
  const root = fixtureRepo();
  try {
    mutate(root, 'internal/fleet/fleet.go', 'SchemaVersion = 1', 'SchemaVersion = 2');
    const before = new Map(CONFIGURATION_SOURCE_PATHS.map((rel) => [rel, readFileSync(path.join(root, rel))]));
    assert.throws(() => collectConfiguration({ repoRoot: root }), ConfigurationError);
    for (const [rel, bytes] of before) {
      assert.deepEqual(readFileSync(path.join(root, rel)), bytes, `drift check wrote ${rel}`);
    }
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});
