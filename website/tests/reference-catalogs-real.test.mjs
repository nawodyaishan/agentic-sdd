// Spec 020 B1/T02 lane B1-W2-B: catalog collector facts against the real
// canonical source tree. Red evidence is a missing/throwing collector; this
// file never skips a failing fact.

import assert from 'node:assert/strict';
import { existsSync, readFileSync, statSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import YAML from 'yaml';

import { catalogsModulePath, repoRoot } from './reference-catalogs-support.mjs';

const { collectCatalogs, CatalogError } = await import(catalogsModulePath);

const EXPECTED_CORE_SKILLS = 18;
const EXPECTED_PACK_AGENTS = 14;
const EXPECTED_PRESETS = 4;
const EXPECTED_ADAPTERS = 2;

const HOST_PATH_MARKERS = [repoRoot, process.cwd(), '/home/', '/Users/', 'C:\\'];

function collect() {
  return collectCatalogs({ repoRoot });
}

function sourceText(rel) {
  return readFileSync(path.join(repoRoot, rel), 'utf8');
}

function frontmatterOf(rel) {
  const text = sourceText(rel).replace(/\r\n/g, '\n');
  const lines = text.split('\n');
  assert.equal(lines[0], '---', `${rel} must start with a frontmatter delimiter`);
  const end = lines.indexOf('---', 1);
  assert.notEqual(end, -1, `${rel} must close its frontmatter`);
  return YAML.parse(lines.slice(1, end).join('\n'));
}

test('collector is an exported sync function returning the two frozen sections', () => {
  assert.equal(typeof collectCatalogs, 'function');
  assert.equal(typeof CatalogError, 'function');
  const result = collect();
  assert.equal(typeof result, 'object');
  assert.deepEqual(Object.keys(result).sort(), ['inventory', 'sections']);
  assert.ok(Array.isArray(result.sections));
  assert.deepEqual(result.sections.map((section) => section.pageId).sort(), [
    'reference/profiles',
    'reference/skills',
  ]);
  for (const section of result.sections) {
    assert.equal(typeof section.title, 'string');
    assert.ok(section.title.trim().length > 0);
    assert.equal(typeof section.markdown, 'string');
    assert.ok(section.markdown.trim().length > 0);
    assert.ok(Array.isArray(section.sources));
  }
});

test('inventory reproduces the T02 acceptance counts from source', () => {
  const { inventory } = collect();
  assert.equal(inventory.skills.core.length, EXPECTED_CORE_SKILLS);
  assert.equal(inventory.skills.orchestrate.length, 1);
  assert.equal(inventory.profiles.agents.length, EXPECTED_PACK_AGENTS);
  assert.equal(inventory.profiles.presets.length, EXPECTED_PRESETS);
  assert.equal(inventory.profiles.adapters.length, EXPECTED_ADAPTERS);
  assert.deepEqual(inventory.skills.counts, {
    core: EXPECTED_CORE_SKILLS,
    orchestrate: 1,
    total: EXPECTED_CORE_SKILLS + 1,
  });
  assert.deepEqual(inventory.profiles.counts, {
    agents: EXPECTED_PACK_AGENTS,
    presets: EXPECTED_PRESETS,
    adapters: EXPECTED_ADAPTERS,
    total: EXPECTED_PACK_AGENTS + EXPECTED_PRESETS + EXPECTED_ADAPTERS,
  });
});

test('skill IDs are unique, sorted and match their directory and frontmatter name', () => {
  const { inventory } = collect();
  const ids = inventory.skills.core.map((skill) => skill.id);
  assert.deepEqual(ids, [...new Set(ids)].sort());
  assert.equal(new Set(ids).size, ids.length);
  for (const skill of inventory.skills.core) {
    const rel = `skills-files/${skill.id}/SKILL.md`;
    assert.equal(skill.source, rel);
    const fields = frontmatterOf(rel);
    assert.equal(fields.name, skill.id);
    assert.equal(skill.name, skill.id);
    assert.equal(skill.description, fields.description);
    assert.equal(skill.install, 'default');
  }
  const orchestrate = inventory.skills.orchestrate[0];
  assert.equal(orchestrate.id, 'agentic-sdd-orchestrate');
  assert.equal(orchestrate.source, 'opencode-pack/skills/agentic-sdd-orchestrate/SKILL.md');
  assert.equal(orchestrate.install, 'with-opencode');
  assert.equal(frontmatterOf(orchestrate.source).name, 'agentic-sdd-orchestrate');
});

test('model-invocation flags come from real frontmatter, not guesses', () => {
  const { inventory } = collect();
  const disabled = inventory.skills.core
    .filter((skill) => skill.modelInvocable === false)
    .map((skill) => skill.id);
  const declared = Object.fromEntries(
    inventory.skills.core.map((skill) => [skill.id, frontmatterOf(skill.source)['disable-model-invocation'] === true]),
  );
  for (const skill of inventory.skills.core) {
    assert.equal(skill.modelInvocable, !declared[skill.id], `${skill.id} modelInvocable`);
  }
  assert.deepEqual(disabled, disabled.slice().sort());
  assert.equal(disabled.length, 4);
  const { sections } = collect();
  const skills = sections.find((section) => section.pageId === 'reference/skills');
  for (const skill of inventory.skills.core) {
    assert.match(skills.markdown, new RegExp(`\`${skill.id}\``));
  }
  for (const id of disabled) {
    assert.match(skills.markdown, new RegExp(`\`${id}\``));
  }
});

test('pack profiles expose mode, pinned model and install classification', () => {
  const { inventory } = collect();
  const agentIds = inventory.profiles.agents.map((profile) => profile.id);
  assert.deepEqual(agentIds, [...new Set(agentIds)].sort());
  assert.equal(new Set(agentIds).size, agentIds.length);
  for (const profile of inventory.profiles.agents) {
    assert.equal(profile.install, 'with-opencode');
    assert.equal(profile.source, `opencode-pack/agents/${profile.id}.md`);
    const fields = frontmatterOf(profile.source);
    assert.equal(profile.mode, fields.mode);
    assert.equal(profile.model, fields.model ?? null);
    assert.equal(profile.modelPinned, fields.model !== undefined);
    assert.ok(['primary', 'subagent'].includes(profile.mode));
  }
  for (const profile of inventory.profiles.presets) {
    assert.equal(profile.install, 'with-opencode+opencode-free-first');
    assert.equal(profile.source, `opencode-pack/presets/free-first/${profile.id}.md`);
  }
  for (const profile of inventory.profiles.adapters) {
    assert.equal(profile.install, 'fleet-adapter-only');
    assert.equal(profile.source, `opencode-pack/adapters/${profile.id}.md`);
  }
  const presetIds = inventory.profiles.presets.map((profile) => profile.id);
  assert.deepEqual(presetIds, ['sdd-implement-1', 'sdd-implement-2', 'sdd-test-implement-1', 'sdd-test-implement-2']);
  const adapterIds = inventory.profiles.adapters.map((profile) => profile.id);
  assert.deepEqual(adapterIds, ['sdd-plan-9r', 'sdd-spec-9r']);
  const installed = new Set(agentIds);
  for (const id of adapterIds) {
    assert.equal(installed.has(id), false, `${id} must not also be an installed agent profile`);
  }
});

test('the unpinned orchestrator means inherit-session, not an error', () => {
  const { inventory } = collect();
  const unpinned = inventory.profiles.agents.filter((profile) => !profile.modelPinned);
  assert.deepEqual(unpinned.map((profile) => profile.id), ['sdd-orchestrator']);
  const orchestrator = unpinned[0];
  assert.equal(orchestrator.mode, 'primary');
  assert.equal(orchestrator.model, null);
  assert.equal(orchestrator.inheritSession, true);
  for (const profile of inventory.profiles.agents) {
    if (profile.id === 'sdd-orchestrator') continue;
    assert.equal(profile.inheritSession, false);
    assert.ok(typeof profile.model === 'string' && profile.model.length > 0, `${profile.id} needs a pinned model`);
    assert.match(profile.model, /^[A-Za-z0-9._-]+(?:\/[A-Za-z0-9._#-]+)+$/);
  }
  const { sections } = collect();
  const profiles = sections.find((section) => section.pageId === 'reference/profiles');
  assert.match(profiles.markdown, /sdd-orchestrator/);
  assert.match(profiles.markdown, /session model/i);
  assert.doesNotMatch(profiles.markdown, /sdd-orchestrator[^\n]*error/i);
});

test('tool policy is derived from ordered permission rules', () => {
  const { inventory } = collect();
  const byId = Object.fromEntries(inventory.profiles.agents.map((profile) => [profile.id, profile]));
  assert.equal(byId['sdd-orchestrator'].policy.base, 'explicit-rules');
  assert.equal(byId['sdd-orchestrator'].policy.writes, 'not-declared');
  assert.equal(byId['sdd-orchestrator'].policy.delegation, 'allow');
  assert.equal(byId['sdd-debug-diagnose'].policy.base, 'deny-by-default');
  assert.equal(byId['sdd-debug-diagnose'].policy.writes, 'deny');
  assert.equal(byId['sdd-debug-diagnose'].policy.shell, 'deny');
  assert.equal(byId['sdd-debug-diagnose'].policy.delegation, 'deny');
  assert.equal(byId['sdd-code-review-9r'].policy.shell, 'scoped');
  assert.equal(byId['sdd-code-review-9r'].policy.externalSearch, 'allow');
  assert.equal(byId['sdd-lightweight'].policy.externalSearch, 'deny');
  assert.equal(byId['sdd-lightweight'].steps, 10);
  for (const profile of [...inventory.profiles.agents, ...inventory.profiles.presets, ...inventory.profiles.adapters]) {
    assert.ok(profile.policy.rules > 0, `${profile.id} rule count`);
    for (const value of Object.values(profile.policy)) {
      assert.doesNotMatch(String(value), /\n/);
    }
  }
});

test('sections declare unique contained regular-file evidence and no host paths', () => {
  const { sections } = collect();
  const registered = new Set();
  for (const section of sections) {
    assert.deepEqual(section.sources, [...new Set(section.sources)], `${section.pageId} duplicate sources`);
    for (const source of section.sources) {
      assert.equal(path.isAbsolute(source), false, `${source} must be repository-relative`);
      assert.equal(source.includes('..'), false, `${source} must stay contained`);
      assert.equal(source.includes('\\'), false, `${source} must use forward slashes`);
      const absolute = path.join(repoRoot, source);
      assert.ok(statSync(absolute).isFile(), `${source} must be a regular file`);
      assert.equal(registered.has(source), false, `${source} registered twice`);
      registered.add(source);
    }
    for (const marker of HOST_PATH_MARKERS) {
      assert.equal(section.markdown.includes(marker), false, `${section.pageId} leaks a host path: ${marker}`);
    }
    assert.doesNotMatch(section.markdown, /\u001B\[/);
    assert.doesNotMatch(section.markdown, /\b\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/);
  }
  const skills = sections.find((section) => section.pageId === 'reference/skills');
  assert.equal(skills.sources.length, EXPECTED_CORE_SKILLS + 1);
  const profiles = sections.find((section) => section.pageId === 'reference/profiles');
  assert.equal(profiles.sources.length, EXPECTED_PACK_AGENTS + EXPECTED_PRESETS + EXPECTED_ADAPTERS);
});

test('generation is deterministic and makes no availability claim', () => {
  const first = collect();
  const second = collect();
  assert.equal(JSON.stringify(first), JSON.stringify(second));
  const profiles = first.sections.find((section) => section.pageId === 'reference/profiles');
  assert.match(profiles.markdown, /no claim|not a claim|does not claim|configuration only/i);
  assert.doesNotMatch(profiles.markdown, /\b(?:is|are) (?:currently )?available\b/i);
  assert.doesNotMatch(profiles.markdown, /\bquota (?:is|was) (?:available|confirmed)\b/i);
  assert.doesNotMatch(profiles.markdown, /https?:\/\//);
  const skills = first.sections.find((section) => section.pageId === 'reference/skills');
  assert.doesNotMatch(skills.markdown, /installed on this machine|verified installed/i);
});

test('collection is read-only and independent of ambient machine profiles', () => {
  const docsDir = path.join(repoRoot, 'website/src/content/docs/reference');
  const before = existsSync(docsDir);
  const ambient = collect();
  assert.equal(existsSync(docsDir), before, 'collector must not create reference pages');

  const previousHome = process.env.HOME;
  const previousUserProfile = process.env.USERPROFILE;
  try {
    process.env.HOME = path.join(repoRoot, 'website', 'tests', 'does-not-exist-home');
    process.env.USERPROFILE = path.join(repoRoot, 'website', 'tests', 'does-not-exist-home');
    assert.equal(JSON.stringify(collect()), JSON.stringify(ambient));
  } finally {
    if (previousHome === undefined) delete process.env.HOME;
    else process.env.HOME = previousHome;
    if (previousUserProfile === undefined) delete process.env.USERPROFILE;
    else process.env.USERPROFILE = previousUserProfile;
  }
});

test('invalid roots fail closed with a typed error', () => {
  assert.throws(() => collectCatalogs({}), (error) => error instanceof CatalogError && error.code === 'invalid-root');
  assert.throws(
    () => collectCatalogs({ repoRoot: path.join(repoRoot, 'website', 'package.json') }),
    (error) => error instanceof CatalogError && error.code === 'invalid-root',
  );
  assert.throws(
    () => collectCatalogs({ repoRoot: path.join(repoRoot, 'website', 'tests', 'no-such-tree') }),
    (error) => error instanceof CatalogError && error.code === 'invalid-root',
  );
});
