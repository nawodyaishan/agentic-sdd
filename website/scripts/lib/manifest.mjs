// SPEC020 B1-W1-C — C1 manifest validation, evidence fingerprints and
// baseline structure inventory.
//
// Contract (frozen for batch B1, see specs/020-documentation-ecosystem/plan.md
// §C1 and /tmp/opencode/spec020-b1/contract.md):
//
//   validateManifest(manifest, {repoRoot, verifySources=false}) -> frozen object
//   loadManifest(manifestPath, options) -> frozen object
//   computeReviewDigest(sources) -> hex SHA-256
//   hashSources(repoRoot, sources) -> [{path, sha256}]
//   collectBaseline(repoRoot, paths) -> {files:[{path, headings, links}]}
//
// Guarantees and limits:
// - Synchronous, Node built-ins only, read-only: no network, no filesystem
//   writes, and no human-review stamping. A changed fingerprint is a
//   `review-digest-mismatch`; only a human may refresh it.
// - File access happens only for `verifySources: true`, `hashSources` and
//   `collectBaseline`. Schema-only validation never touches declared files.
// - Contained paths are POSIX, repository-relative, must avoid `.`/`..`,
//   backslashes and empty segments, and may not resolve through a symlink
//   component or a non-regular file. This is best-effort validation, not a
//   sandbox and not a TOCTOU guarantee.
// - Private scratch, backups, credentials and dependency trees are refused.
//   Public specs/testing reports may be hashed as engineering evidence, but
//   cannot become canonical public pages or old-public-guide baseline entries.
// - `collectBaseline` intentionally implements a small, deterministic subset
//   of CommonMark: ATX and single-line setext headings plus inline links,
//   reference definitions; fenced code, inline code, frontmatter, HTML comments
//   and images are excluded. Reference-style usages are represented by their
//   definitions, and indented code blocks are not distinguished from list
//   continuation content. It never scans recursively and never parses HTML.
//   Explicitly supplied public guide paths anywhere in the repository
//   (README.md, CONTRIBUTING.md, docs/*.md, site content) are in scope and keep
//   their input order; public export scope is a separate decision.

import { createHash } from 'node:crypto';
import { lstatSync, readFileSync, realpathSync, statSync } from 'node:fs';
import path from 'node:path';

export class ManifestError extends Error {
  constructor(code, errorPath, message) {
    super(message);
    this.name = 'ManifestError';
    this.code = code;
    this.path = errorPath ?? '';
  }
}

const PUBLIC_DOCS_ROOT = 'website/src/content/docs/';
const SCHEMA_VERSION = 1;
const DEV_VERSION = 'dev';
const UNRELEASED = 'unreleased';
const TAG_PATTERN = /^v(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/;
const SHA256_PATTERN = /^[0-9a-f]{64}$/;
const COMMIT_PATTERN = /^[0-9a-f]{7,40}$/;
const FULL_COMMIT_PATTERN = /^[0-9a-f]{40}$/;
const DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/;
const ROUTE_PATTERN = /^(?:index|[a-z0-9]+(?:-[a-z0-9]+)*(?:\/[a-z0-9]+(?:-[a-z0-9]+)*)*)$/;
const MAX_MANIFEST_BYTES = 5 * 1024 * 1024;
const MAX_HASH_BYTES = 16 * 1024 * 1024;
const MAX_BASELINE_BYTES = 1024 * 1024;
const MAX_JSON_DEPTH = 8;

const ROOT_FIELDS = new Set([
  'schemaVersion',
  'site',
  'base',
  'repository',
  'docsVersion',
  'pages',
  'releaseEvidence',
  'aliases',
  'examples',
  'diagrams',
]);
const ROOT_REQUIRED = ['schemaVersion', 'site', 'base', 'repository', 'docsVersion', 'pages'];
const PAGE_FIELDS = new Set([
  'id',
  'source',
  'type',
  'audience',
  'class',
  'sources',
  'reviewDigest',
  'reviewStatus',
  'reviewedAt',
  'appliesTo',
  'reviewedCommit',
  'aliases',
  'generatedSections',
  'examples',
  'impactPaths',
  'diagrams',
]);
const PAGE_REQUIRED = ['id', 'source', 'type', 'audience', 'class', 'sources', 'reviewDigest', 'appliesTo'];
const PAGE_OPTIONAL_LISTS = ['aliases', 'generatedSections', 'examples', 'impactPaths', 'diagrams'];
const PAGE_TYPES = new Set(['tutorial', 'how-to', 'reference', 'explanation']);
const PAGE_CLASSES = new Set(['L', 'C', 'A', 'F']);
const EVIDENCE_FIELDS = new Set(['path', 'sha256', 'symbol', 'heading']);
const RELEASE_EVIDENCE_FIELDS = new Set(['tag', 'commit', 'url', 'observedAt', 'verification', 'asset', 'sha256']);
const RELEASE_EVIDENCE_REQUIRED = ['tag', 'commit', 'url', 'observedAt', 'verification'];
const PRIVATE_SEGMENTS = new Set(['backups', '.git', '.agentic-sdd', 'node_modules', 'coverage', 'credentials', 'secrets']);
const PRIVATE_PREFIXES = ['docs/handovers/', 'website/.astro/', 'website/dist/'];
const ENGINEERING_PREFIXES = ['specs/', 'docs/testing/'];
const CREDENTIAL_FILE = /^(?:\.env(?:\..+)?|\.npmrc|\.netrc|id_(?:rsa|ed25519|ecdsa|dsa)(?:\.pub)?)$/i;
const CREDENTIAL_EXTENSION = /\.(?:pem|key|p12|pfx|keystore|jks)$/i;
const RESERVED_KEYS = new Set(['__proto__', 'prototype', 'constructor']);

function isPlainObject(value) {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) return false;
  const prototype = Object.getPrototypeOf(value);
  return prototype === Object.prototype || prototype === null;
}

