// Spec 020 B2-W1-D: the C3 fixed-argv example runner.
//
// The runner executes only the frozen, read-only `first-preview` example against
// an explicitly supplied absolute CLI path. It never searches PATH, never runs
// `apply`, never calls a provider or the network, and keeps every scratch path
// under an owned temporary root that is removed by default.
import { spawn } from 'node:child_process';
import { createHash } from 'node:crypto';
import {
  accessSync,
  chmodSync,
  constants,
  lstatSync,
  mkdirSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  readlinkSync,
  rmSync,
  statSync,
} from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

/** Stable process exit codes. Callers and documentation must rely on these. */
export const EXIT = Object.freeze({
  PASS: 0,
  FAIL: 1,
  REFUSED: 2,
  BLOCKED: 3,
});

export const DEFAULT_TIMEOUT_MS = 20_000;
export const DEFAULT_MAX_BUFFER = 64 * 1024;
export const HOME_PLACEHOLDER = '{home}';

const OWNED_ROOT_PREFIX = 'asdx-examples-';
const OWNED_DIRECTORIES = Object.freeze(['home', 'tmp', 'cache', 'config', 'data', 'state']);
const ALLOWED_EXAMPLE_COMMANDS = Object.freeze(['version', 'preview']);
const CODE_PATTERN = /```|~~~|`/;
const CONTROL_PATTERN = /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u001b\u009b]/g;
const MAX_REPORTED_TEXT = 400;

/** Invalid invocation: unknown ID, Markdown/code input, unsafe path shape. */
export class ExamplesRefusal extends Error {
  constructor(message) {
    super(message);
    this.name = 'ExamplesRefusal';
    this.exitCode = EXIT.REFUSED;
  }
}

/** The environment cannot run the example: missing or non-executable CLI. */
export class ExamplesBlocked extends Error {
  constructor(message) {
    super(message);
    this.name = 'ExamplesBlocked';
    this.exitCode = EXIT.BLOCKED;
  }
}

/**
 * Frozen example table. `argv` is an ordered list of argv templates; the
 * `{home}` token is replaced with the owned fixture home at run time. Manual
 * examples are documentation-only and are never executed.
 */
export const EXAMPLES = Object.freeze([
  Object.freeze({
    id: 'first-preview',
    effect: 'read-only',
    argv: Object.freeze([
      Object.freeze(['version']),
      Object.freeze(['preview', '--home', HOME_PLACEHOLDER]),
    ]),
    expected:
      '`version` and `preview --home <owned-temp>` both exit 0; the owned fixture home stays empty and byte-for-byte unchanged, so no skills were installed.',
  }),
  Object.freeze({
    id: 'direct-fix-illustration',
    effect: 'manual',
    argv: Object.freeze([]),
    expected:
      'Illustrative transcript of one small fix; the learner runs it with an AI agent of their choice.',
    reason:
      'illustrative only: it needs a human-supplied request and an AI agent, which this runner never invokes (no provider or network call).',
  }),
  Object.freeze({
    id: 'feature-approval',
    effect: 'manual',
    argv: Object.freeze([]),
    expected:
      'Illustrative spec/plan/tasks flow ending at one combined human approval; approval records are never fabricated.',
    reason:
      'illustrative only: it needs the learner\u2019s own feature documents and an explicit human approval, so it cannot be executed or certified here.',
  }),
]);

export const EXAMPLE_IDS = Object.freeze(EXAMPLES.map((example) => example.id));

for (const example of EXAMPLES) {
  for (const argv of example.argv) {
    if (!ALLOWED_EXAMPLE_COMMANDS.includes(argv[0])) {
      throw new Error(`Frozen example "${example.id}" invokes a non-allowlisted command: ${argv[0]}`);
    }
    if (argv.includes('apply')) throw new Error(`Frozen example "${example.id}" must never run apply`);
    if (argv[0] === 'preview' && JSON.stringify(argv) !== JSON.stringify(['preview', '--home', HOME_PLACEHOLDER])) {
      throw new Error('The frozen preview argv must stay exactly "preview --home {home}"');
    }
  }
}

