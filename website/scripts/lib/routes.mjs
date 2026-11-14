import fs from 'node:fs';
import path from 'node:path';

const CONTENT_ROOT = 'website/src/content/docs';
const IMAGE_REFERENCE = /\.(?:avif|bmp|gif|ico|jpe?g|png|svg|webp)$/i;
const CONTROL_CHARACTER = /[\u0000-\u001f\u007f]/;
const SCHEME = /^([a-z][a-z0-9+.-]*):/i;
const ALLOWED_EXTERNAL_SCHEMES = new Set(['https', 'mailto']);
const TARGETS = new Set(['html', 'markdown', 'source']);

export class RouteError extends Error {
  constructor(code, message, details = {}) {
    super(message);
    this.name = 'RouteError';
    this.code = code;
    this.source = details.source ?? null;
    this.href = details.href ?? null;
    if (details.cause) this.cause = details.cause;
  }
}

export function resolveLink(options = {}) {
  const {
    manifest,
    repoRoot,
    sourcePath,
    href,
    target = 'html',
    ref = 'main',
  } = options ?? {};

  const sourceForError = typeof sourcePath === 'string' ? sourcePath : null;
  const hrefForError = typeof href === 'string' ? href : null;
  const manifestData = normalizeManifest(manifest, sourceForError, hrefForError);

  if (!TARGETS.has(target)) {
    throw new RouteError('invalid-target', `Unsupported link target "${String(target)}"`, {
      source: sourceForError,
      href: hrefForError,
    });
  }

  const sourceRelative = normalizeSourcePath(repoRoot, sourcePath, hrefForError);
  const parsed = parseHref(href, sourceForError);

  if (parsed.passthrough) {
    return href;
  }

  if (parsed.external) {
    return href;
  }

  if (parsed.image) {
    return href;
  }

  const decodedPath = decodeHrefPath(parsed.pathPart, sourceForError, href);
  const repositoryPath = resolveRepositoryPath(sourceRelative, decodedPath, sourceForError, href);
  assertNoSymlinkComponents(repoRoot, repositoryPath, sourceForError, href);

  const inContentRoot =
    repositoryPath === CONTENT_ROOT || repositoryPath.startsWith(`${CONTENT_ROOT}/`);
  const isMarkdown = repositoryPath.endsWith('.md');

  if (isMarkdown) {
    const page = manifestData.pagesBySource.get(repositoryPath);
    if (page) {
      if (target === 'html') {
        return addSuffix(htmlRoute(manifestData.base, page.id), parsed, sourceForError, href);
      }
      if (target === 'markdown') {
        return addSuffix(markdownRoute(manifestData.base, page.id), parsed, sourceForError, href);
      }
      return addSuffix(
        sourceUrl(manifestData.repository, ref, repositoryPath, sourceForError, href),
        parsed,
        sourceForError,
        href,
      );
    }

    if (inContentRoot) {
      throw new RouteError(
        'unknown-target',
        `No manifest page maps the public Markdown source "${repositoryPath}"`,
        { source: sourceForError, href },
      );
    }
  }

  if (target === 'source' || !inContentRoot) {
    return addSuffix(
      sourceUrl(manifestData.repository, ref, repositoryPath, sourceForError, href),
      parsed,
      sourceForError,
      href,
    );
  }

  throw new RouteError(
    'unsupported-relative-link',
    `Relative non-Markdown public link "${href}" has no supported route; use an image reference or a manifest page`,
    { source: sourceForError, href },
  );
}

function normalizeManifest(manifest, source, href) {
  if (manifest === null || typeof manifest !== 'object' || Array.isArray(manifest)) {
    throw new RouteError('invalid-manifest', 'Manifest must be an object', { source, href });
  }

  const base = normalizeBase(manifest.base, source, href);
  const repository = normalizeRepository(manifest.repository, source, href);

  if (!Array.isArray(manifest.pages)) {
    throw new RouteError('invalid-manifest', 'Manifest pages must be an array', { source, href });
  }

  const pagesBySource = new Map();
  const pagesById = new Map();

  for (const page of manifest.pages) {
    if (page === null || typeof page !== 'object' || Array.isArray(page)) {
      throw new RouteError('invalid-manifest', 'Manifest page entries must be objects', { source, href });
    }
    if (typeof page.id !== 'string' || page.id.length === 0) {
      throw new RouteError('invalid-manifest', 'Manifest page id must be a non-empty string', { source, href });
    }
    if (typeof page.source !== 'string' || page.source.length === 0) {
      throw new RouteError('invalid-manifest', `Manifest page "${page.id}" needs a source`, { source, href });
    }

    const id = normalizeRouteId(page.id, source, href);
    const pageSource = normalizeManifestSource(page.source, page.id, source, href);

    if (pagesById.has(id)) {
      throw new RouteError('duplicate-id', `Duplicate manifest page id "${id}"`, { source, href });
    }
    if (pagesBySource.has(pageSource)) {
      throw new RouteError('duplicate-source', `Duplicate manifest source "${pageSource}"`, { source, href });
    }

    pagesById.set(id, page);
    pagesBySource.set(pageSource, page);
  }

  return { base, repository, pagesBySource, pagesById };
}

