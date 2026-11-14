// Shared glue for the Spec 020 B4-W1-G independent negative tests.
//
// Scope: fixture paths, producer loading, a mutation runner, a front-matter
// claim scanner and honest `blocked` / `integration-pending` reporting. It
// deliberately does not re-implement any production validator: behavioural
// expectations come from specs/020-documentation-ecosystem/plan.md (C1–C4),
// the page contract in sitemap.md and the frozen B4 worker contract.
//
// A producer that is absent from this worktree is reported as
// `integration-pending`; a case whose producer exists but genuinely lacks the
// behaviour its fixture assumes is `blocked` with a reason naming the missing
// capability and its owning task. Neither is ever a behavioural pass, and
// neither is ever converted into a plain skip.

import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import {
  assertApiError,
  capture,
  clone,
  isErrorCode,
  isNonEmptyString,
  loadFrozenApi,
  readJson,
  snapshot,
} from '../../foundation-negative-support/support.mjs';

export { loadFrozenApi, readJson };

export const libDir = path.dirname(fileURLToPath(import.meta.url));
export const fixturesRoot = path.dirname(libDir);
export const testsDir = path.dirname(fixturesRoot);
export const repoRoot = path.resolve(testsDir, '..', '..');

export const PUBLIC_DOCS = 'website/src/content/docs';
export const BASELINE_MANIFEST = path.join(fixturesRoot, 'manifests', 'baseline.json');

/** Real producers exercised by these negative cases. */
export const PRODUCERS = {
  check: 'website/scripts/check.mjs',
  examples: 'website/scripts/examples.mjs',
  export: 'website/scripts/export.mjs',
};

/**
 * Test options for a case whose enforcement point is a not-yet-present
 * producer. `absent` is the honest integration-pending reason; `unadapted`
 * must name the missing capability and its owning task when the producer has
 * landed without implementing that behaviour.
 */
export function blockedOptions(producerKey, absent, unadapted = 'producer present but this adapter is not wired yet') {
  const rel = PRODUCERS[producerKey];
  const reason = fs.existsSync(path.join(repoRoot, rel))
    ? `blocked: ${unadapted} (${rel})`
    : `integration-pending: ${rel} is not present in this worktree; ${absent}`;
  return { skip: reason };
}

/**
 * Load a real producer module. Reports `integration-pending` when the module is
 * absent and fails loudly if it exists but dropped a frozen export.
 */
export async function loadProducer(producerKey, exportNames = []) {
  const rel = PRODUCERS[producerKey];
  if (rel === undefined) throw new Error(`Unknown producer key: ${producerKey}`);
  return loadFrozenApi(rel, exportNames);
}

/** Error entries with one diagnostic code, optionally restricted to one path. */
export function errorsWith(result, code, entryPath) {
  assert.ok(result && Array.isArray(result.errors), 'producer result must expose an errors array');
  return result.errors.filter(
    (entry) => entry.code === code && (entryPath === undefined || entry.path === entryPath),
  );
}

/** Assert at least one diagnostic with a code (and optional path) exists. */
export function requireError(result, code, entryPath) {
  const matches = errorsWith(result, code, entryPath);
  const reported = [...new Set((result?.errors ?? []).map((entry) => entry.code))];
  assert.ok(
    matches.length > 0,
    `${code} was not reported${entryPath === undefined ? '' : ` for ${entryPath}`}; reported: ${reported.join(', ') || '(none)'}`,
  );
  return matches;
}

/**
 * Create an owned temporary repository tree for a producer fixture that needs
 * real files (a manifest plus contained Markdown) rather than a JSON sketch.
 * `cleanup()` is idempotent and must run even when an assertion fails.
 */
export function tempRepo(files) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'asdx-negative-repo-'));
  const cleanup = () => fs.rmSync(root, { recursive: true, force: true });
  try {
    for (const [relative, content] of Object.entries(files)) {
      const normalized = path.posix.normalize(relative);
      assert.ok(
        !path.isAbsolute(relative) && normalized !== '..' && !normalized.startsWith('../'),
        `tempRepo path must stay inside the owned root: ${relative}`,
      );
      const absolute = path.join(root, ...normalized.split('/'));
      fs.mkdirSync(path.dirname(absolute), { recursive: true });
      fs.writeFileSync(absolute, content);
    }
  } catch (error) {
    cleanup();
    throw error;
  }
  return { root, cleanup };
}

/** Register a rejection case against a frozen validator export. */
export function expectRejected(api, label, mutate, options = {}) {
  test(`rejects ${label}`, { skip: api.skip }, () => {
    const mod = api.mod;
    const manifest = clone(options.manifest ?? readJson(BASELINE_MANIFEST));
    mutate(manifest);
    const { threw, error } = capture(() => mod.validateManifest(manifest, options.validate ?? {}));
    assert.ok(threw, `${label}: validateManifest accepted the mutation`);
    assertApiError(error, { name: 'ManifestError', fields: { code: isErrorCode, path: () => true } });
    if (options.pathMentions) {
      assert.match(
        `${error.path} ${error.message}`,
        options.pathMentions,
        `${label}: diagnostic must identify the offending field (${error.path})`,
      );
    }
    return error;
  });
}

// ---------------------------------------------------------------------------
// Front-matter claims
// ---------------------------------------------------------------------------

/**
 * Top-level `key:` names of a YAML front-matter block. Only key detection is
 * implemented: the suite never claims to parse YAML.
 */
export function frontmatterKeys(text) {
  if (!text.startsWith('---\n')) return [];
  const end = text.indexOf('\n---', 4);
  if (end === -1) return [];
  return text
    .slice(4, end)
    .split('\n')
    .filter((line) => /^[A-Za-z0-9_-]+\s*:/.test(line))
    .map((line) => line.slice(0, line.indexOf(':')).trim());
}

/** Claim keys no page may carry without a recorded human review. */
export const FORGED_CLAIM_KEYS = [
  'approval',
  'approvals',
  'approved',
  'approvedBy',
  'acceptedBy',
  'reviewStatus',
  'reviewed',
  'reviewedAt',
  'reviewedBy',
  'humanReview',
];

/** Return the forged claim keys present in one page's front matter. */
export function forgedClaims(text) {
  const keys = frontmatterKeys(text);
  return FORGED_CLAIM_KEYS.filter((key) => keys.includes(key));
}

/** Every file under a directory; symlinks are reported rather than followed. */
export function walkFiles(dir) {
  const out = [];
  if (!fs.existsSync(dir)) return out;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const abs = path.join(dir, entry.name);
    if (entry.isSymbolicLink()) {
      out.push({ path: abs, symlink: true });
    } else if (entry.isDirectory()) {
      out.push(...walkFiles(abs));
    } else if (entry.isFile()) {
      out.push({ path: abs, symlink: false });
    }
  }
  return out;
}

/** Read a fixture file relative to the fixture root. */
export function fixture(relPath) {
  return fs.readFileSync(path.join(fixturesRoot, relPath), 'utf8');
}

/** Read and parse a fixture JSON document relative to the fixture root. */
export function fixtureJson(relPath) {
  return readJson(path.join(fixturesRoot, relPath));
}

export const rel = (abs) => path.relative(repoRoot, abs).split(path.sep).join('/');