function isPlainObject(value) {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) return false;
  const proto = Object.getPrototypeOf(value);
  return proto === Object.prototype || proto === null;
}

function assertNoCodeInput(value, label) {
  if (typeof value !== 'string') throw new ExamplesRefusal(`${label} must be a string`);
  if (CODE_PATTERN.test(value) || /[\r\n]/.test(value)) {
    throw new ExamplesRefusal(
      `Refusing ${label} ${JSON.stringify(value)}: Markdown fences, inline code and multi-line blocks are documentation, not executable examples. Only the frozen ids are runnable: ${EXAMPLE_IDS.join(', ')}.`,
    );
  }
}

/**
 * Validate an explicitly supplied binary path. Never resolves through PATH.
 * Missing/non-executable paths are `blocked`; unsafe shapes are `refused`.
 */
export function inspectBinary(binaryPath) {
  if (typeof binaryPath !== 'string' || binaryPath.trim() === '') {
    throw new ExamplesBlocked(
      'No --binary supplied. Pass an explicit absolute path to a built CLI (PATH search is never used), for example: go build -o /tmp/asdx-core ./cmd/agentic-sdd',
    );
  }
  if (binaryPath.includes('\0')) throw new ExamplesRefusal('--binary must not contain NUL bytes');
  if (!path.isAbsolute(binaryPath)) {
    throw new ExamplesRefusal(
      `--binary must be an explicit absolute path; refusing "${binaryPath}" (PATH search is never used).`,
    );
  }
  let stats;
  try {
    stats = lstatSync(binaryPath);
  } catch (error) {
    throw new ExamplesBlocked(
      `No executable CLI at "${binaryPath}" (${error.code ?? 'missing'}). Build it first (for example: go build -o /tmp/asdx-core ./cmd/agentic-sdd) or pass the correct absolute path; PATH search is never used.`,
    );
  }
  if (stats.isSymbolicLink()) {
    throw new ExamplesRefusal(`Refusing symlink --binary "${binaryPath}"; pass the real file path.`);
  }
  if (!stats.isFile()) {
    throw new ExamplesRefusal(`Refusing --binary "${binaryPath}": it is not a regular file.`);
  }
  try {
    accessSync(binaryPath, constants.X_OK);
  } catch (error) {
    throw new ExamplesBlocked(
      `CLI at "${binaryPath}" is not executable (${error.code ?? 'EACCES'}); chmod +x it or pass an executable build.`,
    );
  }
  return binaryPath;
}

/** Select frozen examples by exact id. Unknown ids and code-like input are refused. */
export function selectExamples({ only } = {}) {
  if (only === undefined || only === null) return EXAMPLES;
  assertNoCodeInput(only, '--only value');
  if (only === '') throw new ExamplesRefusal(`--only requires a value; available ids: ${EXAMPLE_IDS.join(', ')}.`);
  if (!EXAMPLE_IDS.includes(only)) {
    throw new ExamplesRefusal(
      `Unknown example id "${only}". Available ids: ${EXAMPLE_IDS.join(', ')}. Only the frozen example table is executable.`,
    );
  }
  return EXAMPLES.filter((example) => example.id === only);
}