function normalizeBase(base, source, href) {
  if (typeof base !== 'string') {
    throw new RouteError('invalid-manifest', 'Manifest base must be a string', { source, href });
  }
  if (base === '') {
    return '';
  }
  if (!base.startsWith('/') || base.startsWith('//') || base.includes('?') || base.includes('#')) {
    throw new RouteError('invalid-manifest', `Manifest base "${base}" must be a root or project path prefix`, {
      source,
      href,
    });
  }
  if (CONTROL_CHARACTER.test(base) || /\s/u.test(base) || base.includes('\\')) {
    throw new RouteError('invalid-manifest', `Manifest base "${base}" contains unsafe characters`, { source, href });
  }

  const normalized = base === '/' ? '/' : base.replace(/\/+$/, '');
  for (const segment of normalized.split('/').filter(Boolean)) {
    if (segment === '.' || segment === '..' || /%2e|%2f|%5c/i.test(segment)) {
      throw new RouteError('invalid-manifest', `Manifest base "${base}" has an unsafe segment`, { source, href });
    }
  }

  return normalized === '/' ? '' : normalized;
}

function normalizeRepository(repository, source, href) {
  if (typeof repository !== 'string' || repository.length === 0 || CONTROL_CHARACTER.test(repository)) {
    throw new RouteError('invalid-manifest', 'Manifest repository must be an HTTPS GitHub URL', { source, href });
  }

  let parsed;
  try {
    parsed = new URL(repository);
  } catch {
    throw new RouteError('invalid-manifest', `Manifest repository "${repository}" is not a valid URL`, {
      source,
      href,
    });
  }

  if (
    parsed.protocol !== 'https:' ||
    parsed.hostname.toLowerCase() !== 'github.com' ||
    parsed.port !== '' ||
    parsed.username !== '' ||
    parsed.password !== '' ||
    parsed.search !== '' ||
    parsed.hash !== ''
  ) {
    throw new RouteError('invalid-manifest', `Manifest repository "${repository}" must be a fixed HTTPS GitHub URL`, {
      source,
      href,
    });
  }

  const pathname = parsed.pathname.replace(/\/+$/, '');
  if (!/^\/[^/]+\/[^/]+$/.test(pathname)) {
    throw new RouteError('invalid-manifest', `Manifest repository "${repository}" must name one repository`, {
      source,
      href,
    });
  }

  return `${parsed.origin}${pathname}`;
}

function normalizeRouteId(id, source, href) {
  if (CONTROL_CHARACTER.test(id) || id.includes('\\') || id.includes('?') || id.includes('#')) {
    throw new RouteError('invalid-manifest', `Manifest page id "${id}" contains unsafe characters`, { source, href });
  }
  if (id.startsWith('/') || id.endsWith('/')) {
    throw new RouteError('invalid-manifest', `Manifest page id "${id}" must be relative without a trailing slash`, {
      source,
      href,
    });
  }
  for (const segment of id.split('/')) {
    if (segment === '' || segment === '.' || segment === '..') {
      throw new RouteError('invalid-manifest', `Manifest page id "${id}" has an unsafe segment`, { source, href });
    }
  }
  return id;
}

function normalizeManifestSource(sourcePath, id, source, href) {
  let candidate = sourcePath;
  while (candidate.startsWith('./')) {
    candidate = candidate.slice(2);
  }

  if (
    CONTROL_CHARACTER.test(candidate) ||
    candidate.includes('\\') ||
    candidate.startsWith('/') ||
    /^[a-z]:\//i.test(candidate)
  ) {
    throw new RouteError('invalid-manifest', `Manifest page "${id}" source "${sourcePath}" is not repository-relative`, {
      source,
      href,
    });
  }

  const normalized = path.posix.normalize(candidate);
  if (normalized === '.' || normalized === '..' || normalized.startsWith('../')) {
    throw new RouteError('invalid-manifest', `Manifest page "${id}" source "${sourcePath}" escapes the repository`, {
      source,
      href,
    });
  }

  return normalized;
}