function sha256Hex(value) {
  return createHash('sha256').update(value).digest('hex');
}

function byPath(left, right) {
  if (left.path < right.path) return -1;
  if (left.path > right.path) return 1;
  return 0;
}

function deepFreeze(value) {
  if (value !== null && typeof value === 'object') {
    for (const key of Object.keys(value)) deepFreeze(value[key]);
    Object.freeze(value);
  }
  return value;
}

function cloneJson(value, errorPath, depth = 0) {
  if (depth > MAX_JSON_DEPTH) {
    throw new ManifestError('invalid-manifest', errorPath, 'Manifest exceeds the supported nesting depth');
  }
  if (value === null) return null;
  const type = typeof value;
  if (type === 'string' || type === 'boolean') return value;
  if (type === 'number') {
    if (!Number.isFinite(value)) {
      throw new ManifestError('invalid-manifest', errorPath, `Non-finite number at "${errorPath}"`);
    }
    return value;
  }
  if (type !== 'object') {
    throw new ManifestError('invalid-manifest', errorPath, `Unsupported ${type} value at "${errorPath}"`);
  }
  if (Array.isArray(value)) {
    return value.map((item, index) => cloneJson(item, `${errorPath}[${index}]`, depth + 1));
  }
  if (!isPlainObject(value)) {
    throw new ManifestError('invalid-manifest', errorPath, `Expected a plain JSON object at "${errorPath}"`);
  }
  const clone = {};
  for (const key of Object.keys(value)) {
    const childPath = errorPath ? `${errorPath}.${key}` : key;
    if (RESERVED_KEYS.has(key)) {
      throw new ManifestError('unknown-field', childPath, `Refusing reserved object key "${key}"`);
    }
    clone[key] = cloneJson(value[key], childPath, depth + 1);
  }
  return clone;
}

function assertNonEmptyString(value, errorPath, code, label) {
  if (typeof value !== 'string' || value.length === 0 || value.trim() !== value) {
    throw new ManifestError(code, errorPath, `${label} must be a non-empty trimmed string`);
  }
  return value;
}

function assertIsoDate(value, errorPath, code) {
  if (typeof value !== 'string' || !DATE_PATTERN.test(value)) {
    throw new ManifestError(code, errorPath, 'Expected an ISO calendar date (YYYY-MM-DD)');
  }
  const [year, month, day] = value.split('-').map(Number);
  const date = new Date(Date.UTC(year, month - 1, day));
  if (
    year < 1000 ||
    date.getUTCFullYear() !== year ||
    date.getUTCMonth() !== month - 1 ||
    date.getUTCDate() !== day
  ) {
    throw new ManifestError(code, errorPath, `"${value}" is not a real calendar date`);
  }
  return value;
}

function assertHex(value, pattern, errorPath, code, label) {
  if (typeof value !== 'string' || !pattern.test(value)) {
    throw new ManifestError(code, errorPath, `${label} must match ${pattern.source}`);
  }
  return value;
}

// Repository-relative POSIX path. Returns the normalised value; every caller
// supplies the error code that fits its schema context.
function normalizeRelativePath(value, errorPath, code) {
  if (typeof value !== 'string' || value.length === 0) {
    throw new ManifestError(code, errorPath, 'Expected a non-empty repository-relative path');
  }
  if (value.includes('\0')) {
    throw new ManifestError(code, errorPath, 'Path contains a NUL byte');
  }
  if (value.includes('\\')) {
    throw new ManifestError(code, errorPath, 'Use POSIX "/" separators; backslashes are not contained paths');
  }
  if (value.startsWith('/') || /^[A-Za-z]:/.test(value)) {
    throw new ManifestError(code, errorPath, `"${value}" is absolute; expected a repository-relative path`);
  }
  const segments = value.split('/');
  if (segments.some((segment) => segment.length === 0 || segment === '.' || segment === '..')) {
    throw new ManifestError(code, errorPath, `"${value}" traverses or repeats a path segment`);
  }
  return segments.join('/');
}

function isPrivatePath(relative) {
  const segments = relative.split('/');
  if (segments.some((segment) => PRIVATE_SEGMENTS.has(segment))) return true;
  const filename = segments[segments.length - 1];
  if (CREDENTIAL_FILE.test(filename) || CREDENTIAL_EXTENSION.test(filename)) return true;
  return PRIVATE_PREFIXES.some((prefix) => relative.startsWith(prefix));
}

function assertPublicPath(relative, errorPath) {
  if (isPrivatePath(relative)) {
    throw new ManifestError(
      'private-source-path',
      errorPath,
      `"${relative}" is in a private or history tree and cannot be public documentation input`,
    );
  }
  return relative;
}

function resolveRepoRoot(repoRoot) {
  if (typeof repoRoot !== 'string' || repoRoot.length === 0) {
    throw new ManifestError('invalid-repo-root', 'repoRoot', 'Expected a non-empty repository root path');
  }
  let real;
  let stats;
  try {
    real = realpathSync(path.resolve(repoRoot));
    stats = statSync(real);
  } catch (error) {
    throw new ManifestError('invalid-repo-root', 'repoRoot', `Cannot resolve repository root "${repoRoot}": ${error.code ?? error.message}`);
  }
  if (!stats.isDirectory()) {
    throw new ManifestError('invalid-repo-root', 'repoRoot', `Repository root "${repoRoot}" is not a directory`);
  }
  return real;
}