/** Parse CLI flags. Unknown options and positional arguments are refused. */
export function parseArgs(argv) {
  if (!Array.isArray(argv)) throw new ExamplesRefusal('Arguments must be an array of strings');
  const options = { binaryPath: undefined, only: undefined, json: false, keep: false, help: false };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--binary' || arg === '--only') {
      const value = argv[index + 1];
      if (value === undefined || value === '' || value.startsWith('--')) {
        throw new ExamplesRefusal(`Missing value for ${arg}`);
      }
      if (arg === '--binary') options.binaryPath = value;
      else {
        assertNoCodeInput(value, '--only value');
        options.only = value;
      }
      index += 1;
    } else if (arg === '--json') options.json = true;
    else if (arg === '--keep') options.keep = true;
    else if (arg === '--help' || arg === '-h') options.help = true;
    else if (arg.startsWith('--')) throw new ExamplesRefusal(`Unknown example option: ${arg}`);
    else {
      assertNoCodeInput(arg, 'argument');
      throw new ExamplesRefusal(
        `Unexpected argument "${arg}". This runner has no passthrough argv; choose --only <id> from: ${EXAMPLE_IDS.join(', ')}.`,
      );
    }
  }
  return options;
}

function normalizeOptions(options) {
  if (!isPlainObject(options)) throw new Error('runExamples requires a plain options object');
  const timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  const maxBuffer = options.maxBuffer ?? DEFAULT_MAX_BUFFER;
  if (!Number.isInteger(timeoutMs) || timeoutMs <= 0) throw new Error('timeoutMs must be a positive integer');
  if (!Number.isInteger(maxBuffer) || maxBuffer <= 0) throw new Error('maxBuffer must be a positive integer');
  if (typeof options.keep !== 'undefined' && typeof options.keep !== 'boolean') {
    throw new Error('keep must be a boolean');
  }
  if (typeof options.json !== 'undefined' && typeof options.json !== 'boolean') {
    throw new Error('json must be a boolean');
  }
  const tmpBase = options.tmpBase === undefined ? os.tmpdir() : options.tmpBase;
  if (typeof tmpBase !== 'string' || !path.isAbsolute(tmpBase)) throw new Error('tmpBase must be an absolute path');
  return {
    binaryPath: options.binaryPath,
    only: options.only,
    keep: options.keep === true,
    json: options.json === true,
    timeoutMs,
    maxBuffer,
    tmpBase,
  };
}

function createOwnedRoot(tmpBase) {
  let baseStats;
  try {
    baseStats = statSync(tmpBase);
  } catch (error) {
    throw new Error(`Missing tmpBase "${tmpBase}": ${error.code ?? error.message}`);
  }
  if (!baseStats.isDirectory()) throw new Error(`tmpBase is not a directory: "${tmpBase}"`);
  const root = mkdtempSync(path.join(tmpBase, OWNED_ROOT_PREFIX));
  chmodSync(root, 0o700);
  for (const name of OWNED_DIRECTORIES) mkdirSync(path.join(root, name), { mode: 0o700 });
  return {
    root,
    home: path.join(root, 'home'),
  };
}

function removeOwnedRoot(root) {
  if (typeof root !== 'string' || !path.basename(root).startsWith(OWNED_ROOT_PREFIX)) {
    throw new Error(`Refusing to remove a path this runner does not own: "${root}"`);
  }
  rmSync(root, { recursive: true, force: true });
}

function buildEnvironment(owned) {
  return {
    PATH: '/usr/bin:/bin',
    LANG: 'C.UTF-8',
    LC_ALL: 'C.UTF-8',
    TERM: 'dumb',
    NO_COLOR: '1',
    COLUMNS: '80',
    HOME: owned.home,
    TMPDIR: path.join(owned.root, 'tmp'),
    XDG_CACHE_HOME: path.join(owned.root, 'cache'),
    XDG_CONFIG_HOME: path.join(owned.root, 'config'),
    XDG_DATA_HOME: path.join(owned.root, 'data'),
    XDG_STATE_HOME: path.join(owned.root, 'state'),
  };
}

