// SPEC020 B4-W1-D (T09) — deterministic documentation checks.
//
// Contract:
//   checkSource({repoRoot, manifestPath, stubs})            -> frozen report
//   checkBuilt({repoRoot, manifestPath, builtDir})          -> frozen report
//   checkFreshness({repoRoot, manifestPath, changedPaths})  -> frozen report
//   checkDocumentation(options)                             -> combined report
//
// CLI:
//   node website/scripts/check.mjs [--repo DIR] [--manifest FILE]
//     [--built DIR] [--source] [--affected PATH[,PATH...]]
//     [--stub FILE=PAGE_ID[:ANCHOR,...]] [--json] [--help]
//
// Guarantees and limits:
// - Node built-ins only, synchronous, read-only and offline. External links are
//   enumerated as observations and never fetched. No Git command is executed:
//   freshness compares reviewed full-file fingerprints with current bytes and
//   never refreshes them automatically or consults commit ancestry.
// - Markdown structure is parsed with the reviewed `collectBaseline` helper
//   (fences, frontmatter, inline code and HTML comments do not contribute
//   headings or links). This module adds the raw-HTML, generated-marker, stub
//   and fragment rules that helper intentionally leaves to callers. Heading
//   slugs mirror `website/plugins/remark-link-routes.mjs`, so an anchor that
//   this check accepts is one the build transform can also accept. Raw HTML
//   `<a id>` retained anchors are fragment targets, not raw navigation.
// - Page and stub front matter is scanned for top-level review/approval claims
//   (`reviewStatus`, `reviewed*`, `approved*`, `approval(s)`, `acceptedBy`,
//   `humanReview`); content never owns a human decision.
// - Diagnostics are actionable: every entry names a repository-relative path,
//   a 1-based line (1 when a whole-file condition applies), the reason and a
//   concrete fix. Output is sorted by path, line and code and is stable across
//   repeated runs.
// - There is deliberately no ignore list and no glob suppression. `--stub`
//   declarations are the only scoped inputs, and each names one exact file and
//   one exact canonical page. Alias object/fragment semantics remain owned by
//   C1/T08: this checker flags only an exactly repeated alias entry or a
//   recognized old-path key collision, never a guessed fragment mapping.
// - Source mode enumerates the canonical content root and fails any `.md` page
//   missing from the manifest, matching the export producer: an unregistered
//   page must never build and publish while `export.mjs` refuses it.
// - Relative link targets outside the content root are classified by
//   `lib/public-paths.mjs`. Private trees, generated output and history
//   records fail with `links/private-target`; public engineering sources stay
//   linkable. One documented, source-and-target-keyed exception keeps the
//   retained MCP migration stub's historical-spec pointer working.
// - A fragment that cannot be re-parsed is an error when its target exists.
//   An unverifiable anchor is never downgraded to an observation.
// - `--built` resolves from the repository root (so `npm --prefix website run
//   docs:check-built` can pass `website/dist` without depending on the npm cwd);
//   `--manifest` resolves from the invocation directory. The programmatic API
//   resolves `manifestPath`/`builtDir` from `repoRoot` unless absolute.
// - Source-mode link resolution consumes `website/scripts/lib/routes.mjs`;
//   manifest schema/digest loading consumes `website/scripts/lib/manifest.mjs`.
//   This checker does not rewrite either helper.
//
// Freshness disposition model:
// - `changedPaths` (CLI `--affected`) is the current change set. A stale page
//   whose authored source, reviewed evidence or generated input intersects that
//   set fails with `freshness/needs-review` (one error per affected page).
// - A stale page outside that set is pre-existing drift: it produces exactly
//   one consolidated `freshness/unrelated-stale` observation, never a cascade.
// - Omitting `changedPaths` is full-publication mode: every stale page fails.
//   A reviewed page and an unreviewed page are both reported, because the
//   reviewed fingerprint is the contract; `reviewStatus` is included for the
//   human deciding whether to refresh content or record no-public-impact.

import { readFileSync, lstatSync, realpathSync, statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import {
  ManifestError,
  collectBaseline,
  computeReviewDigest,
  hashSources,
  loadManifest,
} from './lib/manifest.mjs';
import { CONTENT_ROOT, PublicTreeError, classifyLinkTarget, scanContentRoot } from './lib/public-paths.mjs';
import { RouteError, resolveLink } from './lib/routes.mjs';

const MODULE_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_REPO_ROOT = path.resolve(MODULE_DIR, '..', '..');
const DEFAULT_MANIFEST = 'website/docs-manifest.json';
const MAX_SOURCE_BYTES = 4 * 1024 * 1024;
const MAX_BUILT_HTML_BYTES = 8 * 1024 * 1024;
const EXTERNAL_SCHEME = /^(https|mailto):/i;
const ANY_SCHEME = /^[a-z][a-z0-9+.-]*:/i;
const MARKER_PATTERN = /<!--\s*generated:([^\s:]+):(start|end)\s*-->/g;
const FENCE_OPEN = /^ {0,3}(`{3,}|~{3,})(.*)$/;
const FENCE_CLOSE = /^ {0,3}(`{3,}|~{3,})[ \t]*$/;
const ORDERED_STEP = /^\s{0,7}\d+[.)][ \t]+\S/;
// An `<a>` element is navigation only when it opens a destination. A retained
// `<a id>` / `<a name>` anchor is not a link, so it must not be rejected as
// unrewritten raw HTML navigation; its id is a valid fragment target.
const RAW_HTML_LINK_START = /<a(?=[\s>]|$)/gi;
const RAW_HTML_HREF = /<a(?=[\s>])[^>]*\bhref\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))/i;
const HTML_ID_ATTRIBUTE = /(?:^|[\s<])id\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))/g;
// Top-level front-matter keys that can only mean "a human reviewed or approved
// this document". Content never owns that decision.
const FORGED_CLAIM_KEY =
  /^(?:approvals?|approved[A-Za-z0-9_-]*|acceptedBy|reviewStatus|reviewed[A-Za-z0-9_-]*|humanReview)$/i;

/* ------------------------------------------------------------------------- */
/* Small deterministic utilities                                              */
/* ------------------------------------------------------------------------- */

function deepFreeze(value) {
  if (value !== null && typeof value === 'object') {
    for (const key of Object.keys(value)) deepFreeze(value[key]);
    Object.freeze(value);
  }
  return value;
}

function compareStrings(left, right) {
  if (left < right) return -1;
  if (left > right) return 1;
  return 0;
}

function byDiagnostic(left, right) {
  return (
    compareStrings(left.path, right.path) ||
    (left.line ?? 0) - (right.line ?? 0) ||
    compareStrings(left.code, right.code) ||
    compareStrings(left.message, right.message)
  );
}

function toPosix(value) {
  return value.split(path.sep).join('/');
}

function normalizeRelative(value, label) {
  if (typeof value !== 'string' || value.length === 0) {
    throw new TypeError(`${label} must be a non-empty repository-relative path`);
  }
  if (value.includes('\0') || value.includes('\\')) {
    throw new TypeError(`${label} must be POSIX and free of control characters: ${JSON.stringify(value)}`);
  }
  if (value.startsWith('/') || /^[A-Za-z]:/.test(value)) {
    throw new TypeError(`${label} must be repository-relative, not absolute: ${value}`);
  }
  const segments = value.split('/');
  if (segments.some((segment) => segment.length === 0 || segment === '.' || segment === '..')) {
    throw new TypeError(`${label} has an empty, "." or ".." segment: ${value}`);
  }
  return segments.join('/');
}

function normalizeOptions(options, allowed, label) {
  if (options === undefined || options === null) return {};
  if (typeof options !== 'object' || Array.isArray(options)) {
    throw new TypeError(`${label} options must be a plain object`);
  }
  for (const key of Object.keys(options)) {
    if (!allowed.includes(key)) throw new TypeError(`Unknown ${label} option "${key}"`);
  }
  return options;
}

function resolveRepoRootOption(options) {
  const candidate = options.repoRoot ?? DEFAULT_REPO_ROOT;
  if (typeof candidate !== 'string' || candidate.length === 0) {
    throw new TypeError('repoRoot must be a non-empty directory path');
  }
  const real = realpathSync(path.resolve(candidate));
  if (!statSync(real).isDirectory()) throw new TypeError(`repoRoot is not a directory: ${candidate}`);
  return real;
}

function resolveManifestPath(repoRoot, options) {
  const candidate = options.manifestPath ?? path.join(repoRoot, DEFAULT_MANIFEST);
  return path.resolve(repoRoot, candidate);
}

function displayManifestPath(repoRoot, manifestPath) {
  const relative = toPosix(path.relative(repoRoot, manifestPath));
  if (relative.startsWith('../') || path.isAbsolute(relative)) return manifestPath;
  return relative;
}

function decodeUriComponent(value) {
  try {
    return { ok: true, value: decodeURIComponent(value) };
  } catch {
    return { ok: false, value: null };
  }
}

function decodeHtmlEntities(value) {
  return value.replace(/&(amp|lt|gt|quot|apos|#39|#x27|#x2F|#47);/gi, (match, name) => {
    const lowered = name.toLowerCase();
    if (lowered === 'amp') return '&';
    if (lowered === 'lt') return '<';
    if (lowered === 'gt') return '>';
    if (lowered === 'quot') return '"';
    if (lowered === 'apos' || lowered === '#39' || lowered === '#x27') return "'";
    return '/';
  });
}

function lineAt(text, index) {
  let line = 1;
  for (let cursor = 0; cursor < index && cursor < text.length; cursor += 1) {
    if (text[cursor] === '\n') line += 1;
  }
  return line;
}

function headingSlug(text) {
  return String(text)
    .normalize('NFKC')
    .trim()
    .toLowerCase()
    .replace(/\s/gu, '-')
    .replace(/[^\p{L}\p{N}\p{M}_-]+/gu, '');
}

function headingInfo(headings) {
  const counts = new Map();
  const duplicates = [];
  for (const heading of headings) {
    const slug = headingSlug(heading.text) || 'section';
    const count = counts.get(slug) ?? 0;
    if (count > 0) duplicates.push({ line: heading.line, slug, text: heading.text });
    counts.set(slug, count + 1);
  }
  return { slugs: new Set(counts.keys()), duplicates };
}

function formatSuggestionList(values, limit = 5) {
  const unique = [...new Set(values)].sort(compareStrings);
  const shown = unique.slice(0, limit).map((value) => `#${value}`).join(', ');
  return unique.length > limit ? `${shown} (+${unique.length - limit} more)` : shown;
}

function splitHref(href) {
  if (typeof href !== 'string' || href.length === 0) return null;
  const hashIndex = href.indexOf('#');
  const fragment = hashIndex === -1 ? null : href.slice(hashIndex + 1);
  const beforeHash = hashIndex === -1 ? href : href.slice(0, hashIndex);
  const queryIndex = beforeHash.indexOf('?');
  const query = queryIndex === -1 ? null : beforeHash.slice(queryIndex + 1);
  const pathPart = queryIndex === -1 ? beforeHash : beforeHash.slice(0, queryIndex);
  return { pathPart, query, fragment };
}

function pathMatchesChange(changedPath, target) {
  if (changedPath === target) return true;
  const directory = changedPath.endsWith('/') ? changedPath : `${changedPath}/`;
  return target.startsWith(directory);
}