// Resolves an already normalised relative path below a real repository root.
// Every component is lstat-checked so symlink parents, symlink leaves and
// special files are refused before a read.
function resolveContainedRegularFile(rootReal, relative, errorPath) {
  const segments = relative.split('/');
  let current = rootReal;
  for (let index = 0; index < segments.length; index += 1) {
    current = path.join(current, segments[index]);
    let stats;
    try {
      stats = lstatSync(current);
    } catch (error) {
      if (error.code === 'ENOENT' || error.code === 'ENOTDIR') {
        throw new ManifestError('missing-source-file', errorPath, `Missing contained file "${relative}"`);
      }
      throw new ManifestError('unreadable-source', errorPath, `Cannot stat "${relative}": ${error.code ?? error.message}`);
    }
    if (stats.isSymbolicLink()) {
      throw new ManifestError(
        'symlink-path',
        errorPath,
        `"${relative}" resolves through symlink "${segments.slice(0, index + 1).join('/')}"`,
      );
    }
    if (index < segments.length - 1) {
      if (!stats.isDirectory()) {
        throw new ManifestError('missing-source-file', errorPath, `"${relative}" is not a directory below "${segments.slice(0, index + 1).join('/')}"`);
      }
      continue;
    }
    if (!stats.isFile()) {
      throw new ManifestError('not-a-regular-file', errorPath, `"${relative}" is not a regular file`);
    }
  }
  let real;
  try {
    real = realpathSync(current);
  } catch (error) {
    throw new ManifestError('unreadable-source', errorPath, `Cannot resolve "${relative}": ${error.code ?? error.message}`);
  }
  if (real !== current || !real.startsWith(rootReal.endsWith(path.sep) ? rootReal : `${rootReal}${path.sep}`)) {
    throw new ManifestError('path-escape', errorPath, `"${relative}" resolves outside the repository root`);
  }
  return current;
}

function readFileWithin(absolute, errorPath, maxBytes) {
  let stats;
  try {
    stats = lstatSync(absolute);
  } catch (error) {
    throw new ManifestError('missing-source-file', errorPath, `Missing contained file "${errorPath}": ${error.code ?? error.message}`);
  }
  if (!stats.isFile()) {
    throw new ManifestError('not-a-regular-file', errorPath, `"${errorPath}" is not a regular file`);
  }
  if (stats.size > maxBytes) {
    throw new ManifestError('file-too-large', errorPath, `"${errorPath}" exceeds ${maxBytes} bytes`);
  }
  return readFileSync(absolute);
}

function validateOptions(options, verifySources) {
  if (options === undefined || options === null) {
    if (options === null) throw new ManifestError('invalid-options', 'options', 'Options must be an object');
    return;
  }
  if (!isPlainObject(options)) {
    throw new ManifestError('invalid-options', 'options', 'Options must be a plain object');
  }
  for (const key of Object.keys(options)) {
    if (key !== 'repoRoot' && key !== 'verifySources') {
      throw new ManifestError('invalid-options', key, `Unknown option "${key}"`);
    }
  }
  if (typeof verifySources !== 'boolean') {
    throw new ManifestError('invalid-options', 'verifySources', 'verifySources must be a boolean');
  }
  if (options.repoRoot !== undefined && (typeof options.repoRoot !== 'string' || options.repoRoot.length === 0)) {
    throw new ManifestError('invalid-options', 'repoRoot', 'repoRoot must be a non-empty string');
  }
}

function validateJsonValue(value, errorPath, code, depth = 0) {
  if (depth > MAX_JSON_DEPTH) {
    throw new ManifestError(code, errorPath, `Value at "${errorPath}" exceeds the supported nesting depth`);
  }
  if (value === null) return;
  const type = typeof value;
  if (type === 'string' || type === 'boolean') return;
  if (type === 'number') {
    if (!Number.isFinite(value)) throw new ManifestError(code, errorPath, `Non-finite number at "${errorPath}"`);
    return;
  }
  if (type !== 'object') {
    throw new ManifestError(code, errorPath, `Unsupported ${type} value at "${errorPath}"`);
  }
  if (Array.isArray(value)) {
    value.forEach((item, index) => validateJsonValue(item, `${errorPath}[${index}]`, code, depth + 1));
    return;
  }
  if (!isPlainObject(value)) {
    throw new ManifestError(code, errorPath, `Expected a plain JSON object at "${errorPath}"`);
  }
  for (const [key, child] of Object.entries(value)) {
    validateJsonValue(child, `${errorPath}.${key}`, code, depth + 1);
  }
}

function validateJsonList(value, errorPath, code) {
  if (!Array.isArray(value)) {
    throw new ManifestError(code, errorPath, `Expected an array at "${errorPath}"`);
  }
  value.forEach((item, index) => {
    const itemPath = `${errorPath}[${index}]`;
    if (typeof item === 'string') {
      if (item.length === 0) throw new ManifestError(code, itemPath, 'Expected a non-empty string');
      return;
    }
    if (!isPlainObject(item)) {
      throw new ManifestError(code, itemPath, `Expected a non-empty string or object at "${itemPath}"`);
    }
    validateJsonValue(item, itemPath, code);
  });
}