function runBounded(binaryPath, argv, { cwd, env, timeoutMs, maxBuffer }) {
  const label = argv.join(' ');
  return new Promise((resolve, reject) => {
    const child = spawn(binaryPath, [...argv], {
      shell: false,
      cwd,
      env,
      stdio: ['ignore', 'pipe', 'pipe'],
      windowsHide: true,
    });
    const stdoutChunks = [];
    const stderrChunks = [];
    let stdoutBytes = 0;
    let stderrBytes = 0;
    let settled = false;
    let killTimer = null;

    const forceKill = () => {
      try {
        child.kill('SIGTERM');
      } catch {
        // Already exited.
      }
      killTimer = setTimeout(() => {
        try {
          child.kill('SIGKILL');
        } catch {
          // Already exited.
        }
      }, 100);
      killTimer.unref();
    };

    const finish = (error, result) => {
      if (settled) return;
      settled = true;
      clearTimeout(timeoutTimer);
      if (killTimer) clearTimeout(killTimer);
      if (error) reject(error);
      else resolve(result);
    };

    const timeoutTimer = setTimeout(() => {
      forceKill();
      finish(new Error(`Example step "${label}" timed out after ${timeoutMs}ms`));
    }, timeoutMs);
    timeoutTimer.unref();

    child.stdout.on('data', (chunk) => {
      stdoutBytes += chunk.length;
      if (stdoutBytes > maxBuffer) {
        forceKill();
        finish(new Error(`Example step "${label}" stdout exceeded maxBuffer (${maxBuffer} bytes)`));
        return;
      }
      stdoutChunks.push(chunk);
    });
    child.stderr.on('data', (chunk) => {
      stderrBytes += chunk.length;
      if (stderrBytes > maxBuffer) {
        forceKill();
        finish(new Error(`Example step "${label}" stderr exceeded maxBuffer (${maxBuffer} bytes)`));
        return;
      }
      stderrChunks.push(chunk);
    });
    child.on('error', (error) => finish(new Error(`Failed to execute example step "${label}": ${error.message}`)));
    child.on('close', (code, signal) => {
      if (signal) {
        finish(new Error(`Example step "${label}" terminated by signal ${signal}`));
        return;
      }
      finish(null, {
        exit: code,
        stdout: Buffer.concat(stdoutChunks).toString('utf8').replace(/\r\n/g, '\n'),
        stderr: Buffer.concat(stderrChunks).toString('utf8').replace(/\r\n/g, '\n'),
      });
    });
  });
}

function sanitizeText(text, replacements) {
  let output = String(text).replace(CONTROL_PATTERN, ' ');
  for (const [needle, label] of replacements) {
    if (needle) output = output.split(needle).join(label);
  }
  output = output.trim();
  if (output.length > MAX_REPORTED_TEXT) output = `${output.slice(0, MAX_REPORTED_TEXT)}\u2026`;
  return output;
}

function snapshotTree(root) {
  const entries = [];
  const walk = (directory, relative) => {
    const names = readdirSync(directory, { withFileTypes: true })
      .map((entry) => entry.name)
      .sort();
    for (const name of names) {
      const absolute = path.join(directory, name);
      const relativePath = relative === '' ? name : `${relative}/${name}`;
      const stats = lstatSync(absolute);
      if (stats.isSymbolicLink()) {
        entries.push({ path: relativePath, type: 'symlink', target: readlinkSync(absolute) });
      } else if (stats.isDirectory()) {
        entries.push({ path: relativePath, type: 'directory' });
        walk(absolute, relativePath);
      } else if (stats.isFile()) {
        entries.push({
          path: relativePath,
          type: 'file',
          size: stats.size,
          sha256: createHash('sha256').update(readFileSync(absolute)).digest('hex'),
        });
      } else {
        entries.push({ path: relativePath, type: 'other' });
      }
    }
  };
  walk(root, '');
  return entries;
}

function treeDigest(entries) {
  return createHash('sha256').update(JSON.stringify(entries)).digest('hex');
}

