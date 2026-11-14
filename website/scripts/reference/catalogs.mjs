// Spec 020 B1/T02 lane B1-W2-B: canonical skill, pack-profile, preset and
// adapter facts for the documentation reference.
//
// Contract (plan C2 and the B1 wave2 module contract): read only canonical
// regular repository source files, parse frontmatter with the pinned `yaml`
// package, return `{sections, inventory}` for `reference/skills` and
// `reference/profiles`, and fail closed on invalid input. It never reads a
// home directory or ambient profile store, never writes, and makes no claim
// about model/provider availability or installation state.
//
// The derived counts below are the T02 acceptance snapshot. A mismatch is a
// failure: either the product change is approved (then update the snapshot in
// a reviewed change) or a source file went missing (then restore it).

import fs from 'node:fs';
import path from 'node:path';
import YAML from 'yaml';

const SKILL_PREFIX = 'agentic-sdd-';
const ORCHESTRATE_PROFILE = 'sdd-orchestrator';
const ID_PATTERN = /^[a-z0-9][a-z0-9-]*$/;
const MODEL_PATTERN = /^[A-Za-z0-9._-]+(?:\/[A-Za-z0-9._#-]+)+$/;
const MODES = new Set(['primary', 'subagent']);
const EFFECTS = new Set(['allow', 'deny']);

const EXPECTED = Object.freeze({
  core: 18,
  orchestrate: 1,
  agents: 14,
  presets: 4,
  adapters: 2,
});

const INSTALL = Object.freeze({
  default: 'default',
  withOpenCode: 'with-opencode',
  withFreeFirst: 'with-opencode+opencode-free-first',
  fleetAdapter: 'fleet-adapter-only',
});

/** Typed, actionable collector failure. */
export class CatalogError extends Error {
  constructor(code, message, { path: at } = {}) {
    super(at === undefined ? message : `${message} (${at})`);
    this.name = 'CatalogError';
    this.code = code;
    if (at !== undefined) this.path = at;
  }
}

function segmentsOf(rel) {
  if (typeof rel !== 'string' || rel === '') {
    throw new CatalogError('unsafe-path', 'Canonical source path must be a non-empty string');
  }
  if (path.isAbsolute(rel) || rel.includes('\\')) {
    throw new CatalogError('unsafe-path', 'Canonical source path must be repository-relative', { path: rel });
  }
  const segments = rel.split('/');
  if (segments.some((segment) => segment === '' || segment === '.' || segment === '..')) {
    throw new CatalogError('unsafe-path', 'Canonical source path must stay contained', { path: rel });
  }
  return segments;
}

/** Resolve and validate the repository root; a symlinked root is refused. */
function resolveRoot(repoRoot) {
  if (typeof repoRoot !== 'string' || repoRoot.trim() === '') {
    throw new CatalogError('invalid-root', 'repoRoot must be an explicit repository path');
  }
  const absolute = path.resolve(repoRoot);
  let stat;
  try {
    stat = fs.lstatSync(absolute);
  } catch {
    throw new CatalogError('invalid-root', 'Repository root does not exist', { path: absolute });
  }
  if (stat.isSymbolicLink()) {
    throw new CatalogError('unsafe-path', 'Repository root must not be a symlink', { path: absolute });
  }
  if (!stat.isDirectory()) {
    throw new CatalogError('invalid-root', 'Repository root must be a directory', { path: absolute });
  }
  return fs.realpathSync(absolute);
}

function containedDirectory(root, rel) {
  const segments = segmentsOf(rel);
  let current = root;
  for (let index = 0; index < segments.length; index += 1) {
    current = path.join(current, segments[index]);
    const partial = segments.slice(0, index + 1).join('/');
    let stat;
    try {
      stat = fs.lstatSync(current);
    } catch {
      throw new CatalogError('missing-source', 'Missing canonical source directory', { path: partial });
    }
    if (stat.isSymbolicLink()) {
      throw new CatalogError('unsafe-path', 'Refusing symlinked canonical source directory', { path: partial });
    }
    if (!stat.isDirectory()) {
      throw new CatalogError('missing-source', 'Canonical source path is not a directory', { path: partial });
    }
  }
  return current;
}

function containedFile(root, rel) {
  const segments = segmentsOf(rel);
  let current = root;
  for (let index = 0; index < segments.length; index += 1) {
    current = path.join(current, segments[index]);
    const partial = segments.slice(0, index + 1).join('/');
    const last = index === segments.length - 1;
    let stat;
    try {
      stat = fs.lstatSync(current);
    } catch {
      throw new CatalogError('missing-source', 'Missing canonical source file', { path: partial });
    }
    if (stat.isSymbolicLink()) {
      throw new CatalogError('unsafe-path', 'Refusing symlinked canonical source path', { path: partial });
    }
    if (last ? !stat.isFile() : !stat.isDirectory()) {
      throw new CatalogError('unsafe-path', 'Canonical source path has an unexpected file type', { path: partial });
    }
  }
  return current;
}

function readContainedText(root, rel) {
  const absolute = containedFile(root, rel);
  try {
    return fs.readFileSync(absolute, 'utf8');
  } catch {
    throw new CatalogError('missing-source', 'Cannot read canonical source file', { path: rel });
  }
}

function listDirectory(root, rel) {
  const absolute = containedDirectory(root, rel);
  let entries;
  try {
    entries = fs.readdirSync(absolute, { withFileTypes: true });
  } catch {
    throw new CatalogError('missing-source', 'Cannot read canonical source directory', { path: rel });
  }
  return entries.sort((left, right) => (left.name < right.name ? -1 : left.name > right.name ? 1 : 0));
}

function firstLine(error) {
  return String(error?.message ?? error).split('\n')[0];
}

/**
 * Read the leading `---` block and parse it with the pinned YAML parser.
 * The delimiter search is line-based; YAML semantics are never reimplemented.
 */
function readFrontmatter(root, rel) {
  const text = readContainedText(root, rel).replace(/\r\n/g, '\n');
  const lines = text.split('\n');
  if (lines[0] !== '---') {
    throw new CatalogError('invalid-frontmatter', 'Frontmatter must open on the first line', { path: rel });
  }
  const end = lines.indexOf('---', 1);
  if (end === -1) {
    throw new CatalogError('invalid-frontmatter', 'Frontmatter is not closed', { path: rel });
  }
  let data;
  try {
    data = YAML.parse(lines.slice(1, end).join('\n'), { uniqueKeys: true });
  } catch (error) {
    throw new CatalogError('invalid-frontmatter', `Invalid YAML frontmatter: ${firstLine(error)}`, { path: rel });
  }
  if (data === null || data === undefined) {
    throw new CatalogError('missing-metadata', 'Frontmatter is empty', { path: rel });
  }
  if (typeof data !== 'object' || Array.isArray(data)) {
    throw new CatalogError('invalid-frontmatter', 'Frontmatter must be a YAML mapping', { path: rel });
  }
  return data;
}

function requireDescription(fields, rel) {
  if (typeof fields.description !== 'string' || fields.description.trim() === '') {
    throw new CatalogError('missing-metadata', 'Frontmatter requires a non-empty description', { path: rel });
  }
  return fields.description.trim().replace(/\s+/g, ' ');
}

function assertSafeId(id, rel) {
  if (!ID_PATTERN.test(id)) {
    throw new CatalogError('unsafe-path', `Source entry name is not a safe catalog ID: ${id}`, { path: rel });
  }
}

function assertCount(label, actual, expected, hint) {
  if (actual !== expected) {
    throw new CatalogError(
      'count-mismatch',
      `Canonical ${label} count is ${actual}, but the B1/T02 acceptance snapshot expects ${expected}. `
        + `If that product change is approved, update the snapshot in website/scripts/reference/catalogs.mjs; `
        + `otherwise restore the missing source under ${hint}.`,
    );
  }
}

function collectSkill(root, dirRel, id, install) {
  assertSafeId(id, dirRel);
  const rel = `${dirRel}/SKILL.md`;
  const fields = readFrontmatter(root, rel);
  if (typeof fields.name !== 'string' || fields.name.trim() === '') {
    throw new CatalogError('missing-metadata', 'Skill frontmatter requires a name', { path: rel });
  }
  if (!ID_PATTERN.test(fields.name)) {
    throw new CatalogError('invalid-metadata', `Skill name is not a safe catalog ID: ${fields.name}`, { path: rel });
  }
  if (fields.name !== id) {
    throw new CatalogError(
      'invalid-metadata',
      `Skill frontmatter name "${fields.name}" does not match directory "${id}"`,
      { path: rel },
    );
  }
  if (fields['disable-model-invocation'] !== undefined && typeof fields['disable-model-invocation'] !== 'boolean') {
    throw new CatalogError('invalid-metadata', 'disable-model-invocation must be a boolean', { path: rel });
  }
  const disableModelInvocation = fields['disable-model-invocation'] === true;
  return {
    id,
    name: fields.name,
    description: requireDescription(fields, rel),
    modelInvocable: !disableModelInvocation,
    disableModelInvocation,
    install,
    source: rel,
  };
}

function collectSkills(root) {
  const core = [];
  const orchestrate = [];
  const seen = new Map();

  for (const entry of listDirectory(root, 'skills-files')) {
    if (!entry.name.startsWith(SKILL_PREFIX)) continue;
    const rel = `skills-files/${entry.name}`;
    if (entry.isSymbolicLink()) {
      throw new CatalogError('unsafe-path', 'Refusing symlinked skill directory', { path: rel });
    }
    if (!entry.isDirectory()) {
      throw new CatalogError('unsafe-path', 'Skill source must be a directory', { path: rel });
    }
    const skill = collectSkill(root, rel, entry.name, INSTALL.default);
    if (seen.has(skill.id)) {
      throw new CatalogError('duplicate-id', `Duplicate skill ID: ${skill.id}`, { path: rel });
    }
    seen.set(skill.id, rel);
    core.push(skill);
  }

  for (const entry of listDirectory(root, 'opencode-pack/skills')) {
    const rel = `opencode-pack/skills/${entry.name}`;
    if (entry.isSymbolicLink()) {
      throw new CatalogError('unsafe-path', 'Refusing symlinked skill directory', { path: rel });
    }
    if (!entry.isDirectory()) {
      throw new CatalogError('unsafe-path', 'Optional skill source must be a directory', { path: rel });
    }
    const skill = collectSkill(root, rel, entry.name, INSTALL.withOpenCode);
    if (seen.has(skill.id)) {
      throw new CatalogError('duplicate-id', `Duplicate skill ID: ${skill.id}`, { path: rel });
    }
    seen.set(skill.id, rel);
    orchestrate.push(skill);
  }

  assertCount('core skill', core.length, EXPECTED.core, 'skills-files/<skill>/SKILL.md');
  assertCount('optional orchestration skill', orchestrate.length, EXPECTED.orchestrate, 'opencode-pack/skills/');

  core.sort((left, right) => (left.id < right.id ? -1 : 1));
  orchestrate.sort((left, right) => (left.id < right.id ? -1 : 1));
  return { core, orchestrate };
}

function requirePermissions(fields, rel) {
  const rules = fields.permissions;
  if (rules === undefined || rules === null) {
    throw new CatalogError('missing-metadata', 'Profile frontmatter requires permissions', { path: rel });
  }
  if (!Array.isArray(rules) || rules.length === 0) {
    throw new CatalogError('missing-metadata', 'Profile permissions must be a non-empty list', { path: rel });
  }
  return rules.map((rule, index) => {
    if (rule === null || typeof rule !== 'object' || Array.isArray(rule)) {
      throw new CatalogError('invalid-metadata', `Permission rule ${index + 1} must be a mapping`, { path: rel });
    }
    for (const key of ['action', 'resource', 'effect']) {
      if (typeof rule[key] !== 'string' || rule[key].trim() === '') {
        throw new CatalogError('invalid-metadata', `Permission rule ${index + 1} requires a ${key}`, { path: rel });
      }
    }
    if (!EFFECTS.has(rule.effect)) {
      throw new CatalogError(
        'invalid-metadata',
        `Permission rule ${index + 1} effect must be allow or deny, got "${rule.effect}"`,
        { path: rel },
      );
    }
    return { action: rule.action, resource: rule.resource, effect: rule.effect };
  });
}

/**
 * Summarize ordered permission rules for one tool. OpenCode applies the last
 * matching rule, so later rules override earlier ones. Only wildcard-resource
 * rules contribute to the summary value; a role with narrower resource-scoped
 * allow rules is reported as `scoped`.
 */
function toolSummary(rules, tool) {
  let effect = 'not-declared';
  let scoped = false;
  for (const rule of rules) {
    const prefix = rule.action.endsWith('*') && rule.action !== '*' ? rule.action.slice(0, -1) : null;
    const matches = rule.action === '*' || rule.action === tool || (prefix !== null && tool.startsWith(prefix));
    if (!matches) continue;
    if (rule.resource === '*') effect = rule.effect;
    else scoped = true;
  }
  if (effect === 'allow') return 'allow';
  if (scoped) return 'scoped';
  return effect;
}

function derivePolicy(rules) {
  const [first] = rules;
  return {
    base: first.action === '*' && first.resource === '*' && first.effect === 'deny' ? 'deny-by-default' : 'explicit-rules',
    rules: rules.length,
    writes: toolSummary(rules, 'edit'),
    shell: toolSummary(rules, 'shell'),
    delegation: toolSummary(rules, 'subagent'),
    externalSearch: toolSummary(rules, 'exa_web_search_exa'),
  };
}

function collectProfile(root, rel, id, install) {
  assertSafeId(id, rel);
  const fields = readFrontmatter(root, rel);
  const description = requireDescription(fields, rel);

  if (fields.mode === undefined || fields.mode === null || fields.mode === '') {
    throw new CatalogError('missing-metadata', 'Profile frontmatter requires a mode', { path: rel });
  }
  if (typeof fields.mode !== 'string' || !MODES.has(fields.mode)) {
    throw new CatalogError(
      'invalid-metadata',
      `Profile mode must be primary or subagent, got "${String(fields.mode)}"`,
      { path: rel },
    );
  }

  const orchestrator = id === 'sdd-orchestrator';
  const hasModel = fields.model !== undefined && fields.model !== null;
  if (hasModel && (typeof fields.model !== 'string' || !MODEL_PATTERN.test(fields.model))) {
    throw new CatalogError(
      'invalid-metadata',
      `Profile model must be a provider/model identifier, got "${String(fields.model)}"`,
      { path: rel },
    );
  }
  if (!hasModel && !orchestrator) {
    throw new CatalogError(
      'missing-metadata',
      `Profile requires a pinned model; only ${ORCHESTRATE_PROFILE} may inherit the session model`,
      { path: rel },
    );
  }
  if (orchestrator && hasModel) {
    throw new CatalogError('invalid-metadata', 'The orchestrator must not pin a model', { path: rel });
  }
  if (orchestrator && fields.mode !== 'primary') {
    throw new CatalogError('invalid-metadata', 'The orchestrator must declare mode primary', { path: rel });
  }

  let steps = null;
  if (fields.steps !== undefined && fields.steps !== null) {
    if (!Number.isInteger(fields.steps) || fields.steps <= 0) {
      throw new CatalogError('invalid-metadata', 'Profile steps must be a positive integer', { path: rel });
    }
    steps = fields.steps;
  }

  const rules = requirePermissions(fields, rel);
  return {
    id,
    mode: fields.mode,
    model: hasModel ? fields.model : null,
    modelPinned: hasModel,
    inheritSession: !hasModel,
    steps,
    description,
    install,
    policy: derivePolicy(rules),
    source: rel,
  };
}

function collectProfiles(root) {
  const groups = [
    { key: 'agents', dir: 'opencode-pack/agents', install: INSTALL.withOpenCode },
    { key: 'presets', dir: 'opencode-pack/presets/free-first', install: INSTALL.withFreeFirst },
    { key: 'adapters', dir: 'opencode-pack/adapters', install: INSTALL.fleetAdapter },
  ];
  const result = { agents: [], presets: [], adapters: [] };
  const seen = new Map();

  for (const group of groups) {
    for (const entry of listDirectory(root, group.dir)) {
      if (entry.isDirectory()) continue;
      if (!entry.name.endsWith('.md')) continue;
      const rel = `${group.dir}/${entry.name}`;
      const id = entry.name.slice(0, -'.md'.length);
      assertSafeId(id, rel);
      if (entry.isSymbolicLink()) {
        throw new CatalogError('unsafe-path', 'Refusing symlinked profile source', { path: rel });
      }
      if (seen.has(id)) {
        throw new CatalogError('duplicate-id', `Duplicate profile ID: ${id}`, { path: rel });
      }
      seen.set(id, rel);
      result[group.key].push(collectProfile(root, rel, id, group.install));
    }
  }

  assertCount('pack agent profile', result.agents.length, EXPECTED.agents, 'opencode-pack/agents/');
  assertCount('free-first preset template', result.presets.length, EXPECTED.presets, 'opencode-pack/presets/free-first/');
  assertCount('fleet adapter profile', result.adapters.length, EXPECTED.adapters, 'opencode-pack/adapters/');

  for (const key of Object.keys(result)) {
    result[key].sort((left, right) => (left.id < right.id ? -1 : 1));
  }
  return result;
}


function escapeCell(value) {
  return String(value).replace(/\s+/g, ' ').replace(/\|/g, '\\|').trim();
}

function code(value) {
  return `\`${escapeCell(value)}\``;
}

function policyCell(profile) {
  const { policy } = profile;
  return `base ${code(policy.base)}, ${policy.rules} rule${policy.rules === 1 ? '' : 's'}`;
}

function skillRows(skills) {
  const lines = ['| Skill ID | Model-invocable | Description |', '| --- | --- | --- |'];
  for (const skill of skills) {
    lines.push(`| ${code(skill.id)} | ${skill.modelInvocable ? 'yes' : 'no'} | ${escapeCell(skill.description)} |`);
  }
  return lines;
}

function profileRows(profiles) {
  const lines = [
    '| Profile | Mode | Model | Policy | Writes | Shell | Delegation | External search |',
    '| --- | --- | --- | --- | --- | --- | --- | --- |',
  ];
  for (const profile of profiles) {
    const model = profile.modelPinned ? code(profile.model) : 'session model (inherit-session)';
    lines.push(
      `| ${code(profile.id)} | ${code(profile.mode)} | ${model} | ${policyCell(profile)} | `
        + `${code(profile.policy.writes)} | ${code(profile.policy.shell)} | ${code(profile.policy.delegation)} | `
        + `${code(profile.policy.externalSearch)} |`,
    );
  }
  return lines;
}

function skillsSection(skills) {
  const disabled = skills.core.filter((skill) => !skill.modelInvocable).map((skill) => code(skill.id));
  const lines = [
    '## Core skills',
    '',
    `${skills.core.length} skills are defined in \`skills-files/\`, one directory with a \`SKILL.md\` each. `
      + 'The default `asdx-core apply` selection (also reachable through the `agentic-sdd` wrapper) selects this set '
      + 'without any opt-in pack flag.',
    '',
    ...skillRows(skills.core),
    '',
    disabled.length
      ? `${disabled.length} of ${skills.core.length} core skills declare \`disable-model-invocation: true\` and are explicitly invocable only: ${disabled.join(', ')}.`
      : 'No core skill declares `disable-model-invocation: true`.',
    '',
    '## Optional orchestration skill',
    '',
    `${skills.orchestrate.length} skill is defined in \`opencode-pack/skills/\`. It is selected only together with the `
      + 'opt-in OpenCode pack (`--with-opencode`) and is not part of the default core selection.',
    '',
    ...skillRows(skills.orchestrate),
    '',
    '## Derived counts',
    '',
    '| Class | Source root | Count |',
    '| --- | --- | --- |',
    `| Core skills | \`skills-files/\` | ${skills.core.length} |`,
    `| Optional orchestration skills | \`opencode-pack/skills/\` | ${skills.orchestrate.length} |`,
    '',
    '## Source evidence',
    '',
    `Facts above are parsed from ${skills.core.length + skills.orchestrate.length} \`SKILL.md\` files with the pinned `
      + '`yaml` parser; fingerprints are recorded in the documentation manifest. This block describes repository source, '
      + 'not the installation state of any machine.',
  ];
  return {
    pageId: 'reference/skills',
    title: 'Skills reference',
    markdown: lines.join('\n'),
    sources: [...skills.core, ...skills.orchestrate].map((skill) => skill.source).sort(),
  };
}

function profilesSection(profiles) {
  const steps = profiles.agents.filter((profile) => profile.steps !== null);
  const stepsNote = steps.length
    ? ` ${steps.map((profile) => `${code(profile.id)} declares \`steps: ${profile.steps}\``).join('; ')}.`
    : '';
  const lines = [
    '## Pack agent profiles',
    '',
    `${profiles.agents.length} agent profiles are defined in \`opencode-pack/agents/\` and selected by `
      + '`--with-opencode`. `sdd-orchestrator` is the human-facing `primary` profile and pins no model: it inherits '
      + 'the session model. A pinned model is configuration only; this page makes no claim about provider, credential, '
      + 'quota or model availability.',
    '',
    'Permission columns summarize the ordered `permissions` rules using last-match-wins selection per tool; '
      + 'resource-scoped allow rules are reported as `scoped` and not enumerated here.'
      + stepsNote,
    '',
    ...profileRows(profiles.agents),
    '',
    '## Free-first preset templates',
    '',
    `${profiles.presets.length} templates are defined in \`opencode-pack/presets/free-first/\`. They are selected only `
      + 'when both `--with-opencode` and `--opencode-free-first` are passed; `--with-opencode` alone does not install them.',
    '',
    ...profileRows(profiles.presets),
    '',
    '## Fleet adapter profiles',
    '',
    `${profiles.adapters.length} profiles are defined in \`opencode-pack/adapters/\`. They are resolver-only render `
      + 'sources for reviewed fleet model routes and are not installed by the ordinary OpenCode pack: `--with-opencode` '
      + 'reads `opencode-pack/agents/` only.',
    '',
    ...profileRows(profiles.adapters),
    '',
    '## Derived counts',
    '',
    '| Class | Selected by | Count |',
    '| --- | --- | --- |',
    `| Pack agent profiles | \`--with-opencode\` | ${profiles.agents.length} |`,
    `| Free-first preset templates | \`--with-opencode --opencode-free-first\` | ${profiles.presets.length} |`,
    `| Fleet adapter profiles | not installed by the pack | ${profiles.adapters.length} |`,
    '',
    '## Source evidence',
    '',
    `Facts above are parsed from `
      + `${profiles.agents.length + profiles.presets.length + profiles.adapters.length} profile files with the pinned `
      + '`yaml` parser; fingerprints are recorded in the documentation manifest. This block describes repository '
      + 'source, not runtime permission enforcement, model routing success or the installation state of any machine.',
  ];
  return {
    pageId: 'reference/profiles',
    title: 'Profiles and presets reference',
    markdown: lines.join('\n'),
    sources: [...profiles.agents, ...profiles.presets, ...profiles.adapters].map((profile) => profile.source).sort(),
  };
}

/**
 * Collect canonical skill and profile facts from one repository checkout.
 *
 * Synchronous and read-only. Throws `CatalogError` with a stable `code` and,
 * for source-specific failures, a repository-relative `path`.
 */
export function collectCatalogs({ repoRoot } = {}) {
  const root = resolveRoot(repoRoot);
  containedDirectory(root, 'skills-files');
  containedDirectory(root, 'opencode-pack');
  containedDirectory(root, 'opencode-pack/agents');

  const skills = collectSkills(root);
  const profiles = collectProfiles(root);

  const inventory = {
    skills: {
      core: skills.core,
      orchestrate: skills.orchestrate,
      counts: {
        core: skills.core.length,
        orchestrate: skills.orchestrate.length,
        total: skills.core.length + skills.orchestrate.length,
      },
    },
    profiles: {
      agents: profiles.agents,
      presets: profiles.presets,
      adapters: profiles.adapters,
      counts: {
        agents: profiles.agents.length,
        presets: profiles.presets.length,
        adapters: profiles.adapters.length,
        total: profiles.agents.length + profiles.presets.length + profiles.adapters.length,
      },
    },
  };

  return { sections: [skillsSection(skills), profilesSection(profiles)], inventory };
}
