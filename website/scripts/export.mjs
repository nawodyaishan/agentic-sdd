// SPEC020 B4-W1-E — deterministic public export from the canonical L/C allowlist.
//
// Contract (frozen for batch B4, see specs/020-documentation-ecosystem/plan.md
// §3/§C1 and /tmp/opencode/spec020-b4/contract.md):
//
//   buildExport({repoRoot, manifestPath?, ref?}) -> {manifest, ref, pages, excluded, files}
//   writeExport({repoRoot, outDir, manifestPath?, ref?}) -> {written, unchanged}
//   checkExport({repoRoot, outDir, manifestPath?, ref?}) -> {ok, differences, firstDifference, expectedCount}
//   node scripts/export.mjs --out <dir> [--check] [--repo <dir>] [--ref <git-ref>] [--json]
//
// Guarantees:
// - Output is exactly `llms.txt`, `index.json` and one Markdown twin per
//   manifest page whose class is L or C. Deferred A/F pages, `specs/`, `docs/`,
//   handovers, evidence, runs, backups and history are never read for output.
// - Every exported page source is a contained regular file under
//   `website/src/content/docs`, registered exactly once, matching the reviewed
//   sha256 in its own `sources[]` entry. Unregistered content files, symlinks,
//   special files and stale digests fail closed.
// - All public URLs and twin routes come from `lib/routes.mjs`; the project base
//   and the page applicability/docsVersion relationship are asserted, never
//   re-derived in prose. Links inside twins become absolute public URLs or
//   pinned repository source links.
// - Deterministic: no timestamps, no absolute paths, sorted output, LF endings.
//   Two runs produce byte-identical files.
// - `--check` reads only and reports the first difference. It never creates the
//   output directory and never deletes unrelated files.
// - Node built-ins plus the already pinned `yaml` parser. No network, no
//   external AI, no vector/BM25/RAG service. `llms-full.txt` is out of scope.

import { createHash } from 'node:crypto';
import {
  existsSync,
  lstatSync,
  mkdirSync,
  readdirSync,
  readFileSync,
  realpathSync,
  renameSync,
  unlinkSync,
  writeFileSync,
} from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import YAML from 'yaml';
import { loadManifest } from './lib/manifest.mjs';
import { CONTENT_ROOT, PublicTreeError, classifyLinkTarget, scanContentRoot } from './lib/public-paths.mjs';
import { resolveLink } from './lib/routes.mjs';
const DEFAULT_MANIFEST = 'website/docs-manifest.json';
const EXPORT_FORMAT_VERSION = 1;
const DEV_VERSION = 'dev';
const UNRELEASED = 'unreleased';
const ALLOWED_CLASSES = new Set(['L', 'C']);
const MAX_SOURCE_BYTES = 1024 * 1024;
const CONTROL_CHARACTER = /[\u0000-\u001f\u007f]/;
const OUTPUT_NAMES = new Set(['llms.txt', 'index.json']);
const OUT_OF_SCOPE_OUTPUTS = new Set(['llms-full.txt']);

// A public export must never mirror a private, historical or engineering tree.
const DENIED_FIRST_SEGMENTS = new Set([
  'specs',
  'docs',
  'website',
  'tests',
  'harness',
  'internal',
  'cmd',
  'backups',
  '.agentic-sdd',
  '.git',
  'node_modules',
  'coverage',
  'credentials',
  'secrets',
  'skills-files',
  'opencode-pack',
  'experiments',
]);