function treeDiff(before, after) {
  const beforeMap = new Map(before.map((entry) => [entry.path, JSON.stringify(entry)]));
  const afterMap = new Map(after.map((entry) => [entry.path, JSON.stringify(entry)]));
  return {
    added: after.filter((entry) => !beforeMap.has(entry.path)).map((entry) => entry.path),
    changed: after
      .filter((entry) => beforeMap.has(entry.path) && beforeMap.get(entry.path) !== afterMap.get(entry.path))
      .map((entry) => entry.path),
    removed: before.filter((entry) => !afterMap.has(entry.path)).map((entry) => entry.path),
  };
}

function manualResult(example) {
  return {
    id: example.id,
    effect: example.effect,
    status: 'manual',
    expected: example.expected,
    reason: example.reason,
    argv: [],
  };
}

function blockedResult(example, reason) {
  return {
    id: example.id,
    effect: example.effect,
    status: 'blocked',
    expected: example.expected,
    reason,
    argv: example.argv.map((argv) => [...argv]),
  };
}

async function runReadOnlyExample(example, binaryPath, owned, options) {
  const before = snapshotTree(owned.home);
  const replacements = [
    [owned.root, '{owned-temp}'],
    [os.homedir(), '{host-home}'],
  ];
  const steps = [];
  const fail = (reason) => {
    let current = before;
    try {
      current = snapshotTree(owned.home);
    } catch {
      // Keep the pre-run snapshot if the home cannot be read.
    }
    return {
      id: example.id,
      effect: example.effect,
      status: 'fail',
      expected: example.expected,
      reason,
      argv: example.argv.map((argv) => [...argv]),
      steps,
      home: { entries: current.length, digest: treeDigest(current) },
    };
  };

  try {
    for (const template of example.argv) {
      const argv = template.map((token) => (token === HOME_PLACEHOLDER ? owned.home : token));
      const run = await runBounded(binaryPath, argv, {
        cwd: owned.root,
        env: buildEnvironment(owned),
        timeoutMs: options.timeoutMs,
        maxBuffer: options.maxBuffer,
      });
      steps.push({
        argv: [...template],
        exit: run.exit,
        stdout: sanitizeText(run.stdout, replacements),
        stderr: sanitizeText(run.stderr, replacements),
      });
      const label = template.join(' ');
      if (run.exit !== 0) {
        const detail = sanitizeText(run.stderr, replacements).split('\n')[0];
        return fail(`Step "${label}" exited with code ${run.exit}${detail ? `: ${detail}` : ''}`);
      }
      if (run.stdout.trim() === '') {
        return fail(`Step "${label}" produced no stdout; the example asserts that the command result is reported.`);
      }
    }
  } catch (error) {
    return fail(error.message);
  }

  if (before.length !== 0) {
    return fail(`The owned fixture home was not empty before the run (${before.length} entries); refusing to judge non-mutation.`);
  }
  const after = snapshotTree(owned.home);
  const diff = treeDiff(before, after);
  if (diff.added.length || diff.changed.length || diff.removed.length) {
    const detail = [
      diff.added.length ? `added ${diff.added.slice(0, 5).join(', ')}` : '',
      diff.changed.length ? `changed ${diff.changed.slice(0, 5).join(', ')}` : '',
      diff.removed.length ? `removed ${diff.removed.slice(0, 5).join(', ')}` : '',
    ]
      .filter(Boolean)
      .join('; ');
    return fail(
      `preview modified the owned fixture home (${detail}). A preview must be read-only; never point --home at a real user home.`,
    );
  }

  return {
    id: example.id,
    effect: example.effect,
    status: 'pass',
    expected: example.expected,
    argv: example.argv.map((argv) => [...argv]),
    steps,
    home: { entries: after.length, digest: treeDigest(after) },
  };
}

/**
 * Run the selected examples. Resolves to a deterministic summary; the caller
 * maps `status` to EXIT. Missing binaries resolve to `blocked`, never a throw.
 */
