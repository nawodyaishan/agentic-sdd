// SPEC020 B4-W1-F — one low-noise documentation impact report.
//
// Contract (frozen for batch B4; plan §6 "Docs impact" and T09):
//
//   impactReport({manifest, changedPaths}) -> frozen report object
//   formatReport(report) -> deterministic text
//   collectGitChanges({repoRoot, base}) -> repository-relative paths
//   parseChangedPaths(text) -> repository-relative paths
//   normalizeRepoPath(value) -> normalized POSIX path
//
// Guarantees and limits:
// - Read-only and offline. The manifest is validated but never written; no
//   AI/model call, no network, no page-content parsing and no file hashing.
// - One observation per changed path. A path maps only through the manifest's
//   `impactPaths[]` globs or `sources[].path`; shared evidence, page-to-page
//   links and generated sections do not expand the set (no cascade).
// - Nothing is dropped. A path the manifest does not map is either covered by
//   an explicit no-impact rule (with its reason) or reported as a tracked gap
//   with a surface and owner. Unknown is never silently ignored.
// - Deterministic. Observations sort by repository path and page IDs by id;
//   no timestamps, no absolute paths and no environment-dependent output.
// - `impactPaths[]` entries must be plain strings. A trailing "/" is a prefix
//   and the only glob operators are "*" (one segment), "**" (any depth) and
//   "?" (one character). Brace, bracket, parenthesis or NUL syntax fails
//   closed instead of being interpreted as a literal.
// - Exit status is 0 for every completed report, including tracked gaps. Only
//   a usage, manifest or I/O error exits non-zero.
//
// The check names below are the planned T09 interfaces from plan §6; this
// module does not run them and does not decide whether a PR may merge.

import { spawnSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { loadManifest, validateManifest } from './lib/manifest.mjs';

export class ImpactError extends Error {
  constructor(code, errorPath, message) {
    super(message);
    this.name = 'ImpactError';
    this.code = code;
    this.path = errorPath ?? '';
  }
}

const REGISTRY_PATH = 'website/docs-manifest.json';
const SOURCE_CHECK = 'npm --prefix website run docs:check';
const CATEGORIES = ['mapped', 'no-impact', 'tracked-gap'];
const TRACKED_GAP_ACTION = 'map this surface with impactPaths, or record a no-public-impact reason with the owner';
const REPORT_SCHEMA_VERSION = 1;

// Explicit no-impact rules, in order. Each one states why the tree cannot
// require a published page edit; an explicit manifest mapping always wins.
const NO_IMPACT_RULES = [
  {
    match: (relative) => relative.startsWith('specs/'),
    reason: 'engineering specification record; not a published page input',
  },
  {
    match: (relative) => relative.startsWith('docs/handovers/'),
    reason: 'private session handover; not a published page input',
  },
  {
    match: (relative) => relative.startsWith('audits/'),
    reason: 'historical audit report; not a published page input',
  },
  {
    match: (relative) => relative.startsWith('backups/') || relative.startsWith('.agentic-sdd/'),
    reason: 'private authored or execution scratch tree',
  },
  {
    match: (relative) => /(^|\/)(?:node_modules|dist|\.astro|coverage)\//.test(relative),
    reason: 'generated or installed dependency tree',
  },
  {
    match: (relative) => relative.startsWith('.github/'),
    reason: 'CI and automation configuration; validated by CI, not by page prose',
  },
  {
    match: (relative) =>
      relative === 'website/package.json' ||
      relative === 'website/package-lock.json' ||
      relative === 'website/.node-version' ||
      relative === 'website/astro.config.mjs' ||
      relative === 'website/src/content.config.ts' ||
      relative === 'website/tsconfig.json',
    reason: 'site build and tooling configuration; validated by the site build, not by page content',
  },
  {
    match: (relative) => relative.startsWith('website/tests/'),
    reason: 'documentation-tooling test; it validates, it does not publish',
  },
  {
    match: (relative) => relative === '.gitignore' || relative === '.gitattributes',
    reason: 'repository metadata',
  },
];

// Gap surfaces used only when nothing maps a path. Owners are governance
// labels, not people; the parent feature record owns their meaning.
const SURFACE_RULES = [
  { match: (relative) => relative.startsWith('cmd/'), surface: 'CLI', owner: 'product-maintainers' },
  { match: (relative) => relative.startsWith('internal/'), surface: 'core service', owner: 'product-maintainers' },
  { match: (relative) => relative.startsWith('experiments/'), surface: 'TUI', owner: 'product-maintainers' },
  { match: (relative) => relative.startsWith('skills-files/'), surface: 'skill assets', owner: 'docs-maintainers' },
  { match: (relative) => relative.startsWith('opencode-pack/'), surface: 'profiles and routing', owner: 'docs-maintainers' },
  { match: (relative) => relative.startsWith('docs/testing/'), surface: 'testing documentation', owner: 'testing-initiative' },
  {
    match: (relative) => relative.startsWith('tests/') || relative.startsWith('harness/'),
    surface: 'testing contract',
    owner: 'testing-initiative',
  },
  { match: (relative) => relative.startsWith('docs/'), surface: 'engineering documentation', owner: 'docs-maintainers' },
  { match: (relative) => relative.startsWith('website/'), surface: 'site tooling', owner: 'docs-maintainers' },
  { match: (relative) => relative.startsWith('scripts/'), surface: 'automation', owner: 'docs-maintainers' },
  { match: (relative) => relative.startsWith('Taskfile'), surface: 'developer commands', owner: 'docs-maintainers' },
  {
    match: (relative) => relative === 'README.md' || relative === 'CONTRIBUTING.md',
    surface: 'public entry point',
    owner: 'docs-maintainers',
  },
];
const DEFAULT_SURFACE = { surface: 'repository surface', owner: 'docs-maintainers' };

export function normalizeRepoPath(value, errorPath = 'path') {
  if (typeof value !== 'string') {
    throw new ImpactError('invalid-path', errorPath, 'Expected a repository-relative path string');
  }
  let normalized = value.trim();
  while (normalized.startsWith('./')) normalized = normalized.slice(2);
  if (normalized.length === 0) {
    throw new ImpactError('invalid-path', errorPath, 'Expected a non-empty repository-relative path');
  }
  if (normalized.includes('\0')) {
    throw new ImpactError('invalid-path', errorPath, 'Path contains a NUL byte');
  }
  if (normalized.includes('\\')) {
    throw new ImpactError('invalid-path', errorPath, 'Use POSIX "/" separators; backslashes are not contained paths');
  }
  if (normalized.startsWith('/') || /^[A-Za-z]:/.test(normalized)) {
    throw new ImpactError('invalid-path', errorPath, `"${normalized}" is absolute; expected a repository-relative path`);
  }
  const segments = normalized.split('/');
  if (segments.some((segment) => segment.length === 0 || segment === '.' || segment === '..')) {
    throw new ImpactError('invalid-path', errorPath, `"${normalized}" traverses or repeats a path segment`);
  }
  return segments.join('/');
}

// Compiles one explicit `impactPaths[]` entry. Unsupported syntax is refused
// rather than treated as a literal, so a typo becomes an actionable error
// instead of a silently unmapped surface.
function compileImpactPattern(pattern, errorPath) {
  if (typeof pattern !== 'string') {
    throw new ImpactError('invalid-impact-path', errorPath, 'impactPaths entries must be plain strings');
  }
  if (pattern.length === 0 || pattern.trim() !== pattern) {
    throw new ImpactError('invalid-impact-path', errorPath, 'impactPaths entries must be non-empty trimmed strings');
  }
  if (pattern.includes('\0') || pattern.includes('\\')) {
    throw new ImpactError('invalid-impact-path', errorPath, 'impactPaths entries use POSIX "/" separators and no NUL bytes');
  }
  if (pattern.startsWith('/') || /^[A-Za-z]:/.test(pattern)) {
    throw new ImpactError('invalid-impact-path', errorPath, `"${pattern}" is absolute; expected a repository-relative pattern`);
  }
  if (/[{}[\]()]/.test(pattern)) {
    throw new ImpactError(
      'unsupported-impact-pattern',
      errorPath,
      `"${pattern}" uses unsupported glob syntax; use "*", "**", "?" or a trailing "/" prefix`,
    );
  }
  const isPrefix = pattern.endsWith('/');
  const body = isPrefix ? pattern.slice(0, -1) : pattern;
  const segments = body.split('/');
  if (segments.some((segment) => segment.length === 0 || segment === '.' || segment === '..')) {
    throw new ImpactError('invalid-impact-path', errorPath, `"${pattern}" traverses or repeats a pattern segment`);
  }

  let source = '^';
  for (let index = 0; index < body.length; index += 1) {
    const character = body[index];
    if (character === '*') {
      if (body[index + 1] === '*') {
        index += 1;
        if (body[index + 1] === '/') {
          index += 1;
          source += '(?:[^/]+/)*';
        } else {
          source += '.*';
        }
      } else {
        source += '[^/]*';
      }
      continue;
    }
    if (character === '?') {
      source += '[^/]';
      continue;
    }
    source += /[.+^$|\\]/.test(character) ? `\\${character}` : character;
  }
  source += isPrefix ? '(?:/.*)?$' : '$';
  return new RegExp(source);
}

function indexManifest(manifest) {
  return manifest.pages.map((page, pageIndex) => ({
    id: page.id,
    sources: new Set(page.sources.map((source) => source.path)),
    patterns: (page.impactPaths ?? []).map((pattern, patternIndex) =>
      compileImpactPattern(pattern, `pages[${pageIndex}].impactPaths[${patternIndex}]`),
    ),
  }));
}

function mappedPages(pageIndex, relative) {
  const ids = pageIndex
    .filter((entry) => entry.sources.has(relative) || entry.patterns.some((pattern) => pattern.test(relative)))
    .map((entry) => entry.id);
  return [...new Set(ids)].sort();
}

function noImpactReason(relative) {
  const rule = NO_IMPACT_RULES.find((candidate) => candidate.match(relative));
  return rule ? rule.reason : undefined;
}

function surfaceFor(relative) {
  return SURFACE_RULES.find((candidate) => candidate.match(relative)) ?? DEFAULT_SURFACE;
}

function classify(relative, pageIndex, allPageIds) {
  // A registry edit changes every page contract at once: a direct mapping of
  // the changed surface, not a transitive expansion.
  if (relative === REGISTRY_PATH) {
    return { path: relative, category: 'mapped', pages: allPageIds, check: SOURCE_CHECK };
  }
  const pages = mappedPages(pageIndex, relative);
  if (pages.length > 0) {
    return { path: relative, category: 'mapped', pages, check: SOURCE_CHECK };
  }
  const reason = noImpactReason(relative);
  if (reason !== undefined) {
    return { path: relative, category: 'no-impact', reason, check: 'none' };
  }
  const { surface, owner } = surfaceFor(relative);
  return {
    path: relative,
    category: 'tracked-gap',
    surface,
    owner,
    nextAction: TRACKED_GAP_ACTION,
    check: 'none',
  };
}

export function impactReport({ manifest, changedPaths } = {}) {
  const registry = validateManifest(manifest);
  if (!Array.isArray(changedPaths)) {
    throw new ImpactError('invalid-changed-paths', 'changedPaths', 'Expected an array of repository-relative changed paths');
  }
  const unique = [...new Set(changedPaths.map((changedPath, index) => normalizeRepoPath(changedPath, `changedPaths[${index}]`)))];
  unique.sort();
  const pageIndex = indexManifest(registry);
  const allPageIds = registry.pages.map((page) => page.id).sort();
  const observations = unique.map((relative) => Object.freeze(classify(relative, pageIndex, allPageIds)));
  const summary = { mapped: 0, 'no-impact': 0, 'tracked-gap': 0 };
  for (const observation of observations) summary[observation.category] += 1;
  return Object.freeze({
    schemaVersion: REPORT_SCHEMA_VERSION,
    changed: observations.length,
    summary: Object.freeze(summary),
    observations: Object.freeze(observations),
  });
}

export function formatReport(report) {
  const summary = report.summary;
  const lines = [
    `docs impact: ${report.changed} changed path(s) | mapped ${summary.mapped} | no-impact ${summary['no-impact']} | tracked-gap ${summary['tracked-gap']}`,
  ];
  if (report.changed === 0) {
    lines.push('', 'nothing to report');
    return lines.join('\n');
  }
  for (const category of CATEGORIES) {
    const items = report.observations.filter((observation) => observation.category === category);
    if (items.length === 0) continue;
    lines.push('', category);
    for (const item of items) {
      lines.push(`  ${item.path}`);
      if (category === 'mapped') {
        lines.push(`    pages: ${item.pages.join(', ')}`);
        lines.push(`    check: ${item.check}`);
      } else if (category === 'no-impact') {
        lines.push(`    reason: ${item.reason}`);
        lines.push('    check: none');
      } else {
        lines.push(`    surface: ${item.surface} | owner: ${item.owner}`);
        lines.push('    check: none');
        lines.push(`    next: ${item.nextAction}`);
      }
    }
  }
  return lines.join('\n');
}

export function parseChangedPaths(text) {
  if (typeof text !== 'string') {
    throw new ImpactError('invalid-changed-paths', 'paths', 'Expected changed-path input text');
  }
  return text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line.length > 0 && !line.startsWith('#'));
}

