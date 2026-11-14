import path from 'node:path';

import { RouteError, resolveLink } from '../scripts/lib/routes.mjs';

const RAW_HTML_NAVIGATION = /<a(?=[\s>])/i;
const RAW_HTML_HREF = /<a(?=[\s>])[^>]*\bhref\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))/i;
const IMAGE_REFERENCE = /\.(?:avif|bmp|gif|ico|jpe?g|png|svg|webp)$/i;
const SCHEME = /^([a-z][a-z0-9+.-]*):/i;

export default function remarkLinkRoutes(options = {}) {
  const manifestPath = options?.manifestPath;
  const repoRoot = options?.repoRoot;

  if (typeof manifestPath !== 'string' || manifestPath.trim() === '') {
    throw new RouteError('plugin-options', 'remarkLinkRoutes requires a manifestPath');
  }
  if (typeof repoRoot !== 'string' || repoRoot.trim() === '') {
    throw new RouteError('plugin-options', 'remarkLinkRoutes requires a repoRoot');
  }

  const root = path.resolve(repoRoot);
  const resolvedManifestPath = path.isAbsolute(manifestPath) ? manifestPath : path.resolve(root, manifestPath);
  let manifestPromise = null;

  const loadManifest = (source) => {
    if (manifestPromise === null) {
      manifestPromise = (async () => {
        let manifestModule;
        try {
          manifestModule = await import('../scripts/lib/manifest.mjs');
        } catch (error) {
          throw new RouteError(
            'manifest-module-unavailable',
            'Cannot load the sibling manifest helper required for link resolution',
            { source, cause: error },
          );
        }

        if (typeof manifestModule.loadManifest !== 'function') {
          throw new RouteError('manifest-module-invalid', 'The sibling manifest helper does not export loadManifest', {
            source,
          });
        }

        try {
          return await manifestModule.loadManifest(resolvedManifestPath, { repoRoot: root, verifySources: false });
        } catch (error) {
          throw new RouteError('manifest-invalid', `Cannot load documentation manifest: ${error.message}`, {
            source,
            cause: error,
          });
        }
      })();
    }
    return manifestPromise;
  };

  return async function transformer(tree, file) {
    const source = sourcePathFor(file, root);
    if (source === null) {
      throw new RouteError('missing-source-context', 'A Markdown vfile path is required for link resolution');
    }

    validateRawHtmlNavigation(tree, source);
    validateUniqueHeadings(tree, source);

    const linkNodes = [];
    walk(tree, (node) => {
      if (node.type === 'link' || node.type === 'definition') {
        linkNodes.push(node);
      }
    });

    const updates = [];
    for (const node of linkNodes) {
      const href = node.url;
      if (typeof href !== 'string' || href.length === 0 || isPassThroughHref(href)) {
        continue;
      }

      let manifest;
      try {
        manifest = await loadManifest(source);
      } catch (error) {
        throw atNode(error, node, source);
      }

      try {
        updates.push({
          node,
          url: resolveLink({
            manifest,
            repoRoot: root,
            sourcePath: source,
            href,
            target: 'html',
            ref: 'main',
          }),
        });
      } catch (error) {
        throw atNode(error, node, source);
      }
    }

    for (const update of updates) {
      update.node.url = update.url;
    }
  };
}

function sourcePathFor(file, repoRoot) {
  const candidate =
    typeof file?.path === 'string' && file.path.length > 0
      ? file.path
      : Array.isArray(file?.history) && typeof file.history[0] === 'string' && file.history[0].length > 0
        ? file.history[0]
        : undefined;
  if (typeof candidate !== 'string' || candidate.length === 0) {
    return null;
  }
  return path.isAbsolute(candidate) ? candidate : path.resolve(repoRoot, candidate);
}

function walk(node, visit) {
  if (node === null || typeof node !== 'object') {
    return;
  }
  visit(node);
  if (Array.isArray(node.children)) {
    for (const child of node.children) {
      walk(child, visit);
    }
  }
}

function validateRawHtmlNavigation(tree, source) {
  walk(tree, (node) => {
    if (node.type !== 'html' || typeof node.value !== 'string' || !RAW_HTML_NAVIGATION.test(node.value)) {
      return;
    }
    throw atNode(
      new RouteError(
        'raw-html-navigation',
        'Raw HTML navigation links are not allowed; use Markdown links so routes can be resolved',
        { source, href: extractRawHref(node.value) },
      ),
      node,
      source,
    );
  });
}

function validateUniqueHeadings(tree, source) {
  const seen = new Map();
  walk(tree, (node) => {
    if (node.type !== 'heading') {
      return;
    }
    const slug = slugify(headingText(node)) || 'section';
    if (seen.has(slug)) {
      throw atNode(
        new RouteError('duplicate-heading', `Duplicate authored heading slug "#${slug}"; use a unique heading`, {
          source,
          href: `#${slug}`,
        }),
        node,
        source,
      );
    }
    seen.set(slug, node);
  });
}

function headingText(node) {
  if (node === null || typeof node !== 'object') {
    return '';
  }
  if (node.type === 'text' || node.type === 'inlineCode') {
    return typeof node.value === 'string' ? node.value : '';
  }
  if (node.type === 'image') {
    return typeof node.alt === 'string' ? node.alt : '';
  }
  if (node.type === 'break') {
    return ' ';
  }
  if (node.type === 'html' && typeof node.value === 'string') {
    return node.value.replace(/<[^>]*>/g, '');
  }
  if (Array.isArray(node.children)) {
    return node.children.map((child) => headingText(child)).join('');
  }
  return '';
}

function slugify(value) {
  return value
    .normalize('NFKC')
    .trim()
    .toLowerCase()
    .replace(/\s/gu, '-')
    .replace(/[^\p{L}\p{N}\p{M}_-]+/gu, '');
}

function extractRawHref(value) {
  const match = RAW_HTML_HREF.exec(value);
  return match ? (match[1] ?? match[2] ?? match[3] ?? null) : null;
}

function isPassThroughHref(href) {
  if (href.startsWith('#')) {
    return true;
  }

  const hashIndex = href.indexOf('#');
  const beforeHash = hashIndex === -1 ? href : href.slice(0, hashIndex);
  const queryIndex = beforeHash.indexOf('?');
  const pathPart = queryIndex === -1 ? beforeHash : beforeHash.slice(0, queryIndex);

  if (pathPart === '') {
    return true;
  }

  const schemeMatch = SCHEME.exec(pathPart);
  if (schemeMatch) {
    return /^(https|mailto)$/i.test(schemeMatch[1]);
  }

  if (pathPart.startsWith('//')) {
    return false;
  }

  return IMAGE_REFERENCE.test(pathPart);
}

function atNode(error, node, source) {
  const routeError =
    error instanceof RouteError
      ? error
      : new RouteError('link-resolution-failed', error?.message ?? 'Link resolution failed', {
          source,
          href: typeof node?.url === 'string' ? node.url : null,
          cause: error,
        });

  if (routeError.source === null) {
    routeError.source = source;
  }
  if (routeError.href === null && typeof node?.url === 'string') {
    routeError.href = node.url;
  }

  routeError.file = source;

  const start = node?.position?.start;
  if (start && Number.isInteger(start.line)) {
    routeError.line = start.line;
    routeError.column = Number.isInteger(start.column) ? start.column : 1;
    routeError.message = `${routeError.message} (${source}:${routeError.line}:${routeError.column})`;
  }

  return routeError;
}