function normalizeEvidenceList(sources, errorPathPrefix = '') {
  if (!Array.isArray(sources)) {
    throw new ManifestError(
      'invalid-evidence-list',
      errorPathPrefix || 'sources',
      'Expected an array of {path, sha256} evidence records',
    );
  }
  const seen = new Set();
  return sources.map((entry, index) => {
    const entryPath = errorPathPrefix ? `${errorPathPrefix}[${index}]` : `[${index}]`;
    if (!isPlainObject(entry)) {
      throw new ManifestError('invalid-evidence', entryPath, 'Expected a {path, sha256} object');
    }
    for (const key of Object.keys(entry)) {
      if (!EVIDENCE_FIELDS.has(key)) {
        throw new ManifestError('unknown-field', `${entryPath}.${key}`, `Unknown evidence field "${key}"`);
      }
    }
    const declared = normalizeRelativePath(entry.path, `${entryPath}.path`, 'invalid-evidence');
    assertPublicPath(declared, `${entryPath}.path`);
    assertHex(entry.sha256, SHA256_PATTERN, `${entryPath}.sha256`, 'invalid-evidence', 'sha256');
    for (const key of ['symbol', 'heading']) {
      if (entry[key] !== undefined) {
        assertNonEmptyString(entry[key], `${entryPath}.${key}`, 'invalid-evidence', key);
      }
    }
    if (seen.has(declared)) {
      throw new ManifestError('duplicate-evidence-path', `${entryPath}.path`, `Duplicate evidence path "${declared}"`);
    }
    seen.add(declared);
    return { path: declared, sha256: entry.sha256, ...(entry.symbol === undefined ? {} : { symbol: entry.symbol }), ...(entry.heading === undefined ? {} : { heading: entry.heading }) };
  });
}

// Hashing accepts either a repository-relative string or a partial record. A
// supplied sha256 is verified against the file rather than trusted.
function normalizeSourceInputs(sources) {
  if (!Array.isArray(sources)) {
    throw new ManifestError('invalid-evidence-list', 'sources', 'Expected an array of source paths or records');
  }
  const seen = new Set();
  return sources.map((entry, index) => {
    const entryPath = `[${index}]`;
    let declaredInput;
    let declaredSha;
    if (typeof entry === 'string') {
      declaredInput = entry;
    } else if (isPlainObject(entry)) {
      for (const key of Object.keys(entry)) {
        if (!EVIDENCE_FIELDS.has(key)) {
          throw new ManifestError('unknown-field', `${entryPath}.${key}`, `Unknown source field "${key}"`);
        }
      }
      declaredInput = entry.path;
      declaredSha = entry.sha256;
    } else {
      throw new ManifestError('invalid-evidence', entryPath, 'Expected a source path string or {path, sha256} object');
    }
    const pathErrorPath = typeof declaredInput === 'string' ? declaredInput : `${entryPath}.path`;
    const declared = normalizeRelativePath(declaredInput, pathErrorPath, 'path-escape');
    assertPublicPath(declared, declared);
    if (declaredSha !== undefined) {
      assertHex(declaredSha, SHA256_PATTERN, `${entryPath}.sha256`, 'invalid-evidence', 'sha256');
    }
    if (seen.has(declared)) {
      throw new ManifestError('duplicate-evidence-path', declared, `Duplicate source path "${declared}"`);
    }
    seen.add(declared);
    return { path: declared, sha256: declaredSha };
  });
}

function reviewDigest(records) {
  const canonical = [...records]
    .sort(byPath)
    .map(({ path: sourcePath, sha256 }) => ({ path: sourcePath, sha256 }));
  return sha256Hex(JSON.stringify(canonical));
}

export function computeReviewDigest(sources) {
  return reviewDigest(normalizeEvidenceList(sources));
}

export function hashSources(repoRoot, sources) {
  const rootReal = resolveRepoRoot(repoRoot);
  const inputs = normalizeSourceInputs(sources);
  const results = inputs.map((input) => {
    const absolute = resolveContainedRegularFile(rootReal, input.path, input.path);
    const digest = sha256Hex(readFileWithin(absolute, input.path, MAX_HASH_BYTES));
    if (input.sha256 !== undefined && input.sha256 !== digest) {
      throw new ManifestError(
        'source-sha-mismatch',
        input.path,
        `Declared sha256 for "${input.path}" does not match the contained file (expected ${input.sha256}, computed ${digest})`,
      );
    }
    return { path: input.path, sha256: digest };
  });
  return results.sort(byPath);
}

function validateSite(site) {
  if (typeof site !== 'string' || site.length === 0) {
    throw new ManifestError('invalid-site', 'site', 'site must be the approved HTTPS origin');
  }
  let url;
  try {
    url = new URL(site);
  } catch {
    throw new ManifestError('invalid-site', 'site', `"${site}" is not an absolute URL`);
  }
  if (url.protocol !== 'https:') {
    throw new ManifestError('invalid-site', 'site', 'site must use HTTPS');
  }
  if (url.username || url.password) {
    throw new ManifestError('invalid-site', 'site', 'site must not carry credentials');
  }
  if (url.pathname !== '/' || url.search !== '' || url.hash !== '') {
    throw new ManifestError('invalid-site', 'site', 'site must be a bare origin without path, query or fragment');
  }
  if (!url.hostname || site !== url.origin) {
    throw new ManifestError('invalid-site', 'site', `site must be the normalised origin "${url.origin}" without a trailing slash`);
  }
}

function validateBase(base) {
  if (typeof base !== 'string' || !base.startsWith('/')) {
    throw new ManifestError('invalid-base', 'base', 'base must start with "/" (use "/" for a root site)');
  }
  if (base === '/') return;
  if (base.endsWith('/')) {
    throw new ManifestError('invalid-base', 'base', 'base must not end with "/"');
  }
  const segments = base.slice(1).split('/');
  if (segments.some((segment) => !/^[A-Za-z0-9][A-Za-z0-9._~-]*$/.test(segment))) {
    throw new ManifestError('invalid-base', 'base', 'base must be a normalised "/" separated project prefix without empty, "." or ".." segments');
  }
}

