// Deploy-artifact accounting for Feature 020 plan §Q3: total artifact bytes,
// bucket sizes, and the compressed JS+CSS actually referenced by a page.
//
// Everything here is an observation: no size or timing threshold is asserted.
// Plan §6-§7 budgets remain targets until the parent measures and compares them.

import { createHash } from 'node:crypto';
import { existsSync, lstatSync, readdirSync, readFileSync } from 'node:fs';
import { isAbsolute, join, relative, resolve, sep } from 'node:path';
import { brotliCompressSync, constants as zlibConstants, gzipSync } from 'node:zlib';

import { toPageId } from './pagefind.mjs';

export const ATTRIBUTE_RE = /([a-zA-Z_:][-a-zA-Z0-9_:.]*)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>]+))/g;

export function sha256File(path) {
	return createHash('sha256').update(readFileSync(path)).digest('hex');
}

function bucketFor(relPath) {
	if (relPath.startsWith('pagefind/')) return 'pagefind';
	if (relPath.startsWith('_astro/')) return 'astro-assets';
	if (relPath.toLowerCase().endsWith('.html')) return 'html';
	return 'other';
}

// Deterministic, symlink-free walk. Symlinks are rejected rather than followed so
// a supplied dist cannot pull in files from outside the build.
export function walkArtifact(distDir) {
	const root = resolve(distDir);
	const files = [];
	const stack = [{ abs: root, rel: '' }];
	while (stack.length > 0) {
		const current = stack.pop();
		const entries = readdirSync(current.abs, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name, 'en'));
		for (const entry of entries) {
			const abs = join(current.abs, entry.name);
			const relPath = current.rel === '' ? entry.name : `${current.rel}/${entry.name}`;
			const stat = lstatSync(abs);
			if (stat.isSymbolicLink() || entry.isSymbolicLink()) {
				throw new Error(`artifact walk refused symlink: ${relPath}`);
			}
			if (stat.isDirectory()) {
				stack.push({ abs, rel: relPath });
				continue;
			}
			if (!stat.isFile()) {
				throw new Error(`artifact walk refused special file: ${relPath}`);
			}
			files.push({ relPath, bytes: stat.size, sha256: sha256File(abs), bucket: bucketFor(relPath) });
		}
	}
	files.sort((a, b) => (a.relPath < b.relPath ? -1 : a.relPath > b.relPath ? 1 : 0));

	const byBucket = { pagefind: { files: 0, bytes: 0 }, 'astro-assets': { files: 0, bytes: 0 }, html: { files: 0, bytes: 0 }, other: { files: 0, bytes: 0 } };
	let totalBytes = 0;
	for (const file of files) {
		byBucket[file.bucket].files += 1;
		byBucket[file.bucket].bytes += file.bytes;
		totalBytes += file.bytes;
	}

	const digest = createHash('sha256');
	for (const file of files) {
		digest.update(`${file.relPath}\u0000${file.bytes}\u0000${file.sha256}\n`);
	}

	return {
		distDir: root,
		files,
		htmlFiles: files.filter((file) => file.relPath.toLowerCase().endsWith('.html')).map((file) => file.relPath),
		fileCount: files.length,
		totalBytes,
		byBucket,
		digest: digest.digest('hex'),
	};
}

export function compressSize(bytes) {
	const gzip = gzipSync(bytes, { level: 9 }).length;
	const brotli = brotliCompressSync(bytes, {
		params: { [zlibConstants.BROTLI_PARAM_QUALITY]: zlibConstants.BROTLI_MAX_QUALITY },
	}).length;
	return { raw: bytes.length, gzip, brotli };
}

export function parseAttributes(tagSource) {
	const attributes = new Map();
	for (const match of tagSource.matchAll(ATTRIBUTE_RE)) {
		const name = match[1].toLowerCase();
		const value = match[2] ?? match[3] ?? match[4] ?? '';
		if (!attributes.has(name)) {
			attributes.set(name, decodeHtmlEntities(value));
		}
	}
	return attributes;
}