function readContainedText(repoRoot, relative, maxBytes, label) {
  const normalized = normalizeRelative(relative, label);
  const absolute = path.join(repoRoot, normalized);
  let stats;
  try {
    stats = lstatSync(absolute);
  } catch (error) {
    const wrapped = new Error(`Cannot stat "${normalized}": ${error.code ?? error.message}`);
    wrapped.code = error.code === 'ENOENT' || error.code === 'ENOTDIR' ? 'missing' : 'unreadable';
    throw wrapped;
  }
  if (stats.isSymbolicLink()) {
    const wrapped = new Error(`"${normalized}" is a symlink; managed documentation inputs must be regular files`);
    wrapped.code = 'symlink';
    throw wrapped;
  }
  if (!stats.isFile()) {
    const wrapped = new Error(`"${normalized}" is not a regular file`);
    wrapped.code = 'not-file';
    throw wrapped;
  }
  if (stats.nlink > 1) {
    const wrapped = new Error(
      `"${normalized}" is a hard link (${stats.nlink} names); managed documentation inputs must be single-linked regular files`,
    );
    wrapped.code = 'hardlink';
    throw wrapped;
  }
  if (stats.size > maxBytes) {
    const wrapped = new Error(`"${normalized}" exceeds ${maxBytes} bytes`);
    wrapped.code = 'too-large';
    throw wrapped;
  }
  let real;
  try {
    real = realpathSync(absolute);
  } catch (error) {
    const wrapped = new Error(`Cannot resolve "${normalized}": ${error.code ?? error.message}`);
    wrapped.code = 'unreadable';
    throw wrapped;
  }
  const prefix = repoRoot.endsWith(path.sep) ? repoRoot : `${repoRoot}${path.sep}`;
  if (real !== repoRoot && !real.startsWith(prefix)) {
    const wrapped = new Error(`"${normalized}" resolves outside the repository root`);
    wrapped.code = 'escape';
    throw wrapped;
  }
  return readFileSync(real, 'utf8');
}

function inspectContainedTarget(repoRoot, relative) {
  const normalized = path.posix.normalize(relative);
  if (normalized === '.' || normalized === '..' || normalized.startsWith('../') || normalized.startsWith('/')) {
    return { kind: 'escape' };
  }
  const segments = normalized.split('/');
  let current = repoRoot;
  for (let index = 0; index < segments.length; index += 1) {
    current = path.join(current, segments[index]);
    let stats;
    try {
      stats = lstatSync(current);
    } catch (error) {
      if (error.code === 'ENOENT' || error.code === 'ENOTDIR') return { kind: 'missing' };
      return { kind: 'unreadable' };
    }
    if (stats.isSymbolicLink()) return { kind: 'symlink' };
    if (index < segments.length - 1 && !stats.isDirectory()) return { kind: 'missing' };
    if (index === segments.length - 1) {
      if (stats.isDirectory()) return { kind: 'directory' };
      if (!stats.isFile()) return { kind: 'special' };
      return { kind: stats.nlink > 1 ? 'hardlink' : 'file' };
    }
  }
  return { kind: 'missing' };
}

/* ------------------------------------------------------------------------- */
/* Report                                                                    */
/* ------------------------------------------------------------------------- */

class CheckReport {
  constructor() {
    this.entries = [];
    this.metrics = {
      pages: 0,
      links: 0,
      external: 0,
      stubs: 0,
      builtPages: 0,
    };
  }

  error(code, entryPath, line, message, fix, details = {}) {
    this.push('error', code, entryPath, line, message, fix, details);
  }

  observation(code, entryPath, line, message, fix, details = {}) {
    this.push('observation', code, entryPath, line, message, fix, details);
  }

  push(severity, code, entryPath, line, message, fix, details) {
    if (typeof code !== 'string' || code.length === 0) throw new Error('Internal: diagnostic code is required');
    if (typeof entryPath !== 'string' || entryPath.length === 0) throw new Error(`Internal: ${code} needs a path`);
    if (typeof message !== 'string' || message.length === 0) throw new Error(`Internal: ${code} needs a message`);
    if (typeof fix !== 'string' || fix.length === 0) throw new Error(`Internal: ${code} needs an actionable fix`);
    this.entries.push(
      deepFreeze({
        severity,
        code,
        path: entryPath,
        line: Number.isInteger(line) && line > 0 ? line : 1,
        message,
        fix,
        details: { ...details },
      }),
    );
  }

  result() {
    const sorted = [...this.entries].sort(byDiagnostic);
    const errors = Object.freeze(sorted.filter((entry) => entry.severity === 'error'));
    const observations = Object.freeze(sorted.filter((entry) => entry.severity === 'observation'));
    const summary = Object.freeze({
      ...this.metrics,
      errorCount: errors.length,
      observationCount: observations.length,
    });
    return deepFreeze({
      ok: errors.length === 0,
      errors,
      observations,
      summary,
    });
  }
}

/* ------------------------------------------------------------------------- */
/* Manifest loading and diagnostic line lookup                               */
/* ------------------------------------------------------------------------- */

function manifestFixFor(code) {
  switch (code) {
    case 'duplicate-page-source':
      return 'Keep exactly one page entry per canonical source; move the other content or remove the duplicate page.';
    case 'duplicate-page-id':
      return 'Give each page a unique route id; one URL has one owner.';
    case 'review-digest-mismatch':
      return 'Review the listed evidence, then update reviewDigest to the path-sorted SHA-256 digest of the corrected source list. Never auto-refresh it.';
    case 'source-sha-mismatch':
      return 'Re-review the changed input, then update its sha256 and the page reviewDigest together.';
    case 'missing-source-file':
      return 'Restore the contained source file or remove/replace the manifest reference after review.';
    case 'private-source-path':
      return 'Use a public documentation input; private scratch, history, backups and credentials cannot be reviewed page evidence.';
    case 'symlink-path':
      return 'Replace the symlink with a real contained file; managed inputs must not resolve through symlinks.';
    default:
      return 'Correct docs-manifest.json to the reviewed C1 schema (schemaVersion 1); rerun this command for the exact page and field.';
  }
}

function locateManifestLine(text, errorPath) {
  if (typeof errorPath !== 'string' || errorPath.length === 0) return 1;
  const lines = text.split(/\r?\n/);
  const pageMatch = /^pages\[(\d+)\]/.exec(errorPath);
  let start = 0;
  let end = lines.length;
  if (pageMatch) {
    const pageIndex = Number(pageMatch[1]);
    const idLines = [];
    lines.forEach((line, index) => {
      if (/^\s*"id"\s*:/.test(line)) idLines.push(index);
    });
    if (idLines.length > 0) {
      start = idLines[Math.min(pageIndex, idLines.length - 1)];
      end = idLines[pageIndex + 1] ?? lines.length;
    }
  }
  const indexMatches = [...errorPath.matchAll(/\[(\d+)\]/g)];
  const requestedIndex = indexMatches.length > 0 ? Number(indexMatches[indexMatches.length - 1][1]) : 0;
  const fieldMatch = /([A-Za-z0-9_]+)$/.exec(errorPath);
  if (fieldMatch) {
    const pattern = new RegExp(`^\\s*"${fieldMatch[1]}"\\s*:`);
    const matches = [];
    for (let index = start; index < end; index += 1) {
      if (pattern.test(lines[index])) matches.push(index);
    }
    if (matches.length > 0) {
      return (matches[Math.min(requestedIndex, matches.length - 1)] ?? matches[0]) + 1;
    }
    for (let index = 0; index < lines.length; index += 1) {
      if (pattern.test(lines[index])) return index + 1;
    }
  }
  const literal = `"${errorPath}"`;
  for (let index = 0; index < lines.length; index += 1) {
    if (lines[index].includes(literal)) return index + 1;
  }
  if (pageMatch) return start + 1;
  return 1;
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function readPageIndex(manifestText, pageId) {
  const lines = manifestText.split(/\r?\n/);
  const pattern = new RegExp(`^\\s*"id"\\s*:\\s*"${escapeRegExp(pageId)}"\\s*,?\\s*$`);
  let index = -1;
  for (const line of lines) {
    if (/^\s*"id"\s*:/.test(line)) index += 1;
    if (pattern.test(line)) return index;
  }
  return -1;
}

function manifestLineFor(manifestText, pageId, field) {
  if (typeof manifestText !== 'string') {
    return locateManifestLine('', field);
  }
  const pageIndex = readPageIndex(manifestText, pageId);
  const fakePath = pageIndex >= 0 ? `pages[${pageIndex}].${field}` : field;
  return locateManifestLine(manifestText, fakePath);
}

function readManifest(report, repoRoot, manifestPath) {
  let text;
  try {
    text = readFileSync(manifestPath, 'utf8');
  } catch (error) {
    report.error(
      'manifest/unreadable',
      displayManifestPath(repoRoot, manifestPath),
      1,
      `Cannot read the documentation manifest: ${error.code ?? error.message}`,
      'Restore website/docs-manifest.json or pass --manifest with the reviewed registry path.',
    );
    return null;
  }
  try {
    const manifest = loadManifest(manifestPath, { repoRoot, verifySources: false });
    return { manifest, text };
  } catch (error) {
    if (!(error instanceof ManifestError)) throw error;
    report.error(
      `manifest/${error.code}`,
      displayManifestPath(repoRoot, manifestPath),
      locateManifestLine(text, error.path),
      `${error.path ? `${error.path}: ` : ''}${error.message}`,
      manifestFixFor(error.code),
      { manifestPath: error.path ?? '' },
    );
    return null;
  }
}

/* ------------------------------------------------------------------------- */
/* Markdown line view: frontmatter, fences, inline code and comments         */
/* ------------------------------------------------------------------------- */

function maskInlineCode(line, spans = null) {
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
    if (spans !== null) spans.push({ start: index, end: closing + runLength });
    index = closing + runLength;
  }
  return characters.join('');
}

function analyseMarkdown(text) {
  const lines = text.replace(/^\uFEFF/, '').split(/\r?\n/);
  const masked = lines.slice();
  const kinds = new Array(lines.length).fill('text');
  const fences = [];
  const inlineCode = [];
  let fence = null;
  let inComment = false;

  let frontmatterEnd = -1;
  if (lines.length > 0 && lines[0].trim() === '---') {
    for (let index = 1; index < lines.length; index += 1) {
      const trimmed = lines[index].trim();
      if (trimmed === '---' || trimmed === '...') {
        frontmatterEnd = index;
        break;
      }
    }
    if (frontmatterEnd !== -1) {
      for (let index = 0; index <= frontmatterEnd; index += 1) {
        kinds[index] = 'frontmatter';
        masked[index] = '';
      }
    }
  }

  for (let index = frontmatterEnd + 1; index < lines.length; index += 1) {
    const line = lines[index];
    if (fence !== null) {
      kinds[index] = 'fence';
      masked[index] = '';
      const closing = FENCE_CLOSE.exec(line);
      if (closing && closing[1][0] === fence.character && closing[1].length >= fence.length) fence = null;
      continue;
    }
    const opening = FENCE_OPEN.exec(line);
    if (opening && !(opening[1][0] === '`' && opening[2].includes('`'))) {
      fence = { character: opening[1][0], length: opening[1].length };
      fences.push({ line: index + 1, marker: opening[1], info: opening[2].trim() });
      kinds[index] = 'fence';
      masked[index] = '';
      continue;
    }
    const spans = [];
    inlineCode[index] = spans;
    let view = maskInlineCode(line, spans);
    if (inComment) {
      const end = view.indexOf('-->');
      if (end === -1) {
        kinds[index] = 'comment';
        masked[index] = '';
        continue;
      }
      view = `${' '.repeat(end + 3)}${view.slice(end + 3)}`;
      inComment = false;
    }
    let cursor = 0;
    while (cursor < view.length) {
      const start = view.indexOf('<!--', cursor);
      if (start === -1) break;
      const end = view.indexOf('-->', start + 4);
      if (end === -1) {
        view = `${view.slice(0, start)}${' '.repeat(view.length - start)}`;
        kinds[index] = 'comment';
        inComment = true;
        break;
      }
      view = `${view.slice(0, start)}${' '.repeat(end + 3 - start)}${view.slice(end + 3)}`;
      cursor = end + 3;
    }
    masked[index] = view;
  }

  return { lines, masked, kinds, fences, inlineCode };
}