export async function runExamples(options = {}) {
  const parsed = normalizeOptions(options);
  const selected = selectExamples({ only: parsed.only });
  const requiresBinary = selected.some((example) => example.effect !== 'manual');

  let binaryPath = null;
  let blockedReason = null;
  if (requiresBinary) {
    try {
      binaryPath = inspectBinary(parsed.binaryPath);
    } catch (error) {
      if (error instanceof ExamplesRefusal) throw error;
      blockedReason = error.message;
    }
  }

  const results = [];
  let owned = null;
  try {
    for (const example of selected) {
      if (example.effect === 'manual') {
        results.push(manualResult(example));
        continue;
      }
      if (blockedReason) {
        results.push(blockedResult(example, blockedReason));
        continue;
      }
      if (!owned) owned = createOwnedRoot(parsed.tmpBase);
      results.push(await runReadOnlyExample(example, binaryPath, owned, parsed));
    }
  } finally {
    if (owned && !parsed.keep) removeOwnedRoot(owned.root);
  }

  const status = results.some((result) => result.status === 'fail')
    ? 'fail'
    : results.some((result) => result.status === 'blocked')
      ? 'blocked'
      : 'pass';

  return {
    status,
    ok: status === 'pass',
    binaryPath: requiresBinary ? (parsed.binaryPath ?? null) : null,
    only: parsed.only ?? null,
    tempRoot: owned && parsed.keep ? owned.root : null,
    examples: results,
  };
}

export function exitCodeForStatus(status) {
  if (status === 'pass') return EXIT.PASS;
  if (status === 'fail') return EXIT.FAIL;
  if (status === 'blocked') return EXIT.BLOCKED;
  throw new Error(`Unknown example status: ${status}`);
}

function formatText(summary) {
  const lines = [`examples: ${summary.status}`];
  if (summary.binaryPath) lines.push(`binary: ${summary.binaryPath} (explicit; no PATH search)`);
  if (summary.tempRoot) lines.push(`temp root kept: ${summary.tempRoot}`);
  for (const result of summary.examples) {
    lines.push(`[${result.status}] ${result.id} (${result.effect})`);
    if (result.reason) lines.push(`  ${result.reason}`);
    for (const step of result.steps ?? []) {
      lines.push(`  ${step.argv.join(' ')} -> exit ${step.exit}`);
      if (step.stdout) lines.push(`    stdout: ${step.stdout.split('\n')[0]}`);
      if (step.exit !== 0 && step.stderr) lines.push(`    stderr: ${step.stderr.split('\n')[0]}`);
    }
    if (result.home) {
      lines.push(`  fixture home: ${result.home.entries} entries, digest ${result.home.digest.slice(0, 16)}`);
    }
  }
  if (summary.status !== 'pass') {
    lines.push(`Result: ${summary.status}. No state changed outside the owned temp root; nothing was installed.`);
  }
  return lines.join('\n');
}

const HELP_TEXT = [
  'Usage: node scripts/examples.mjs --binary <absolute-path> [--only <id>] [--json] [--keep]',
  '',
  'Runs only the frozen, read-only examples; manual examples are reported, never executed.',
  `Available ids: ${EXAMPLE_IDS.join(', ')}`,
  '',
  'Exit codes: 0 pass, 1 fail, 2 refused (bad input/unknown id/Markdown), 3 blocked (missing CLI).',
  'The binary path must be absolute; PATH is never searched. Use --keep to retain the owned temp root.',
].join('\n');

async function main(argv) {
  const options = parseArgs(argv);
  if (options.help) {
    console.log(HELP_TEXT);
    return EXIT.PASS;
  }
  const summary = await runExamples(options);
  if (options.json) console.log(JSON.stringify(summary, null, 2));
  else console.log(formatText(summary));
  return exitCodeForStatus(summary.status);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main(process.argv.slice(2)).then(
    (code) => {
      process.exitCode = code;
    },
    (error) => {
      const code = error instanceof ExamplesRefusal ? EXIT.REFUSED : EXIT.FAIL;
      console.error(`examples: ${error.message}`);
      process.exitCode = code;
    },
  );
}
