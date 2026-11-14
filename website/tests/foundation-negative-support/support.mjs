// Shared support for the Spec 020 B1-W1-D independent counterexample tests.
//
// Scope: fixture/loading glue only. It never re-implements the production
// manifest or routing algorithms; expectations come from the frozen contract in
// specs/020-documentation-ecosystem/plan.md C1 and the B1 worker contract.
//
// Missing sibling modules (or their npm dependencies) are reported as
// `integration-pending` and skip the affected assertions. A skipped test is
// never a behavioural pass, and a missing import is never red evidence.

import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

export const supportDir = path.dirname(fileURLToPath(import.meta.url));
export const testsDir = path.dirname(supportDir);
export const fixturesRoot = path.join(testsDir, 'foundation-negative-fixtures');
export const fixtureTree = path.join(fixturesRoot, 'tree');
export const manifestsDir = path.join(fixturesRoot, 'manifests');
export const repoRoot = path.resolve(testsDir, '..', '..');

export const PUBLIC_DOCS = 'website/src/content/docs';
export const INDEX_SOURCE = `${PUBLIC_DOCS}/index.md`;
export const VALID_MANIFEST = path.join(manifestsDir, 'valid.json');

/** Read and parse a JSON fixture. */
export function readJson(file) {
  return JSON.parse(fs.readFileSync(file, 'utf8'));
}

/**
 * Load a frozen-contract module. A missing module file, a missing transitive
 * dependency, or a module-resolution failure is `integration-pending`.
 * Any other failure (syntax error, throwing module body) propagates: that is
 * real red evidence about a module that does exist.
 */
export async function loadFrozenApi(relPath, exportNames = []) {
  const abs = path.join(repoRoot, relPath);
  if (!fs.existsSync(abs)) {
    return {
      ok: false,
      skip: `integration-pending: ${relPath} is not present in this worktree`,
      mod: null,
    };
  }

  let mod;
  try {
    mod = await import(pathToFileURL(abs).href);
  } catch (error) {
    if (error?.code === 'ERR_MODULE_NOT_FOUND' || error?.code === 'ERR_UNSUPPORTED_DIR_IMPORT') {
      return {
        ok: false,
        skip: `integration-pending: ${relPath} cannot resolve a module (${firstLine(error.message)})`,
        mod: null,
      };
    }
    throw error;
  }

  const missing = exportNames.filter((name) => mod[name] === undefined);
  if (missing.length > 0) {
    // A module that exists but no longer exports the frozen contract has drifted.
    assert.fail(`${relPath} is missing frozen export(s): ${missing.join(', ')}`);
  }

  return { ok: true, skip: false, mod };
}

function firstLine(text) {
  return String(text ?? '').split('\n')[0];
}

/** Console note so an integration-pending file is visible in raw output. */
export function reportPending(file, api) {
  if (!api.ok) console.error(`[foundation-negative] ${file}: ${api.skip}`);
}

/** Deep clone a fixture object. */
export function clone(value) {
  return structuredClone(value);
}

/** Recursively freeze a fixture so accidental input mutation surfaces. */
export function deepFreeze(value) {
  if (value && typeof value === 'object' && !Object.isFrozen(value)) {
    Object.freeze(value);
    for (const key of Object.keys(value)) deepFreeze(value[key]);
  }
  return value;
}

/** Snapshot a JSON-like value for mutation checks. */
export function snapshot(value) {
  return JSON.stringify(value);
}

/** Capture either a thrown error or a returned value. */
export function capture(fn) {
  try {
    return { threw: false, value: fn() };
  } catch (error) {
    return { threw: true, error };
  }
}

/** Async variant of capture for a possibly-async call. */
export async function captureAsync(fn) {
  try {
    return { threw: false, value: await fn() };
  } catch (error) {
    return { threw: true, error };
  }
}

const errorName = (error) => error?.name ?? error?.constructor?.name;

/**
 * Assert an error carries the frozen contract's diagnostic fields.
 *
 * The contract fixes the error *names* (ManifestError / RouteError) and the
 * required fields, not extra structure, so the class check tolerates a module
 * that exposes only the name.
 */
export function assertApiError(error, { name, fields = {} }) {
  assert.ok(error instanceof Error, `expected ${name}, received ${String(error)}`);
  assert.equal(errorName(error), name, `expected error name ${name}, received ${errorName(error)}`);
  for (const [field, check] of Object.entries(fields)) {
    assert.ok(field in error, `${name} must expose "${field}" (received keys: ${Object.keys(error).join(', ')})`);
    if (check !== null) {
      assert.ok(check(error[field]), `${name}.${field} failed check: ${String(error[field])}`);
    }
  }
  assert.equal(typeof error.message, 'string');
  assert.ok(error.message.length > 0, `${name} must include a useful message`);
}

export const isNonEmptyString = (value) => typeof value === 'string' && value.trim().length > 0;
export const isErrorCode = (value) => typeof value === 'string' && /^[a-z0-9][a-z0-9_-]*$/i.test(value);

/** Crypto digest of a file's raw bytes — the documented source-hash meaning. */
export function fileDigest(absPath) {
  return createHash('sha256').update(fs.readFileSync(absPath)).digest('hex');
}

/** Owned scratch root for runtime fixtures (symlinks, temporary trees). */
export function scratchRoot() {
  const base = process.env.FOUNDATION_NEGATIVE_TMP || '/tmp/opencode';
  fs.mkdirSync(base, { recursive: true });
  return fs.mkdtempSync(path.join(base, 'spec020-foundation-'));
}