function validateRepository(repository) {
  if (typeof repository !== 'string') {
    throw new ManifestError('invalid-repository', 'repository', 'repository must be a fixed HTTPS GitHub repository URL');
  }
  const match = /^https:\/\/github\.com\/([A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?)\/([A-Za-z0-9._-]+)$/.exec(repository);
  if (!match || match[2] === '.' || match[2] === '..' || match[2].endsWith('.git')) {
    throw new ManifestError(
      'invalid-repository',
      'repository',
      'repository must be "https://github.com/<owner>/<repo>" without ".git", subpaths or trailing slash',
    );
  }
}

function validateReleaseEvidence(manifest) {
  const tags = new Set();
  const entries = manifest.releaseEvidence;
  if (entries === undefined) return tags;
  if (!Array.isArray(entries)) {
    throw new ManifestError('invalid-release-evidence', 'releaseEvidence', 'releaseEvidence must be an array');
  }
  entries.forEach((entry, index) => {
    const base = `releaseEvidence[${index}]`;
    if (!isPlainObject(entry)) {
      throw new ManifestError('invalid-release-evidence', base, 'Expected a release evidence object');
    }
    for (const key of Object.keys(entry)) {
      if (!RELEASE_EVIDENCE_FIELDS.has(key)) {
        throw new ManifestError('unknown-field', `${base}.${key}`, `Unknown release evidence field "${key}"`);
      }
    }
    for (const key of RELEASE_EVIDENCE_REQUIRED) {
      if (!Object.hasOwn(entry, key)) {
        throw new ManifestError('missing-field', `${base}.${key}`, `Release evidence is missing "${key}"`);
      }
    }
    if (typeof entry.tag !== 'string' || !TAG_PATTERN.test(entry.tag)) {
      throw new ManifestError('invalid-release-evidence', `${base}.tag`, 'tag must be an evidenced "v<semver>" release tag');
    }
    if (tags.has(entry.tag)) {
      throw new ManifestError('duplicate-release-evidence', `${base}.tag`, `Duplicate release evidence for "${entry.tag}"`);
    }
    assertHex(entry.commit, FULL_COMMIT_PATTERN, `${base}.commit`, 'invalid-release-evidence', 'commit');
    const expectedUrl = `${manifest.repository}/releases/tag/${entry.tag}`;
    if (typeof entry.url !== 'string' || entry.url !== expectedUrl) {
      throw new ManifestError('invalid-release-evidence', `${base}.url`, `Release URL must be the fixed public URL "${expectedUrl}"`);
    }
    assertIsoDate(entry.observedAt, `${base}.observedAt`, 'invalid-release-evidence');
    assertNonEmptyString(entry.verification, `${base}.verification`, 'invalid-release-evidence', 'verification');
    if (entry.asset !== undefined) {
      assertNonEmptyString(entry.asset, `${base}.asset`, 'invalid-release-evidence', 'asset');
      assertHex(entry.sha256, SHA256_PATTERN, `${base}.sha256`, 'invalid-release-evidence', 'sha256');
    } else if (entry.sha256 !== undefined) {
      throw new ManifestError('invalid-release-evidence', `${base}.sha256`, 'sha256 is only valid together with an asset name');
    }
    tags.add(entry.tag);
  });
  return tags;
}

function validateDocsVersion(docsVersion, evidenceTags) {
  if (typeof docsVersion !== 'string' || docsVersion.length === 0) {
    throw new ManifestError('invalid-docs-version', 'docsVersion', 'docsVersion must be "dev" or an evidenced release tag');
  }
  if (docsVersion === DEV_VERSION) return;
  if (!TAG_PATTERN.test(docsVersion)) {
    throw new ManifestError('invalid-docs-version', 'docsVersion', 'docsVersion must be "dev" or an evidenced "v<semver>" tag');
  }
  if (!evidenceTags.has(docsVersion)) {
    throw new ManifestError(
      'unevidenced-version',
      'docsVersion',
      `docsVersion "${docsVersion}" needs a matching releaseEvidence entry; never infer "latest"`,
    );
  }
}

function validatePage(page, index, evidenceTags) {
  const base = `pages[${index}]`;
  if (!isPlainObject(page)) {
    throw new ManifestError('invalid-page', base, 'Expected a page object');
  }
  for (const key of Object.keys(page)) {
    if (!PAGE_FIELDS.has(key)) {
      throw new ManifestError('unknown-field', `${base}.${key}`, `Unknown page field "${key}"`);
    }
  }
  for (const key of PAGE_REQUIRED) {
    if (!Object.hasOwn(page, key)) {
      throw new ManifestError('missing-field', `${base}.${key}`, `Page is missing required field "${key}"`);
    }
  }

  if (
    typeof page.id !== 'string' ||
    !ROUTE_PATTERN.test(page.id) ||
    (page.id !== 'index' && page.id.split('/').some((segment) => segment === 'index'))
  ) {
    throw new ManifestError(
      'invalid-id',
      `${base}.id`,
      'id must be "index" or a slash-separated lowercase kebab route without extension, traversal or "index" segments',
    );
  }

  const source = normalizeRelativePath(page.source, `${base}.source`, 'invalid-source-path');
  if (
    !source.startsWith(PUBLIC_DOCS_ROOT) ||
    source.length <= PUBLIC_DOCS_ROOT.length ||
    !source.endsWith('.md')
  ) {
    throw new ManifestError(
      'invalid-source-path',
      `${base}.source`,
      `source must be a repository-relative ".md" file under "${PUBLIC_DOCS_ROOT}"`,
    );
  }
  page.source = source;

  if (!PAGE_TYPES.has(page.type)) {
    throw new ManifestError('invalid-type', `${base}.type`, 'type must be tutorial, how-to, reference or explanation');
  }

  if (!Array.isArray(page.audience) || page.audience.length === 0) {
    throw new ManifestError('invalid-audience', `${base}.audience`, 'audience must be a non-empty array of strings');
  }
  const audiences = new Set();
  page.audience.forEach((entry, entryIndex) => {
    assertNonEmptyString(entry, `${base}.audience[${entryIndex}]`, 'invalid-audience', 'audience entry');
    if (audiences.has(entry)) {
      throw new ManifestError('invalid-audience', `${base}.audience[${entryIndex}]`, `Duplicate audience "${entry}"`);
    }
    audiences.add(entry);
  });

  if (!PAGE_CLASSES.has(page.class)) {
    throw new ManifestError('invalid-class', `${base}.class`, 'class must be L, C, A or F');
  }

  const sources = normalizeEvidenceList(page.sources, `${base}.sources`);
  page.sources = sources;

  if (typeof page.reviewDigest !== 'string' || !SHA256_PATTERN.test(page.reviewDigest)) {
    throw new ManifestError('invalid-review-digest', `${base}.reviewDigest`, 'reviewDigest must be a lowercase SHA-256 hex digest');
  }
  const computed = reviewDigest(sources);
  if (computed !== page.reviewDigest) {
    throw new ManifestError(
      'review-digest-mismatch',
      `${base}.reviewDigest`,
      `reviewDigest does not match the path-sorted {path,sha256} evidence list (expected ${computed}); a human must review and refresh it`,
    );
  }

  if (page.reviewStatus !== undefined) {
    if (!['unreviewed', 'reviewed'].includes(page.reviewStatus)) {
      throw new ManifestError('invalid-review-status', `${base}.reviewStatus`, 'reviewStatus must be unreviewed or reviewed');
    }
    if (page.reviewStatus === 'unreviewed' && page.reviewedAt !== undefined) {
      throw new ManifestError('invalid-review-status', `${base}.reviewedAt`, 'unreviewed pages must not claim a human review date');
    }
    if (page.reviewStatus === 'reviewed' && page.reviewedAt === undefined) {
      throw new ManifestError('invalid-review-status', `${base}.reviewedAt`, 'reviewed pages require a human review date');
    }
  } else if (page.reviewedAt !== undefined) {
    assertIsoDate(page.reviewedAt, `${base}.reviewedAt`, 'invalid-reviewed-at');
  }

  if (page.appliesTo !== UNRELEASED) {
    if (typeof page.appliesTo !== 'string' || !TAG_PATTERN.test(page.appliesTo) || !evidenceTags.has(page.appliesTo)) {
      throw new ManifestError(
        'invalid-applies-to',
        `${base}.appliesTo`,
        'appliesTo must be "unreleased" or an evidenced "v<semver>" release tag',
      );
    }
  }

  if (page.reviewedCommit !== undefined) {
    assertHex(page.reviewedCommit, COMMIT_PATTERN, `${base}.reviewedCommit`, 'invalid-reviewed-commit', 'reviewedCommit');
  }

  for (const field of PAGE_OPTIONAL_LISTS) {
    if (page[field] !== undefined) {
      validateJsonList(page[field], `${base}.${field}`, 'invalid-list');
    }
  }
}

export function validateManifest(manifest, options = {}) {
  if (!isPlainObject(manifest) || Array.isArray(manifest)) {
    throw new ManifestError('invalid-manifest', '', 'Manifest must be a plain JSON object');
  }
  const { repoRoot, verifySources = false } = options ?? {};
  validateOptions(options, verifySources);
  if (verifySources && repoRoot === undefined) {
    throw new ManifestError('missing-repo-root', 'repoRoot', 'verifySources requires repoRoot');
  }

  const clone = cloneJson(manifest, '');
  for (const key of Object.keys(clone)) {
    if (!ROOT_FIELDS.has(key)) {
      throw new ManifestError('unknown-field', key, `Unknown root field "${key}"`);
    }
  }
  for (const key of ROOT_REQUIRED) {
    if (!Object.hasOwn(clone, key)) {
      throw new ManifestError('missing-field', key, `Manifest is missing required root field "${key}"`);
    }
  }
  if (clone.schemaVersion !== SCHEMA_VERSION) {
    throw new ManifestError('invalid-schema-version', 'schemaVersion', `Unsupported schemaVersion ${JSON.stringify(clone.schemaVersion)}`);
  }

  validateSite(clone.site);
  validateBase(clone.base);
  validateRepository(clone.repository);

  const evidenceTags = validateReleaseEvidence(clone);
  validateDocsVersion(clone.docsVersion, evidenceTags);

  for (const field of ['aliases', 'examples', 'diagrams']) {
    if (clone[field] !== undefined) validateJsonList(clone[field], field, 'invalid-list');
  }

  if (!Array.isArray(clone.pages)) {
    throw new ManifestError('invalid-pages', 'pages', 'pages must be an array');
  }
  if (clone.pages.length === 0) {
    throw new ManifestError('empty-pages', 'pages', 'pages must contain at least one page; an empty manifest is not a valid C1 registry');
  }

  const ids = new Set();
  const sources = new Set();
  clone.pages.forEach((page, index) => {
    validatePage(page, index, evidenceTags);
    if (ids.has(page.id)) {
      throw new ManifestError('duplicate-page-id', `pages[${index}].id`, `Duplicate page id "${page.id}"`);
    }
    ids.add(page.id);
    if (sources.has(page.source)) {
      throw new ManifestError('duplicate-page-source', `pages[${index}].source`, `Duplicate canonical source "${page.source}"`);
    }
    sources.add(page.source);
  });

  if (verifySources) {
    const rootReal = resolveRepoRoot(repoRoot);
    for (const page of clone.pages) {
      resolveContainedRegularFile(rootReal, page.source, page.source);
      for (const evidence of page.sources) {
        const absolute = resolveContainedRegularFile(rootReal, evidence.path, evidence.path);
        const digest = sha256Hex(readFileWithin(absolute, evidence.path, MAX_HASH_BYTES));
        if (digest !== evidence.sha256) {
          throw new ManifestError(
            'source-sha-mismatch',
            evidence.path,
            `Evidence sha256 for "${evidence.path}" does not match the contained file (expected ${evidence.sha256}, computed ${digest})`,
          );
        }
      }
    }
  }

  return deepFreeze(clone);
}

export function loadManifest(manifestPath, options = {}) {
  // repoRoot defaults to the parent of the manifest's directory, matching
  // <repoRoot>/website/docs-manifest.json; pass repoRoot explicitly otherwise.
  if (typeof manifestPath !== 'string' || manifestPath.length === 0) {
    throw new ManifestError('invalid-options', 'manifestPath', 'Expected a non-empty manifest path');
  }
  const { repoRoot, verifySources = false } = options ?? {};
  validateOptions(options, verifySources);

  const absolute = path.resolve(manifestPath);
  let stats;
  try {
    stats = lstatSync(absolute);
  } catch (error) {
    throw new ManifestError('manifest-missing', absolute, `Cannot stat manifest "${manifestPath}": ${error.code ?? error.message}`);
  }
  if (stats.isSymbolicLink()) {
    throw new ManifestError('manifest-symlink', absolute, 'Manifest path must not be a symlink');
  }
  if (!stats.isFile()) {
    throw new ManifestError('manifest-not-regular-file', absolute, 'Manifest path must be a regular file');
  }
  if (stats.size > MAX_MANIFEST_BYTES) {
    throw new ManifestError('manifest-too-large', absolute, `Manifest exceeds ${MAX_MANIFEST_BYTES} bytes`);
  }

  let text;
  try {
    text = readFileSync(absolute, 'utf8');
  } catch (error) {
    throw new ManifestError('unreadable-manifest', absolute, `Cannot read manifest: ${error.code ?? error.message}`);
  }
  let parsed;
  try {
    parsed = JSON.parse(text);
  } catch (error) {
    throw new ManifestError('invalid-json', absolute, `Manifest is not valid JSON: ${error.message}`);
  }

  const resolvedRoot = repoRoot ?? path.dirname(path.dirname(absolute));
  return validateManifest(parsed, { repoRoot: resolvedRoot, verifySources });
}

// __BASELINE_SECTION__

const ATX_HEADING = /^ {0,3}(#{1,6})(?:[ \t]+(.*?))?[ \t]*$/;
const SETEXT_UNDERLINE = /^ {0,3}(=+|-+)[ \t]*$/;
const FENCE_OPEN = /^ {0,3}(`{3,}|~{3,})(.*)$/;
const FENCE_CLOSE = /^ {0,3}(`{3,}|~{3,})[ \t]*$/;
const INLINE_LINK = /(!?)\[([^\]\n]*)\]\([ \t]*(<[^<>\n]*>|(?:[^()\s]|\([^()\s]*\))*)(?:[ \t]+(?:"[^"\n]*"|'[^'\n]*'|\([^()\n]*\)))?[ \t]*\)/g;
const LINK_DEFINITION = /^ {0,3}\[([^\]\n]+)\]:[ \t]*(<[^>\n]*>|\S+)/;

function maskInlineCode(line) {
  const characters = line.split('');
  let index = 0;
  while (index < line.length) {
    if (line[index] !== '`') {
      index += 1;
      continue;
    }
    let runLength = 1;
    while (line[index + runLength] === '`') runLength += 1;
    const marker = '`'.repeat(runLength);
    let searchFrom = index + runLength;
    let closing = -1;
    while (true) {
      const candidate = line.indexOf(marker, searchFrom);
      if (candidate === -1) break;
      let runEnd = candidate + runLength;
      let total = runLength;
      while (line[runEnd] === '`') {
        runEnd += 1;
        total += 1;
      }
      if (total === runLength) {
        closing = candidate;
        break;
      }
      searchFrom = runEnd;
    }
    if (closing === -1) {
      index += runLength;
      continue;
    }
    for (let cursor = index; cursor < closing + runLength; cursor += 1) characters[cursor] = ' ';
    index = closing + runLength;
  }
  return characters.join('');
}

function maskHtmlComments(masked) {
  let inside = false;
  for (let index = 0; index < masked.length; index += 1) {
    let line = masked[index];
    let cursor = 0;
    while (cursor < line.length) {
      if (inside) {
        const end = line.indexOf('-->', cursor);
        if (end === -1) {
          line = `${line.slice(0, cursor)}${' '.repeat(line.length - cursor)}`;
          break;
        }
        line = `${line.slice(0, cursor)}${' '.repeat(end + 3 - cursor)}${line.slice(end + 3)}`;
        cursor = end + 3;
        inside = false;
        continue;
      }
      const start = line.indexOf('<!--', cursor);
      if (start === -1) break;
      const end = line.indexOf('-->', start + 4);
      if (end === -1) {
        line = `${line.slice(0, start)}${' '.repeat(line.length - start)}`;
        cursor = line.length;
        inside = true;
        break;
      }
      line = `${line.slice(0, start)}${' '.repeat(end + 3 - start)}${line.slice(end + 3)}`;
      cursor = end + 3;
    }
    masked[index] = line;
  }
}

// Builds per-line views for structure parsing: fenced blocks, frontmatter,
// inline code and HTML comments are blanked so they cannot contribute headings
// or links. Blanking preserves line numbering.
function buildStructureView(text) {
  const lines = text.replace(/^\uFEFF/, '').split(/\r?\n/);
  const masked = lines.map((line) => line);
  const skipped = new Set();

  if (lines.length > 0 && lines[0].trim() === '---') {
    for (let index = 1; index < lines.length; index += 1) {
      const trimmed = lines[index].trim();
      if (trimmed === '---' || trimmed === '...') {
        for (let cursor = 0; cursor <= index; cursor += 1) {
          skipped.add(cursor);
          masked[cursor] = '';
        }
        break;
      }
    }
  }

  let fence = null;
  for (let index = 0; index < lines.length; index += 1) {
    if (skipped.has(index)) continue;
    const line = lines[index];
    if (fence) {
      skipped.add(index);
      masked[index] = '';
      const closing = FENCE_CLOSE.exec(line);
      if (closing && closing[1][0] === fence.character && closing[1].length >= fence.length) {
        fence = null;
      }
      continue;
    }
    const opening = FENCE_OPEN.exec(line);
    if (opening && !(opening[1][0] === '`' && opening[2].includes('`'))) {
      fence = { character: opening[1][0], length: opening[1].length };
      skipped.add(index);
      masked[index] = '';
      continue;
    }
    masked[index] = maskInlineCode(line);
  }

  maskHtmlComments(masked);
  return { lines, masked, skipped };
}

function renderedText(markdown) {
  return markdown
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/\[([^\]]*)\]\[[^\]]*\]/g, '$1')
    .replace(/`+([^`]*)`+/g, '$1')
    .replace(/<[^>]*>/g, '')
    .replace(/\\([\\`*_[\]()#+\-.!{}])/g, '$1')
    .replace(/[*~]/g, '')
    .trim();
}

