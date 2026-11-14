// SPEC020 T09/T10 (F5, F10) — one public-tree contract shared by the
// documentation checker and the public export.
//
// Guarantees:
// - `scanContentRoot` enumerates the canonical public content root exactly as
//   the export producer does: only regular `.md` files, sorted, with symlinks,
//   special files, `.mdx` and a directory named `*.md` refused. A page that is
//   not enumerated here can neither be checked nor exported.
// - `classifyLinkTarget` is the single private/history classifier for
//   repository-relative link targets. Public engineering sources
//   (`skills-files/`, `opencode-pack/`, `cmd/`, `internal/`, `tests/`, …) stay
//   linkable through their repository source URL; private trees (`backups/`,
//   `.agentic-sdd/`, credentials/secrets), generated output (`website/dist/`,
//   `website/.astro/`) and history/engineering records (`specs/`,
//   `docs/handovers/`, `docs/testing/`) are refused.
// - Exactly one documented exception keeps the retained MCP migration stub's
//   deliberate historical-spec pointer working. It is keyed by both the source
//   page and the exact target, so no content page can reuse it. Every other
//   `specs/` link fails closed.
//
// Node built-ins only, synchronous, read-only and offline.

import { lstatSync, readdirSync } from 'node:fs';
import path from 'node:path';

export const CONTENT_ROOT = 'website/src/content/docs';

const PRIVATE_SEGMENTS = new Set([
  'backups',
  '.git',
  '.agentic-sdd',
  'node_modules',
  'coverage',
  'credentials',
  'secrets',
]);
const PRIVATE_PREFIXES = ['website/.astro/', 'website/dist/'];
const HISTORY_PREFIXES = ['specs/', 'docs/handovers/', 'docs/testing/'];
const CREDENTIAL_FILE = /^(?:\.env(?:\..+)?|\.npmrc|\.netrc|id_(?:rsa|ed25519|ecdsa|dsa)(?:\.pub)?)$/i;
const CREDENTIAL_EXTENSION = /\.(?:pem|key|p12|pfx|keystore|jks)$/i;

// The retained MCP migration stub records its historical Feature 015
// requirements source on purpose. This is the only permitted private/history
// link, and only from that exact retained file; public pages never inherit it.
export const DOCUMENTED_LINK_EXCEPTIONS = Object.freeze([
  Object.freeze({
    source: 'docs/mcp-tools.md',
    target: 'specs/015-fleet-manager-tui/spec.md',
    reason:
      'Retained MCP migration stub: the unchanged historical Feature 015 spec is the recorded requirements source. ' +
      'Public pages must state the fact instead of linking private/history material.',
  }),
]);

export class PublicTreeError extends Error {
  constructor(code, errorPath, message) {
    super(message);
    this.name = 'PublicTreeError';
    this.code = code;
    this.path = errorPath;
  }
}

function compareText(left, right) {
  if (left < right) return -1;
  if (left > right) return 1;
  return 0;
}

function isUnder(relative, prefix) {
  return relative === prefix.slice(0, -1) || relative.startsWith(prefix);
}

/**
 * Lists the Markdown sources below the canonical public content root. Symlinks
 * and special files are refused at any depth; only `.md` sources can be
 * canonical pages, and `.mdx` is refused rather than silently skipped.
 */
export function scanContentRoot(rootReal) {
  const contentAbs = path.join(rootReal, CONTENT_ROOT);
  const found = [];
  const visit = (absolute, relativeDir) => {
    let entries;
    try {
      entries = readdirSync(absolute, { withFileTypes: true });
    } catch (error) {
      if (error.code === 'ENOENT') {
        throw new PublicTreeError('missing-content-root', CONTENT_ROOT, `Missing canonical content root "${CONTENT_ROOT}"`);
      }
      throw new PublicTreeError(
        'unreadable-content-root',
        CONTENT_ROOT,
        `Cannot read "${CONTENT_ROOT}": ${error.code ?? error.message}`,
      );
    }
    entries.sort((left, right) => compareText(left.name, right.name));
    for (const entry of entries) {
      const relative = relativeDir ? `${relativeDir}/${entry.name}` : entry.name;
      const entryAbs = path.join(absolute, entry.name);
      const stats = lstatSync(entryAbs);
      if (stats.isSymbolicLink()) {
        throw new PublicTreeError(
          'symlink-path',
          `${CONTENT_ROOT}/${relative}`,
          `Refusing symlinked public content entry "${relative}"`,
        );
      }
      if (stats.isDirectory()) {
        if (entry.name.toLowerCase().endsWith('.md')) {
          throw new PublicTreeError(
            'not-a-regular-file',
            `${CONTENT_ROOT}/${relative}`,
            'A ".md" content entry must be a regular page source, not a directory',
          );
        }
        visit(entryAbs, relative);
        continue;
      }
      if (!stats.isFile()) {
        throw new PublicTreeError('not-a-regular-file', `${CONTENT_ROOT}/${relative}`, `Refusing special file "${relative}"`);
      }
      if (entry.name.toLowerCase().endsWith('.mdx')) {
        throw new PublicTreeError(
          'unsupported-source',
          `${CONTENT_ROOT}/${relative}`,
          'Canonical public pages are ".md"; ".mdx" is not in the published allowlist',
        );
      }
      if (entry.name.endsWith('.md')) found.push(`${CONTENT_ROOT}/${relative}`);
    }
  };
  visit(contentAbs, '');
  return found.sort(compareText);
}

/**
 * Classifies one repository-relative target. Returns
 * `{ allowed: true, category: 'public' | 'documented-exception' }` or
 * `{ allowed: false, category: 'private' | 'history' }`. `source` is the
 * repository-relative Markdown file holding the link, when known.
 */
export function classifyLinkTarget(relative, { source = null } = {}) {
  if (typeof relative !== 'string' || relative.length === 0) {
    return { allowed: false, category: 'private' };
  }
  const segments = relative.split('/');
  const filename = segments[segments.length - 1];
  let category = null;
  if (
    segments.some((segment) => PRIVATE_SEGMENTS.has(segment)) ||
    CREDENTIAL_FILE.test(filename) ||
    CREDENTIAL_EXTENSION.test(filename)
  ) {
    category = 'private';
  } else if (HISTORY_PREFIXES.some((prefix) => isUnder(relative, prefix))) {
    category = 'history';
  } else if (PRIVATE_PREFIXES.some((prefix) => isUnder(relative, prefix))) {
    category = 'private';
  }
  if (category === null) return { allowed: true, category: 'public' };
  if (
    source !== null &&
    DOCUMENTED_LINK_EXCEPTIONS.some((entry) => entry.source === source && entry.target === relative)
  ) {
    return { allowed: true, category: 'documented-exception' };
  }
  return { allowed: false, category };
}