export function collectGitChanges({ repoRoot, base } = {}) {
  if (typeof repoRoot !== 'string' || repoRoot.length === 0) {
    throw new ImpactError('invalid-repo-root', 'repoRoot', 'Expected a non-empty repository root path');
  }
  if (typeof base !== 'string' || base.length === 0 || base.startsWith('-')) {
    throw new ImpactError('invalid-base', 'base', 'Expected a non-empty git revision or range that does not start with "-"');
  }
  const result = spawnSync(
    'git',
    ['-C', repoRoot, 'diff', '--name-only', '-z', '--no-renames', '--diff-filter=ACDMR', base, '--'],
    { encoding: 'utf8', maxBuffer: 16 * 1024 * 1024 },
  );
  if (result.error) {
    throw new ImpactError('git-unavailable', base, `Cannot run git: ${result.error.message}`);
  }
  if (result.status !== 0) {
    const detail = (result.stderr ?? '').trim() || `exit ${result.status}`;
    throw new ImpactError('git-diff-failed', base, `git diff failed for "${base}": ${detail}`);
  }
  return (result.stdout ?? '').split('\0').filter((entry) => entry.length > 0);
}

function usage() {
  return [
    'Usage: node website/scripts/impact.mjs (--base <git-rev-or-range> | --paths <file|->) [options]',
    '',
    'Reports the registered documentation impact of changed repository paths.',
    '',
    '  --base      compare a git revision or range with the working tree',
    '  --paths     read one repository-relative changed path per line ("-" reads stdin)',
    '  --repo      repository root (default: the checkout that contains this script)',
    '  --manifest  manifest path (default: <repo>/website/docs-manifest.json)',
    '  --json      print the machine-readable report',
    '  --help      print this usage',
    '',
    'Exit status is 0 for any completed report; non-zero only on usage, manifest or I/O errors.',
  ].join('\n');
}

