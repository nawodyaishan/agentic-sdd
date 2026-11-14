// Source-derived configuration facts for the Spec 020 B1 reference pages.
//
// The collector reads a bounded, explicit set of Go parser/type/constant files
// from the current checkout and refuses to emit facts when an expected
// contract anchor drifted. It never reads a home directory, runs a product
// command, performs network I/O or writes any file.
//
// The five pages are separate subsystem contracts on purpose: fleet.json
// schema version 1, the skill backup manifest formats 1-3, the fleet sidecar
// format 4, the MCP ownership store (format 1, manifest version 1) and the
// execution schema version 1 are not one unified configuration schema.

import { lstatSync, readFileSync, realpathSync } from 'node:fs';
import path from 'node:path';

const MAX_SOURCE_BYTES = 2 * 1024 * 1024;

const PRESENTATION_ENVIRONMENT = new Set([
  'AGENTIC_SDD_ANIMATE',
  'AGENTIC_SDD_ASCII',
  'AGENTIC_SDD_FORCE_INTERACTIVE',
  'AGENTIC_SDD_REDUCED_MOTION',
  'AGENTIC_SDD_TRACE',
  'COLUMNS',
  'NO_COLOR',
  'TERM',
]);

export class ConfigurationError extends Error {
  constructor(message, { code = 'invalid_input', path: sourcePath, detail } = {}) {
    super(message);
    this.name = 'ConfigurationError';
    this.code = code;
    this.path = sourcePath;
    this.detail = detail;
  }
}

export const CONFIGURATION_SOURCE_PATHS = Object.freeze([
  'cmd/agentic-sdd/env.go',
  'cmd/agentic-sdd/execution_cli.go',
  'cmd/agentic-sdd/fleet.go',
  'cmd/agentic-sdd/fleet_plan.go',
  'cmd/agentic-sdd/main.go',
  'cmd/agentic-sdd/mcp.go',
  'cmd/agentic-sdd/review.go',
  'cmd/agentic-sdd/run.go',
  'cmd/agentic-sdd/trace.go',
  'cmd/agentic-sdd/tui.go',
  'experiments/tui/cmd/internal/tuirun/run.go',
  'internal/buildinfo/buildinfo.go',
  'internal/execution/checkpoint.go',
  'internal/execution/checks.go',
  'internal/execution/record.go',
  'internal/execution/reviewpackage.go',
  'internal/execution/taskbrief.go',
  'internal/execution/types.go',
  'internal/execution/workspace.go',
  'internal/fleet/fleet.go',
  'internal/fleet/manifest.go',
  'internal/fleet/parse.go',
  'internal/fleet/resolve.go',
  'internal/skillsync/clients.go',
  'internal/skillsync/manifest.go',
  'internal/skillsync/mcp.go',
  'internal/skillsync/mcp_config.go',
  'internal/skillsync/mcp_transaction.go',
  'internal/skillsync/restore.go',
  'internal/skillsync/source.go',
  'internal/skillsync/sync.go',
]);

function fail(code, message, options = {}) {
  throw new ConfigurationError(message, { code, ...options });
}

function normalizeRepoRoot(repoRoot) {
  if (typeof repoRoot !== 'string' || repoRoot.trim() === '') {
    fail('invalid_repo_root', 'repoRoot must be a non-empty repository directory path');
  }
  const resolved = path.resolve(repoRoot);
  let info;
  try {
    info = lstatSync(resolved);
  } catch {
    fail('invalid_repo_root', `repository root does not exist: ${repoRoot}`);
  }
  let root;
  try {
    root = realpathSync(resolved);
  } catch {
    fail('invalid_repo_root', `repository root cannot be resolved: ${repoRoot}`);
  }
  if (!lstatSync(root).isDirectory()) {
    fail('invalid_repo_root', `repository root is not a directory: ${repoRoot}`);
  }
  return root;
}

function readSources(root) {
  const sources = new Map();
  for (const relative of CONFIGURATION_SOURCE_PATHS) {
    const absolute = path.join(root, ...relative.split('/'));
    let info;
    try {
      info = lstatSync(absolute);
    } catch {
      fail('missing_source', `required configuration source is missing: ${relative}`, { path: relative });
    }
    if (info.isSymbolicLink()) {
      fail('unsafe_source', `required configuration source is a symlink: ${relative}`, { path: relative });
    }
    if (!info.isFile()) {
      fail('unsafe_source', `required configuration source is not a regular file: ${relative}`, { path: relative });
    }
    if (info.size > MAX_SOURCE_BYTES) {
      fail('source_too_large', `required configuration source exceeds ${MAX_SOURCE_BYTES} bytes: ${relative}`, { path: relative });
    }
    let cursor = root;
    for (const segment of relative.split('/')) {
      cursor = path.join(cursor, segment);
      if (lstatSync(cursor).isSymbolicLink()) {
        fail('unsafe_source', `required configuration source traverses a symlink: ${relative}`, { path: relative });
      }
    }
    const real = realpathSync(absolute);
    const contained = path.relative(root, real);
    if (contained === '' || contained.startsWith('..') || path.isAbsolute(contained)) {
      fail('unsafe_source', `required configuration source escapes the repository root: ${relative}`, { path: relative });
    }
    sources.set(relative, readFileSync(absolute, 'utf8'));
  }
  return sources;
}

function factError(file, factId, message) {
  throw new ConfigurationError(`Configuration source drift: ${factId} in ${file}: ${message}`, {
    code: 'configuration_drift',
    path: file,
    detail: factId,
  });
}

function requireMatch(text, file, factId, pattern, description) {
  const match = text.match(pattern);
  if (!match) factError(file, factId, description ?? `expected anchor ${pattern} was not found`);
  return match;
}

function expectEqual(actual, expected, file, factId) {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    factError(file, factId, `expected ${JSON.stringify(expected)}, found ${JSON.stringify(actual)}`);
  }
  return actual;
}

function expectPresent(text, file, factId, pattern, description) {
  requireMatch(text, file, factId, pattern, description ?? `expected anchor ${pattern} was not found`);
  return true;
}

function stringConst(text, file, factId, name) {
  return requireMatch(text, file, factId, new RegExp(`\\b${name}\\s*=\\s*"([^"]*)"`), `constant ${name} was not found`)[1];
}

function typedStringConst(text, file, factId, name) {
  return requireMatch(text, file, factId, new RegExp(`\\b${name}\\s+[A-Za-z0-9_.]+\\s*=\\s*"([^"]*)"`), `typed constant ${name} was not found`)[1];
}

function intConst(text, file, factId, name) {
  return Number(requireMatch(text, file, factId, new RegExp(`\\b${name}\\s*=\\s*(\\d+)\\b`), `constant ${name} was not found`)[1]);
}

function shiftConst(text, file, factId, name) {
  const match = requireMatch(text, file, factId, new RegExp(`\\b${name}\\s*=\\s*(\\d+)\\s*<<\\s*(\\d+)`), `constant ${name} was not found`);
  return Number(match[1]) * 2 ** Number(match[2]);
}

function durationConst(text, file, factId, name, unit = 'time.Minute') {
  const match = requireMatch(text, file, factId, new RegExp(`\\b${name}\\s*=\\s*(\\d+)\\s*\\*\\s*${unit.replace('.', '\\.')}`), `constant ${name} was not found`);
  return Number(match[1]) * (unit === 'time.Minute' ? 60 : 1);
}

function regexpConst(text, file, factId, name) {
  return requireMatch(text, file, factId, new RegExp(`\\b${name}\\s*=\\s*regexp\\.MustCompile\\(\`([^\`]*)\`\\)`), `regexp constant ${name} was not found`)[1];
}

function enumStrings(text, file, factId, startName) {
  const start = text.indexOf(startName);
  if (start < 0) factError(file, factId, `enum starting at ${startName} was not found`);
  const end = text.indexOf('\n)', start);
  if (end < 0) factError(file, factId, `enum starting at ${startName} is not closed`);
  return [...text.slice(start, end).matchAll(/=\s*"([^"]*)"/g)].map((match) => match[1]);
}

function enumStringsWithPrefix(text, file, factId, startName, namePrefix) {
  const start = text.indexOf(startName);
  if (start < 0) factError(file, factId, `enum starting at ${startName} was not found`);
  const end = text.indexOf('\n)', start);
  if (end < 0) factError(file, factId, `enum starting at ${startName} is not closed`);
  return [...text.slice(start, end).matchAll(new RegExp(`\\b${namePrefix}[A-Za-z]*\\s*=\\s*"([^"]*)"`, 'g'))]
    .map((match) => match[1]);
}

function structJSONFields(text, file, factId, typeName) {
  const body = requireMatch(
    text,
    file,
    factId,
    new RegExp(`type ${typeName} struct \\{([\\s\\S]*?)\\n\\}`),
    `type ${typeName} was not found`,
  )[1];
  return [...body.matchAll(/json:"([^",]+)(?:,[^"]*)?"/g)].map((match) => match[1]);
}

function funcBody(text, file, factId, signature) {
  return requireMatch(
    text,
    file,
    factId,
    new RegExp(`${signature}\\s*\\{([\\s\\S]*?)\\n\\}`),
    `function ${signature} was not found`,
  )[1];
}