function normalizeSourcePath(repoRoot, sourcePath, href) {
  if (typeof sourcePath !== 'string' || sourcePath.trim() === '') {
    throw new RouteError('missing-source-context', 'A current Markdown source path is required', {
      source: null,
      href,
    });
  }
  if (CONTROL_CHARACTER.test(sourcePath)) {
    throw new RouteError('invalid-source', 'The current source path contains unsafe characters', {
      source: sourcePath,
      href,
    });
  }

  let relative;
  if (path.isAbsolute(sourcePath)) {
    if (typeof repoRoot !== 'string' || repoRoot.length === 0) {
      throw new RouteError('path-escape', 'An absolute source path requires repoRoot for containment', {
        source: sourcePath,
        href,
      });
    }
    const root = path.resolve(repoRoot);
    const absolute = path.resolve(sourcePath);
    const nativeRelative = path.relative(root, absolute);
    if (
      nativeRelative === '' ||
      nativeRelative === '..' ||
      nativeRelative.startsWith(`..${path.sep}`) ||
      path.isAbsolute(nativeRelative)
    ) {
      throw new RouteError('path-escape', `Source path "${sourcePath}" is outside repoRoot`, {
        source: sourcePath,
        href,
      });
    }
    relative = nativeRelative.split(path.sep).join('/');
  } else {
    relative = sourcePath;
  }

  const normalized = normalizeRepositoryRelative(relative, sourcePath, href);
  if (normalized === '') {
    throw new RouteError('missing-source-context', 'A current Markdown source file is required', {
      source: sourcePath,
      href,
    });
  }

  assertNoSymlinkComponents(repoRoot, normalized, sourcePath, href);
  return normalized;
}

function normalizeRepositoryRelative(value, source, href) {
  let candidate = value;
  while (candidate.startsWith('./')) {
    candidate = candidate.slice(2);
  }

  if (
    candidate.includes('\\') ||
    candidate.startsWith('/') ||
    /^[a-z]:\//i.test(candidate) ||
    CONTROL_CHARACTER.test(candidate)
  ) {
    throw new RouteError('path-escape', `Path "${value}" is not a safe repository-relative path`, { source, href });
  }

  const normalized = path.posix.normalize(candidate);
  if (normalized === '.' || normalized === '..' || normalized.startsWith('../')) {
    throw new RouteError('path-escape', `Path "${value}" escapes the repository`, { source, href });
  }

  return normalized;
}

function parseHref(href, source) {
  if (typeof href !== 'string' || href.trim() === '') {
    throw new RouteError('invalid-href', 'Link href must be a non-empty string', { source, href });
  }
  if (CONTROL_CHARACTER.test(href)) {
    throw new RouteError('unsafe-href', 'Link href contains control characters', { source, href });
  }

  const hashIndex = href.indexOf('#');
  const fragment = hashIndex === -1 ? null : href.slice(hashIndex + 1);
  const beforeHash = hashIndex === -1 ? href : href.slice(0, hashIndex);
  const queryIndex = beforeHash.indexOf('?');
  const query = queryIndex === -1 ? null : beforeHash.slice(queryIndex + 1);
  const pathPart = queryIndex === -1 ? beforeHash : beforeHash.slice(0, queryIndex);

  if (pathPart === '') {
    return { passthrough: true, pathPart, query, fragment };
  }

  const schemeMatch = SCHEME.exec(pathPart);
  if (schemeMatch) {
    if (ALLOWED_EXTERNAL_SCHEMES.has(schemeMatch[1].toLowerCase())) {
      return { external: true, pathPart, query, fragment };
    }
    throw new RouteError('unsafe-scheme', `Unsafe link scheme "${schemeMatch[1]}:"`, { source, href });
  }

  if (pathPart.startsWith('//')) {
    throw new RouteError('unsafe-scheme', 'Protocol-relative links are not allowed', { source, href });
  }
  if (IMAGE_REFERENCE.test(pathPart)) {
    return { image: true, pathPart, query, fragment };
  }
  if (pathPart.startsWith('/')) {
    throw new RouteError('path-escape', `Root-absolute link "${pathPart}" is not a repository-relative path`, {
      source,
      href,
    });
  }

  return { pathPart, query, fragment };
}