function decodeHtmlEntities(value) {
	return value
		.replace(/&#(\d+);/g, (_, code) => String.fromCodePoint(Number(code)))
		.replace(/&#x([0-9a-f]+);/gi, (_, code) => String.fromCodePoint(Number.parseInt(code, 16)))
		.replace(/&quot;/g, '"')
		.replace(/&#39;|&apos;/g, "'")
		.replace(/&lt;/g, '<')
		.replace(/&gt;/g, '>')
		.replace(/&amp;/g, '&');
}

// Extract the resources a page actually loads: external scripts, stylesheets and
// inline script/style blocks.
export function extractPageAssets(html) {
	const scripts = [];
	for (const match of html.matchAll(/<script\b([^>]*)>([\s\S]*?)<\/script>/gi)) {
		const attributes = parseAttributes(match[1]);
		const type = (attributes.get('type') ?? 'text/javascript').toLowerCase();
		const isJavaScript = type === '' || type.includes('javascript') || type === 'module';
		const src = attributes.get('src');
		if (src) {
			if (isJavaScript) {
				scripts.push(src);
			}
			continue;
		}
		if (isJavaScript) {
			scripts.push({ inline: match[2] });
		}
	}

	const stylesheets = [];
	for (const match of html.matchAll(/<link\b([^>]*?)\/?>/gi)) {
		const attributes = parseAttributes(match[1]);
		const rel = (attributes.get('rel') ?? '').toLowerCase().split(/\s+/);
		if (rel.includes('stylesheet') && attributes.get('href')) {
			stylesheets.push(attributes.get('href'));
		}
	}

	const inlineStyles = [];
	for (const match of html.matchAll(/<style\b[^>]*>([\s\S]*?)<\/style>/gi)) {
		inlineStyles.push(match[1]);
	}

	return { scripts, stylesheets, inlineStyles };
}

function stripQueryAndFragment(href) {
	return href.split('#')[0].split('?')[0];
}

function decodePath(value) {
	try {
		return decodeURIComponent(value);
	} catch {
		return value;
	}
}

// Resolve a page-relative or base-prefixed URL to a file inside the dist. Returns
// null for external/data references; throws when a local path escapes the dist.
export function resolveAssetPath(href, { distDir, base = '', pageDir = '' }) {
	const cleaned = stripQueryAndFragment(String(href).trim());
	if (cleaned === '' || cleaned.startsWith('#') || cleaned.startsWith('//')) {
		return null;
	}
	if (/^[a-z][a-z0-9+.-]*:/i.test(cleaned)) {
		return null;
	}
	let decoded = decodePath(cleaned);
	const normalizedBase = `/${String(base).replace(/^\/+|\/+$/g, '')}`;
	const root = resolve(distDir);
	const isRootRelative = decoded.startsWith('/');
	if (isRootRelative && normalizedBase !== '/' && (decoded === normalizedBase || decoded.startsWith(`${normalizedBase}/`))) {
		decoded = decoded.slice(normalizedBase.length);
	}
	const relativePath = decoded.replace(/^\/+/, '');
	const candidate = isRootRelative ? resolve(root, relativePath) : resolve(root, pageDir, relativePath);
	const relativeToRoot = relative(root, candidate);
	if (relativeToRoot === '' || relativeToRoot.startsWith('..') || isAbsolute(relativeToRoot)) {
		throw new Error(`asset path escapes the built site: ${href}`);
	}
	return { absPath: candidate, relPath: relativeToRoot.split(sep).join('/') };
}

// Compressed JS+CSS actually loaded by one page. Missing files are recorded, not
// silently skipped.
export function measurePageWeight({ distDir, base = '', relPath }) {
	const absHtml = resolve(distDir, relPath);
	if (!existsSync(absHtml)) {
		throw new Error(`page not found in built site: ${relPath}`);
	}
	const html = readFileSync(absHtml, 'utf8');
	const pageDir = relPath.includes('/') ? relPath.slice(0, relPath.lastIndexOf('/')) : '';
	const extracted = extractPageAssets(html);
	const missing = [];
	const seen = new Set();

	function measureExternal(href) {
		const resolved = resolveAssetPath(href, { distDir, base, pageDir });
		if (resolved === null || seen.has(resolved.relPath)) {
			return null;
		}
		seen.add(resolved.relPath);
		if (!existsSync(resolved.absPath)) {
			missing.push(resolved.relPath);
			return { href, relPath: resolved.relPath, missing: true, raw: 0, gzip: 0, brotli: 0 };
		}
		const sizes = compressSize(readFileSync(resolved.absPath));
		return { href, relPath: resolved.relPath, missing: false, ...sizes };
	}

	const scripts = extracted.scripts.map((entry) => (typeof entry === 'string' ? measureExternal(entry) : null)).filter(Boolean);
	const stylesheets = extracted.stylesheets.map(measureExternal).filter(Boolean);
	const inlineScriptBytes = extracted.scripts
		.filter((entry) => typeof entry === 'object')
		.reduce((total, entry) => total + Buffer.byteLength(entry.inline, 'utf8'), 0);
	const inlineStyleBytes = extracted.inlineStyles.reduce((total, content) => total + Buffer.byteLength(content, 'utf8'), 0);

	function totalsFor(entries) {
		return entries.reduce(
			(total, entry) => ({ raw: total.raw + entry.raw, gzip: total.gzip + entry.gzip, brotli: total.brotli + entry.brotli }),
			{ raw: 0, gzip: 0, brotli: 0 },
		);
	}

	const js = totalsFor(scripts);
	const css = totalsFor(stylesheets);
	return {
		pageId: toPageId(`/${relPath}`),
		relPath,
		htmlBytes: Buffer.byteLength(html, 'utf8'),
		js: { files: scripts, ...js },
		css: { files: stylesheets, ...css },
		inline: { scriptBytes: inlineScriptBytes, styleBytes: inlineStyleBytes },
		totals: {
			raw: js.raw + css.raw + inlineScriptBytes + inlineStyleBytes,
			gzip: js.gzip + css.gzip,
			brotli: js.brotli + css.brotli,
		},
		missing,
	};
}

export function pickDeepPage(htmlFiles, preferences = ['reference/cli/index.html', 'reference/cli.html']) {
	const candidates = htmlFiles.filter((relPath) => relPath !== 'index.html' && relPath !== '404.html');
	for (const preferred of preferences) {
		if (candidates.includes(preferred)) {
			return { relPath: preferred, selected: 'preferred' };
		}
	}
	if (candidates.length === 0) {
		return { relPath: null, selected: 'absent' };
	}
	return { relPath: [...candidates].sort()[0], selected: 'first-sorted' };
}

export function measureArtifact({ distDir, base = '', deepPagePreferences } = {}) {
	const artifact = walkArtifact(distDir);
	const deep = pickDeepPage(artifact.htmlFiles, deepPagePreferences);
	const pagePaths = ['index.html', ...(deep.relPath ? [deep.relPath] : [])];
	const pages = pagePaths
		.filter((relPath) => artifact.htmlFiles.includes(relPath))
		.map((relPath) => measurePageWeight({ distDir, base, relPath }));
	return {
		artifact: {
			fileCount: artifact.fileCount,
			totalBytes: artifact.totalBytes,
			byBucket: artifact.byBucket,
			htmlFiles: artifact.htmlFiles.length,
			digest: artifact.digest,
		},
		deepPage: deep,
		pages,
		missingAssets: pages.flatMap((page) => page.missing.map((relPath) => ({ page: page.pageId, relPath }))),
	};
}

export function round(value, digits = 3) {
	return Number(Number(value).toFixed(digits));
}

export function summarizeSamples(samples) {
	if (!Array.isArray(samples) || samples.length === 0) {
		return { count: 0, min: null, median: null, max: null };
	}
	const sorted = [...samples].map((value) => round(value)).sort((a, b) => a - b);
	const middle = Math.floor(sorted.length / 2);
	const median = sorted.length % 2 === 0 ? round((sorted[middle - 1] + sorted[middle]) / 2) : sorted[middle];
	return { count: sorted.length, min: sorted[0], median, max: sorted[sorted.length - 1] };
}

export function timeSync(fn) {
	const started = process.hrtime.bigint();
	const value = fn();
	return { value, elapsedMs: round(Number(process.hrtime.bigint() - started) / 1e6) };
}

export async function timeAsync(fn) {
	const started = process.hrtime.bigint();
	const value = await fn();
	return { value, elapsedMs: round(Number(process.hrtime.bigint() - started) / 1e6) };
}