function registeredFlags(text, file, factId, funcName) {
  const body = funcBody(text, file, factId, `func ${funcName}\\([^)]*\\) int`);
  return [...body.matchAll(/fs\.(?:String|Bool)Var\([^,]+,\s*"([^"]+)"/g)].map((match) => `--${match[1]}`);
}

function quotedValues(block) {
  return [...block.matchAll(/"([^"]*)"/g)].map((match) => match[1]);
}

function verifyFleet(sources) {
  const coreFile = 'internal/fleet/fleet.go';
  const core = sources.get(coreFile);
  const schemaVersion = expectEqual(
    intConst(core, coreFile, 'fleet.schemaVersion', 'SchemaVersion'),
    1,
    coreFile,
    'fleet.schemaVersion',
  );
  const maxConfigBytes = expectEqual(
    shiftConst(core, coreFile, 'fleet.maxConfigBytes', 'MaxConfigBytes'),
    1024 * 1024,
    coreFile,
    'fleet.maxConfigBytes',
  );
  const sourceKinds = [
    stringConst(core, coreFile, 'fleet.sourceKinds', 'KindAgent'),
    stringConst(core, coreFile, 'fleet.sourceKinds', 'KindPreset'),
  ];
  expectEqual(sourceKinds, ['agent', 'preset'], coreFile, 'fleet.sourceKinds');
  const instanceIdPattern = expectEqual(
    regexpConst(core, coreFile, 'fleet.instanceIdPattern', 'instanceIDRe'),
    '^[a-z0-9][a-z0-9-]{0,63}$',
    coreFile,
    'fleet.instanceIdPattern',
  );
  const sourceRefPattern = expectEqual(
    regexpConst(core, coreFile, 'fleet.sourceRefPattern', 'sourceRefRe'),
    '^(agent|preset):([a-z0-9][a-z0-9-]*)$',
    coreFile,
    'fleet.sourceRefPattern',
  );
  const modelPattern = expectEqual(
    regexpConst(core, coreFile, 'fleet.modelPattern', 'modelRe'),
    '^[a-z0-9][a-z0-9_-]{0,31}/[A-Za-z0-9][A-Za-z0-9._/-]{0,127}(#[A-Za-z0-9][A-Za-z0-9._-]{0,31})?$',
    coreFile,
    'fleet.modelPattern',
  );
  expectPresent(core, coreFile, 'fleet.rootFields', /must not contain path separators or traversal/);
  expectPresent(core, coreFile, 'fleet.instanceFields', /source is required/);
  const rootFields = expectEqual(
    structJSONFields(core, coreFile, 'fleet.rootFields', 'Config'),
    ['schemaVersion', 'fleet', 'preset', 'instances'],
    coreFile,
    'fleet.rootFields',
  );
  const instanceFields = expectEqual(
    structJSONFields(core, coreFile, 'fleet.instanceFields', 'Instance'),
    ['source', 'model', 'enabled'],
    coreFile,
    'fleet.instanceFields',
  );
  const reservedBlock = requireMatch(
    core,
    coreFile,
    'fleet.reservedNames',
    /ReservedNames = map\[string\]bool\{([\s\S]*?)\n\}/,
    'ReservedNames map was not found',
  )[1];
  const reservedNames = expectEqual(
    [...reservedBlock.matchAll(/"([^"]+)":\s*true/g)].map((match) => match[1]),
    ['spec', 'plan', 'vision'],
    coreFile,
    'fleet.reservedNames',
  );
  const enabledBody = funcBody(core, coreFile, 'fleet.enabledDefault', 'func \\(i Instance\\) IsEnabled\\(\\) bool');
  const enabledDefault = expectEqual(
    /\breturn true\b/.test(enabledBody),
    true,
    coreFile,
    'fleet.enabledDefault',
  );
  expectPresent(
    core,
    coreFile,
    'fleet.modelOverrideEmpty',
    /empty string \(""\), omitted field, or null value means "leave the canonical/,
    'the documented empty-model semantics were not found',
  );

  const parseFile = 'internal/fleet/parse.go';
  const parse = sources.get(parseFile);
  expectPresent(parse, parseFile, 'fleet.parseLimit', /io\.LimitReader\(r, MaxConfigBytes\+1\)/);
  expectPresent(parse, parseFile, 'fleet.parseLimit', /len\(data\) > MaxConfigBytes/);
  expectPresent(parse, parseFile, 'fleet.parseUTF8', /if !utf8\.Valid\(data\)/);
  expectPresent(parse, parseFile, 'fleet.parseUnknownFields', /dec\.DisallowUnknownFields\(\)/);
  expectPresent(parse, parseFile, 'fleet.parseDuplicateKeys', /detectDuplicateKeys\(data\)/);
  expectPresent(parse, parseFile, 'fleet.parseDepth', /if depth > 32/);
  expectPresent(parse, parseFile, 'fleet.parseTrailingData', /trailing data after config document/);

  const resolveFile = 'internal/fleet/resolve.go';
  const resolve = sources.get(resolveFile);
  const sourceRoots = ['agents', 'adapters', 'presets/free-first'];
  for (const root of sourceRoots) {
    expectPresent(
      resolve,
      resolveFile,
      'fleet.sourceRoots',
      new RegExp(`\\{Kind(?:Agent|Preset), "${root.replaceAll('/', '\\/')}"\\}`),
      `fleet source root ${root} was not found`,
    );
  }
  expectPresent(resolve, resolveFile, 'fleet.sourceRoots', /strings\.HasSuffix\(name, "\.md"\)/);
  expectPresent(resolve, resolveFile, 'fleet.sourceRoots', /instance-to-instance aliases, cycles and unknown sources are rejected/);
  expectPresent(resolve, resolveFile, 'fleet.sourceRoots', /duplicate source %q in %s and %s/);
  expectPresent(resolve, resolveFile, 'fleet.sourceRoots', /unknown source %q/);
  expectPresent(core, coreFile, 'fleet.modelPattern', /empty path segment/);
  expectPresent(core, coreFile, 'fleet.writeBoundary', /write only when called explicitly/);

  const manifestFile = 'internal/fleet/manifest.go';
  const manifest = sources.get(manifestFile);
  const manifestFormat = expectEqual(
    intConst(manifest, manifestFile, 'fleet.manifestFormat', 'ManifestFormat'),
    4,
    manifestFile,
    'fleet.manifestFormat',
  );
  const manifestFileName = expectEqual(
    stringConst(manifest, manifestFile, 'fleet.manifestFileName', 'ManifestFileName'),
    'fleet-manifest.json',
    manifestFile,
    'fleet.manifestFileName',
  );
  const scopeGlobal = expectEqual(
    stringConst(manifest, manifestFile, 'fleet.scope', 'ScopeGlobal'),
    'global',
    manifestFile,
    'fleet.scope',
  );
  const provenance = expectEqual(
    enumStrings(manifest, manifestFile, 'fleet.provenance', 'ProvenanceManaged'),
    ['managed', 'adopted', 'restored', 'legacy', 'unknown', 'removed'],
    manifestFile,
    'fleet.provenance',
  );
  const states = expectEqual(
    enumStrings(manifest, manifestFile, 'fleet.manifestStates', 'ManifestStateApplying'),
    ['applying', 'applied', 'failed'],
    manifestFile,
    'fleet.manifestStates',
  );
  const manifestFields = expectEqual(
    structJSONFields(manifest, manifestFile, 'fleet.manifestFields', 'Manifest'),
    [
      'format', 'id', 'operation', 'restoredFrom', 'state', 'createdAt', 'home', 'backupRoot',
      'backupPath', 'targetRoot', 'scope', 'sourceRev', 'configDigest', 'rendererVersion',
      'rollback', 'files',
    ],
    manifestFile,
    'fleet.manifestFields',
  );
  const manifestFileFields = expectEqual(
    structJSONFields(manifest, manifestFile, 'fleet.manifestFileFields', 'FileRecord'),
    [
      'instanceID', 'sourceRef', 'sourceDigest', 'configDigest', 'rendererVersion', 'model',
      'desiredDigest', 'installedDigest', 'backupRef', 'backupDigest', 'scope', 'target', 'provenance',
    ],
    manifestFile,
    'fleet.manifestFileFields',
  );
  const targetRootBody = funcBody(manifest, manifestFile, 'fleet.targetRoot', 'func TargetRootForHome\\(home string\\) string');
  const targetRootSegments = expectEqual(
    quotedValues(targetRootBody),
    ['.config', 'opencode', 'agents'],
    manifestFile,
    'fleet.targetRoot',
  );
  expectPresent(manifest, manifestFile, 'fleet.agentsExcluded', /strings\.EqualFold\(file, "AGENTS\.md"\)/);
  expectPresent(manifest, manifestFile, 'fleet.targetRoot', /refusing to manage AGENTS\.md/);
  expectPresent(manifest, manifestFile, 'fleet.manifestFileName', /ManifestFileName is the sidecar file name inside a backup directory/);

  const cliFile = 'cmd/agentic-sdd/fleet.go';
  const cli = sources.get(cliFile);
  const planFile = 'cmd/agentic-sdd/fleet_plan.go';
  const planCLI = sources.get(planFile);
  const cliCommands = [
    {
      command: 'fleet status',
      usage: 'asdx-core fleet status --config PATH [--home PATH] [--repo PATH]',
      flags: ['--config', '--home', '--repo'],
      required: ['--config'],
      source: cliFile,
      func: 'runFleetStatus',
      registered: ['--config', '--home', '--repo'],
    },
    {
      command: 'fleet plan',
      usage: 'asdx-core fleet plan --config PATH [--output PATH] [--adopt] [--home PATH] [--repo PATH]',
      flags: ['--config', '--output', '--adopt', '--home', '--repo'],
      required: ['--config'],
      source: planFile,
      func: 'runFleetPlan',
      registered: ['--config', '--output', '--adopt', '--home', '--repo'],
    },
    {
      command: 'fleet apply',
      usage: 'asdx-core fleet apply --plan PATH [--home PATH]',
      flags: ['--plan', '--home'],
      required: ['--plan'],
      source: cliFile,
      func: 'runFleetApply',
      registered: ['--plan', '--home'],
    },
    {
      command: 'fleet restore',
      usage: 'asdx-core fleet restore --backup ID [--home PATH]',
      flags: ['--backup', '--home'],
      required: ['--backup'],
      source: cliFile,
      func: 'runFleetRestore',
      registered: ['--backup', '--home'],
    },
    {
      command: 'fleet doctor',
      usage: 'asdx-core fleet doctor [--opencode-bin PATH] [--config PATH]',
      flags: ['--opencode-bin', '--config'],
      required: [],
      source: cliFile,
      func: 'runFleetDoctor',
      registered: ['--opencode-bin', '--config'],
    },
  ];
  for (const entry of cliCommands) {
    const text = entry.source === planFile ? planCLI : cli;
    expectPresent(text, entry.source, 'fleet.cli', new RegExp(entry.usage.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
    expectEqual(registeredFlags(text, entry.source, 'fleet.cli', entry.func), entry.registered, entry.source, 'fleet.cli');
    for (const flag of entry.required) {
      expectPresent(
        text,
        entry.source,
        'fleet.cli',
        new RegExp(`${flag.replace('-', '\\-')} [A-Z]+ {2}[^\\n]*\\(required\\)`),
        `${entry.command} does not document ${flag} as required`,
      );
      expectPresent(text, entry.source, 'fleet.cli', new RegExp(`${flag} is required`), `${entry.command} does not enforce ${flag}`);
    }
  }
  expectPresent(cli, cliFile, 'fleet.exitCodes', /0 {2}Success/);
  expectPresent(cli, cliFile, 'fleet.exitCodes', /1 {2}Runtime error, lock held, stale plan, or unmanaged conflict/);
  expectPresent(cli, cliFile, 'fleet.exitCodes', /2 {2}Usage error, flag error, invalid configuration, or malformed plan/);

  const buildFile = 'internal/buildinfo/buildinfo.go';
  const build = sources.get(buildFile);
  const capabilityMarkers = expectEqual(
    [
      typedStringConst(build, buildFile, 'fleet.tuiCapability', 'FleetCapabilityPreview'),
      typedStringConst(build, buildFile, 'fleet.tuiCapability', 'FleetCapabilityLive'),
    ],
    ['preview', 'live'],
    buildFile,
    'fleet.tuiCapability',
  );
  expectPresent(build, buildFile, 'fleet.tuiCapability', /FleetCapabilityMarker = string\(FleetCapabilityPreview\)/);
  expectPresent(build, buildFile, 'fleet.tuiCapability', /agentic-sdd\/internal\/buildinfo\.<Name>=<value>/);
  expectPresent(build, buildFile, 'fleet.tuiCapability', /if !marker\.Valid\(\) \{\s*\n\t\treturn FleetCapabilityPreview/);
  expectPresent(build, buildFile, 'fleet.tuiCapability', /environment, config, CLI or key override anywhere in the process/);

  return {
    schemaVersion,
    configFile: 'fleet.json',
    configFlag: '--config',
    configRequired: true,
    maxConfigBytes,
    parseRules: [
      'valid UTF-8 required',
      'duplicate JSON object keys rejected',
      'unknown fields rejected',
      'trailing data rejected',
      'JSON nesting depth limited to 32',
    ],
    rootFields,
    instanceFields,
    instanceIdPattern,
    sourceRefPattern,
    sourceKinds,
    sourceRoots,
    modelPattern,
    modelOverrideEmpty: 'inherit the canonical source model',
    enabledDefault,
    reservedNames,
    target: { scope: scopeGlobal, root: `<home>/${targetRootSegments.join('/')}` },
    manifest: {
      file: manifestFileName,
      format: manifestFormat,
      provenance,
      states,
      fields: manifestFields,
      fileFields: manifestFileFields,
    },
    cli: cliCommands.map((entry) => ({
      command: entry.command,
      flags: entry.flags,
      required: entry.required,
    })),
    tui: {
      default: capabilityMarkers[0],
      markers: capabilityMarkers,
      override: 'none',
      injection: '-X agentic-sdd/internal/buildinfo.FleetCapabilityMarker',
    },
  };
}

function verifyMCP(sources) {
  const coreFile = 'internal/skillsync/mcp.go';
  const core = sources.get(coreFile);
  const states = expectEqual(
    enumStrings(core, coreFile, 'mcp.states', 'MCPStateNotConfigured'),
    [
      'not-configured', 'configured', 'disabled', 'missing-prerequisite', 'auth-not-checked',
      'conflict', 'schema-mismatch', 'unsafe-target', 'unknown',
    ],
    coreFile,
    'mcp.states',
  );
  const kinds = expectEqual(
    enumStrings(core, coreFile, 'mcp.kinds', 'MCPServerLocal'),
    ['local', 'remote'],
    coreFile,
    'mcp.kinds',
  );
  const defaultsBody = funcBody(
    core,
    coreFile,
    'mcp.defaultServers',
    'func MCPDefaultServers\\(\\) \\[\\]MCPServerSpec',
  );
  const names = [...defaultsBody.matchAll(/Name:\s+"([^"]+)"/g)].map((match) => match[1]);
  const serverKinds = [...defaultsBody.matchAll(/Kind:\s+MCPServer(Local|Remote)/g)].map((match) => match[1].toLowerCase());
  const purposes = [...defaultsBody.matchAll(/Purpose:\s+"([^"]*)"/g)].map((match) => match[1]);
  const prerequisites = [...defaultsBody.matchAll(/Prerequisite:\s+"([^"]*)"/g)].map((match) => match[1]);
  const recipes = [...defaultsBody.matchAll(/Recipe:\s+`([^`]*)`/g)].map((match) => match[1]);
  expectEqual(names, ['codegraph', 'context7', 'exa'], coreFile, 'mcp.defaultServers');
  expectEqual(serverKinds, ['local', 'remote', 'remote'], coreFile, 'mcp.defaultServers');
  expectEqual(prerequisites, ['codegraph CLI', '', ''], coreFile, 'mcp.defaultServers');
  expectEqual(recipes.length, 3, coreFile, 'mcp.defaultServers');
  const servers = names.map((name, index) => ({
    name,
    kind: serverKinds[index],
    purpose: purposes[index],
    prerequisite: prerequisites[index],
    recipe: recipes[index],
  }));
  const maxConfigMatch = requireMatch(
    core,
    coreFile,
    'mcp.maxConfigBytes',
    /maxMCPConfigFileBytes = (\d+) \* (\d+)/,
    'maxMCPConfigFileBytes was not found',
  );
  const maxConfigBytes = expectEqual(
    Number(maxConfigMatch[1]) * Number(maxConfigMatch[2]),
    1024 * 1024,
    coreFile,
    'mcp.maxConfigBytes',
  );
  expectPresent(
    core,
    coreFile,
    'mcp.targetResolution',
    /if ext != "\.json" && ext != "\.jsonc"/,
    'explicit MCP config extension guard was not found',
  );
  expectPresent(core, coreFile, 'mcp.targetResolution', /both opencode\.json and opencode\.jsonc exist/);
  expectPresent(core, coreFile, 'mcp.targetResolution', /if !hasJSON && !hasJSONC \{\s*\n\t\treturn "", "none", "", "", false/);

  const configFile = 'internal/skillsync/mcp_config.go';
  const config = sources.get(configFile);
  const configDir = expectEqual(
    stringConst(config, configFile, 'mcp.configDir', 'mcpConfigDirRel'),
    '.config/opencode',
    configFile,
    'mcp.configDir',
  );
  const defaultJSON = expectEqual(
    stringConst(config, configFile, 'mcp.configFiles', 'mcpDefaultJSON'),
    'opencode.json',
    configFile,
    'mcp.configFiles',
  );
  const defaultJSONC = expectEqual(
    stringConst(config, configFile, 'mcp.configFiles', 'mcpDefaultJSONC'),
    'opencode.jsonc',
    configFile,
    'mcp.configFiles',
  );
  const stateDir = expectEqual(
    stringConst(config, configFile, 'mcp.stateDir', 'mcpStateDirRel'),
    '.agentic-sdd/mcp',
    configFile,
    'mcp.stateDir',
  );
  const ownershipFile = expectEqual(
    stringConst(config, configFile, 'mcp.ownershipFile', 'mcpOwnershipFile'),
    'manifest.json',
    configFile,
    'mcp.ownershipFile',
  );
  const formats = expectEqual(
    [
      stringConst(config, configFile, 'mcp.formats', 'mcpFormatJSON'),
      stringConst(config, configFile, 'mcp.formats', 'mcpFormatJSONC'),
    ],
    ['json', 'jsonc'],
    configFile,
    'mcp.formats',
  );
  const schemas = expectEqual(
    [
      stringConst(config, configFile, 'mcp.schemas', 'mcpSchemaNone'),
      stringConst(config, configFile, 'mcp.schemas', 'mcpSchemaV2'),
    ],
    ['none', 'v2'],
    configFile,
    'mcp.schemas',
  );
  const planVersion = expectEqual(
    stringConst(config, configFile, 'mcp.planVersion', 'mcpPlanVersion'),
    'mcp-plan-v1',
    configFile,
    'mcp.planVersion',
  );
  expectPresent(config, configFile, 'mcp.planVersion', /func mcpPlanDigest\(/);
  expectPresent(config, configFile, 'mcp.planVersion', /Version:\s+mcpPlanVersion/);
  const authModes = expectEqual(
    [
      stringConst(config, configFile, 'mcp.authModes', 'mcpAuthOAuth'),
      stringConst(config, configFile, 'mcp.authModes', 'mcpAuthAPIKey'),
      stringConst(config, configFile, 'mcp.authModes', 'mcpAuthKeyless'),
    ],
    ['oauth', 'api-key', 'keyless'],
    configFile,
    'mcp.authModes',
  );
  const managedList = requireMatch(
    config,
    configFile,
    'mcp.managedServers',
    /func mcpManagedNames\(\) \[\]string \{\s*\n\treturn \[\]string\{([^}]*)\}/,
    'mcpManagedNames was not found',
  )[1];
  const managedServers = expectEqual(
    quotedValues(managedList),
    ['codegraph', 'context7', 'exa'],
    configFile,
    'mcp.managedServers',
  );
  const operations = expectEqual(
    enumStrings(config, configFile, 'mcp.operations', 'mcpOpAdd'),
    ['add', 'change', 'leave', 'disabled', 'conflict', 'restore', 'remove', 'missing'],
    configFile,
    'mcp.operations',
  );
  expectPresent(
    config,
    configFile,
    'mcp.legacySchema',
    /legacy V1 mcp\.<name> schema detected/,
    'legacy V1 MCP schema refusal was not found',
  );
  expectPresent(config, configFile, 'mcp.legacySchema', /OpenCode V2 requires mcp\.servers/);
  expectPresent(config, configFile, 'mcp.jsoncRefusal', /comment\/format preservation cannot be/);
  expectPresent(core, coreFile, 'mcp.unsafeTarget', /target is a symlink/);
  expectPresent(core, coreFile, 'mcp.unsafeTarget', /target is not a regular file/);
  expectPresent(config, configFile, 'mcp.unsafeTarget', /outside the supported global scope/);
  expectPresent(config, configFile, 'mcp.authPlaceholders', /"Bearer \{env:CONTEXT7_API_KEY\}"/);
  expectPresent(config, configFile, 'mcp.authPlaceholders', /"\{env:EXA_API_KEY\}"/);

  const transactionFile = 'internal/skillsync/mcp_transaction.go';
  const transaction = sources.get(transactionFile);
  expectPresent(transaction, transactionFile, 'mcp.ownershipPrivacy', /No entry text, header value or credential is stored/);
  const ownershipFormat = expectEqual(
    intConst(transaction, transactionFile, 'mcp.ownershipFormat', 'mcpManifestFormat'),
    1,
    transactionFile,
    'mcp.ownershipFormat',
  );
  const ownershipVersion = expectEqual(
    intConst(transaction, transactionFile, 'mcp.ownershipVersion', 'mcpManifestVersion'),
    1,
    transactionFile,
    'mcp.ownershipVersion',
  );
  const snapshotFormat = expectEqual(
    intConst(transaction, transactionFile, 'mcp.snapshotFormat', 'mcpSnapshotFormat'),
    1,
    transactionFile,
    'mcp.snapshotFormat',
  );
  const backupsDirName = expectEqual(
    stringConst(transaction, transactionFile, 'mcp.backupDir', 'mcpBackupsDirName'),
    'backups',
    transactionFile,
    'mcp.backupDir',
  );
  const lockFileName = expectEqual(
    stringConst(transaction, transactionFile, 'mcp.lockFile', 'mcpLockFileName'),
    '.lock',
    transactionFile,
    'mcp.lockFile',
  );
  const snapshotFileName = expectEqual(
    stringConst(transaction, transactionFile, 'mcp.snapshotFile', 'mcpSnapshotFile'),
    'snapshot.json',
    transactionFile,
    'mcp.snapshotFile',
  );
  const intentFileName = expectEqual(
    stringConst(transaction, transactionFile, 'mcp.intentFile', 'mcpIntentFile'),
    'intent.json',
    transactionFile,
    'mcp.intentFile',
  );
  const intentPhases = expectEqual(
    enumStringsWithPrefix(transaction, transactionFile, 'mcp.intentPhases', 'mcpIntentPrepared', 'mcpIntent'),
    ['prepared', 'renamed', 'manifest-written', 'complete', 'rolled-back', 'needs-recovery'],
    transactionFile,
    'mcp.intentPhases',
  );
  const runtimeEvidence = expectEqual(
    stringConst(transaction, transactionFile, 'mcp.runtimeEvidence', 'mcpRuntimeEvidenceLocal'),
    'local-probe',
    transactionFile,
    'mcp.runtimeEvidence',
  );
  const snapshotKinds = expectEqual(
    enumStringsWithPrefix(transaction, transactionFile, 'mcp.snapshotKinds', 'mcpSnapshotKindApply', 'mcpSnapshotKind'),
    ['apply', 'restore'],
    transactionFile,
    'mcp.snapshotKinds',
  );
  const ownershipFields = structJSONFields(
    transaction,
    transactionFile,
    'mcp.ownershipFields',
    'mcpOwnershipManifest',
  );
  expectEqual(
    ownershipFields,
    [
      'format', 'manifest_version', 'tool_version', 'created', 'target', 'target_format', 'schema',
      'base_digest', 'installed_digest', 'backup_id', 'backup_path', 'runtime_version',
      'runtime_evidence', 'actions', 'entries',
    ],
    transactionFile,
    'mcp.ownershipFields',
  );
  const ownershipEntryFields = expectEqual(
    structJSONFields(transaction, transactionFile, 'mcp.ownershipEntryFields', 'mcpManifestEntry'),
    ['name', 'installed_digest', 'backup_id'],
    transactionFile,
    'mcp.ownershipEntryFields',
  );
  expectPresent(
    transaction,
    transactionFile,
    'mcp.separateStore',
    /separate from the skillsync backup manifest and from Feature 014's fleet/,
    'the separate MCP ownership store statement was not found',
  );

  const cliFile = 'cmd/agentic-sdd/mcp.go';
  const cli = sources.get(cliFile);
  const commands = [
    { command: 'mcp status', usage: 'asdx-core mcp [status] [--home PATH] [--config PATH] [--project PATH] [--json]' },
    { command: 'mcp doctor', usage: 'asdx-core mcp doctor [--home PATH] [--config PATH] [--project PATH]' },
    { command: 'mcp preview', usage: 'asdx-core mcp preview [--servers NAME,...] [--auth SERVER=MODE]' },
    { command: 'mcp apply', usage: 'asdx-core mcp apply --expect-config SHA256 [--opencode-bin PATH]' },
    { command: 'mcp restore list', usage: 'asdx-core mcp restore list [--home PATH] [--json]' },
    { command: 'mcp restore preview', usage: 'asdx-core mcp restore preview ID [--home PATH] [--config PATH] [--json]' },
    { command: 'mcp restore apply', usage: 'asdx-core mcp restore apply ID --expect-config SHA256' },
  ];
  for (const entry of commands) {
    expectPresent(cli, cliFile, 'mcp.cli', new RegExp(entry.usage.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  }
  expectPresent(cli, cliFile, 'mcp.cli', /0 {2}Success, a produced report, or a real no-op/);
  expectPresent(cli, cliFile, 'mcp.cli', /1 {2}Stale plan, conflict, unsupported runtime, rollback or engine\/I\/O error/);
  expectPresent(cli, cliFile, 'mcp.cli', /2 {2}Usage error, flag error, validation, unsafe target or unknown subcommand/);
  expectPresent(cli, cliFile, 'mcp.noCreate', /a missing\s+configuration is never created and there is no replace or force option/);
  expectPresent(cli, cliFile, 'mcp.offline', /No command performs a network call, OAuth flow, package installation or/);
  expectPresent(cli, cliFile, 'mcp.restoreScope', /Whole-file restore is not part of this build/);
  expectPresent(cli, cliFile, 'mcp.planDigest', /Base digest the reviewed preview printed/);
  expectPresent(cli, cliFile, 'mcp.authPlaceholders', /OpenCode owns OAuth credentials/);

  const stateRoot = `<home>/${stateDir}`;
  return {
    maxConfigBytes,
    target: {
      defaultDir: `<home>/${configDir}`,
      files: [defaultJSON, defaultJSONC],
      explicitExtensions: ['.json', '.jsonc'],
      bothPresent: 'conflict',
      neitherPresent: 'none (no file is created)',
    },
    formats,
    schemas,
    legacySchema: 'legacy v1 mcp.<name> entries are refused and never migrated to mcp.servers',
    servers,
    states,
    kinds,
    operations,
    authModes,
    plan: { version: planVersion, digestBound: true },
    ownership: {
      path: `${stateRoot}/${ownershipFile}`,
      format: ownershipFormat,
      manifestVersion: ownershipVersion,
      fields: ownershipFields,
      entryFields: ownershipEntryFields,
      backups: `${stateRoot}/${backupsDirName}`,
      lock: `${stateRoot}/${lockFileName}`,
      snapshotFile: snapshotFileName,
      snapshotFormat,
      intentFile: intentFileName,
      intentPhases,
      runtimeEvidence,
      snapshotKinds,
    },
    cli: commands.map((entry) => entry.command),
  };
}

function verifyExecution(sources) {
  const typesFile = 'internal/execution/types.go';
  const types = sources.get(typesFile);
  const schemaVersion = expectEqual(
    intConst(types, typesFile, 'execution.schemaVersion', 'SchemaVersion'),
    1,
    typesFile,
    'execution.schemaVersion',
  );
  const states = expectEqual(
    enumStrings(types, typesFile, 'execution.states', 'StatePrepared'),
    [
      'prepared', 'running', 'observed-partial', 'observed-failed', 'awaiting-verification',
      'verified-evidence', 'stale', 'blocked',
    ],
    typesFile,
    'execution.states',
  );
  const claimResults = expectEqual(
    enumStrings(types, typesFile, 'execution.claimResults', 'ClaimPassed'),
    ['passed', 'failed', 'partial', 'blocked', 'needs_input'],
    typesFile,
    'execution.claimResults',
  );
  const checkResults = expectEqual(
    enumStrings(types, typesFile, 'execution.checkResults', 'CheckPassed'),
    ['passed', 'failed', 'timed_out', 'interrupted', 'output_limit', 'not_started'],
    typesFile,
    'execution.checkResults',
  );
  const scopeKinds = expectEqual(
    [
      typedStringConst(types, typesFile, 'execution.scopeKinds', 'ScopeFeature'),
      typedStringConst(types, typesFile, 'execution.scopeKinds', 'ScopeDirect'),
    ],
    ['feature', 'direct'],
    typesFile,
    'execution.scopeKinds',
  );
  const fields = {
    runIdentity: expectEqual(
      structJSONFields(types, typesFile, 'execution.runIdentityFields', 'RunIdentity'),
      [
        'schemaVersion', 'kind', 'run_id', 'created_at', 'tool_version', 'run_root', 'run_dir',
        'scope', 'baseline', 'approval_reference', 'workspace_git_visible',
      ],
      typesFile,
      'execution.runIdentityFields',
    ),
    checkpoint: expectEqual(
      structJSONFields(types, typesFile, 'execution.checkpointFields', 'Checkpoint'),
      [
        'schemaVersion', 'run_id', 'updated_at', 'state', 'last_observation', 'tasks', 'checks',
        'records', 'blockers', 'review_packages',
      ],
      typesFile,
      'execution.checkpointFields',
    ),
    claimRecord: expectEqual(
      structJSONFields(types, typesFile, 'execution.claimRecordFields', 'ClaimRecord'),
      [
        'schemaVersion', 'record_id', 'task_id', 'result', 'summary', 'changed_paths', 'checks',
        'blockers', 'agent', 'unknowns', 'received_at', 'raw_path', 'raw_digest',
      ],
      typesFile,
      'execution.claimRecordFields',
    ),
    reviewManifest: expectEqual(
      structJSONFields(types, typesFile, 'execution.reviewManifestFields', 'ReviewManifest'),
      [
        'schemaVersion', 'kind', 'package_id', 'run_id', 'created_at', 'repo_root', 'base', 'head',
        'head_requested', 'branch', 'git_version', 'scope', 'complete', 'partial_reasons', 'files',
        'omissions', 'counts', 'patch_path', 'untracked_dir', 'run_state', 'stale_reasons',
      ],
      typesFile,
      'execution.reviewManifestFields',
    ),
  };
  const limits = {
    tasksBytes: expectEqual(shiftConst(types, typesFile, 'execution.limits', 'MaxTasksFileBytes'), 4 * 1024 * 1024, typesFile, 'execution.limits'),
    specBytes: expectEqual(shiftConst(types, typesFile, 'execution.limits', 'MaxSpecFileBytes'), 8 * 1024 * 1024, typesFile, 'execution.limits'),
    recordBytes: expectEqual(shiftConst(types, typesFile, 'execution.limits', 'MaxRecordBytes'), 1024 * 1024, typesFile, 'execution.limits'),
    checkTimeoutSeconds: expectEqual(durationConst(types, typesFile, 'execution.limits', 'DefaultCheckTimeout'), 900, typesFile, 'execution.limits'),
    outputBytes: expectEqual(shiftConst(types, typesFile, 'execution.limits', 'DefaultMaxOutputBytes'), 64 * 1024 * 1024, typesFile, 'execution.limits'),
    reviewFileBytes: expectEqual(shiftConst(types, typesFile, 'execution.limits', 'DefaultMaxFileBytes'), 2 * 1024 * 1024, typesFile, 'execution.limits'),
    reviewTotalBytes: expectEqual(shiftConst(types, typesFile, 'execution.limits', 'DefaultMaxTotalBytes'), 32 * 1024 * 1024, typesFile, 'execution.limits'),
    lockStaleSeconds: expectEqual(durationConst(types, typesFile, 'execution.limits', 'LockStaleAfter'), 1800, typesFile, 'execution.limits'),
  };

  const workspaceFile = 'internal/execution/workspace.go';
  const workspace = sources.get(workspaceFile);
  const runRootParentDir = expectEqual(
    stringConst(workspace, workspaceFile, 'execution.runRootParent', 'runRootParentDir'),
    '.agentic-sdd',
    workspaceFile,
    'execution.runRootParent',
  );
  const runRootDir = expectEqual(
    stringConst(workspace, workspaceFile, 'execution.runRootDir', 'runRootDirName'),
    'runs',
    workspaceFile,
    'execution.runRootDir',
  );
  const runIdentityName = expectEqual(
    stringConst(workspace, workspaceFile, 'execution.runIdentityName', 'runIdentityName'),
    'run.json',
    workspaceFile,
    'execution.runIdentityName',
  );
  const checkpointName = expectEqual(
    stringConst(workspace, workspaceFile, 'execution.checkpointName', 'checkpointName'),
    'checkpoint.json',
    workspaceFile,
    'execution.checkpointName',
  );
  const lockName = expectEqual(
    stringConst(workspace, workspaceFile, 'execution.lockName', 'lockFileName'),
    '.lock',
    workspaceFile,
    'execution.lockName',
  );
  const ignoreName = expectEqual(
    stringConst(workspace, workspaceFile, 'execution.ignoreName', 'ignoreFileName'),
    '.gitignore',
    workspaceFile,
    'execution.ignoreName',
  );
  const ignoreData = expectEqual(
    stringConst(workspace, workspaceFile, 'execution.ignoreData', 'ignoreFileData'),
    '*',
    workspaceFile,
    'execution.ignoreData',
  );
  expectPresent(workspace, workspaceFile, 'execution.atomicWrites', /writeFileAtomic\(filepath\.Join\(runDir, runIdentityName\), append\(data, '\\n'\), 0o600\)/);
  expectPresent(sources.get('internal/execution/checkpoint.go'), 'internal/execution/checkpoint.go', 'execution.atomicWrites', /writeFileAtomic\(filepath\.Join\(runDir, checkpointName\), append\(data, '\\n'\), 0o600\)/);
  expectPresent(workspace, workspaceFile, 'execution.privateModes', /os\.Mkdir\(runDir, 0o700\)/);
  expectPresent(workspace, workspaceFile, 'execution.privateModes', /os\.MkdirAll\(runRoot, 0o700\)/);
  expectPresent(types, typesFile, 'execution.boundary', /never approves work, calls a model, or acts as/);
  const runIdPattern = expectEqual(
    regexpConst(workspace, workspaceFile, 'execution.runIdPattern', 'runIDPattern'),
    '^run-[0-9]{8}T[0-9]{6}Z-[0-9a-f]{6}$',
    workspaceFile,
    'execution.runIdPattern',
  );
  expectPresent(sources.get('internal/execution/checks.go'), 'internal/execution/checks.go', 'execution.artifacts', /ensureArtifactDir\(runDir, "checks"\)/);
  expectPresent(sources.get('internal/execution/taskbrief.go'), 'internal/execution/taskbrief.go', 'execution.artifacts', /filepath\.Join\(runDir, "briefs", extracted\.TaskID\+"\.md"\)/);
  expectPresent(sources.get('internal/execution/record.go'), 'internal/execution/record.go', 'execution.artifacts', /ensureArtifactDir\(runDir, "records"\)/);
  expectPresent(sources.get('internal/execution/reviewpackage.go'), 'internal/execution/reviewpackage.go', 'execution.artifacts', /ensureArtifactDir\(runDir, "review"\)/);
  expectPresent(sources.get('internal/execution/reviewpackage.go'), 'internal/execution/reviewpackage.go', 'execution.artifacts', /filepath\.Join\(pkgDir, "review\.json"\)/);
  expectPresent(sources.get('internal/execution/reviewpackage.go'), 'internal/execution/reviewpackage.go', 'execution.artifacts', /filepath\.Join\(pkgDir, "review\.patch"\)/);
  const runKind = expectEqual(
    requireMatch(workspace, workspaceFile, 'execution.kinds', /Kind:\s+"(execution\.run\.identity)"/, 'run identity kind was not found')[1],
    'execution.run.identity',
    workspaceFile,
    'execution.kinds',
  );
  const taskbriefFile = 'internal/execution/taskbrief.go';
  const taskbrief = sources.get(taskbriefFile);
  const briefKind = expectEqual(
    requireMatch(taskbrief, taskbriefFile, 'execution.kinds', /Kind:\s+"(execution\.task\.brief)"/, 'task brief kind was not found')[1],
    'execution.task.brief',
    taskbriefFile,
    'execution.kinds',
  );
  const reviewFile = 'internal/execution/reviewpackage.go';
  const review = sources.get(reviewFile);
  const reviewKind = expectEqual(
    requireMatch(review, reviewFile, 'execution.kinds', /Kind:\s+"(execution\.review\.package)"/, 'review package kind was not found')[1],
    'execution.review.package',
    reviewFile,
    'execution.kinds',
  );
  const packagePrefix = expectEqual(
    requireMatch(review, reviewFile, 'execution.artifacts', /newArtifactID\("([a-z]+)", now\)/, 'review package ID prefix was not found')[1],
    'pkg',
    reviewFile,
    'execution.artifacts',
  );

  const checksFile = 'internal/execution/checks.go';
  const checks = sources.get(checksFile);
  const checkFiles = ['stdoutLogName', 'stderrLogName', 'checkMetaName', 'checkArgvName'].map((name) => (
    stringConst(checks, checksFile, 'execution.artifacts', name)
  ));
  expectEqual(checkFiles, ['stdout.log', 'stderr.log', 'meta.json', 'argv.json'], checksFile, 'execution.artifacts');
  const recordPrefix = expectEqual(
    requireMatch(sources.get('internal/execution/record.go'), 'internal/execution/record.go', 'execution.artifacts', /newArtifactID\("(rec)", now\)/, 'record ID prefix was not found')[1],
    'rec',
    'internal/execution/record.go',
    'execution.artifacts',
  );

  const runCLIFile = 'cmd/agentic-sdd/run.go';
  const runCLI = sources.get(runCLIFile);
  const reviewCLIFile = 'cmd/agentic-sdd/review.go';
  const reviewCLI = sources.get(reviewCLIFile);
  const executionCLIFile = 'cmd/agentic-sdd/execution_cli.go';
  const executionCLI = sources.get(executionCLIFile);
  for (const usage of [
    'asdx-core run init --feature DIR --batch ID',
    'asdx-core run status --run ID',
    'asdx-core run record --run ID',
    'asdx-core run clean --run ID',
    'asdx-core run check --run ID',
  ]) {
    expectPresent(runCLI, runCLIFile, 'execution.cli', new RegExp(usage.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
  }
  expectPresent(reviewCLI, reviewCLIFile, 'execution.cli', /asdx-core review package --run ID --base COMMIT/);
  const commonFlagsBody = funcBody(executionCLI, executionCLIFile, 'execution.cli', 'func newExecutionFlagSet\\(name string, common \\*executionCommonFlags\\) \\*flag\\.FlagSet');
  const commonFlags = ['repo', 'run-root', 'format'].map((name) => (
    expectPresent(commonFlagsBody, executionCLIFile, 'execution.cli', new RegExp(`"${name}"`)) && `--${name}`
  ));
  expectPresent(commonFlagsBody, executionCLIFile, 'execution.cli', /"format", "text", "output format: text or json"/);
  expectPresent(runCLI, runCLIFile, 'execution.exitCodes', /return 124/);
  expectPresent(reviewCLI, reviewCLIFile, 'execution.exitCodes', /return 3/);
  expectPresent(reviewCLI, reviewCLIFile, 'execution.exitCodes', /writeExecutionJSON\(stdout, manifest\)/);
  expectPresent(executionCLI, executionCLIFile, 'execution.exitCodes', /errors\.Is\(err, execution\.ErrPartial\):\s*\n\t\treturn 3/);
  expectPresent(executionCLI, executionCLIFile, 'execution.exitCodes', /return 2/);
  expectPresent(executionCLI, executionCLIFile, 'execution.exitCodes', /return 1/);

  return {
    schemaVersion,
    kinds: { run: runKind, brief: briefKind, review: reviewKind },
    runRoot: {
      default: `<repo>/${runRootParentDir}/${runRootDir}`,
      flag: '--run-root',
    },
    ignore: { file: ignoreName, content: ignoreData },
    artifacts: {
      identity: runIdentityName,
      checkpoint: checkpointName,
      lock: lockName,
      briefs: 'briefs/<task>.md',
      checks: 'checks/<check-id>/',
      records: `records/<${recordPrefix}-id>.json`,
      review: `review/<${packagePrefix}-id>/`,
      reviewManifest: 'review.json',
      reviewPatch: 'review.patch',
      checkFiles: [
        checkFiles[2],
        checkFiles[3],
        checkFiles[0],
        checkFiles[1],
      ],
    },
    runIdPattern,
    states,
    claimResults,
    checkResults,
    scopeKinds,
    fields,
    limits,
    cli: {
      commonFlags,
      formatDefault: 'text',
      exitCodes: [0, 1, 2, 3, 124],
    },
  };
}

function verifyFilesAndBackups(sources, { fleet, mcp }) {
  const clientsFile = 'internal/skillsync/clients.go';
  const clientsSource = sources.get(clientsFile);
  const supportedList = requireMatch(
    clientsSource,
    clientsFile,
    'files.clients',
    /var supportedClientIDs = \[\]string\{([^}]*)\}/,
    'supportedClientIDs was not found',
  )[1];
  const clientIds = expectEqual(
    quotedValues(supportedList),
    ['agents', 'codex', 'claude', 'agy'],
    clientsFile,
    'files.clients',
  );

  const syncFile = 'internal/skillsync/sync.go';
  const sync = sources.get(syncFile);
  const clientTargetsBody = funcBody(sync, syncFile, 'files.clients', 'func clientTargets\\(home string\\) \\[\\]target');
  const clientPaths = [
    ['agents', '.agents/skills'],
    ['codex', '.codex/skills'],
    ['claude', '.claude/skills'],
    ['agy', '.gemini/antigravity-cli/skills'],
  ].map(([id, relative]) => {
    const segments = relative.split('/');
    const pattern = new RegExp(
      `filepath\\.Join\\(home, ${segments.map((segment) => `"${segment.replace('.', '\\.')}"`).join(', ')}\\)`,
    );
    expectPresent(clientTargetsBody, syncFile, 'files.clients', pattern, `client destination ${relative} was not found`);
    return { id, path: `<home>/${relative}` };
  });
  expectEqual(clientPaths.map((client) => client.id), clientIds, syncFile, 'files.clients');

  const openCodeTargetBody = funcBody(sync, syncFile, 'files.openCodeAgents', 'func openCodeAgentsTarget\\(home string\\) target');
  const openCodeJoin = requireMatch(
    openCodeTargetBody,
    syncFile,
    'files.openCodeAgents',
    /filepath\.Join\(([^)]*)\)/,
    'opencode agents target path was not found',
  )[1];
  const openCodeSegments = expectEqual(
    quotedValues(openCodeJoin),
    ['.config', 'opencode', 'agents'],
    syncFile,
    'files.openCodeAgents',
  );
  const openCodeClient = expectEqual(
    requireMatch(sync, syncFile, 'files.openCodeAgents', /name: "(opencode-agents)"/, 'opencode-agents target name was not found')[1],
    'opencode-agents',
    syncFile,
    'files.openCodeAgents',
  );

  const backupRootBody = funcBody(sync, syncFile, 'files.backupRoots', 'func backupRootFor\\(repo, home string\\) \\(root, sourceLabel string\\)');
  const backupSegments = quotedValues(backupRootBody).filter((value) => value !== '');
  expectEqual(backupSegments, ['backups', 'skills-files', '.agentic-sdd', 'backups', 'embedded'], syncFile, 'files.backupRoots');
  const backupIdPattern = expectEqual(
    requireMatch(sync, syncFile, 'files.backupIdPattern', /stamp := now\.Format\("([^"]+)"\)/, 'backup stamp format was not found')[1],
    '20060102T150405.000000000Z',
    syncFile,
    'files.backupIdPattern',
  );
  const formatTwo = expectEqual(
    Number(requireMatch(sync, syncFile, 'skillsync.formatTwo', /format := (\d+)/, 'baseline manifest format was not found')[1]),
    2,
    syncFile,
    'skillsync.formatTwo',
  );
  const formatThree = expectEqual(
    Number(requireMatch(sync, syncFile, 'skillsync.formatThree', /if hasFileChanges \{\s*\n\t\tformat = (\d+)/, 'format 3 condition was not found')[1]),
    3,
    syncFile,
    'skillsync.formatThree',
  );
  const entryKindsBody = [...sync.matchAll(/e\.Kind = "([^"]+)"/g)].map((match) => match[1]);
  const entryKinds = expectEqual(entryKindsBody, ['opencode-agent', 'skill'], syncFile, 'files.entryKinds');

  const manifestFile = 'internal/skillsync/manifest.go';
  const manifest = sources.get(manifestFile);
  const allFields = structJSONFields(manifest, manifestFile, 'skillsync.manifestFields', 'manifest');
  const formatOne = expectEqual(allFields.slice(0, 4), ['created', 'source', 'targets', 'skills'], manifestFile, 'skillsync.manifestFields');
  const additive = expectEqual(
    allFields.slice(4),
    [
      'format', 'id', 'created_at', 'operation', 'restored_from', 'tool', 'home', 'backup_root',
      'opencode_targets', 'entries',
    ],
    manifestFile,
    'skillsync.manifestFields',
  );
  const toolFields = expectEqual(
    structJSONFields(manifest, manifestFile, 'skillsync.toolFields', 'toolInfo'),
    ['version', 'commit', 'date', 'go_version'],
    manifestFile,
    'skillsync.toolFields',
  );
  const entryFields = expectEqual(
    structJSONFields(manifest, manifestFile, 'skillsync.entryFields', 'entry'),
    ['kind', 'client', 'skill', 'path', 'action', 'backup', 'sha256'],
    manifestFile,
    'skillsync.entryFields',
  );
  const actions = expectEqual(
    [
      stringConst(manifest, manifestFile, 'skillsync.actions', 'actionReplaced'),
      stringConst(manifest, manifestFile, 'skillsync.actions', 'actionInstalled'),
    ],
    ['replaced', 'installed'],
    manifestFile,
    'skillsync.actions',
  );
  expectPresent(manifest, manifestFile, 'skillsync.treeDigest', /fmt\.Fprintf\(h, "d %s\\n", rel\)/);
  expectPresent(manifest, manifestFile, 'skillsync.treeDigest', /fmt\.Fprintf\(h, "f %s\\n", rel\)/);
  expectPresent(manifest, manifestFile, 'skillsync.fileDigest', /refusing special file %s/);
  expectPresent(manifest, manifestFile, 'skillsync.legacy', /return m\.Format < 2/);

  const restoreFile = 'internal/skillsync/restore.go';
  const restore = sources.get(restoreFile);
  const managedPrefix = expectEqual(
    requireMatch(restore, restoreFile, 'files.managedPrefix', /strings\.HasPrefix\(e\.Skill, "(agentic-sdd-)"\)/, 'managed skill prefix was not found')[1],
    'agentic-sdd-',
    restoreFile,
    'files.managedPrefix',
  );
  expectPresent(sources.get('internal/skillsync/source.go'), 'internal/skillsync/source.go', 'files.managedPrefix', /that "agentic-sdd-<name>" entries sit at its top level/);
  expectPresent(restore, restoreFile, 'files.restoreHome', /backup %s was made for a different home/);
  expectPresent(restore, restoreFile, 'files.restoreHome', /refusing special manifest\.json in backup/);

  return {
    clients: clientPaths,
    managedPrefix,
    openCodeAgents: {
      client: openCodeClient,
      path: `<home>/${openCodeSegments.join('/')}`,
      manifestKind: entryKinds[0],
    },
    backupRoots: [
      { when: 'embedded skills (no explicit --repo)', path: '<home>/.agentic-sdd/backups' },
      { when: 'explicit --repo checkout', path: '<repo>/backups' },
    ],
    backupSourceLabels: ['<repo>/skills-files', 'embedded'],
    backupIdPattern,
    manifestFile: 'manifest.json',
    formats: {
      one: formatOne,
      two: additive,
      three: { marker: 'opencode_targets', entryKind: 'opencode-agent', digest: 'single-file SHA-256' },
      baseline: formatTwo,
      fileChange: formatThree,
    },
    toolFields,
    entryFields,
    actions,
    entryKinds,
    digests: {
      tree: 'relative paths, entry kinds (file or directory) and file contents in stable lexical order',
      file: 'raw bytes of one regular file; special files are refused',
    },
    restore: { currentHomeRequired: true, validatesSavedContentBeforeUse: true },
    fleetSidecar: {
      file: fleet.manifest.file,
      format: fleet.manifest.format,
      owner: 'internal/fleet',
    },
    mcpOwnership: {
      path: mcp.ownership.path,
      format: mcp.ownership.format,
      manifestVersion: mcp.ownership.manifestVersion,
      owner: 'internal/skillsync MCP-B',
    },
  };
}

function verifyConfiguration(sources, parts) {
  const envFile = 'cmd/agentic-sdd/env.go';
  const env = sources.get(envFile);
  const cliEnvNames = [...new Set([...env.matchAll(/getenv\("([^"]+)"\)/g)].map((match) => match[1]))];
  expectEqual(
    cliEnvNames,
    ['AGENTIC_SDD_FORCE_INTERACTIVE', 'TERM', 'NO_COLOR', 'AGENTIC_SDD_ANIMATE', 'COLUMNS'],
    envFile,
    'configuration.environment',
  );
  expectPresent(env, envFile, 'configuration.environment', /forceInteractive := getenv\("AGENTIC_SDD_FORCE_INTERACTIVE"\) == "1"/);
  expectPresent(env, envFile, 'configuration.environment', /animate := getenv\("AGENTIC_SDD_ANIMATE"\) == "1"/);
  expectPresent(env, envFile, 'configuration.environment', /color := outTTY && getenv\("NO_COLOR"\) == "" && !ascii/);
  expectPresent(env, envFile, 'configuration.environment', /ascii := term == "dumb"/);
  expectPresent(env, envFile, 'configuration.environment', /width := 80/);
  expectPresent(env, envFile, 'configuration.environment', /AGENTIC_SDD_ANIMATE=1 opt-in; further gates apply/);
  expectPresent(sources.get('cmd/agentic-sdd/trace.go'), 'cmd/agentic-sdd/trace.go', 'configuration.environment', /one\s*\n\/\/ JSON object per event/);

  const mainFile = 'cmd/agentic-sdd/main.go';
  const main = sources.get(mainFile);
  const traceVariable = expectEqual(
    requireMatch(main, mainFile, 'configuration.traceVariable', /strings\.TrimSpace\(os\.Getenv\("([^"]+)"\)\)/, 'trace environment variable was not found')[1],
    'AGENTIC_SDD_TRACE',
    mainFile,
    'configuration.traceVariable',
  );
  const homeBody = funcBody(main, mainFile, 'configuration.homeDefault', 'func resolveHome\\(home string\\) \\(string, error\\)');
  expectPresent(homeBody, mainFile, 'configuration.homeDefault', /os\.UserHomeDir\(\)/);

  const sourceFile = 'internal/skillsync/source.go';
  const source = sources.get(sourceFile);
  expectPresent(source, sourceFile, 'configuration.repoDefault', /return fs\.Sub\(skillsfiles\.Files, "skills-files"\)/);
  expectPresent(source, sourceFile, 'configuration.repoDefault', /return fs\.Sub\(os\.DirFS\(repo\), "skills-files"\)/);

  const tuiRunFile = 'experiments/tui/cmd/internal/tuirun/run.go';
  const tuiRun = sources.get(tuiRunFile);
  const tuiEnv = expectEqual(
    [
      requireMatch(tuiRun, tuiRunFile, 'configuration.environment', /Getenv\("(AGENTIC_SDD_REDUCED_MOTION)"\)/, 'TUI reduced-motion variable was not found')[1],
      requireMatch(tuiRun, tuiRunFile, 'configuration.environment', /Getenv\("(AGENTIC_SDD_ASCII)"\)/, 'TUI ASCII variable was not found')[1],
    ],
    ['AGENTIC_SDD_REDUCED_MOTION', 'AGENTIC_SDD_ASCII'],
    tuiRunFile,
    'configuration.environment',
  );

  const allNames = [...new Set([
    ...cliEnvNames,
    traceVariable,
    ...tuiEnv,
  ])].sort();
  expectEqual(
    allNames,
    ['AGENTIC_SDD_ANIMATE', 'AGENTIC_SDD_ASCII', 'AGENTIC_SDD_FORCE_INTERACTIVE', 'AGENTIC_SDD_REDUCED_MOTION', 'AGENTIC_SDD_TRACE', 'COLUMNS', 'NO_COLOR', 'TERM'],
    envFile,
    'configuration.environment',
  );
  const unexpected = allNames.filter((name) => !PRESENTATION_ENVIRONMENT.has(name));
  expectEqual(unexpected, [], envFile, 'configuration.pathOverrides');

  const executionCLIFile = 'cmd/agentic-sdd/execution_cli.go';
  expectPresent(sources.get(executionCLIFile), executionCLIFile, 'configuration.outputs', /"format", "text", "output format: text or json"/);
  const mcpCLIFile = 'cmd/agentic-sdd/mcp.go';
  expectPresent(sources.get(mcpCLIFile), mcpCLIFile, 'configuration.outputs', /--json\s+Emit stable, script-compatible JSON instead of text\./);
  const fleetPlanFile = 'cmd/agentic-sdd/fleet_plan.go';
  expectPresent(sources.get(fleetPlanFile), fleetPlanFile, 'configuration.outputs', /--output PATH {2}Write canonical JSON plan to specified file/);

  const environment = [
    {
      name: 'AGENTIC_SDD_TRACE',
      values: '<path>',
      effect: 'append the opt-in JSON-Lines flow trace; unset or empty disables it',
      scope: 'CLI',
    },
    {
      name: 'AGENTIC_SDD_FORCE_INTERACTIVE',
      values: 'exactly 1',
      effect: 'treat stdin, stdout and stderr as terminals for scripted terminal behavior',
      scope: 'CLI presentation and TUI launch gate',
    },
    {
      name: 'AGENTIC_SDD_ANIMATE',
      values: 'exactly 1',
      effect: 'opt in to animation; further terminal gates apply',
      scope: 'CLI guide presentation',
    },
    {
      name: 'AGENTIC_SDD_ASCII',
      values: 'exactly 1',
      effect: 'use ASCII borders and symbols',
      scope: 'TUI companion',
    },
    {
      name: 'AGENTIC_SDD_REDUCED_MOTION',
      values: 'exactly 1',
      effect: 'disable TUI animation',
      scope: 'TUI companion',
    },
    {
      name: 'NO_COLOR',
      values: 'any non-empty value',
      effect: 'disable ANSI color',
      scope: 'CLI and TUI presentation',
    },
    {
      name: 'TERM',
      values: 'dumb',
      effect: 'disable color and select ASCII-only glyphs',
      scope: 'CLI presentation',
    },
    {
      name: 'COLUMNS',
      values: 'positive integer',
      effect: 'set terminal width; otherwise 80',
      scope: 'CLI presentation',
    },
  ];

  return {
    precedence: [
      {
        subsystem: 'skill installation',
        selector: '--repo',
        default: 'skills-files embedded in the binary',
        note: 'an explicit checkout reads the skills-files/ directory under the repository root',
      },
      {
        subsystem: 'skill installation and fleet',
        selector: '--home',
        default: 'the OS user home directory (os.UserHomeDir)',
        note: 'all client, agent, backup and MCP state paths derive from this value',
      },
      {
        subsystem: 'fleet',
        selector: `${parts.fleet.configFlag} PATH`,
        default: `none: personal ${parts.fleet.configFile} is required`,
        note: 'no product environment variable or implicit default path selects it',
      },
      {
        subsystem: 'MCP',
        selector: '--config PATH',
        default: `${parts.mcp.target.defaultDir}/${parts.mcp.target.files[0]} or ${parts.mcp.target.defaultDir}/${parts.mcp.target.files[1]}`,
        note: 'an explicit path wins; both default files present is an ambiguity conflict; neither present is format none and no file is created',
      },
      {
        subsystem: 'execution',
        selector: `${parts.execution.runRoot.flag} DIR`,
        default: parts.execution.runRoot.default,
        note: '--repo defaults to the current working tree',
      },
      {
        subsystem: 'skill backups',
        selector: '--repo',
        default: `${parts.filesAndBackups.backupRoots[0].path} when assets are embedded`,
        note: 'an explicit checkout keeps backups under its own backups/ directory',
      },
    ],
    environment,
    pathOverrides: { productEnvironmentVariables: false },
    outputs: {
      execution: { format: ['text', 'json'], schemaVersion: parts.execution.schemaVersion, kindField: true },
      mcp: { jsonFlag: '--json' },
      fleet: { planFile: '--output PATH writes canonical JSON', config: 'fleet.json' },
    },
    homeDefault: 'os.UserHomeDir',
    repoDefault: 'embedded skills-files',
  };
}

function buildInventory(fleet, mcp, execution, filesAndBackups, configuration, pages) {
  const sourcesFor = (id) => pages.find((page) => page.pageId === `reference/${id}`).sources;
  return {
    systems: [
      { id: 'configuration', pageId: 'reference/configuration', sources: sourcesFor('configuration'), facts: configuration },
      { id: 'fleet', pageId: 'reference/fleet', sources: sourcesFor('fleet'), facts: fleet },
      { id: 'mcp', pageId: 'reference/mcp', sources: sourcesFor('mcp'), facts: mcp },
      { id: 'execution', pageId: 'reference/execution', sources: sourcesFor('execution'), facts: execution },
      { id: 'files-and-backups', pageId: 'reference/files-and-backups', sources: sourcesFor('files-and-backups'), facts: filesAndBackups },
    ],
  };
}

const PAGE_SOURCES = {
  'reference/configuration': [
    'cmd/agentic-sdd/env.go',
    'cmd/agentic-sdd/execution_cli.go',
    'cmd/agentic-sdd/fleet.go',
    'cmd/agentic-sdd/main.go',
    'cmd/agentic-sdd/mcp.go',
    'cmd/agentic-sdd/trace.go',
    'cmd/agentic-sdd/tui.go',
    'experiments/tui/cmd/internal/tuirun/run.go',
    'internal/execution/workspace.go',
    'internal/fleet/fleet.go',
    'internal/skillsync/mcp_config.go',
    'internal/skillsync/source.go',
  ],
  'reference/fleet': [
    'cmd/agentic-sdd/fleet.go',
    'cmd/agentic-sdd/fleet_plan.go',
    'internal/buildinfo/buildinfo.go',
    'internal/fleet/fleet.go',
    'internal/fleet/manifest.go',
    'internal/fleet/parse.go',
    'internal/fleet/resolve.go',
  ],
  'reference/mcp': [
    'cmd/agentic-sdd/mcp.go',
    'internal/skillsync/mcp.go',
    'internal/skillsync/mcp_config.go',
    'internal/skillsync/mcp_transaction.go',
  ],
  'reference/execution': [
    'cmd/agentic-sdd/execution_cli.go',
    'cmd/agentic-sdd/review.go',
    'cmd/agentic-sdd/run.go',
    'internal/execution/checkpoint.go',
    'internal/execution/checks.go',
    'internal/execution/record.go',
    'internal/execution/reviewpackage.go',
    'internal/execution/taskbrief.go',
    'internal/execution/types.go',
    'internal/execution/workspace.go',
  ],
  'reference/files-and-backups': [
    'internal/execution/workspace.go',
    'internal/fleet/manifest.go',
    'internal/skillsync/clients.go',
    'internal/skillsync/manifest.go',
    'internal/skillsync/mcp_transaction.go',
    'internal/skillsync/restore.go',
    'internal/skillsync/source.go',
    'internal/skillsync/sync.go',
  ],
};

const PAGE_TITLES = {
  'reference/configuration': 'Configuration and precedence',
  'reference/fleet': 'Fleet configuration',
  'reference/mcp': 'MCP configuration',
  'reference/execution': 'Execution runs',
  'reference/files-and-backups': 'Files, destinations and backups',
};

function code(value) {
  return `\`${value}\``;
}

function table(headers, rows) {
  const render = (cells) => `| ${cells.map((cell) => String(cell).replaceAll('|', '\\|')).join(' | ')} |`;
  return [render(headers), render(headers.map(() => '---')), ...rows.map(render)].join('\n');
}

function bullet(items) {
  return items.map((item) => `- ${item}`).join('\n');
}

function bytes(value) {
  const mib = value / (1024 * 1024);
  return Number.isInteger(mib) ? `${value} bytes (${mib} MiB)` : `${value} bytes`;
}

function duration(totalSeconds) {
  const minutes = totalSeconds / 60;
  return Number.isInteger(minutes) ? `${totalSeconds} seconds (${minutes} minutes)` : `${totalSeconds} seconds`;
}

function configurationMarkdown(facts) {
  return `Source-derived configuration facts for the current checkout. They do not certify runtime behavior, provider availability or release support.

## Precedence by subsystem

${table(
    ['Subsystem', 'Selector', 'Default', 'Boundary'],
    facts.precedence.map((entry) => [entry.subsystem, code(entry.selector), code(entry.default), entry.note]),
  )}

No product-specific environment variable in the inspected CLI and TUI sources overrides these selectors. The only implicit home source is the OS user-home lookup behind ${code('--home')}.

## Environment variables

${table(
    ['Variable', 'Accepted values', 'Effect', 'Scope'],
    facts.environment.map((entry) => [code(entry.name), code(entry.values), entry.effect, entry.scope]),
  )}

These variables affect presentation, tracing or the TUI launch gate. They do not change the fleet config path, the MCP config path, the run root or a backup root.

## Output contracts

${table(
    ['Subsystem', 'Contract'],
    [
      ['Execution', `${code('--format text|json')}; JSON carries ${code('schemaVersion')} ${facts.outputs.execution.schemaVersion} and a kind value`],
      ['MCP', `${code('--json')} emits stable, script-compatible JSON instead of text`],
      ['Fleet', `${code('--output PATH')} writes the canonical plan JSON; ${code('fleet.json')} remains the input contract`],
    ],
  )}

## Boundary

Configuration is read from explicit or documented locations scoped by subsystem. There is no single unified configuration schema: each subsystem owns its parser, constants and defaults, and the reference pages keep those contracts separate.
`;
}

function fleetMarkdown(facts) {
  return `Source-derived fleet facts for the current checkout. Preview planning is read-only; apply and restore remain explicit write operations.

## ${code('fleet.json')} schema

Only schemaVersion 1 is accepted. The document is strict JSON, bounded to ${bytes(facts.maxConfigBytes)}.

${table(
    ['Root field', 'Contract'],
    [
      ['schemaVersion', `Must equal ${facts.schemaVersion}`],
      ['fleet', 'Optional fleet label; no path separators or traversal'],
      ['preset', 'Optional preset label; no path separators or traversal'],
      ['instances', 'Map of instance ID to instance object'],
    ],
  )}

${table(
    ['Instance field', 'Contract'],
    [
      ['source', `Required one-hop reference matching ${code(facts.sourceRefPattern)}`],
      ['model', `Optional override matching ${code(facts.modelPattern)}`],
      ['enabled', `Optional boolean; omitted or null defaults to ${facts.enabledDefault}`],
    ],
  )}

Instance IDs match ${code(facts.instanceIdPattern)}. Reserved instance names are ${facts.reservedNames.map(code).join(', ')}.

${bullet(facts.parseRules)}

## Model grammar

${code(facts.modelPattern)}

An empty or omitted model leaves the canonical source model untouched. The optional ${code('#variant')} suffix is part of the same model token, and empty path segments are rejected.

## Source references

${bullet([
    `Allowed namespaces: ${facts.sourceKinds.map(code).join(' and ')}`,
    `Walked source roots: ${facts.sourceRoots.map(code).join(', ')}`,
    'Exactly one hop is resolved; instance-to-instance aliases and cycles are rejected',
    'A name may exist in exactly one walked root; duplicate agent/adapter names fail resolution',
    'Unknown sources fail instead of rendering an empty agent',
  ])}

## Target and sidecar

${table(
    ['Item', 'Contract'],
    [
      ['Scope', code(facts.target.scope)],
      ['Target root', code(facts.target.root)],
      ['Sidecar file', `${code(facts.manifest.file)} inside the backup directory`],
      ['Sidecar format', 'format ' + facts.manifest.format],
      ['Transaction states', facts.manifest.states.map(code).join(', ')],
      ['Per-file provenance', facts.manifest.provenance.map(code).join(', ')],
    ],
  )}

The sidecar fields are ${facts.manifest.fields.map(code).join(', ')}. Per-file records carry ${facts.manifest.fileFields.map(code).join(', ')}. Fleet never writes the user's ${code('AGENTS.md')}; names are bare ${code('.md')} file names under the target root.

## CLI

${table(
    ['Command', 'Flags', 'Required'],
    facts.cli.map((entry) => [code(entry.command), entry.flags.map(code).join(', '), entry.required.length ? entry.required.map(code).join(', ') : 'none']),
  )}

Documented exit codes are 0 (success), 1 (runtime, lock, stale plan or unmanaged conflict) and 2 (usage, flags, invalid configuration or malformed plan).

## TUI build capability

The Fleet Manager live controls are a build-time marker, not a user preference. Valid markers are ${facts.tui.markers.map(code).join(' and ')}; the shipped default is ${code(facts.tui.default)}, and an absent, malformed or unknown marker fails safe to preview. The marker is injected with ${code(facts.tui.injection)}; there is ${facts.tui.override === 'none' ? 'no environment, config, CLI or key override' : facts.tui.override}.

## Separate contracts

The ${code('fleet.json')} schema (version ${facts.schemaVersion}) and the ${code(facts.manifest.file)} sidecar (format ${facts.manifest.format}) are distinct from the skill backup ${code('manifest.json')} formats 1-3 and from the MCP ownership store. They do not share a schema version.
`;
}

function mcpMarkdown(facts) {
  return `Source-derived MCP facts for the current checkout. Status, doctor and preview are read-only and offline; apply and restore require the expected base digest from a reviewed preview before they write.

## Target selection

${table(
    ['Case', 'Result'],
    [
      ['Explicit ' + code('--config'), `Must end in ${facts.target.explicitExtensions.map(code).join(' or ')}; an explicit missing path is a missing target, not a new file`],
      ['Default JSON', code(`${facts.target.defaultDir}/${facts.target.files[0]}`)],
      ['Default JSONC', code(`${facts.target.defaultDir}/${facts.target.files[1]}`)],
      ['Both default files', 'Ambiguity conflict; no target is chosen'],
      ['Neither default file', facts.target.neitherPresent],
    ],
  )}

A symlink, special file or out-of-scope target is refused. The configuration read is bounded to ${bytes(facts.maxConfigBytes)}.

## Schema and format

${bullet([
    `Accepted target formats: ${facts.formats.map(code).join(', ')}`,
    `Observed schema values: ${facts.schemas.map(code).join(', ')}`,
    `Managed entries live under ${code('mcp.servers')} in the OpenCode V2 shape`,
    `${facts.legacySchema}; there is no automatic rewrite`,
    'A JSONC edit is refused when comment and formatting preservation cannot be proven',
  ])}

## Default managed servers

${table(
    ['Server', 'Kind', 'Purpose', 'Prerequisite'],
    facts.servers.map((server) => [code(server.name), server.kind, server.purpose, server.prerequisite || 'none']),
  )}

## Plan, apply and restore

Plan identity is ${code(facts.plan.version)}. A plan is bound to the base configuration digest and its own plan digest; an apply must present the expected base digest or fail as stale.

${bullet([
    `Plan operations: ${facts.operations.map(code).join(', ')}`,
    `Observed server states: ${facts.states.map(code).join(', ')}`,
    `Authentication modes: ${facts.authModes.map(code).join(', ')}`,
    `Snapshot kinds: ${facts.ownership.snapshotKinds.map(code).join(', ')}`,
    'A missing configuration is never created; there is no replace or force option',
    'No command performs a network call, OAuth flow, package installation or model validation',
  ])}

## Ownership and recovery store

${table(
    ['Item', 'Contract'],
    [
      ['Ownership manifest', code(facts.ownership.path)],
      ['Manifest format', `${facts.ownership.format} (manifest_version ${facts.ownership.manifestVersion})`],
      ['Manifest fields', facts.ownership.fields.map(code).join(', ')],
      ['Per-entry fields', facts.ownership.entryFields.map(code).join(', ')],
      ['Backups', code(facts.ownership.backups)],
      ['Lock', code(facts.ownership.lock)],
      ['Snapshot', `${code(facts.ownership.snapshotFile)} format ${facts.ownership.snapshotFormat}`],
      ['Intent record', code(facts.ownership.intentFile)],
      ['Intent phases', facts.ownership.intentPhases.map(code).join(', ')],
      ['Runtime evidence', code(facts.ownership.runtimeEvidence)],
    ],
  )}

The MCP ownership manifest is a separate store, not the skill backup manifest and not the fleet sidecar. Restore is entry-level only; whole-file replacement is not part of this build.

## CLI

${facts.cli.map((command) => `- ${code(command)}`).join('\n')}

Documented exit codes are 0 (success, a produced report or a real no-op), 1 (stale plan, conflict, unsupported runtime, rollback or engine/I/O error) and 2 (usage, flags, validation, unsafe target or unknown subcommand).

## Boundary

The manifest copies only managed-entry text and hashes. Unrelated servers, comments where preservation is unproven and credentials remain outside the managed write; remote entries use environment placeholders and OpenCode owns OAuth credentials.
`;
}

function executionMarkdown(facts) {
  return `Source-derived execution-run facts for the current checkout. A run, checkpoint, check or record is an observation: it is never approval, batch state, verification or a sandbox.

## Schema

All emitted records use ${code('schemaVersion')} ${facts.schemaVersion}.

${table(
    ['Record kind', 'Value'],
    [
      ['Run identity', code(facts.kinds.run)],
      ['Task brief', code(facts.kinds.brief)],
      ['Review package', code(facts.kinds.review)],
    ],
  )}

## Workspace

The default root is ${code(facts.runRoot.default)}; ${code(facts.runRoot.flag)} selects another private root. A generated ${code(facts.ignore.file)} containing ${code(facts.ignore.content)} keeps the run tree out of Git without editing the tracked root ignore file.

${table(
    ['Artifact', 'Path'],
    [
      ['Run identity', code(facts.artifacts.identity)],
      ['Checkpoint', code(facts.artifacts.checkpoint)],
      ['Writer lock', code(facts.artifacts.lock)],
      ['Task brief', code(facts.artifacts.briefs)],
      ['Check directory', code(facts.artifacts.checks)],
      ['Check metadata', facts.artifacts.checkFiles.map(code).join(', ')],
      ['Caller record', code(facts.artifacts.records)],
      ['Review package', code(facts.artifacts.review)],
      ['Review manifest', code(facts.artifacts.reviewManifest)],
      ['Review patch', code(facts.artifacts.reviewPatch)],
    ],
  )}

Run IDs match ${code(facts.runIdPattern)}. The run root and run directory use mode 0700, identity and checkpoint files use mode 0600, and those two writes are atomic.

## Observed state

${bullet([
    `Checkpoint states: ${facts.states.map(code).join(', ')}`,
    `Caller claim results: ${facts.claimResults.map(code).join(', ')}`,
    `Observed check results: ${facts.checkResults.map(code).join(', ')}`,
    `Scope kinds: ${facts.scopeKinds.map(code).join(', ')}`,
  ])}

Precedence, severity and reconciliation belong to ${code('run status')}; a reported state is an observation even when it is stale or blocked.

## Limits

${table(
    ['Limit', 'Default'],
    [
      ['tasks.md read', bytes(facts.limits.tasksBytes)],
      ['spec.md read', bytes(facts.limits.specBytes)],
      ['Caller record', bytes(facts.limits.recordBytes)],
      ['run check wall clock', duration(facts.limits.checkTimeoutSeconds)],
      ['Per-stream stdout/stderr', bytes(facts.limits.outputBytes)],
      ['Review package per file', bytes(facts.limits.reviewFileBytes)],
      ['Review package total', bytes(facts.limits.reviewTotalBytes)],
      ['Stale writer lock age', duration(facts.limits.lockStaleSeconds)],
    ],
  )}

## CLI

Common flags are ${facts.cli.commonFlags.map(code).join(', ')}; ${code('--format')} defaults to ${code(facts.cli.formatDefault)} and accepts text or json. The command families are ${code('run init|status|record|clean|check')}, ${code('task brief')} and ${code('review package')}.

Documented exit codes are ${facts.cli.exitCodes.map(code).join(', ')}. ${code('124')} is the check-timeout status, and a partial review package uses ${code('3')} while still emitting its result object.

## Boundary

The service observes local state and writes only private run artifacts. It does not approve work, call a model, or act as a sandbox; the repository, batch authorization and human review remain authoritative.
`;
}

function filesMarkdown(facts) {
  return `Source-derived destination and backup facts for the current checkout.

## Client skill destinations

${table(
    ['Client', 'Destination'],
    facts.clients.map((client) => [code(client.id), code(client.path)]),
  )}

The restore reader accepts only saved ${code(facts.managedPrefix + '*')} skill trees; other skill names are refused.

## OpenCode agent profiles

${table(
    ['Item', 'Contract'],
    [
      ['Client name', code(facts.openCodeAgents.client)],
      ['Target', code(facts.openCodeAgents.path)],
      ['Manifest entry kind', code(facts.openCodeAgents.manifestKind)],
    ],
  )}

## Backup roots

${table(
    ['Source selection', 'Backup root'],
    facts.backupRoots.map((entry) => [entry.when, code(entry.path)]),
  )}

Backup IDs use the UTC layout ${code(facts.backupIdPattern)}. Each backup directory contains ${code(facts.manifestFile)}.

## ${code('manifest.json')} formats

${table(
    ['Format', 'Fields'],
    [
      ['Format 1 (original)', facts.formats.one.map(code).join(', ')],
      ['Formats 2 and 3 (additive)', facts.formats.two.map(code).join(', ')],
      ['Format 2 baseline', 'Used when the transaction changes only skill directories'],
      ['Format 3 file changes', `${code(facts.formats.three.marker)} is recorded; OpenCode agent entries use kind ${code(facts.formats.three.entryKind)}`],
    ],
  )}

${table(
    ['Record part', 'Fields'],
    [
      ['Tool metadata', facts.toolFields.map(code).join(', ')],
      ['Per-entry', facts.entryFields.map(code).join(', ')],
      ['Entry actions', facts.actions.map(code).join(', ')],
    ],
  )}

## Digests and restore

${bullet([
    `Tree digest: ${facts.digests.tree}`,
    `File digest: ${facts.digests.file}`,
    'A restored backup must match the current home identity and its saved content is validated before use',
    `Format 1 is legacy: ${code('format < 2')} keeps the older reader path without per-entry records`,
  ])}

## Separate stores

${table(
    ['Store', 'Owner', 'Identity'],
    [
      ['Skill backup ' + code('manifest.json'), 'internal/skillsync', 'Formats 1-3'],
      ['Fleet sidecar ' + code('fleet-manifest.json'), 'internal/fleet', 'Format 4'],
      ['MCP ownership ' + code('manifest.json'), 'internal/skillsync MCP-B', 'Format 1, manifest_version 1'],
      ['Execution run scratch', 'internal/execution', 'schemaVersion 1'],
    ],
  )}

These files live in different roots and must not be read as one schema. The MCP ownership manifest records only managed entry names and installed digests; the fleet sidecar records per-file provenance; the skill backup manifest records skill destinations and saved copies.
`;
}

export function collectConfiguration({ repoRoot } = {}) {
  const root = normalizeRepoRoot(repoRoot);
  const sources = readSources(root);

  const fleet = verifyFleet(sources);
  const mcp = verifyMCP(sources);
  const execution = verifyExecution(sources);
  const filesAndBackups = verifyFilesAndBackups(sources, { fleet, mcp });
  const configuration = verifyConfiguration(sources, { fleet, mcp, execution, filesAndBackups });

  const builders = {
    'reference/configuration': configurationMarkdown,
    'reference/fleet': fleetMarkdown,
    'reference/mcp': mcpMarkdown,
    'reference/execution': executionMarkdown,
    'reference/files-and-backups': filesMarkdown,
  };
  const facts = {
    'reference/configuration': configuration,
    'reference/fleet': fleet,
    'reference/mcp': mcp,
    'reference/execution': execution,
    'reference/files-and-backups': filesAndBackups,
  };

  const sections = Object.keys(PAGE_TITLES).map((pageId) => {
    const pageSources = PAGE_SOURCES[pageId];
    for (const source of pageSources) {
      if (!sources.has(source)) {
        fail('missing_source', `page ${pageId} references an unread source: ${source}`, { path: source });
      }
    }
    const markdown = builders[pageId](facts[pageId]);
    if (!markdown.trim()) fail('invalid_inventory', `page ${pageId} produced empty facts`);
    return {
      pageId,
      title: PAGE_TITLES[pageId],
      markdown,
      sources: [...pageSources].sort(),
    };
  });

  const inventory = buildInventory(fleet, mcp, execution, filesAndBackups, configuration, sections);
  return { sections, inventory };
}