function frontmatterAndBody(text) {
  const lines = text.replace(/^\uFEFF/, '').split(/\r?\n/);
  let start = 0;
  if (lines.length > 0 && lines[0].trim() === '---') {
    for (let index = 1; index < lines.length; index += 1) {
      const trimmed = lines[index].trim();
      if (trimmed === '---' || trimmed === '...') {
        start = index + 1;
        break;
      }
    }
  }
  return lines.slice(start).join('\n');
}

function normalizedBody(text) {
  return frontmatterAndBody(text)
    .replace(/<!--\s*generated:[^\s:]+:start\s*-->[\s\S]*?<!--\s*generated:[^\s:]+:end\s*-->/g, '')
    .replace(/\s+/gu, ' ')
    .trim();
}

/**
 * Top-level front-matter keys that claim a human review or approval. Only the
 * outer block is scanned and indented child keys are ignored, so prose such as
 * a nested `reviewedBy` value cannot masquerade as a top-level claim.
 */
function frontmatterClaimKeys(text) {
  const lines = text.replace(/^\uFEFF/, '').split(/\r?\n/);
  if (lines.length === 0 || lines[0].trim() !== '---') return [];
  const claims = [];
  for (let index = 1; index < lines.length; index += 1) {
    const trimmed = lines[index].trim();
    if (trimmed === '---' || trimmed === '...') break;
    const match = /^['"]?([A-Za-z0-9_-]+)['"]?\s*:/.exec(lines[index]);
    if (match !== null && FORGED_CLAIM_KEY.test(match[1])) claims.push({ key: match[1], line: index + 1 });
  }
  return claims;
}

function checkFrontmatterClaims(report, entryPath, text, label) {
  for (const claim of frontmatterClaimKeys(text)) {
    report.error(
      'frontmatter/forged-claim',
      entryPath,
      claim.line,
      `${label} front matter declares "${claim.key}", which claims a human review or approval this repository never recorded.`,
      `Remove "${claim.key}" from the front matter. Review status lives only in docs-manifest.json and the feature approval record, never in page or stub content.`,
      { key: claim.key },
    );
  }
}

/** Raw HTML `id` values in prose, used as fragment targets beside headings. */
function htmlAnchorIds(view) {
  const ids = [];
  for (const line of view.masked) {
    if (!line.includes('<')) continue;
    for (const match of line.matchAll(HTML_ID_ATTRIBUTE)) {
      const value = decodeHtmlEntities(match[1] ?? match[2] ?? match[3] ?? '');
      if (value.length > 0) ids.push(value);
    }
  }
  return ids;
}

/**
 * Read one Markdown source through the reviewed baseline parser. The returned
 * `slugs` set matches the build plugin's slug rules plus raw HTML ids, and
 * `duplicates` make the transform's duplicate-heading refusal explicit in
 * source mode.
 */
function parseMarkdownSource(repoRoot, relative) {
  const parsed = collectBaseline(repoRoot, [relative]);
  const file = parsed.files[0];
  const text = readContainedText(repoRoot, relative, MAX_SOURCE_BYTES, relative);
  const info = headingInfo(file.headings);
  const view = analyseMarkdown(text);
  const slugs = new Set(info.slugs);
  for (const heading of file.headings) slugs.add(heading.slug);
  for (const id of htmlAnchorIds(view)) slugs.add(id);
  return {
    path: relative,
    text,
    headings: file.headings,
    links: file.links,
    slugs,
    duplicateHeadings: info.duplicates,
    view,
    body: normalizedBody(text),
  };
}

/* ------------------------------------------------------------------------- */
/* Ownership: URL, source, body and alias uniqueness                         */
/* ------------------------------------------------------------------------- */

function canonicalUrl(manifest, id) {
  const base = manifest.base === '/' ? '' : manifest.base;
  const route = id === 'index' ? '/' : `/${id}/`;
  return `${manifest.site}${base}${route}`;
}

function canonicalRoute(manifest, id) {
  const base = manifest.base === '/' ? '' : manifest.base;
  return `${base}${id === 'index' ? '/' : `/${id}/`}`;
}

function aliasFrom(entry) {
  if (typeof entry === 'string' && entry.length > 0) return entry;
  if (entry !== null && typeof entry === 'object' && !Array.isArray(entry)) {
    for (const key of ['from', 'oldPath', 'url']) {
      if (typeof entry[key] === 'string' && entry[key].length > 0) return entry[key];
    }
  }
  return null;
}

function checkOwnership(report, manifest, manifestPath, manifestText, sources, bodies) {
  const urls = new Map();
  const sourceOwners = new Map();
  for (const page of manifest.pages) {
    const url = canonicalUrl(manifest, page.id);
    if (urls.has(page.id) || [...urls.values()].includes(url)) {
      report.error(
        'ownership/duplicate-url',
        manifestPath,
        manifestLineFor(manifestText, page.id, 'id'),
        `Page "${page.id}" would produce the canonical URL ${url} more than once.`,
        'Keep one page per canonical URL; remove the duplicate id or redirect through an exact alias.',
      );
    }
    urls.set(page.id, url);
    if (sourceOwners.has(page.source)) {
      report.error(
        'ownership/duplicate-source',
        manifestPath,
        manifestLineFor(manifestText, page.id, 'source'),
        `Source "${page.source}" is already owned by page "${sourceOwners.get(page.source)}".`,
        'Keep exactly one canonical source per public document; move the second body or remove the duplicate page.',
      );
    } else {
      sourceOwners.set(page.source, page.id);
    }
  }

  const budgets = new Set(manifest.pages.map((page) => page.source));
  for (const stub of sources.stubs ?? []) {
    if (budgets.has(stub.path)) {
      report.error(
        'ownership/stub-shadows-source',
        stub.path,
        1,
        `Declared stub "${stub.path}" is also a canonical manifest page source.`,
        'A path is either a canonical page or a compatibility stub, never both; remove one declaration.',
      );
    }
  }

  const bodyOwners = new Map();
  for (const entry of bodies) {
    if (entry.body.length === 0) continue;
    const owner = bodyOwners.get(entry.body);
    if (owner !== undefined) {
      report.error(
        'ownership/duplicate-body',
        entry.path,
        1,
        `Body content duplicates "${owner}"; one public document body has a single owner.`,
        'Keep one canonical body and link the other path to it instead of copying prose.',
      );
    } else {
      bodyOwners.set(entry.body, entry.path);
    }
  }

  const aliases = Array.isArray(manifest.aliases) ? manifest.aliases : [];
  const aliasOwners = new Map();
  for (const entry of aliases) {
    // The alias object schema belongs to C1/T08. Only an exactly repeated
    // entry is unambiguous duplicate ownership here; destination/fragment
    // semantics are not guessed by this checker.
    const aliasKey = typeof entry === 'string' ? `string:${entry}` : `object:${JSON.stringify(entry)}`;
    if (aliasOwners.has(aliasKey)) {
      report.error(
        'ownership/duplicate-alias',
        manifestPath,
        locateManifestLine(manifestText, 'aliases'),
        'The same legacy alias entry is declared more than once.',
        'Give every migrated old path exactly one destination alias; remove the repeated entry.',
      );
    } else {
      aliasOwners.set(aliasKey, true);
    }
    const from = aliasFrom(entry);
    if (from === null) continue;
    if (sourceOwners.has(from)) {
      report.error(
        'ownership/alias-shadows-source',
        manifestPath,
        locateManifestLine(manifestText, 'aliases'),
        `Legacy alias "${from}" collides with canonical page source "${from}".`,
        'Rename or remove the alias; an old path cannot also be a live canonical source.',
      );
    }
    if ([...urls.values()].includes(from)) {
      report.error(
        'ownership/alias-shadows-url',
        manifestPath,
        locateManifestLine(manifestText, 'aliases'),
        `Legacy alias "${from}" collides with a current canonical URL.`,
        'Keep the current URL unique and point the alias at a different (old) path.',
      );
    }
  }
}

/* ------------------------------------------------------------------------- */
/* Generated-section marker ownership                                        */
/* ------------------------------------------------------------------------- */

function checkGeneratedMarkers(report, page, sourceText, view, manifestPath, manifestText) {
  const sections = Array.isArray(page.generatedSections) ? page.generatedSections : [];
  const registered = new Map();
  for (const section of sections) {
    const id = typeof section.id === 'string' ? section.id : '';
    const expectedStart = `<!-- generated:${id}:start -->`;
    const expectedEnd = `<!-- generated:${id}:end -->`;
    if (!/^[a-z0-9][a-z0-9/-]*[a-z0-9]$|^[a-z0-9]$/.test(id)) {
      report.error(
        'generated/invalid-id',
        manifestPath,
        manifestLineFor(manifestText, page.id, 'generatedSections'),
        `Generated section id ${JSON.stringify(section.id)} is not a lowercase route-shaped identifier.`,
        'Use the canonical page id (for example "reference/cli") as the generated section id.',
      );
      continue;
    }
    registered.set(id, section);
    if (section.markerStart !== undefined && section.markerStart !== expectedStart) {
      report.error(
        'generated/marker-registration',
        manifestPath,
        manifestLineFor(manifestText, page.id, 'generatedSections'),
        `markerStart for "${id}" must be exactly ${expectedStart}.`,
        'Register the exact generated markers used by the producer; do not hand-edit a variant.',
      );
    }
    if (section.markerEnd !== undefined && section.markerEnd !== expectedEnd) {
      report.error(
        'generated/marker-registration',
        manifestPath,
        manifestLineFor(manifestText, page.id, 'generatedSections'),
        `markerEnd for "${id}" must be exactly ${expectedEnd}.`,
        'Register the exact generated markers used by the producer; do not hand-edit a variant.',
      );
    }

    const startIndexes = occurrencesOutsideCode(sourceText, expectedStart, view);
    const endIndexes = occurrencesOutsideCode(sourceText, expectedEnd, view);
    if (startIndexes.length === 0) {
      report.error(
        'generated/marker-missing',
        page.source,
        1,
        `Generated section "${id}" is registered but its start marker is absent.`,
        `Add ${expectedStart} and ${expectedEnd} around the generated block, or remove the generatedSections entry after review.`,
      );
    } else if (startIndexes.length > 1) {
      report.error(
        'generated/marker-duplicate',
        page.source,
        lineAt(sourceText, startIndexes[1]),
        `Generated start marker for "${id}" appears ${startIndexes.length} times.`,
        'Keep exactly one generated block per registered id; remove the duplicate marker pair.',
      );
    }
    if (endIndexes.length === 0) {
      report.error(
        'generated/marker-missing',
        page.source,
        1,
        `Generated section "${id}" is registered but its end marker is absent.`,
        `Add ${expectedStart} and ${expectedEnd} around the generated block, or remove the generatedSections entry after review.`,
      );
    } else if (endIndexes.length > 1) {
      report.error(
        'generated/marker-duplicate',
        page.source,
        lineAt(sourceText, endIndexes[1]),
        `Generated end marker for "${id}" appears ${endIndexes.length} times.`,
        'Keep exactly one generated block per registered id; remove the duplicate marker pair.',
      );
    }
    if (startIndexes.length === 1 && endIndexes.length === 1 && endIndexes[0] < startIndexes[0]) {
      report.error(
        'generated/marker-order',
        page.source,
        lineAt(sourceText, startIndexes[0]),
        `Generated start marker for "${id}" appears after its end marker.`,
        'Place the start marker before the end marker and regenerate the section.',
      );
    }

    const declaredInputs = new Set(page.sources.map((source) => source.path));
    for (const input of Array.isArray(section.inputs) ? section.inputs : []) {
      if (typeof input !== 'string' || !declaredInputs.has(input)) {
        report.error(
          'generated/input-unregistered',
          page.source,
          1,
          `Generated input ${JSON.stringify(input)} is not in this page's reviewed sources.`,
          'Add the producer input to the page sources with its reviewed sha256, or remove it from the generated section inputs.',
        );
      }
    }
  }

  for (const match of sourceText.matchAll(MARKER_PATTERN)) {
    const line = lineAt(sourceText, match.index);
    const lineKind = view.kinds[line - 1] ?? 'text';
    if (lineKind === 'fence' || lineKind === 'frontmatter') continue;
    if (insideInlineCode(view, sourceText, match.index)) continue;
    const id = match[1];
    const kind = match[2];
    if (!registered.has(id)) {
      report.error(
        'generated/marker-unregistered',
        page.source,
        lineAt(sourceText, match.index),
        `Generated ${kind} marker for "${id}" has no matching generatedSections entry.`,
        'Register the section producer/inputs in docs-manifest.json, or remove the hand-written generated block.',
      );
    } else {
      const expected = kind === 'start' ? `<!-- generated:${id}:start -->` : `<!-- generated:${id}:end -->`;
      if (match[0] !== expected) {
        report.error(
          'generated/marker-variant',
          page.source,
          lineAt(sourceText, match.index),
          `Generated marker "${match[0]}" is not the canonical form ${expected}.`,
          'Use the exact marker emitted by the reviewed producer; whitespace variants break the owned block.',
        );
      }
    }
  }
}

function insideInlineCode(view, sourceText, index) {
  const line = lineAt(sourceText, index) - 1;
  const lineStart = sourceText.lastIndexOf('\n', index - 1) + 1;
  const column = index - lineStart;
  return (view.inlineCode[line] ?? []).some((span) => column >= span.start && column < span.end);
}

function occurrencesOutsideCode(haystack, needle, view) {
  return occurrences(haystack, needle).filter((index) => {
    const kind = view.kinds[lineAt(haystack, index) - 1] ?? 'text';
    return kind !== 'fence' && kind !== 'frontmatter' && !insideInlineCode(view, haystack, index);
  });
}

function occurrences(haystack, needle) {
  const indexes = [];
  let from = 0;
  while (from <= haystack.length) {
    const index = haystack.indexOf(needle, from);
    if (index === -1) break;
    indexes.push(index);
    from = index + needle.length;
  }
  return indexes;
}

/* ------------------------------------------------------------------------- */
/* Source-mode links, fragments and external reporting                       */
/* ------------------------------------------------------------------------- */

function routeFixFor(code, base) {
  const baseLabel = base === '' || base === '/' ? '/' : base;
  switch (code) {
    case 'unknown-target':
      return 'Point the link at a registered manifest page source, or add the page to docs-manifest.json after review.';
    case 'path-escape':
      return `Use a repository-relative ".md" path or fragment; do not hard-code the project base "${baseLabel}".`;
    case 'unsafe-scheme':
      return 'Use https:, mailto: or a repository-relative Markdown link; other schemes are refused by the build transform.';
    case 'invalid-encoding':
      return 'Percent-encode the path or fragment correctly (for example %20 for a space, %23 for a literal #).';
    case 'unsupported-relative-link':
      return 'Link a registered ".md" page, use an image reference, or resolve the engineering file through the repository source URL.';
    case 'symlink-path':
      return 'Replace the symlink with a regular contained file before linking it as documentation evidence.';
    case 'hardlink-target':
      return 'Replace the hard-linked target with a single-named regular file before linking it as documentation evidence.';
    case 'missing-source-context':
      return 'Keep the link inside a page whose source path is known to the manifest.';
    default:
      return 'Use a relative target under website/src/content/docs or a supported https:/mailto: URL.';
  }
}

function fragmentFailure(report, entryPath, line, href, fragment, targetSlugs, targetPath) {
  const decoded = decodeUriComponent(fragment);
  if (!decoded.ok) {
    report.error(
      'links/invalid-encoding',
      entryPath,
      line,
      `Fragment in "${href}" is not valid percent-encoding.`,
      'Percent-encode the fragment once (for example %20 for a space); do not double-encode it.',
      { href },
    );
    return;
  }
  if (decoded.value === '') return;
  if (targetSlugs.has(decoded.value)) return;
  const suggestions = formatSuggestionList(targetSlugs);
  report.error(
    'links/broken-anchor',
    entryPath,
    line,
    `Anchor "#${decoded.value}" is not a heading in "${targetPath}".`,
    suggestions.length > 0
      ? `Link an existing heading (${suggestions}) or add that heading to the target page.`
      : 'Add the referenced heading to the target page, or remove the fragment.',
    { href, fragment: decoded.value, target: targetPath },
  );
}

function decodeRelativePath(report, entryPath, line, href, pathPart) {
  const decoded = decodeUriComponent(pathPart);
  if (!decoded.ok) {
    report.error(
      'links/invalid-encoding',
      entryPath,
      line,
      `Link path in "${href}" is not valid percent-encoding.`,
      'Percent-encode the path once (for example %20 for a space); do not double-encode it.',
      { href },
    );
    return null;
  }
  if (/[\u0000-\u001f\u007f]/.test(decoded.value) || decoded.value.includes('\\')) {
    report.error(
      'links/unsafe-path',
      entryPath,
      line,
      `Link path in "${href}" contains unsafe characters.`,
      'Use a normal relative POSIX path under the repository.',
      { href },
    );
    return null;
  }
  return decoded.value;
}

function rawNavigationTags(view) {
  const found = [];
  const text = view.masked.join('\n');
  let offset = 0;
  for (let index = 0; index < view.masked.length; index += 1) {
    const line = view.masked[index];
    for (const start of line.matchAll(RAW_HTML_LINK_START)) {
      const absolute = offset + start.index;
      const closing = text.indexOf('>', absolute);
      const tag = closing === -1 ? text.slice(absolute, absolute + 4096) : text.slice(absolute, closing + 1);
      const href = RAW_HTML_HREF.exec(tag);
      if (href !== null) {
        found.push({ line: index + 1, href: href[1] ?? href[2] ?? href[3] ?? null });
      } else if (closing === -1) {
        // An opening tag that never closes is not a retained anchor, so it
        // stays subject to the navigation rule rather than slipping through.
        found.push({ line: index + 1, href: null });
      }
    }
    offset += line.length + 1;
  }
  return found;
}

function checkRawHtmlNavigation(report, page, view) {
  for (const navigation of rawNavigationTags(view)) {
    report.error(
      'html/raw-navigation',
      page.source,
      navigation.line,
      'Raw HTML navigation is not rewritten by the build transform.',
      'Use a Markdown link (or a reference definition) so the target resolves, the base is applied and the anchor is checked.',
      { href: navigation.href },
    );
  }
}

function checkSourceLinks({ report, page, parsed, manifest, repoRoot, pagesBySource, cache }) {
  const base = manifest.base === '/' ? '' : manifest.base;
  const external = [];
  parsed.view.kinds.forEach((kind, index) => {
    if (kind !== 'text') return;
    for (const match of parsed.view.masked[index].matchAll(/<(https?:\/\/[^>\s]+|mailto:[^>\s]+)>/gi)) {
      report.metrics.links += 1;
      external.push({ line: index + 1, href: match[1] });
    }
  });
  const links = [...parsed.links].sort(
    (left, right) => left.line - right.line || compareStrings(left.href, right.href),
  );

  for (const link of links) {
    report.metrics.links += 1;
    const href = link.href;
    if (typeof href !== 'string' || href.length === 0) {
      report.error(
        'links/empty-target',
        page.source,
        link.line,
        'Markdown link has an empty destination.',
        'Remove the link or point it at a registered page, heading or URL.',
      );
      continue;
    }
    if (href.startsWith('#')) {
      fragmentFailure(report, page.source, link.line, href, href.slice(1), parsed.slugs, page.source);
      continue;
    }
    const split = splitHref(href);
    if (split.pathPart === '') {
      if (split.fragment !== null && split.fragment !== '') {
        fragmentFailure(report, page.source, link.line, href, split.fragment, parsed.slugs, page.source);
        continue;
      }
      report.error(
        'links/empty-target',
        page.source,
        link.line,
        `Link "${href}" has no path or heading target.`,
        'Remove the empty link or point it at a registered page, heading or URL.',
        { href },
      );
      continue;
    }
    if (EXTERNAL_SCHEME.test(split.pathPart)) {
      external.push({ line: link.line, href });
      continue;
    }
    if (ANY_SCHEME.test(split.pathPart)) {
      report.error(
        'links/unsafe-scheme',
        page.source,
        link.line,
        `Unsafe link scheme in "${href}".`,
        routeFixFor('unsafe-scheme', base),
        { href },
      );
      continue;
    }
    if (split.pathPart.startsWith('//')) {
      report.error(
        'links/unsafe-scheme',
        page.source,
        link.line,
        `Protocol-relative link "${href}" is not allowed.`,
        routeFixFor('unsafe-scheme', base),
        { href },
      );
      continue;
    }
    if (split.pathPart.startsWith('/')) {
      const normalizedBase = base === '' ? '/' : base;
      report.error(
        'links/path-escape',
        page.source,
        link.line,
        `Root-absolute link "${split.pathPart}" bypasses the project base and cannot be resolved from source.`,
        split.pathPart.startsWith(`${normalizedBase}/`) || split.pathPart === normalizedBase
          ? `Use a relative ".md" link; the transform adds the project base "${normalizedBase}" exactly once.`
          : routeFixFor('path-escape', base),
        { href },
      );
      continue;
    }

    let resolved;
    try {
      resolved = resolveLink({
        manifest,
        repoRoot,
        sourcePath: page.source,
        href,
        target: 'html',
        ref: 'main',
      });
    } catch (error) {
      if (!(error instanceof RouteError)) throw error;
      report.error(
        `links/${error.code}`,
        page.source,
        link.line,
        `${error.message} (${href})`,
        routeFixFor(error.code, base),
        { href },
      );
      continue;
    }

    const decodedPath = decodeRelativePath(report, page.source, link.line, href, split.pathPart);
    if (decodedPath === null) continue;
    const targetRelative = path.posix.normalize(
      path.posix.join(path.posix.dirname(page.source), decodedPath),
    );
    if (
      targetRelative === '.' ||
      targetRelative === '..' ||
      targetRelative.startsWith('../') ||
      targetRelative.startsWith('/')
    ) {
      report.error(
        'links/path-escape',
        page.source,
        link.line,
        `Link "${href}" escapes the repository.`,
        routeFixFor('path-escape', base),
        { href },
      );
      continue;
    }

    const classification = classifyLinkTarget(targetRelative, { source: page.source });
    if (!classification.allowed) {
      report.error(
        'links/private-target',
        page.source,
        link.line,
        `Link "${href}" targets "${targetRelative}", which is in a private or history tree and must not become a public source URL.`,
        'Link a registered public page under website/src/content/docs, or state the fact in the page instead of pointing at private/history material. ' +
          'The retained docs/mcp-tools.md historical-spec pointer is the only documented exception.',
        { href, target: targetRelative, category: classification.category },
      );
      continue;
    }

    const targetPage = pagesBySource.get(targetRelative);
    if (targetPage !== undefined) {
      if (base !== '' && !resolved.startsWith(`${base}/`)) {
        report.error(
          'links/base-prefix',
          page.source,
          link.line,
          `Resolved link "${resolved}" is missing the project base "${base}".`,
          `Let the transform add the base; do not hard-code or strip "${base}".`,
          { href },
        );
      }
      if (base !== '' && resolved.startsWith(`${base}${base}/`)) {
        report.error(
          'links/double-base',
          page.source,
          link.line,
          `Resolved link "${resolved}" applies the project base twice.`,
          'Use a relative ".md" link and let the transform add the base exactly once.',
          { href },
        );
      }
      if (split.fragment !== null && split.fragment !== '') {
        try {
          const target = loadTargetParsed(cache, repoRoot, targetRelative);
          fragmentFailure(report, page.source, link.line, href, split.fragment, target.slugs, targetRelative);
        } catch (error) {
          report.error(
            'links/fragment-unverified',
            page.source,
            link.line,
            `Anchor in "${href}" could not be verified for the existing target "${targetRelative}": ${error.message}`,
            'Reduce the target below the parse cap or link an existing heading in a reviewed public page; an existing target with an unverifiable anchor fails closed.',
            { href, target: targetRelative },
          );
        }
      }
      continue;
    }

    const targetInspection = inspectContainedTarget(repoRoot, targetRelative);
    if (targetInspection.kind === 'missing') {
      report.error(
        'links/missing-file',
        page.source,
        link.line,
        `Link target "${targetRelative}" does not exist in the repository.`,
        'Create the referenced file, or change the link to an existing contained target.',
        { href, target: targetRelative },
      );
      continue;
    }
    if (targetInspection.kind === 'symlink') {
      report.error(
        'links/symlink-target',
        page.source,
        link.line,
        `Link target "${targetRelative}" resolves through a symlink.`,
        'Use a regular contained file or directory; managed documentation links must not traverse symlinks.',
        { href, target: targetRelative },
      );
      continue;
    }
    if (targetInspection.kind === 'hardlink') {
      report.error(
        'links/hardlink-target',
        page.source,
        link.line,
        `Link target "${targetRelative}" is a hard link with more than one name.`,
        'Keep one name per contained file; deploy artifacts must not contain symlinks or hardlinks.',
        { href, target: targetRelative },
      );
      continue;
    }
    if (targetInspection.kind === 'special' || targetInspection.kind === 'unreadable') {
      report.error(
        'links/unreadable-target',
        page.source,
        link.line,
        `Link target "${targetRelative}" is not a readable regular file or directory.`,
        'Replace it with a regular file or directory, or remove the link.',
        { href, target: targetRelative },
      );
      continue;
    }

    if (targetRelative.endsWith('.md') && (split.fragment !== null && split.fragment !== '')) {
      try {
        const target = loadTargetParsed(cache, repoRoot, targetRelative);
        fragmentFailure(report, page.source, link.line, href, split.fragment, target.slugs, targetRelative);
      } catch (error) {
        report.error(
          'links/fragment-unverified',
          page.source,
          link.line,
          `Anchor in "${href}" could not be verified for the existing target "${targetRelative}": ${error.message}`,
          'Reduce the target below the parse cap or move the anchor to a reviewed public page; an existing target with an unverifiable anchor fails closed.',
          { href, target: targetRelative },
        );
      }
    }
  }

  if (external.length > 0) {
    external.sort((left, right) => left.line - right.line || compareStrings(left.href, right.href));
    report.metrics.external += external.length;
    report.observation(
      'links/external',
      page.source,
      external[0].line,
      `${external.length} external link(s) were not fetched (${external.map((entry) => entry.href).sort(compareStrings).join(', ')}).`,
      'External availability is not verified offline; keep the URL and check it manually when connectivity matters.',
      { hrefs: external.map((entry) => entry.href).sort(compareStrings) },
    );
  }
}

function loadTargetParsed(cache, repoRoot, relative) {
  if (cache.has(relative)) {
    const cached = cache.get(relative);
    if (cached === null) throw new Error(`"${relative}" could not be parsed earlier`);
    return cached;
  }
  const parsed = parseMarkdownSource(repoRoot, relative);
  cache.set(relative, parsed);
  return parsed;
}

/* ------------------------------------------------------------------------- */
/* Stub shape and canonical-link ownership                                   */
/* ------------------------------------------------------------------------- */

// Accepted stub declarations: `{path, canonicalId, anchors?}` or the exact
// CLI form `"path=page-id[:anchor,...]"`. Declarations are explicit; the
// checker never guesses which files are stubs.
function parseStubSpec(value, index) {
  if (typeof value === 'string') {
    const separator = value.indexOf('=');
    if (separator <= 0 || separator === value.length - 1) {
      throw new TypeError(`stubs[${index}] must be "path=page-id[:anchor,...]"`);
    }
    const stubPath = value.slice(0, separator);
    const rest = value.slice(separator + 1);
    const anchorSeparator = rest.lastIndexOf(':');
    if (anchorSeparator === -1) return { path: stubPath, canonicalId: rest };
    return {
      path: stubPath,
      canonicalId: rest.slice(0, anchorSeparator),
      anchors: rest.slice(anchorSeparator + 1).split(',').filter((anchor) => anchor.length > 0),
    };
  }
  if (value === null || typeof value !== 'object' || Array.isArray(value)) {
    throw new TypeError(`stubs[${index}] must be an object or "path=page-id" string`);
  }
  for (const key of Object.keys(value)) {
    if (!['path', 'canonicalId', 'anchors'].includes(key)) {
      throw new TypeError(`Unknown stubs[${index}] field "${key}"`);
    }
  }
  if (!Array.isArray(value.anchors ?? [])) throw new TypeError(`stubs[${index}].anchors must be an array`);
  for (const anchor of value.anchors ?? []) {
    if (typeof anchor !== 'string' || anchor.length === 0) {
      throw new TypeError(`stubs[${index}].anchors entries must be non-empty strings`);
    }
  }
  return {
    path: value.path,
    canonicalId: value.canonicalId,
    anchors: value.anchors ?? [],
  };
}

function checkStubs({ report, stubSpecs, manifest, repoRoot, bodies }) {
  const pagesById = new Map(manifest.pages.map((page) => [page.id, page]));
  const parsedStubs = [];
  for (const spec of stubSpecs) {
    let stubPath;
    try {
      stubPath = normalizeRelative(spec.path, 'stub path');
    } catch (error) {
      report.error('stub/invalid-path', typeof spec.path === 'string' ? spec.path : '<stub>', 1, error.message, 'Declare an exact repository-relative stub path.');
      continue;
    }
    report.metrics.stubs += 1;
    let parsed;
    try {
      parsed = parseMarkdownSource(repoRoot, stubPath);
    } catch (error) {
      report.error(
        'stub/unreadable',
        stubPath,
        1,
        `Cannot inspect declared stub: ${error.message}`,
        'Restore the compatibility page as a contained Markdown file, or remove the --stub declaration.',
      );
      continue;
    }
    parsedStubs.push({ spec, stubPath, parsed });
    bodies.push({ path: stubPath, body: parsed.body });
    checkFrontmatterClaims(report, stubPath, parsed.text, 'Compatibility stub');
    for (const duplicate of parsed.duplicateHeadings) {
      report.error(
        'stub/duplicate-heading',
        stubPath,
        duplicate.line,
        `Stub heading "${duplicate.text}" produces duplicate slug "#${duplicate.slug}".`,
        'Rename one retained heading so every stub anchor is unambiguous.',
        { slug: duplicate.slug },
      );
    }
  }

  for (const { spec, stubPath, parsed } of parsedStubs) {
    const target = pagesById.get(spec.canonicalId);
    if (target === undefined) {
      report.error(
        'stub/unknown-canonical',
        stubPath,
        1,
        `Declared canonical page "${spec.canonicalId}" is not registered in docs-manifest.json.`,
        'Use an existing manifest page id, or author the destination before declaring the stub.',
        { canonicalId: spec.canonicalId },
      );
    }

    for (const fence of parsed.view.fences) {
      report.error(
        'stub/code-fence',
        stubPath,
        fence.line,
        `Compatibility stub contains a code fence (${JSON.stringify(fence.info || fence.marker)}).`,
        'Remove the code fence; a stub contains only title/status, canonical links and retained anchors, never procedures or copied commands.',
      );
    }
    parsed.view.kinds.forEach((kind, index) => {
      if (kind !== 'text') return;
      if (ORDERED_STEP.test(parsed.view.masked[index])) {
        report.error(
          'stub/procedure',
          stubPath,
          index + 1,
          'Compatibility stub contains a numbered procedure step.',
          'Replace the step with a direct link to the canonical guide; procedures live in the canonical page, not in a stub.',
        );
      }
    });
    for (const navigation of rawNavigationTags(parsed.view)) {
      report.error(
        'stub/raw-html-navigation',
        stubPath,
        navigation.line,
        'Compatibility stub uses raw HTML navigation, which the build transform does not rewrite.',
        'Use a Markdown link so the canonical destination is validated and base-prefixed.',
        { href: navigation.href },
      );
    }

    if (target !== undefined) {
      const expectedSource = target.source;
      const expectedRoute = canonicalRoute(manifest, target.id);
      const expectedUrl = canonicalUrl(manifest, target.id);
      let canonical = null;
      for (const link of parsed.links) {
        const split = splitHref(link.href);
        if (split === null || split.pathPart === '' || EXTERNAL_SCHEME.test(split.pathPart)) {
          if (split !== null && EXTERNAL_SCHEME.test(split.pathPart)) {
            if (split.pathPart === expectedUrl) {
              canonical = link;
              break;
            }
            const repositoryPrefix = `${manifest.repository}/blob/`;
            if (split.pathPart.startsWith(repositoryPrefix)) {
              const parts = split.pathPart.slice(repositoryPrefix.length).split('/');
              if (parts.length > 1 && parts.slice(1).join('/') === expectedSource) {
                canonical = link;
                break;
              }
            }
          }
          continue;
        }
        const decoded = decodeUriComponent(split.pathPart);
        if (!decoded.ok) continue;
        const joined = path.posix.normalize(
          path.posix.join(path.posix.dirname(stubPath), decoded.value),
        );
        if (joined === expectedSource) {
          canonical = link;
          break;
        }
        try {
          const resolved = resolveLink({
            manifest,
            repoRoot,
            sourcePath: stubPath,
            href: link.href,
            target: 'html',
            ref: 'main',
          });
          if (resolved === expectedRoute || resolved === expectedUrl) {
            canonical = link;
            break;
          }
        } catch {
          // A broken stub link is reported below only when it is the declared
          // canonical link; other stub links do not need to resolve for shape.
        }
      }
      if (canonical === null) {
        const relativeTarget = path.posix.relative(path.posix.dirname(stubPath), expectedSource);
        report.error(
          'stub/missing-canonical',
          stubPath,
          1,
          `Compatibility stub does not link its declared canonical page "${target.id}".`,
          `Add a Markdown link to "${relativeTarget}" (canonical source) or to "${expectedRoute}".`,
          { canonicalId: target.id },
        );
      }
    }

    for (const anchor of spec.anchors ?? []) {
      const normalized = anchor.replace(/^#/, '');
      if (!parsed.slugs.has(normalized)) {
        report.error(
          'stub/missing-anchor',
          stubPath,
          1,
          `Retained anchor "#${normalized}" is absent from the stub.`,
          `Restore a heading that produces "#${normalized}", or remove it from the retained-anchor declaration.`,
          { anchor: normalized },
        );
      }
    }
  }
  return parsedStubs;
}

/* ------------------------------------------------------------------------- */
/* Content-root coverage: every public page has exactly one manifest owner   */
/* ------------------------------------------------------------------------- */

function contentRootFixFor(code) {
  switch (code) {
    case 'missing-content-root':
    case 'unreadable-content-root':
      return `Restore the canonical public content root "${CONTENT_ROOT}" as a readable directory of page sources.`;
    case 'symlink-path':
      return 'Replace the symlink with a real contained page source; public pages are regular files under the content root.';
    case 'unsupported-source':
      return 'Author canonical public pages as ".md"; ".mdx" is not in the published page allowlist.';
    default:
      return 'Replace the special or non-regular entry with a regular ".md" page source.';
  }
}

/**
 * Enumerates the canonical public content root and fails every Markdown page the
 * manifest does not own. `export.mjs` already refuses such a page; without this
 * enumeration `docs:check` would pass content that can still build and publish.
 */
function checkContentRootCoverage(report, repoRoot, manifest) {
  let scanned;
  try {
    scanned = scanContentRoot(repoRoot);
  } catch (error) {
    if (!(error instanceof PublicTreeError)) throw error;
    report.error(
      `source/${error.code}`,
      error.path,
      1,
      error.message,
      contentRootFixFor(error.code),
    );
    return;
  }
  const registered = new Set(manifest.pages.map((page) => page.source));
  for (const source of scanned) {
    if (registered.has(source)) continue;
    report.error(
      'source/unregistered-page',
      source,
      1,
      `Public content page "${source}" is not registered in docs-manifest.json.`,
      'Register the page in docs-manifest.json with human review (id, type, audience, class and reviewed sources), or remove the file; an unregistered page is refused by export and must not publish.',
      { source },
    );
  }
}

/* ------------------------------------------------------------------------- */
/* Freshness: reviewed full-file fingerprints versus current bytes           */
/* ------------------------------------------------------------------------- */

function normalizeChangedPaths(value) {
  if (value === undefined) return null;
  if (!Array.isArray(value)) throw new TypeError('changedPaths must be an array of repository-relative paths');
  const normalized = [];
  for (const entry of value) {
    if (typeof entry !== 'string' || entry.trim().length === 0) {
      throw new TypeError('changedPaths entries must be non-empty strings');
    }
    const trimmed = entry.trim();
    if (trimmed.endsWith('/')) {
      normalized.push(normalizeRelative(trimmed.slice(0, -1), 'changedPaths entry') + '/');
    } else {
      normalized.push(normalizeRelative(trimmed, 'changedPaths entry'));
    }
  }
  return normalized.sort(compareStrings);
}

function pageTouchedBy(page, changedPaths, manifestRelative) {
  if (changedPaths === null) return true;
  if (changedPaths.some((changedPath) => pathMatchesChange(changedPath, manifestRelative))) return true;
  const owned = [page.source];
  for (const source of page.sources) owned.push(source.path);
  for (const section of Array.isArray(page.generatedSections) ? page.generatedSections : []) {
    for (const input of Array.isArray(section.inputs) ? section.inputs : []) owned.push(input);
  }
  return changedPaths.some((changedPath) => owned.some((target) => pathMatchesChange(changedPath, target)));
}

function runFreshnessChecks(report, { repoRoot, manifest, manifestPath, manifestText, changedPaths }) {
  const manifestRelative = displayManifestPath(repoRoot, manifestPath);
  const stale = [];
  for (const page of [...manifest.pages].sort((left, right) => compareStrings(left.id, right.id))) {
    // A fingerprint list that omits the page's own source can never detect a
    // body change, so registration is part of the freshness contract.
    if (!page.sources.some((source) => source.path === page.source)) {
      report.error(
        'freshness/unregistered-source',
        page.source,
        1,
        `Page "${page.id}" does not register its canonical source "${page.source}" in its reviewed sources[] entry.`,
        `Add "${page.source}" to sources[] with its reviewed sha256 and refresh reviewDigest after human review; freshness cannot fail closed on a body change it never hashes.`,
        { pageId: page.id, source: page.source },
      );
      continue;
    }
    let actual;
    try {
      actual = hashSources(
        repoRoot,
        page.sources.map((source) => source.path),
      );
    } catch (error) {
      if (!(error instanceof ManifestError)) throw error;
      report.error(
        'freshness/missing-input',
        displayManifestPath(repoRoot, manifestPath),
        locateManifestLine(manifestText, error.path ?? page.id),
        `Cannot verify reviewed inputs for page "${page.id}": ${error.message}`,
        'Restore the missing/unreadable input, or remove it from sources and recompute reviewDigest after review.',
        { pageId: page.id, input: error.path ?? '' },
      );
      continue;
    }
    const declared = new Map(page.sources.map((source) => [source.path, source.sha256]));
    const actualMap = new Map(actual.map((record) => [record.path, record.sha256]));
    const changed = actual
      .map((record) => record.path)
      .filter((sourcePath) => declared.get(sourcePath) !== actualMap.get(sourcePath))
      .sort(compareStrings);
    if (changed.length === 0) continue;
    stale.push({ page, actual, changed });
  }

  const affected = [];
  const unrelated = [];
  for (const entry of stale) {
    if (pageTouchedBy(entry.page, changedPaths, manifestRelative)) affected.push(entry);
    else unrelated.push(entry);
  }

  for (const entry of affected) {
    const recommended = computeReviewDigest(entry.actual);
    const status = entry.page.reviewStatus === 'reviewed'
      ? `reviewed page (${entry.page.reviewedAt ?? 'review date absent'})`
      : 'page with an unreviewed source fingerprint';
    report.error(
      'freshness/needs-review',
      entry.page.source,
      1,
      `Reviewed source fingerprint for "${entry.page.id}" (${status}) changed without a human disposition: ${entry.changed.join(', ')}.`,
      `Review the changed input(s). Then either update content and intentionally refresh the page's sha256 entries and reviewDigest (${recommended}), or record a no-public-impact explanation and refresh it. This check never refreshes fingerprints automatically.`,
      {
        pageId: entry.page.id,
        changed: entry.changed,
        recommendedReviewDigest: recommended,
        reviewStatus: entry.page.reviewStatus ?? 'unreviewed',
      },
    );
  }

  if (unrelated.length > 0) {
    const pages = unrelated
      .map((entry) => ({ pageId: entry.page.id, source: entry.page.source, changed: entry.changed }))
      .sort((left, right) => compareStrings(left.pageId, right.pageId));
    report.observation(
      'freshness/unrelated-stale',
      displayManifestPath(repoRoot, manifestPath),
      1,
      `Pre-existing reviewed-input drift affects ${pages.length} page(s) outside this change set: ${pages.map((entry) => entry.pageId).join(', ')}.`,
      'This is one consolidated observation, not a failure of the current change. Review/refresh those page fingerprints when each page is next touched; never auto-refresh.',
      { pages },
    );
  }
}

/* ------------------------------------------------------------------------- */
/* Built-site checks                                                         */
/* ------------------------------------------------------------------------- */

function maskBuiltRegions(html) {
  const characters = html.split('');
  const blank = (start, end) => {
    for (let index = start; index < end; index += 1) {
      if (characters[index] !== '\n' && characters[index] !== '\r') characters[index] = ' ';
    }
  };
  for (const pattern of [/<!--[\s\S]*?-->/g, /<script\b[\s\S]*?<\/script\s*>/gi, /<style\b[\s\S]*?<\/style\s*>/gi]) {
    for (const match of html.matchAll(pattern)) blank(match.index, match.index + match[0].length);
  }
  return characters.join('');
}

function parseAttributes(tag) {
  const attributes = {};
  const pattern = /([^\s=/>]+)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>`]+)))?/g;
  for (const match of tag.matchAll(pattern)) {
    const name = match[1].toLowerCase();
    if (name === 'a' || name === 'link' || name === 'img' || Object.hasOwn(attributes, name)) continue;
    attributes[name] = decodeHtmlEntities(match[2] ?? match[3] ?? match[4] ?? '');
  }
  return attributes;
}

function extractBuiltDocument(html) {
  const masked = maskBuiltRegions(html);
  const ids = [];
  const idCounts = new Map();
  for (const match of masked.matchAll(/(?:^|[\s<])id\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))/g)) {
    const raw = match[1] ?? match[2] ?? match[3] ?? '';
    const value = decodeHtmlEntities(raw);
    ids.push({ value, line: lineAt(masked, match.index) });
    idCounts.set(value, (idCounts.get(value) ?? 0) + 1);
  }
  const idsSet = new Set(ids.map((entry) => entry.value));
  const duplicateIds = [...idCounts.entries()]
    .filter(([, count]) => count > 1)
    .map(([value]) => value)
    .sort(compareStrings);

  const canonical = [];
  for (const match of masked.matchAll(/<link\b[^>]*>/gi)) {
    const attributes = parseAttributes(match[0]);
    const rel = (attributes.rel ?? '').toLowerCase().split(/\s+/).filter(Boolean);
    if (!rel.includes('canonical')) continue;
    canonical.push({ href: attributes.href ?? '', line: lineAt(masked, match.index) });
  }

  const links = [];
  for (const match of masked.matchAll(/<a\b[^>]*>/gi)) {
    const attributes = parseAttributes(match[0]);
    if (attributes.href === undefined) continue;
    links.push({ href: attributes.href, line: lineAt(masked, match.index) });
  }
  return { idsSet, ids, duplicateIds, canonical, links };
}

function readBuiltDocument(cache, absolute) {
  const key = toPosix(absolute);
  if (cache.has(key)) return cache.get(key);
  let document = null;
  try {
    const stats = lstatSync(absolute);
    if (stats.isSymbolicLink() || !stats.isFile() || stats.nlink > 1) {
      throw new Error('not a single-linked regular file');
    }
    if (stats.size > MAX_BUILT_HTML_BYTES) {
      throw new Error(`exceeds ${MAX_BUILT_HTML_BYTES} bytes`);
    }
    const html = readFileSync(absolute, 'utf8');
    document = extractBuiltDocument(html);
  } catch {
    document = null;
  }
  cache.set(key, document);
  return document;
}

function displayBuiltPath(repoRoot, absolute) {
  const relative = toPosix(path.relative(repoRoot, absolute));
  if (relative.startsWith('../') || path.isAbsolute(relative)) return absolute;
  return relative;
}

function builtPathForUrl(builtRoot, base, pathname) {
  const decodedSegments = [];
  for (const segment of pathname.split('/')) {
    if (segment === '') {
      decodedSegments.push('');
      continue;
    }
    const decoded = decodeUriComponent(segment);
    if (!decoded.ok) return { error: 'invalid-encoding' };
    if (
      decoded.value === '.' ||
      decoded.value === '..' ||
      decoded.value.includes('/') ||
      decoded.value.includes('\\') ||
      /[\u0000-\u001f\u007f]/.test(decoded.value)
    ) {
      return { error: 'unsafe-path' };
    }
    decodedSegments.push(decoded.value);
  }
  let cleaned = decodedSegments.join('/');
  const basePrefix = base === '/' ? '' : base;
  if (basePrefix !== '') {
    if (cleaned !== basePrefix && !cleaned.startsWith(`${basePrefix}/`)) return { wrongBase: true };
    cleaned = cleaned.slice(basePrefix.length) || '/';
  }
  const relative = cleaned.replace(/^\/+/, '');
  const candidates = [];
  if (relative === '') {
    candidates.push('index.html');
  } else if (relative.endsWith('/')) {
    candidates.push(`${relative}index.html`);
  } else {
    candidates.push(relative);
    if (path.posix.extname(relative) === '') candidates.push(`${relative}/index.html`);
  }
  return { candidates };
}

function builtFixFor(code, base) {
  switch (code) {
    case 'wrong-base':
      return `Prefix internal URLs with the manifest base "${base === '/' ? '/' : base}" exactly once, or remove the hard-coded base.`;
    case 'broken-link':
      return 'Build the linked page or asset, or change the link to an emitted output path.';
    case 'symlink-output':
      return 'Replace the symlink with a regular emitted file; deploy artifacts must not contain symlinks.';
    case 'hardlink-output':
      return 'Replace the hard link with a single-named regular emitted file; deploy artifacts must not contain hardlinks.';
    case 'broken-anchor':
      return 'Add the missing id/heading to the target page or link an existing rendered fragment.';
    case 'invalid-encoding':
      return 'Percent-encode the URL once; do not double-encode or encode a literal path separator.';
    case 'unsafe-path':
      return 'Use a normal same-origin path without traversal or encoded separators.';
    case 'empty-target':
      return 'Remove the empty href or point it at an emitted page, asset or fragment.';
    default:
      return 'Use emitted static output paths under the configured base.';
  }
}

function checkBuiltLinks({ report, page, document, pageUrl, builtRoot, base, cache }) {
  const siteOrigin = new URL(pageUrl).origin;
  const external = [];
  const links = [...document.links].sort(
    (left, right) => left.line - right.line || compareStrings(left.href, right.href),
  );
  for (const link of links) {
    report.metrics.links += 1;
    const href = link.href;
    if (href.length === 0) {
      report.error('built/empty-target', document.displayPath, link.line, 'Rendered link has an empty href.', builtFixFor('empty-target', base));
      continue;
    }
    if (href.startsWith('#')) {
      checkBuiltFragment(report, document, link, href.slice(1), document, pageUrl, base);
      continue;
    }
    const split = splitHref(href);
    if (split.pathPart === '') {
      if (split.fragment !== null && split.fragment !== '') {
        checkBuiltFragment(report, document, link, split.fragment, document, pageUrl, base);
      } else {
        report.error('built/empty-target', document.displayPath, link.line, `Rendered link "${href}" has no path.`, builtFixFor('empty-target', base), { href });
      }
      continue;
    }
    const scheme = /^([a-z][a-z0-9+.-]*):/i.exec(split.pathPart);
    if (scheme !== null && !['http', 'https', 'mailto'].includes(scheme[1].toLowerCase())) {
      report.error('built/unsafe-scheme', document.displayPath, link.line, `Rendered link "${href}" uses unsupported scheme "${scheme[1]}:"`, 'Use an https:, mailto: or same-site path.', { href });
      continue;
    }
    if (scheme !== null && scheme[1].toLowerCase() === 'mailto') {
      external.push(href);
      continue;
    }
    let target;
    try {
      target = new URL(href, pageUrl);
    } catch {
      report.error('built/invalid-url', document.displayPath, link.line, `Rendered link "${href}" is not a valid URL.`, 'Use a valid absolute or site-relative URL.', { href });
      continue;
    }
    if (target.protocol === 'http:' && target.hostname === new URL(siteOrigin).hostname) {
      report.error('built/unsafe-scheme', document.displayPath, link.line, `Rendered same-host link "${href}" downgrades HTTPS.`, 'Use the HTTPS site origin and let the project base resolve.', { href });
      continue;
    }
    if (target.origin !== siteOrigin) {
      external.push(href);
      continue;
    }
    const resolved = builtPathForUrl(builtRoot, base, target.pathname);
    if (resolved.wrongBase === true) {
      report.error('built/wrong-base', document.displayPath, link.line, `Rendered link "${href}" does not start with the manifest base "${base === '/' ? '/' : base}".`, builtFixFor('wrong-base', base), { href });
      continue;
    }
    if (resolved.error !== undefined) {
      report.error(`built/${resolved.error}`, document.displayPath, link.line, `Rendered link "${href}" cannot map to a contained static file.`, builtFixFor(resolved.error, base), { href });
      continue;
    }
    let targetAbsolute = null;
    let symlinkTarget = null;
    let hardlinkTarget = null;
    for (const candidate of resolved.candidates) {
      const absolute = path.join(builtRoot, candidate);
      const relative = toPosix(path.relative(builtRoot, absolute));
      if (relative.startsWith('../') || path.isAbsolute(relative)) continue;
      const inspection = inspectContainedTarget(builtRoot, candidate);
      if (inspection.kind === 'symlink') {
        symlinkTarget = candidate;
        break;
      }
      if (inspection.kind === 'hardlink') {
        hardlinkTarget = candidate;
        break;
      }
      if (inspection.kind !== 'file') continue;
      if (readBuiltDocument(cache, absolute) !== null) {
        targetAbsolute = absolute;
        break;
      }
    }
    if (symlinkTarget !== null) {
      report.error('built/symlink-output', document.displayPath, link.line, `Rendered link "${href}" resolves through a symlink (${symlinkTarget}).`, builtFixFor('symlink-output', base), { href, target: symlinkTarget });
      continue;
    }
    if (hardlinkTarget !== null) {
      report.error('built/hardlink-output', document.displayPath, link.line, `Rendered link "${href}" resolves through a hard link (${hardlinkTarget}).`, builtFixFor('hardlink-output', base), { href, target: hardlinkTarget });
      continue;
    }
    if (targetAbsolute === null) {
      report.error('built/broken-link', document.displayPath, link.line, `Rendered link "${href}" points at a page or asset that was not emitted.`, builtFixFor('broken-link', base), { href, target: toPosix(path.relative(builtRoot, path.join(builtRoot, resolved.candidates[0]))) });
      continue;
    }
    if (target.hash !== '') {
      const targetDocument = readBuiltDocument(cache, targetAbsolute);
      if (targetDocument !== null) {
        checkBuiltFragment(report, document, link, target.hash.slice(1), targetDocument, href, base);
      }
    }
  }
  if (external.length > 0) {
    report.metrics.external += external.length;
    report.observation(
      'built/external',
      document.displayPath,
      document.links.find((link) => external.includes(link.href))?.line ?? 1,
      `${external.length} external rendered link(s) were not fetched (${[...new Set(external)].sort(compareStrings).join(', ')}).`,
      'External availability is not verified offline; keep the URL and check it manually when connectivity matters.',
      { hrefs: [...new Set(external)].sort(compareStrings) },
    );
  }
}

function checkBuiltFragment(report, sourceDocument, link, rawFragment, targetDocument, targetLabel, base) {
  const decoded = decodeUriComponent(rawFragment);
  if (!decoded.ok) {
    report.error('built/invalid-encoding', sourceDocument.displayPath, link.line, `Rendered fragment in "${link.href}" is not valid percent-encoding.`, builtFixFor('invalid-encoding', base), { href: link.href });
    return;
  }
  if (decoded.value === '') return;
  if (targetDocument.idsSet.has(decoded.value)) return;
  const suggestions = formatSuggestionList(targetDocument.idsSet);
  report.error(
    'built/broken-anchor',
    sourceDocument.displayPath,
    link.line,
    `Rendered fragment "#${decoded.value}" is absent from ${targetLabel}.`,
    suggestions.length > 0
      ? `Link an emitted id (${suggestions}) or add the target heading/id.`
      : 'Add the target heading/id or remove the fragment.',
    { href: link.href, fragment: decoded.value },
  );
}

function runBuiltChecks(report, { repoRoot, manifest, builtDir }) {
  const resolvedBuilt = path.resolve(repoRoot, builtDir);
  let builtRoot;
  try {
    builtRoot = realpathSync(resolvedBuilt);
  } catch (error) {
    report.error(
      'built/missing-directory',
      displayBuiltPath(repoRoot, resolvedBuilt),
      1,
      `Built directory does not exist: ${error.code ?? error.message}`,
      'Build the site first (npm --prefix website run build) and pass --built website/dist.',
    );
    return;
  }
  if (!statSync(builtRoot).isDirectory()) {
    report.error(
      'built/not-directory',
      displayBuiltPath(repoRoot, resolvedBuilt),
      1,
      'Built output is not a directory.',
      'Pass --built with the static output directory (for example website/dist).',
    );
    return;
  }
  const cache = new Map();
  for (const page of [...manifest.pages].sort((left, right) => compareStrings(left.id, right.id))) {
    const relative = page.id === 'index' ? 'index.html' : `${page.id}/index.html`;
    const absolute = path.join(builtRoot, relative);
    const display = displayBuiltPath(repoRoot, absolute);
    const inspection = inspectContainedTarget(builtRoot, relative);
    if (inspection.kind === 'symlink') {
      report.error(
        'built/symlink-output',
        display,
        1,
        `Manifest page "${page.id}" is emitted through a symlink.`,
        'Replace the symlink with a regular emitted file; deploy artifacts must not contain symlinks.',
        { pageId: page.id },
      );
      continue;
    }
    if (inspection.kind === 'hardlink') {
      report.error(
        'built/hardlink-output',
        display,
        1,
        `Manifest page "${page.id}" is emitted as a hard link.`,
        'Replace the hard link with a single-named regular emitted file; deploy artifacts must not contain hardlinks.',
        { pageId: page.id },
      );
      continue;
    }
    const document = readBuiltDocument(cache, absolute);
    if (document === null) {
      report.error(
        'built/missing-page',
        display,
        1,
        `Manifest page "${page.id}" has no emitted HTML at "${relative}".`,
        'Build the page and verify Starlight/trailingSlash output; do not pass a partial or stale output directory.',
        { pageId: page.id },
      );
      continue;
    }
    report.metrics.builtPages += 1;
    document.displayPath = display;
    for (const id of document.duplicateIds) {
      const line = document.ids.find((entry) => entry.value === id)?.line ?? 1;
      report.error('built/duplicate-id', display, line, `Rendered HTML repeats id "${id}".`, 'Give each rendered element a unique id; duplicate ids make fragments ambiguous.', { id });
    }
    const expectedCanonical = canonicalUrl(manifest, page.id);
    if (document.canonical.length === 0) {
      report.error('built/canonical-missing', display, 1, `Manifest page "${page.id}" has no canonical link.`, `Emit exactly one <link rel="canonical" href="${expectedCanonical}">.`, { expected: expectedCanonical });
    } else {
      if (document.canonical.length > 1) {
        report.error('built/canonical-duplicate', display, document.canonical[1].line, `Manifest page "${page.id}" has ${document.canonical.length} canonical links.`, 'Emit exactly one canonical link per page.', { expected: expectedCanonical });
      }
      for (const canonical of document.canonical) {
        if (canonical.href !== expectedCanonical) {
          report.error(
            'built/canonical-mismatch',
            display,
            canonical.line,
            `Canonical URL "${canonical.href}" does not match the manifest URL "${expectedCanonical}".`,
            `Derive the canonical from manifest.site + manifest.base + the page id; do not hard-code an origin or base.`,
            { expected: expectedCanonical, actual: canonical.href, pageId: page.id },
          );
        }
      }
    }
    const pageUrl = expectedCanonical;
    checkBuiltLinks({ report, page, document, pageUrl, builtRoot, base: manifest.base, cache });
  }
}

/* ------------------------------------------------------------------------- */
/* Orchestration                                                             */
/* ------------------------------------------------------------------------- */

function analyze(options, modes, label) {
  const repoRoot = resolveRepoRootOption(options);
  const manifestPath = resolveManifestPath(repoRoot, options);
  const report = new CheckReport();
  const loaded = readManifest(report, repoRoot, manifestPath);
  if (loaded === null) return report.result();
  const { manifest, text: manifestText } = loaded;
  report.metrics.pages = manifest.pages.length;
  const manifestDisplay = displayManifestPath(repoRoot, manifestPath);

  let stubSpecs = [];
  if (modes.source) {
    const rawStubs = options.stubs ?? [];
    if (!Array.isArray(rawStubs)) {
      throw new TypeError(`${label}: stubs must be an array of exact stub declarations`);
    }
    stubSpecs = rawStubs.map(parseStubSpec).sort((left, right) => compareStrings(left.path, right.path));
  }

  if (modes.source) {
    checkContentRootCoverage(report, repoRoot, manifest);
    const pagesBySource = new Map(manifest.pages.map((page) => [page.source, page]));
    const bodies = [];
    const cache = new Map();
    const parsedPages = new Map();
    for (const page of [...manifest.pages].sort((left, right) => compareStrings(left.id, right.id))) {
      let parsed;
      try {
        parsed = parseMarkdownSource(repoRoot, page.source);
      } catch (error) {
        report.error(
          error.code === 'hardlink' ? 'source/hardlink' : 'source/unreadable',
          page.source,
          1,
          `Cannot inspect canonical page source: ${error.message}`,
          'Restore the contained Markdown source named by the manifest as a single-linked regular file, or point the page at an existing file.',
          { pageId: page.id },
        );
        continue;
      }
      parsedPages.set(page.id, parsed);
      cache.set(page.source, parsed);
      bodies.push({ path: page.source, body: parsed.body });
      checkFrontmatterClaims(report, page.source, parsed.text, 'Page');
      for (const duplicate of parsed.duplicateHeadings) {
        report.error(
          'headings/duplicate',
          page.source,
          duplicate.line,
          `Heading "${duplicate.text}" produces duplicate slug "#${duplicate.slug}".`,
          'Rename one heading so every authored slug is unique; historical fragments belong in exact aliases, not duplicate headings.',
          { slug: duplicate.slug },
        );
      }
      checkRawHtmlNavigation(report, page, parsed.view);
      checkGeneratedMarkers(report, page, parsed.text, parsed.view, manifestDisplay, manifestText);
    }

    const parsedStubs = checkStubs({ report, stubSpecs, manifest, repoRoot, bodies });
    checkOwnership(report, manifest, manifestDisplay, manifestText, { stubs: stubSpecs }, bodies);
    for (const page of [...manifest.pages].sort((left, right) => compareStrings(left.id, right.id))) {
      const parsed = parsedPages.get(page.id);
      if (parsed === undefined) continue;
      checkSourceLinks({ report, page, parsed, manifest, repoRoot, pagesBySource, cache });
    }
    for (const { stubPath, parsed } of parsedStubs) {
      checkSourceLinks({
        report,
        page: { source: stubPath },
        parsed,
        manifest,
        repoRoot,
        pagesBySource,
        cache,
      });
    }
  }

  if (modes.freshness) {
    runFreshnessChecks(report, {
      repoRoot,
      manifest,
      manifestPath,
      manifestText,
      changedPaths: normalizeChangedPaths(options.changedPaths),
    });
  }

  if (modes.built) {
    if (typeof options.builtDir !== 'string' || options.builtDir.length === 0) {
      throw new TypeError(`${label}: builtDir must be a non-empty directory path`);
    }
    runBuiltChecks(report, { repoRoot, manifest, builtDir: options.builtDir });
  }

  return report.result();
}

export function checkSource(options = {}) {
  const normalized = normalizeOptions(options, ['repoRoot', 'manifestPath', 'stubs'], 'checkSource');
  return analyze(normalized, { source: true, freshness: false, built: false }, 'checkSource');
}

export function checkFreshness(options = {}) {
  const normalized = normalizeOptions(options, ['repoRoot', 'manifestPath', 'changedPaths'], 'checkFreshness');
  return analyze(normalized, { source: false, freshness: true, built: false }, 'checkFreshness');
}

export function checkBuilt(options = {}) {
  const normalized = normalizeOptions(options, ['repoRoot', 'manifestPath', 'builtDir'], 'checkBuilt');
  if (typeof normalized.builtDir !== 'string' || normalized.builtDir.length === 0) {
    throw new TypeError('checkBuilt requires builtDir');
  }
  return analyze(normalized, { source: false, freshness: false, built: true }, 'checkBuilt');
}

export function checkDocumentation(options = {}) {
  const normalized = normalizeOptions(
    options,
    ['repoRoot', 'manifestPath', 'stubs', 'changedPaths', 'builtDir'],
    'checkDocumentation',
  );
  return analyze(
    normalized,
    { source: true, freshness: true, built: typeof normalized.builtDir === 'string' },
    'checkDocumentation',
  );
}

/* ------------------------------------------------------------------------- */
/* CLI                                                                       */
/* ------------------------------------------------------------------------- */

const USAGE = `Usage: node website/scripts/check.mjs [options]

Checks documentation links, fragments, generated markers, ownership, stub
shape, manifest freshness and (optionally) built HTML metadata.

Options:
  --repo DIR          Repository root (default: this checkout)
  --manifest FILE     Manifest path (cwd-relative; default:
                      website/docs-manifest.json)
  --built DIR         Validate built HTML output (built-only mode; resolves
                      from the repository root, e.g. website/dist)
  --source            With --built, also run source/freshness checks
  --affected PATHS    Repository-relative current change set, comma-separated
                      or repeated. Stale pages outside it become one
                      observation instead of a failure (default: every stale
                      page fails)
  --stub FILE=PAGE_ID[:ANCHOR,...]
                      Exact repository-relative compatibility-stub declaration;
                      may be repeated
  --json              Emit the frozen programmatic report as JSON
  -h, --help          Show this help

Exit: 0 pass, 1 documentation error, 2 usage/input error.
No external link is fetched, no fingerprint is refreshed and no file is written.

Composition: \`npm --prefix website run docs:check\` runs this script for the
source and freshness rules, and \`docs:check-built\` runs the same script so
\`-- --built <dir>\` can be appended for emitted output. The generated reference
catalogs are verified separately with \`node website/scripts/reference.mjs
--check\`, because they require the Go/CLI inputs this offline checker does not
read.`;

function parseCli(argv) {
  const options = {
    repoRoot: undefined,
    manifestPath: undefined,
    builtDir: undefined,
    stubs: [],
    changedPaths: undefined,
    json: false,
    help: false,
    source: false,
  };
  const affected = [];
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--help' || arg === '-h') {
      options.help = true;
      continue;
    }
    if (arg === '--json') {
      options.json = true;
      continue;
    }
    if (arg === '--source') {
      options.source = true;
      continue;
    }
    if (arg === '--repo' || arg === '--manifest' || arg === '--built' || arg === '--affected' || arg === '--stub') {
      const value = argv[index + 1];
      if (value === undefined || value.startsWith('--')) {
        throw new Error(`Missing value for ${arg}`);
      }
      index += 1;
      if (arg === '--repo') options.repoRoot = value;
      else if (arg === '--manifest') options.manifestPath = path.resolve(value);
      else if (arg === '--built') options.builtDir = value;
      else if (arg === '--affected') {
        for (const entry of value.split(',')) {
          if (entry.trim().length > 0) affected.push(entry.trim());
        }
      } else {
        options.stubs.push(value);
      }
      continue;
    }
    throw new Error(`Unknown option: ${arg}`);
  }
  if (affected.length > 0) options.changedPaths = affected;
  return options;
}

function printHuman(result) {
  const lines = [
    `docs-check: ${result.ok ? 'PASS' : 'FAIL'} — ${result.summary.errorCount} error(s), ${result.summary.observationCount} observation(s)`,
    `  pages=${result.summary.pages} links=${result.summary.links} external=${result.summary.external} stubs=${result.summary.stubs} builtPages=${result.summary.builtPages}`,
  ];
  for (const entry of [...result.errors, ...result.observations]) {
    lines.push(`${entry.severity === 'error' ? 'ERROR' : 'NOTE '} [${entry.code}] ${entry.path}:${entry.line}`);
    lines.push(`  ${entry.message}`);
    lines.push(`  fix: ${entry.fix}`);
  }
  return lines.join('\n');
}

function isCliEntry() {
  const entry = process.argv[1];
  if (typeof entry !== 'string' || entry.length === 0) return false;
  return path.resolve(entry) === fileURLToPath(import.meta.url);
}

if (isCliEntry()) {
  let options;
  try {
    options = parseCli(process.argv.slice(2));
  } catch (error) {
    console.error(`docs-check: ${error.message}`);
    process.exitCode = 2;
  }
  if (options !== undefined) {
    if (options.help) {
      console.log(USAGE);
    } else {
      try {
        if (options.builtDir !== undefined && options.source === false) {
          if (options.changedPaths !== undefined) {
            throw new Error('--affected requires source checks; add --source or omit --built');
          }
          if (options.stubs.length > 0) {
            throw new Error('--stub requires source checks; add --source or omit --built');
          }
        }
        const baseOptions = {
          repoRoot: options.repoRoot,
          manifestPath: options.manifestPath,
        };
        for (const key of Object.keys(baseOptions)) {
          if (baseOptions[key] === undefined) delete baseOptions[key];
        }
        const result =
          options.builtDir !== undefined && options.source === false
            ? checkBuilt({ ...baseOptions, builtDir: options.builtDir })
            : checkDocumentation({
                ...baseOptions,
                ...(options.builtDir === undefined ? {} : { builtDir: options.builtDir }),
                ...(options.stubs.length === 0 ? {} : { stubs: options.stubs }),
                ...(options.changedPaths === undefined ? {} : { changedPaths: options.changedPaths }),
              });
        if (options.json) console.log(JSON.stringify(result, null, 2));
        else console.log(printHuman(result));
        if (!result.ok) process.exitCode = 1;
      } catch (error) {
        console.error(`docs-check: ${error.message}`);
        process.exitCode = 2;
      }
    }
  }
}