const INLINE_LINK = /(!?)\[([^\]\n]*)\]\([ \t]*(<[^<>\n]*>|(?:[^()\s]|\([^()\s]*\))*)([ \t]+(?:"[^"\n]*"|'[^'\n]*'|\([^()\n]*\)))?[ \t]*\)/g;
const LINK_DEFINITION = /^( {0,3})\[([^\]\n]+)\]:([ \t]*)(<[^>\n]*>|\S+)(.*)$/g;
const FENCE_OPEN = /^ {0,3}(`{3,}|~{3,})(.*)$/;

export class ExportError extends Error {
  constructor(code, errorPath, message) {
    super(message);
    this.name = 'ExportError';
    this.code = code;
    this.path = errorPath ?? '';
  }
}

function compareText(left, right) {
  if (left < right) return -1;
  if (left > right) return 1;
  return 0;
}

function firstLine(error) {
  return String(error?.message ?? error).split('\n')[0];
}

function singleLine(value) {
  return value.trim().replace(/\s+/g, ' ');
}

function sha256Hex(bytes) {
  return createHash('sha256').update(bytes).digest('hex');
}

function absoluteUrl(site, routePath, errorPath) {
  let url;
  try {
    url = new URL(routePath, site);
  } catch {
    throw new ExportError('base-parity', errorPath, `Cannot compose a public URL from "${routePath}"`);
  }
  if (url.origin !== site || url.search !== '' || url.hash !== '') {
    throw new ExportError('base-parity', errorPath, `"${routePath}" is not a plain path below the approved origin`);
  }
  return url.href;
}

function absoluteLinkUrl(site, base, routePath, errorPath) {
  const pathOnly = routePath.split('#')[0].split('?')[0];
  assertBaseContains(base, pathOnly, errorPath);
  let url;
  try {
    url = new URL(routePath, site);
  } catch {
    throw new ExportError('base-parity', errorPath, `Cannot compose a public link from "${routePath}"`);
  }
  if (url.origin !== site) {
    throw new ExportError('base-parity', errorPath, `"${routePath}" is not below the approved origin`);
  }
  return url.href;
}

function assertBaseContains(base, routePath, errorPath) {
  const ok = base === '/' ? routePath.startsWith('/') : routePath === base || routePath.startsWith(`${base}/`);
  if (!ok) {
    throw new ExportError('base-parity', errorPath, `Route "${routePath}" does not live below the project base "${base}"`);
  }
  return routePath;
}

// Independent expectation for the canonical HTML route: the shared resolver
// must agree with it, so a double prefix or a missing base fails here.
function expectedHtmlRoute(base, id) {
  const prefix = base === '/' ? '/' : `${base}/`;
  if (id === 'index') return prefix;
  const encoded = id.split('/').map((segment) => encodeURIComponent(segment)).join('/');
  return `${prefix}${encoded}/`;
}

function expectedMarkdownName(id) {
  if (id === 'index') return 'index.md';
  const encoded = id.split('/').map((segment) => encodeURIComponent(segment)).join('/');
  return `${encoded}/index.md`;
}

function normalizeOutputPath(relative, errorPath) {
  if (typeof relative !== 'string' || relative.length === 0) {
    throw new ExportError('unsafe-output-path', errorPath, 'Expected a non-empty relative output path');
  }
  if (CONTROL_CHARACTER.test(relative) || relative.includes('\\')) {
    throw new ExportError('unsafe-output-path', errorPath, `Output path "${relative}" contains unsafe characters`);
  }
  if (relative.startsWith('/') || /^[A-Za-z]:/.test(relative)) {
    throw new ExportError('unsafe-output-path', errorPath, `Output path "${relative}" must be relative`);
  }
  const segments = relative.split('/');
  if (segments.some((segment) => segment.length === 0 || segment === '.' || segment === '..')) {
    throw new ExportError('unsafe-output-path', errorPath, `Output path "${relative}" repeats or traverses a segment`);
  }
  if (DENIED_FIRST_SEGMENTS.has(segments[0])) {
    throw new ExportError(
      'denied-output-path',
      errorPath,
      `"${relative}" starts with the private or engineering tree "${segments[0]}/" and cannot be exported`,
    );
  }
  if (OUT_OF_SCOPE_OUTPUTS.has(relative)) {
    throw new ExportError('out-of-scope-output', errorPath, `"${relative}" is explicitly deferred and must not be generated`);
  }
  return segments.join('/');
}

function resolveRepoRoot(repoRoot) {
  if (typeof repoRoot !== 'string' || repoRoot.length === 0) {
    throw new ExportError('invalid-repo-root', 'repoRoot', 'Expected a non-empty repository root path');
  }
  let real;
  let stats;
  try {
    real = realpathSync(path.resolve(repoRoot));
    stats = lstatSync(real);
  } catch (error) {
    throw new ExportError('invalid-repo-root', 'repoRoot', `Cannot resolve repository root "${repoRoot}": ${error.code ?? error.message}`);
  }
  if (!stats.isDirectory()) {
    throw new ExportError('invalid-repo-root', 'repoRoot', `Repository root "${repoRoot}" is not a directory`);
  }
  return real;
}

// Reads a repository-relative path below an already resolved root. Every
// component is lstat-checked, so symlink parents, symlink leaves and special
// files are refused before any bytes are read.
function readContainedRegularFile(rootReal, relative, errorPath) {
  const segments = relative.split('/');
  let current = rootReal;
  for (let index = 0; index < segments.length; index += 1) {
    current = path.join(current, segments[index]);
    let stats;
    try {
      stats = lstatSync(current);
    } catch (error) {
      if (error.code === 'ENOENT' || error.code === 'ENOTDIR') {
        throw new ExportError('missing-page-source', errorPath, `Missing contained file "${relative}"`);
      }
      throw new ExportError('unreadable-source', errorPath, `Cannot stat "${relative}": ${error.code ?? error.message}`);
    }
    if (stats.isSymbolicLink()) {
      throw new ExportError('symlink-path', errorPath, `"${relative}" resolves through a symlink`);
    }
    if (index < segments.length - 1) {
      if (!stats.isDirectory()) {
        throw new ExportError('missing-page-source', errorPath, `"${relative}" is not a directory`);
      }
      continue;
    }
    if (!stats.isFile()) {
      throw new ExportError('not-a-regular-file', errorPath, `"${relative}" is not a regular file`);
    }
    if (stats.nlink > 1) {
      throw new ExportError(
        'hardlink-path',
        errorPath,
        `"${relative}" is a hard link (${stats.nlink} names); export inputs must be single-linked regular files`,
      );
    }
    if (stats.size > MAX_SOURCE_BYTES) {
      throw new ExportError('file-too-large', errorPath, `"${relative}" exceeds ${MAX_SOURCE_BYTES} bytes`);
    }
  }
  let bytes;
  try {
    bytes = readFileSync(current);
  } catch (error) {
    throw new ExportError('unreadable-source', errorPath, `Cannot read "${relative}": ${error.code ?? error.message}`);
  }
  return bytes;
}

function decodeUtf8(bytes, errorPath) {
  const text = bytes.toString('utf8');
  if (!Buffer.from(text, 'utf8').equals(bytes)) {
    throw new ExportError('invalid-encoding', errorPath, 'Public page sources must be valid UTF-8');
  }
  return text;
}

/** Maps the shared content-root refusal onto this producer's error contract. */
function scanPublicContentRoot(rootReal) {
  try {
    return scanContentRoot(rootReal);
  } catch (error) {
    if (error instanceof PublicTreeError) throw new ExportError(error.code, error.path, error.message);
    throw error;
  }
}

/** Reads the leading `---` block with the pinned YAML parser; never reimplements YAML. */
function parsePageDocument(text, page) {
  const normalized = text.replace(/\r\n/g, '\n');
  const lines = normalized.split('\n');
  if (lines[0] !== '---') {
    throw new ExportError('invalid-frontmatter', page.id, `Public page "${page.source}" must open with YAML frontmatter`);
  }
  const end = lines.indexOf('---', 1);
  if (end === -1) {
    throw new ExportError('invalid-frontmatter', page.id, `Frontmatter in "${page.source}" is not closed`);
  }
  let fields;
  try {
    fields = YAML.parse(lines.slice(1, end).join('\n'), { uniqueKeys: true });
  } catch (error) {
    throw new ExportError('invalid-frontmatter', page.id, `Invalid YAML frontmatter in "${page.source}": ${firstLine(error)}`);
  }
  if (fields === null || typeof fields !== 'object' || Array.isArray(fields)) {
    throw new ExportError('invalid-frontmatter', page.id, `Frontmatter in "${page.source}" must be a mapping`);
  }
  if (typeof fields.title !== 'string' || fields.title.trim() === '') {
    throw new ExportError('missing-metadata', page.id, `Public page "${page.source}" requires a non-empty title`);
  }
  if (typeof fields.description !== 'string' || fields.description.trim() === '') {
    throw new ExportError('missing-metadata', page.id, `Public page "${page.source}" requires a non-empty description`);
  }
  const body = lines.slice(end + 1).join('\n').replace(/^\n+/, '').replace(/\n+$/, '');
  return {
    title: singleLine(fields.title),
    description: singleLine(fields.description),
    body,
  };
}

function maskFences(lines) {
  const masked = [...lines];
  let open = null;
  for (let index = 0; index < lines.length; index += 1) {
    const match = FENCE_OPEN.exec(lines[index]);
    if (open !== null) {
      masked[index] = ' '.repeat(lines[index].length);
      if (match && match[1][0] === open.marker && match[1].length >= open.length && match[2].trim() === '') {
        open = null;
      }
      continue;
    }
    if (match) {
      open = { marker: match[1][0], length: match[1].length };
      masked[index] = ' '.repeat(lines[index].length);
    }
  }
  return masked;
}

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

function maskHtmlComments(lines) {
  const masked = [...lines];
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
        inside = true;
        break;
      }
      line = `${line.slice(0, start)}${' '.repeat(end + 3 - start)}${line.slice(end + 3)}`;
      cursor = end + 3;
    }
    masked[index] = line;
  }
  return masked;
}

function proseMask(lines) {
  return maskHtmlComments(maskFences(lines).map(maskInlineCode));
}

function unwrapDestination(raw) {
  return raw.startsWith('<') && raw.endsWith('>') ? raw.slice(1, -1) : raw;
}

function wrapDestination(original, value) {
  if (original.startsWith('<') || /\s/.test(value)) return `<${value}>`;
  return value;
}

/**
 * The repository-relative target of a relative destination, or `null` when the
 * destination is not a contained relative path (external, absolute, fragment,
 * unencoded). Mirrors the resolver's own join so classification inspects the
 * same target that becomes a repository source URL.
 */
function repositoryRelativeTarget(sourcePath, destination) {
  const hashIndex = destination.indexOf('#');
  const beforeHash = hashIndex === -1 ? destination : destination.slice(0, hashIndex);
  const queryIndex = beforeHash.indexOf('?');
  const pathPart = queryIndex === -1 ? beforeHash : beforeHash.slice(0, queryIndex);
  if (pathPart === '' || pathPart.startsWith('/') || /^[a-z][a-z0-9+.-]*:/i.test(pathPart)) return null;
  let decoded;
  try {
    decoded = decodeURIComponent(pathPart);
  } catch {
    return null;
  }
  if (decoded.startsWith('/') || decoded.includes('\\')) return null;
  return path.posix.normalize(path.posix.join(path.posix.dirname(sourcePath), decoded));
}

function rewriteDestination(rawDestination, context) {
  const destination = unwrapDestination(rawDestination);
  if (destination.trim() === '') return null;
  let resolved;
  try {
    resolved = resolveLink({
      manifest: context.manifest,
      repoRoot: context.repoRoot,
      sourcePath: context.page.source,
      href: destination,
      target: 'html',
      ref: context.ref,
    });
  } catch (error) {
    throw new ExportError(
      error.code ?? 'link-error',
      context.page.id,
      `Cannot rewrite link "${destination}" in "${context.page.source}": ${error.message}`,
    );
  }
  const repositoryTarget = repositoryRelativeTarget(context.page.source, destination);
  if (repositoryTarget !== null) {
    const classification = classifyLinkTarget(repositoryTarget, { source: context.page.source });
    if (!classification.allowed) {
      throw new ExportError(
        'private-link-target',
        context.page.id,
        `Cannot export link "${destination}" in "${context.page.source}": "${repositoryTarget}" is in a private or history tree`,
      );
    }
  }
  if (resolved === destination) return null;
  if (resolved.startsWith('/')) {
    return absoluteLinkUrl(context.manifest.site, context.manifest.base, resolved, context.page.id);
  }
  return resolved;
}

/** Rewrites inline links and reference definitions to absolute public URLs. */
function rewriteLinks(body, context) {
  const lines = body.split('\n');
  const masked = proseMask(lines);
  const rewritten = lines.map((line, index) => {
    const matches = [];
    for (const match of masked[index].matchAll(INLINE_LINK)) matches.push({ match, kind: 'inline' });
    for (const match of masked[index].matchAll(LINK_DEFINITION)) matches.push({ match, kind: 'definition' });
    matches.sort((left, right) => left.match.index - right.match.index);
    let result = '';
    let last = 0;
    for (const { match, kind } of matches) {
      if (match.index < last) continue;
      let replacement = null;
      if (kind === 'inline') {
        if (match[1] !== '!') {
          const target = rewriteDestination(match[3], context);
          if (target !== null) {
            replacement = `${match[1]}[${match[2]}](${wrapDestination(match[3], target)}${match[4] ?? ''})`;
          }
        }
      } else {
        const target = rewriteDestination(match[4], context);
        if (target !== null) {
          replacement = `${match[1]}[${match[2]}]:${match[3]}${wrapDestination(match[4], target)}${match[5]}`;
        }
      }
      if (replacement === null) continue;
      result += line.slice(last, match.index) + replacement;
      last = match.index + match[0].length;
    }
    return result + line.slice(last);
  });
  return rewritten.join('\n');
}

function escapeLinkText(value) {
  return value.replace(/\\/g, '\\\\').replace(/\[/g, '\\[').replace(/\]/g, '\\]');
}

/** Collects one allowlisted page entry and its Markdown twin. */
function collectPage(page, context, bytes) {
  if (!ALLOWED_CLASSES.has(page.class)) {
    throw new ExportError('not-allowlisted', page.id, `Page "${page.id}" has class ${JSON.stringify(page.class)}; only L and C are public`);
  }
  if (!page.source.startsWith(`${CONTENT_ROOT}/`) || !page.source.endsWith('.md')) {
    throw new ExportError(
      'not-allowlisted',
      page.id,
      `Page "${page.id}" is not a canonical source under "${CONTENT_ROOT}/"`,
    );
  }
  const selfSource = page.sources.find((entry) => entry.path === page.source);
  if (selfSource === undefined) {
    throw new ExportError(
      'stale-source',
      page.id,
      `Canonical source "${page.source}" is not registered in its own reviewed sources[] list`,
    );
  }
  const digest = sha256Hex(bytes);
  if (selfSource.sha256 !== digest) {
    throw new ExportError(
      'stale-source',
      page.id,
      `Reviewed sha256 for "${page.source}" is ${selfSource.sha256}, but the file is ${digest}; human review must refresh it`,
    );
  }

  const document = parsePageDocument(decodeUtf8(bytes, page.id), page);

  const expectedAppliesTo = context.manifest.docsVersion === DEV_VERSION ? UNRELEASED : context.manifest.docsVersion;
  if (page.appliesTo !== expectedAppliesTo) {
    throw new ExportError(
      'version-parity',
      page.id,
      `Page "${page.id}" applies to ${JSON.stringify(page.appliesTo)}, but docsVersion ${JSON.stringify(context.manifest.docsVersion)} requires ${JSON.stringify(expectedAppliesTo)}`,
    );
  }

  const selfHref = `./${path.posix.basename(page.source)}`;
  let htmlRoute;
  let markdownRoute;
  let sourceUrl;
  try {
    htmlRoute = resolveLink({
      manifest: context.manifest,
      repoRoot: context.repoRoot,
      sourcePath: page.source,
      href: selfHref,
      target: 'html',
      ref: context.ref,
    });
    markdownRoute = resolveLink({
      manifest: context.manifest,
      repoRoot: context.repoRoot,
      sourcePath: page.source,
      href: selfHref,
      target: 'markdown',
      ref: context.ref,
    });
    sourceUrl = resolveLink({
      manifest: context.manifest,
      repoRoot: context.repoRoot,
      sourcePath: page.source,
      href: selfHref,
      target: 'source',
      ref: context.ref,
    });
  } catch (error) {
    throw new ExportError(
      error.code ?? 'route-error',
      page.id,
      `Cannot resolve the canonical route for "${page.id}": ${error.message}`,
    );
  }

  const routeExpectation = expectedHtmlRoute(context.manifest.base, page.id);
  if (htmlRoute !== routeExpectation) {
    throw new ExportError(
      'base-parity',
      page.id,
      `Canonical route "${htmlRoute}" does not match the expected project-base route "${routeExpectation}"`,
    );
  }
  assertBaseContains(context.manifest.base, htmlRoute, page.id);
  assertBaseContains(context.manifest.base, markdownRoute, page.id);

  const basePrefix = context.manifest.base === '/' ? '/' : `${context.manifest.base}/`;
  const twinName = normalizeOutputPath(markdownRoute.slice(basePrefix.length), page.id);
  if (twinName !== expectedMarkdownName(page.id)) {
    throw new ExportError(
      'base-parity',
      page.id,
      `Markdown twin path "${twinName}" does not match the expected route "${expectedMarkdownName(page.id)}"`,
    );
  }

  const htmlUrl = absoluteUrl(context.manifest.site, htmlRoute, page.id);
  const markdownUrl = absoluteUrl(context.manifest.site, markdownRoute, page.id);

  const rewrittenBody = rewriteLinks(document.body, { ...context, page });
  const status = page.reviewStatus ?? null;
  const provenance = [
    `Canonical HTML: <${htmlUrl}>  `,
    `Source: <${sourceUrl}>  `,
    `Documentation version: ${context.manifest.docsVersion} · Applicability: ${page.appliesTo} · Reviewed source sha256: ${selfSource.sha256}${status === null ? '' : ` · Review status: ${status}`}`,
  ].join('\n');
  const twinContent = `<!-- generated-by: website/scripts/export.mjs; do-not-edit -->\n\n# ${document.title}\n\n${provenance}\n\n${rewrittenBody}\n`;

  return {
    entry: {
      id: page.id,
      class: page.class,
      type: page.type,
      audience: [...page.audience],
      title: document.title,
      description: document.description,
      url: htmlUrl,
      markdownUrl,
      source: page.source,
      sourceUrl,
      appliesTo: page.appliesTo,
      reviewStatus: status,
      reviewDigest: page.reviewDigest,
    },
    file: { path: twinName, content: twinContent },
  };
}

function canonicalOrder(left, right) {
  if (left.id === right.id) return 0;
  if (left.id === 'index') return -1;
  if (right.id === 'index') return 1;
  return compareText(left.id, right.id);
}

function baseFilePath(base, name) {
  return base === '/' ? `/${name}` : `${base}/${name}`;
}

function renderIndex(manifest, ref, entries) {
  const document = {
    schemaVersion: EXPORT_FORMAT_VERSION,
    kind: 'agentic-sdd-public-docs-index',
    site: manifest.site,
    base: manifest.base,
    docsVersion: manifest.docsVersion,
    sourceRef: ref,
    llms: absoluteUrl(manifest.site, baseFilePath(manifest.base, 'llms.txt'), 'index'),
    pages: entries.map((entry) => ({
      id: entry.id,
      class: entry.class,
      type: entry.type,
      audience: [...entry.audience],
      title: entry.title,
      description: entry.description,
      url: entry.url,
      markdownUrl: entry.markdownUrl,
      source: entry.source,
      sourceUrl: entry.sourceUrl,
      appliesTo: entry.appliesTo,
      reviewStatus: entry.reviewStatus,
      reviewDigest: entry.reviewDigest,
    })),
  };
  return `${JSON.stringify(document, null, 2)}\n`;
}

function renderLlms(manifest, entries) {
  const home = absoluteUrl(manifest.site, manifest.base === '/' ? '/' : `${manifest.base}/`, 'llms');
  const applicable = manifest.docsVersion === DEV_VERSION ? UNRELEASED : manifest.docsVersion;
  const lines = [
    '# Agentic-SDD',
    '',
    '> Public documentation for Agentic-SDD, generated from the canonical L/C page allowlist.',
    '> Historical, private and engineering-only material is excluded.',
    '>',
    `> Canonical HTML: ${home}`,
    `> Documentation version: ${manifest.docsVersion} · Page applicability: ${applicable}`,
    `> Machine-readable index: ${absoluteUrl(manifest.site, baseFilePath(manifest.base, 'index.json'), 'llms')}`,
    '',
    '## Pages',
    '',
  ];
  for (const entry of entries) {
    lines.push(`- [${escapeLinkText(entry.title)}](${entry.url}): ${entry.description}`);
  }
  return `${lines.join('\n')}\n`;
}

function validateOptions(options) {
  if (options === null || typeof options !== 'object' || Array.isArray(options)) {
    throw new ExportError('invalid-options', 'options', 'Expected an options object');
  }
  for (const key of Object.keys(options)) {
    if (!['repoRoot', 'manifestPath', 'ref', 'outDir'].includes(key)) {
      throw new ExportError('invalid-options', key, `Unknown option "${key}"`);
    }
  }
  if (typeof options.repoRoot !== 'string' || options.repoRoot.length === 0) {
    throw new ExportError('invalid-options', 'repoRoot', 'repoRoot must be a non-empty string');
  }
  if (options.manifestPath !== undefined && (typeof options.manifestPath !== 'string' || options.manifestPath.length === 0)) {
    throw new ExportError('invalid-options', 'manifestPath', 'manifestPath must be a non-empty string');
  }
  if (options.ref !== undefined && (typeof options.ref !== 'string' || options.ref.length === 0)) {
    throw new ExportError('invalid-options', 'ref', 'ref must be a non-empty Git ref');
  }
  if (options.outDir !== undefined && (typeof options.outDir !== 'string' || options.outDir.length === 0)) {
    throw new ExportError('invalid-options', 'outDir', 'outDir must be a non-empty string');
  }
}

/**
 * Builds the complete export in memory. Read-only apart from importing the
 * manifest and page sources; no output directory is created.
 */
export function buildExport(options = {}) {
  validateOptions(options);
  const repoRoot = resolveRepoRoot(options.repoRoot);
  const manifestPath = path.resolve(repoRoot, options.manifestPath ?? DEFAULT_MANIFEST);
  const manifest = loadManifest(manifestPath, { repoRoot, verifySources: false });

  const scanned = scanPublicContentRoot(repoRoot);
  const registered = new Map(manifest.pages.map((page) => [page.source, page]));
  for (const source of scanned) {
    if (!registered.has(source)) {
      throw new ExportError(
        'unregistered-page',
        source,
        `Content file "${source}" is not registered in the manifest; add it with human review or remove it`,
      );
    }
  }

  const ref = options.ref ?? (manifest.docsVersion === DEV_VERSION ? 'main' : manifest.docsVersion);
  const entries = [];
  const files = [];
  const excluded = [];
  const outputPaths = new Set();

  for (const page of manifest.pages) {
    if (!ALLOWED_CLASSES.has(page.class)) {
      excluded.push({ id: page.id, class: page.class });
      continue;
    }
    const bytes = readContainedRegularFile(repoRoot, page.source, page.id);
    const collected = collectPage(page, { manifest, repoRoot, ref }, bytes);
    normalizeOutputPath(collected.file.path, page.id);
    if (outputPaths.has(collected.file.path)) {
      throw new ExportError('duplicate-output', collected.file.path, `Duplicate export path "${collected.file.path}"`);
    }
    outputPaths.add(collected.file.path);
    entries.push(collected.entry);
    files.push(collected.file);
  }

  entries.sort(canonicalOrder);

  files.push({ path: 'index.json', content: renderIndex(manifest, ref, entries) });
  files.push({ path: 'llms.txt', content: renderLlms(manifest, entries) });
  files.sort((left, right) => compareText(left.path, right.path));

  for (const file of files) normalizeOutputPath(file.path, file.path);
  for (const name of OUTPUT_NAMES) {
    if (!files.some((file) => file.path === name)) {
      throw new ExportError('missing-output', name, `Required export file "${name}" was not produced`);
    }
  }

  return {
    manifest,
    ref,
    pages: entries,
    excluded,
    files,
  };
}

function assertNoSymlinkAncestors(absolute) {
  const parsed = path.parse(absolute);
  let current = parsed.root;
  for (const segment of absolute.slice(parsed.root.length).split(path.sep).filter((entry) => entry.length > 0)) {
    current = path.join(current, segment);
    let stats;
    try {
      stats = lstatSync(current);
    } catch (error) {
      if (error.code === 'ENOENT') return;
      throw new ExportError('unreadable-output', absolute, `Cannot inspect "${current}": ${error.code ?? error.message}`);
    }
    if (stats.isSymbolicLink()) {
      throw new ExportError('symlink-output', absolute, `Output path resolves through symlink "${current}"`);
    }
  }
}

function assertSafeOutDir(repoRoot, outDir) {
  const absolute = path.resolve(outDir);
  if (absolute === repoRoot) {
    throw new ExportError('unsafe-out-dir', outDir, 'The output directory must not be the repository root');
  }
  const contentRoot = path.join(repoRoot, CONTENT_ROOT);
  const relative = path.relative(contentRoot, absolute);
  if (relative === '' || (!relative.startsWith('..') && !path.isAbsolute(relative))) {
    throw new ExportError('unsafe-out-dir', outDir, 'The output directory must not be inside the canonical content root');
  }
  assertNoSymlinkAncestors(absolute);
  let stats;
  try {
    stats = lstatSync(absolute);
  } catch (error) {
    if (error.code === 'ENOENT') return { absolute, exists: false };
    throw new ExportError('unreadable-output', outDir, `Cannot inspect "${outDir}": ${error.code ?? error.message}`);
  }
  if (stats.isSymbolicLink()) {
    throw new ExportError('symlink-output', outDir, 'The output directory must not be a symlink');
  }
  if (!stats.isDirectory()) {
    throw new ExportError('not-a-directory', outDir, 'The output path exists and is not a directory');
  }
  return { absolute, exists: true };
}

function writeContainedFile(outRoot, relative, content, index) {
  const segments = relative.split('/');
  let current = outRoot;
  for (const segment of segments.slice(0, -1)) {
    current = path.join(current, segment);
    let stats;
    try {
      stats = lstatSync(current);
    } catch (error) {
      if (error.code !== 'ENOENT') {
        throw new ExportError('unreadable-output', relative, `Cannot inspect "${current}": ${error.code ?? error.message}`);
      }
      mkdirSync(current);
      continue;
    }
    if (stats.isSymbolicLink()) {
      throw new ExportError('symlink-output', relative, `Refusing symlinked output directory "${current}"`);
    }
    if (!stats.isDirectory()) {
      throw new ExportError('not-a-directory', relative, `"${current}" is not a directory`);
    }
  }
  const target = path.join(outRoot, ...segments);
  try {
    const stats = lstatSync(target);
    if (stats.isSymbolicLink()) {
      throw new ExportError('symlink-output', relative, `Refusing to overwrite symlink "${target}"`);
    }
    if (!stats.isFile()) {
      throw new ExportError('not-a-regular-file', relative, `Refusing to overwrite non-file "${target}"`);
    }
  } catch (error) {
    if (error instanceof ExportError) throw error;
    if (error.code !== 'ENOENT') {
      throw new ExportError('unreadable-output', relative, `Cannot inspect "${target}": ${error.code ?? error.message}`);
    }
  }
  const temporary = `${target}.tmp-${process.pid}-${index}`;
  try {
    writeFileSync(temporary, content, { flag: 'wx', mode: 0o644 });
    renameSync(temporary, target);
  } finally {
    try {
      unlinkSync(temporary);
    } catch (error) {
      if (error.code !== 'ENOENT') throw error;
    }
  }
}

/** Writes the export into `outDir`, refusing symlinks and special files. */
export function writeExport(options = {}) {
  const { outDir } = options;
  if (typeof outDir !== 'string' || outDir.length === 0) {
    throw new ExportError('missing-out', '--out', '--out <dir> is required');
  }
  const plan = buildExport(pickBuildOptions(options));
  const { absolute, exists } = assertSafeOutDir(resolveRepoRoot(options.repoRoot), outDir);
  if (!exists) mkdirSync(absolute, { recursive: true });
  const written = [];
  const unchanged = [];
  plan.files.forEach((file, index) => {
    const target = path.join(absolute, ...file.path.split('/'));
    let current = null;
    try {
      const stats = lstatSync(target);
      if (stats.isSymbolicLink()) {
        throw new ExportError('symlink-output', file.path, `Refusing to overwrite symlink "${target}"`);
      }
      if (!stats.isFile()) {
        throw new ExportError('not-a-regular-file', file.path, `Refusing to overwrite non-file "${target}"`);
      }
      if (stats.nlink > 1) {
        throw new ExportError('hardlink-output', file.path, `Refusing to overwrite hard-linked "${target}"`);
      }
      current = readFileSync(target, 'utf8');
    } catch (error) {
      if (error instanceof ExportError) throw error;
      if (error.code !== 'ENOENT') {
        throw new ExportError('unreadable-output', file.path, `Cannot read "${target}": ${error.code ?? error.message}`);
      }
    }
    if (current === file.content) {
      unchanged.push(file.path);
      return;
    }
    writeContainedFile(absolute, file.path, file.content, index);
    written.push(file.path);
  });
  return { written, unchanged };
}

function pickBuildOptions(options) {
  return {
    repoRoot: options.repoRoot,
    manifestPath: options.manifestPath,
    ref: options.ref,
  };
}

function readOutputFile(outRoot, relative) {
  const segments = relative.split('/');
  let current = outRoot;
  for (const segment of segments) {
    current = path.join(current, segment);
    let stats;
    try {
      stats = lstatSync(current);
    } catch (error) {
      if (error.code === 'ENOENT' || error.code === 'ENOTDIR') return { reason: 'missing' };
      return { reason: 'unreadable' };
    }
    if (stats.isSymbolicLink()) return { reason: 'symlink' };
    if (segment !== segments.at(-1) && !stats.isDirectory()) return { reason: 'not-a-directory' };
  }
  const stats = lstatSync(path.join(outRoot, ...segments));
  if (!stats.isFile()) return { reason: 'not-a-regular-file' };
  if (stats.nlink > 1) return { reason: 'hardlink' };
  return { reason: null, content: readFileSync(path.join(outRoot, ...segments), 'utf8') };
}

/**
 * Walks the whole output directory and reports every entry outside the exact
 * planned set. Symlinks are reported rather than followed; the deferred
 * `llms-full.txt` keeps its own diagnostic above.
 */
function listUnexpectedEntries(outRoot, expected) {
  const found = [];
  const visit = (dir, prefix) => {
    let entries;
    try {
      entries = readdirSync(dir, { withFileTypes: true });
    } catch {
      return;
    }
    entries.sort((left, right) => compareText(left.name, right.name));
    for (const entry of entries) {
      const relative = prefix ? `${prefix}/${entry.name}` : entry.name;
      const absolute = path.join(dir, entry.name);
      let stats;
      try {
        stats = lstatSync(absolute);
      } catch {
        continue;
      }
      if (stats.isSymbolicLink()) {
        if (!expected.has(relative)) found.push({ path: relative, reason: 'symlink-output' });
        continue;
      }
      if (stats.isDirectory()) {
        visit(absolute, relative);
        continue;
      }
      if (expected.has(relative) || OUT_OF_SCOPE_OUTPUTS.has(relative)) continue;
      found.push({ path: relative, reason: 'unexpected-output' });
    }
  };
  visit(outRoot, '');
  return found.sort((left, right) => compareText(left.path, right.path));
}

/**
 * Compares the planned export with `outDir` without writing anything. The first
 * difference is reported by relative path and reason. Every entry outside the
 * exact planned set — extra files, nested extras and symlinks — is reported
 * rather than ignored; the deferred `llms-full.txt` keeps its own reason.
 */
export function checkExport(options = {}) {
  const { outDir } = options;
  if (typeof outDir !== 'string' || outDir.length === 0) {
    throw new ExportError('missing-out', '--out', '--out <dir> is required');
  }
  const plan = buildExport(pickBuildOptions(options));
  const { absolute, exists } = assertSafeOutDir(resolveRepoRoot(options.repoRoot), outDir);
  const differences = [];
  if (!exists) {
    differences.push({ path: '.', reason: 'missing-output-directory' });
  } else {
    for (const file of plan.files) {
      const observed = readOutputFile(absolute, file.path);
      if (observed.reason !== null) {
        differences.push({ path: file.path, reason: observed.reason });
        continue;
      }
      if (observed.content !== file.content) differences.push({ path: file.path, reason: 'differs' });
    }
    if (existsSync(path.join(absolute, 'llms-full.txt'))) {
      differences.push({ path: 'llms-full.txt', reason: 'deferred-output' });
    }
    const expected = new Set(plan.files.map((file) => file.path));
    for (const unexpected of listUnexpectedEntries(absolute, expected)) {
      differences.push(unexpected);
    }
  }
  return {
    ok: differences.length === 0,
    differences,
    firstDifference: differences[0] ?? null,
    expectedCount: plan.files.length,
  };
}

function parseArgs(args) {
  const options = { check: false, json: false };
  for (let index = 0; index < args.length; index += 1) {
    const arg = args[index];
    if (arg === '--check') {
      if (options.check) throw new ExportError('invalid-options', arg, 'Choose --check at most once');
      options.check = true;
      continue;
    }
    if (arg === '--json') {
      options.json = true;
      continue;
    }
    if (arg === '--out' || arg === '--repo' || arg === '--ref') {
      const value = args[index + 1];
      if (!value || value.startsWith('--')) throw new ExportError('invalid-options', arg, `Missing value for ${arg}`);
      index += 1;
      const key = arg === '--out' ? 'outDir' : arg === '--repo' ? 'repoRoot' : 'ref';
      if (options[key] !== undefined) throw new ExportError('invalid-options', arg, `${arg} may be given once`);
      options[key] = value;
      continue;
    }
    throw new ExportError('invalid-options', arg, `Unknown export option "${arg}"`);
  }
  if (options.outDir === undefined) {
    throw new ExportError('missing-out', '--out', '--out <dir> is required; the export never defaults to a writable location');
  }
  options.repoRoot ??= path.resolve(fileURLToPath(new URL('../..', import.meta.url)));
  options.outDir = path.resolve(options.outDir);
  return options;
}

async function main(args) {
  const options = parseArgs(args);
  const runOptions = {
    repoRoot: options.repoRoot,
    outDir: options.outDir,
    ...(options.ref === undefined ? {} : { ref: options.ref }),
  };
  if (options.check) {
    const result = checkExport(runOptions);
    if (options.json) {
      console.log(JSON.stringify({ mode: 'check', ...result }, null, 2));
    } else if (result.ok) {
      console.log(`export: ${result.expectedCount} file(s) match`);
    } else {
      const first = result.firstDifference;
      console.error(`export: drift in ${first.path}: ${first.reason}`);
    }
    if (!result.ok) process.exitCode = 1;
    return;
  }
  const result = writeExport(runOptions);
  if (options.json) {
    console.log(JSON.stringify({ mode: 'write', ...result }, null, 2));
  } else {
    console.log(`export: ${result.written.length} file(s) written, ${result.unchanged.length} unchanged`);
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main(process.argv.slice(2)).catch((error) => {
    console.error(`export: ${error.message}`);
    process.exitCode = 1;
  });
}