function readChangedPathFile(source) {
  if (source === '-') {
    try {
      return readFileSync(0, 'utf8');
    } catch (error) {
      throw new ImpactError('unreadable-paths', 'stdin', `Cannot read changed paths from stdin: ${error.code ?? error.message}`);
    }
  }
  try {
    return readFileSync(path.resolve(source), 'utf8');
  } catch (error) {
    throw new ImpactError('unreadable-paths', source, `Cannot read changed paths from "${source}": ${error.code ?? error.message}`);
  }
}

export async function main(argv) {
  const defaultRepoRoot = path.resolve(fileURLToPath(new URL('../..', import.meta.url)));
  let repoRoot = defaultRepoRoot;
  let manifestPath;
  let base;
  let pathsSource;
  let json = false;
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === '--help' || argument === '-h') {
      console.log(usage());
      return 0;
    }
    if (argument === '--json') {
      json = true;
      continue;
    }
    if (['--base', '--paths', '--repo', '--manifest'].includes(argument)) {
      const value = argv[index + 1];
      if (value === undefined || value.length === 0 || value.startsWith('--')) {
        throw new ImpactError('missing-value', argument, `Missing value for ${argument}`);
      }
      index += 1;
      if (argument === '--base') base = value;
      else if (argument === '--paths') pathsSource = value;
      else if (argument === '--repo') repoRoot = path.resolve(value);
      else manifestPath = path.resolve(value);
      continue;
    }
    throw new ImpactError('unknown-argument', argument, `Unknown impact option "${argument}"`);
  }
  if (base !== undefined && pathsSource !== undefined) {
    throw new ImpactError('conflicting-input', 'input', 'Choose --base or --paths, not both');
  }
  if (base === undefined && pathsSource === undefined) {
    throw new ImpactError('missing-input', 'input', 'A change input is required: --base <git-rev> or --paths <file>');
  }

  const manifest = loadManifest(manifestPath ?? path.join(repoRoot, 'website/docs-manifest.json'));
  const changedPaths = base === undefined ? parseChangedPaths(readChangedPathFile(pathsSource)) : collectGitChanges({ repoRoot, base });
  const report = impactReport({ manifest, changedPaths });
  console.log(json ? JSON.stringify(report, null, 2) : formatReport(report));
  return 0;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main(process.argv.slice(2)).catch((error) => {
    console.error(`impact: ${error.message}`);
    process.exitCode = 1;
  });
}