// GitHub-style heading slug. Duplicate headings receive -1, -2, ... suffixes,
// matching old anchor fragments rather than framework-specific behaviour.
function githubSlug(text) {
  return text
    .trim()
    .toLowerCase()
    .replace(/[^\p{L}\p{N}\p{M}\p{Pc}\- ]/gu, '')
    .replace(/ /g, '-');
}

function parseMarkdownStructure(text) {
  const { lines, masked, skipped } = buildStructureView(text);
  const headings = [];
  const links = [];
  const slugCounts = new Map();

  const pushHeading = (lineIndex, level, rawText) => {
    const plain = renderedText(rawText);
    if (plain.length === 0) return;
    const baseSlug = githubSlug(plain);
    const previous = slugCounts.get(baseSlug) ?? 0;
    slugCounts.set(baseSlug, previous + 1);
    const slug = previous === 0 ? baseSlug : `${baseSlug}-${previous}`;
    headings.push({ line: lineIndex + 1, level, text: plain, slug });
  };

  for (let index = 0; index < lines.length; index += 1) {
    if (skipped.has(index)) continue;
    const line = masked[index];
    if (line.trim().length === 0) continue;

    const matched = ATX_HEADING.exec(line);
    if (matched) {
      const original = ATX_HEADING.exec(lines[index]);
      let raw = original?.[2] ?? '';
      raw = raw.replace(/[ \t]+#+[ \t]*$/, '').trim();
      if (raw.length > 0) pushHeading(index, matched[1].length, raw);
      continue;
    }

    if (index + 1 >= lines.length) continue;
    const underline = SETEXT_UNDERLINE.exec(masked[index + 1]);
    if (!underline) continue;
    if (index > 0 && masked[index - 1].trim().length > 0) continue;
    if (/^ {0,3}(?:[-*+][ \t]|>|\d+[.)][ \t]|\|)/.test(line)) continue;
    pushHeading(index, underline[1][0] === '=' ? 1 : 2, lines[index].trim());
  }

  for (let index = 0; index < lines.length; index += 1) {
    if (skipped.has(index)) continue;
    const line = masked[index];
    if (line.trim().length === 0) continue;

    const candidates = [];
    for (const match of line.matchAll(INLINE_LINK)) {
      if (match[1] === '!') continue;
      const destination = match[3].startsWith('<') && match[3].endsWith('>') ? match[3].slice(1, -1) : match[3];
      if (destination.length === 0) continue;
      candidates.push({ column: match.index, href: destination, text: match[2] });
    }
    const definition = LINK_DEFINITION.exec(line);
    if (definition) {
      const destination = definition[2].startsWith('<') && definition[2].endsWith('>')
        ? definition[2].slice(1, -1)
        : definition[2];
      if (destination.length > 0) {
        candidates.push({ column: definition.index, href: destination, text: definition[1] });
      }
    }
    candidates.sort((left, right) => left.column - right.column);
    for (const candidate of candidates) {
      links.push({ line: index + 1, href: candidate.href, text: candidate.text });
    }
  }

  return { headings, links };
}