function decodeHrefPath(pathPart, source, href) {
  let decoded;
  try {
    decoded = decodeURIComponent(pathPart);
  } catch {
    throw new RouteError('invalid-encoding', `Link path "${pathPart}" is not valid percent-encoding`, { source, href });
  }

  if (CONTROL_CHARACTER.test(decoded) || decoded.includes('\\')) {
    throw new RouteError('unsafe-path', `Link path "${pathPart}" contains unsafe characters`, { source, href });
  }
  if (decoded.startsWith('/') || /^[a-z]:\//i.test(decoded)) {
    throw new RouteError('path-escape', `Link path "${pathPart}" is not repository-relative`, { source, href });
  }

  return decoded;
}

function resolveRepositoryPath(sourceRelative, decodedPath, source, href) {
  const joined = path.posix.normalize(path.posix.join(path.posix.dirname(sourceRelative), decodedPath));
  if (joined === '.' || joined === '..' || joined.startsWith('../') || joined.startsWith('/')) {
    throw new RouteError('path-escape', `Link path "${decodedPath}" escapes the repository`, { source, href });
  }
  return joined;
}

function assertNoSymlinkComponents(repoRoot, relativePath, source, href) {
  if (typeof repoRoot !== 'string' || repoRoot.length === 0) {
    return;
  }

  const root = path.resolve(repoRoot);
  let current = root;
  for (const segment of relativePath.split('/')) {
    current = path.join(current, segment);
    let stats;
    try {
      stats = fs.lstatSync(current);
    } catch (error) {
      if (error.code === 'ENOENT' || error.code === 'ENOTDIR') {
        return;
      }
      throw new RouteError('path-check-failed', `Could not inspect repository path "${relativePath}"`, {
        source,
        href,
        cause: error,
      });
    }
    if (stats.isSymbolicLink()) {
      throw new RouteError('symlink-path', `Symlinked repository path component is not allowed: "${relativePath}"`, {
        source,
        href,
      });
    }
  }
}

function htmlRoute(base, id) {
  if (id === 'index') {
    return `${base}/`;
  }
  return `${base}/${encodeRouteId(id)}/`;
}

function markdownRoute(base, id) {
  if (id === 'index') {
    return `${base}/index.md`;
  }
  return `${base}/${encodeRouteId(id)}/index.md`;
}

function encodeRouteId(id) {
  return id.split('/').map((segment) => encodeURIComponent(segment)).join('/');
}

function sourceUrl(repository, ref, repositoryPath, source, href) {
  return `${repository}/blob/${encodeRef(ref, source, href)}/${encodeRepositoryPath(repositoryPath)}`;
}

function encodeRef(ref, source, href) {
  if (typeof ref !== 'string' || ref.length === 0 || CONTROL_CHARACTER.test(ref) || ref.includes('\\')) {
    throw new RouteError('invalid-ref', 'Source link ref must be a non-empty Git ref', { source, href });
  }
  const segments = ref.split('/');
  for (const segment of segments) {
    if (segment === '' || segment === '.' || segment === '..') {
      throw new RouteError('invalid-ref', `Source link ref "${ref}" has an unsafe segment`, { source, href });
    }
  }
  return segments.map((segment) => encodeURIComponent(segment)).join('/');
}

function encodeRepositoryPath(repositoryPath) {
  return repositoryPath.split('/').map((segment) => encodeURIComponent(segment)).join('/');
}

function addSuffix(url, parsed, source, href) {
  let result = url;
  if (parsed.query !== null) {
    result += `?${encodeQuery(parsed.query, source, href)}`;
  }
  if (parsed.fragment !== null) {
    result += `#${encodeFragment(parsed.fragment, source, href)}`;
  }
  return result;
}

function encodeQuery(query, source, href) {
  try {
    return encodePreservingEscapes(query);
  } catch {
    throw new RouteError('invalid-encoding', 'Link query cannot be URL-encoded', { source, href });
  }
}

function encodeFragment(fragment, source, href) {
  try {
    return encodePreservingEscapes(fragment).replace(/#/g, '%23');
  } catch {
    throw new RouteError('invalid-encoding', 'Link fragment cannot be URL-encoded', { source, href });
  }
}

function encodePreservingEscapes(value) {
  let encoded = '';
  let lastIndex = 0;
  const escape = /%[0-9a-f]{2}/gi;
  let match = escape.exec(value);
  while (match !== null) {
    encoded += encodeURI(value.slice(lastIndex, match.index));
    encoded += match[0];
    lastIndex = match.index + match[0].length;
    match = escape.exec(value);
  }
  encoded += encodeURI(value.slice(lastIndex));
  return encoded;
}