export function collectBaseline(repoRoot, paths) {
  if (!Array.isArray(paths)) {
    throw new ManifestError('invalid-baseline-list', 'paths', 'Expected an explicit array of public Markdown paths');
  }
  const rootReal = resolveRepoRoot(repoRoot);
  const seen = new Set();
  const files = [];

  paths.forEach((declared, index) => {
    if (typeof declared !== 'string') {
      throw new ManifestError('invalid-baseline-path', `[${index}]`, 'Expected an explicit repository-relative Markdown path');
    }
    const relative = normalizeRelativePath(declared, declared, 'path-escape');
    if (!relative.endsWith('.md') || relative.length <= '.md'.length) {
      throw new ManifestError('invalid-baseline-path', relative, 'Baseline inventory accepts explicit ".md" paths only');
    }
    assertPublicPath(relative, relative);
    if (ENGINEERING_PREFIXES.some((prefix) => relative.startsWith(prefix))) {
      throw new ManifestError('private-source-path', relative, 'Engineering evidence is hashable but is not a public-guide baseline entry');
    }
    if (seen.has(relative)) return;
    seen.add(relative);

    const absolute = resolveContainedRegularFile(rootReal, relative, relative);
    const text = readFileWithin(absolute, relative, MAX_BASELINE_BYTES).toString('utf8');
    files.push({ path: relative, ...parseMarkdownStructure(text) });
  });

  return { files };
}
