import { spawn } from 'node:child_process';
import { accessSync, constants, lstatSync, realpathSync, statSync } from 'node:fs';
import path from 'node:path';

const ALLOWED_OPTION_KEYS = new Set(['repoRoot', 'binaryPath', 'timeoutMs', 'maxBuffer', 'expectedCommit']);
const ANSI_CONTROL_PATTERN = /[\u001b\u009b]/;
const FLAG_TOKEN_PATTERN = /(?:^|[\s\[,])(-h|--[a-z0-9]+(?:-[a-z0-9]+)*)/g;

const SOURCE_PATHS = Object.freeze([
  'cmd/agentic-sdd/env.go',
  'cmd/agentic-sdd/execution_cli.go',
  'cmd/agentic-sdd/fleet.go',
  'cmd/agentic-sdd/fleet_doctor.go',
  'cmd/agentic-sdd/fleet_plan.go',
  'cmd/agentic-sdd/guide.go',
  'cmd/agentic-sdd/helptext.go',
  'cmd/agentic-sdd/logs.go',
  'cmd/agentic-sdd/main.go',
  'cmd/agentic-sdd/mcp.go',
  'cmd/agentic-sdd/mcp_write.go',
  'cmd/agentic-sdd/opencode_disabled.go',
  'cmd/agentic-sdd/review.go',
  'cmd/agentic-sdd/run.go',
  'cmd/agentic-sdd/task.go',
  'cmd/agentic-sdd/trace.go',
  'cmd/agentic-sdd/tui.go',
  'internal/buildinfo/buildinfo.go',
]);

const ALLOWLISTED_COMMANDS = Object.freeze([
  Object.freeze({ id: 'top', family: 'top', argv: Object.freeze(['--help']), source: 'cmd/agentic-sdd/main.go', flagMode: 'flags-only' }),
  Object.freeze({ id: 'help-preview', family: 'preview', argv: Object.freeze(['help', 'preview']), source: 'cmd/agentic-sdd/helptext.go', flagMode: 'flags-only' }),
  Object.freeze({ id: 'help-apply', family: 'apply', argv: Object.freeze(['help', 'apply']), source: 'cmd/agentic-sdd/helptext.go', flagMode: 'flags-only' }),
  Object.freeze({ id: 'help-backups', family: 'backups', argv: Object.freeze(['help', 'backups']), source: 'cmd/agentic-sdd/helptext.go', flagMode: 'flags-only' }),
  Object.freeze({ id: 'help-restore', family: 'restore', argv: Object.freeze(['help', 'restore']), source: 'cmd/agentic-sdd/helptext.go', flagMode: 'flags-only' }),
  Object.freeze({ id: 'help-logs', family: 'logs', argv: Object.freeze(['help', 'logs']), source: 'cmd/agentic-sdd/helptext.go', flagMode: 'flags-only' }),
  Object.freeze({ id: 'help-guide', family: 'guide', argv: Object.freeze(['help', 'guide']), source: 'cmd/agentic-sdd/helptext.go', flagMode: 'flags-only' }),
  Object.freeze({ id: 'help-version', family: 'version', argv: Object.freeze(['help', 'version']), source: 'cmd/agentic-sdd/helptext.go', flagMode: 'flags-only' }),
  Object.freeze({ id: 'help-tui', family: 'tui', argv: Object.freeze(['help', 'tui']), source: 'cmd/agentic-sdd/helptext.go', flagMode: 'flags-only' }),
  Object.freeze({ id: 'tui', family: 'tui', argv: Object.freeze(['tui', '--help']), source: 'cmd/agentic-sdd/tui.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'orchestrate', family: 'orchestrate', argv: Object.freeze(['help', 'orchestrate']), source: 'cmd/agentic-sdd/main.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'fleet', family: 'fleet', argv: Object.freeze(['fleet', '--help']), source: 'cmd/agentic-sdd/fleet.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'fleet-status', family: 'fleet', argv: Object.freeze(['fleet', 'status', '--help']), source: 'cmd/agentic-sdd/fleet.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'fleet-plan', family: 'fleet', argv: Object.freeze(['fleet', 'plan', '--help']), source: 'cmd/agentic-sdd/fleet_plan.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'fleet-apply', family: 'fleet', argv: Object.freeze(['fleet', 'apply', '--help']), source: 'cmd/agentic-sdd/fleet.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'fleet-restore', family: 'fleet', argv: Object.freeze(['fleet', 'restore', '--help']), source: 'cmd/agentic-sdd/fleet.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'fleet-doctor', family: 'fleet', argv: Object.freeze(['fleet', 'doctor', '--help']), source: 'cmd/agentic-sdd/fleet_doctor.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'mcp', family: 'mcp', argv: Object.freeze(['mcp', 'help']), source: 'cmd/agentic-sdd/mcp.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'mcp-status', family: 'mcp', argv: Object.freeze(['mcp', 'status', '--help']), source: 'cmd/agentic-sdd/mcp.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'mcp-doctor', family: 'mcp', argv: Object.freeze(['mcp', 'doctor', '--help']), source: 'cmd/agentic-sdd/mcp.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'mcp-preview', family: 'mcp', argv: Object.freeze(['mcp', 'preview', '--help']), source: 'cmd/agentic-sdd/mcp.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'mcp-apply', family: 'mcp', argv: Object.freeze(['mcp', 'apply', '--help']), source: 'cmd/agentic-sdd/mcp_write.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'mcp-restore', family: 'mcp', argv: Object.freeze(['mcp', 'restore', 'help']), source: 'cmd/agentic-sdd/mcp_write.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'help-run', family: 'run', argv: Object.freeze(['help', 'run']), source: 'cmd/agentic-sdd/helptext.go', flagMode: 'flags-only' }),
  Object.freeze({ id: 'run', family: 'run', argv: Object.freeze(['run', 'help']), source: 'cmd/agentic-sdd/run.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'help-task', family: 'task', argv: Object.freeze(['help', 'task']), source: 'cmd/agentic-sdd/helptext.go', flagMode: 'flags-only' }),
  Object.freeze({ id: 'task', family: 'task', argv: Object.freeze(['task', 'help']), source: 'cmd/agentic-sdd/task.go', flagMode: 'usage-and-flags' }),
  Object.freeze({ id: 'help-review', family: 'review', argv: Object.freeze(['help', 'review']), source: 'cmd/agentic-sdd/helptext.go', flagMode: 'flags-only' }),
  Object.freeze({ id: 'review', family: 'review', argv: Object.freeze(['review', 'help']), source: 'cmd/agentic-sdd/review.go', flagMode: 'usage-and-flags' }),
]);

const ALLOWLISTED_ARGV_KEYS = new Set(ALLOWLISTED_COMMANDS.map((entry) => JSON.stringify(entry.argv)));

// The identity probe is the only non-help argv the collector may execute. The
// observed commit binds generated facts to the binary built from the inspected
// revision; a missing or unexpected commit fails closed.
const ALLOWLISTED_PROBE_ARGV = Object.freeze([Object.freeze(['version'])]);
const ALLOWLISTED_PROBE_KEYS = new Set(ALLOWLISTED_PROBE_ARGV.map((argv) => JSON.stringify(argv)));
const COMMIT_PATTERN = /^[0-9a-f]{7,40}$/i;

const SANITIZED_ENV = Object.freeze({
  PATH: '/usr/bin:/bin',
  LANG: 'C.UTF-8',
  LC_ALL: 'C.UTF-8',
  TERM: 'dumb',
  NO_COLOR: '1',
  COLUMNS: '80',
});

const DISCREPANCIES = [
  {
    id: 'mcp-top-level-help',
    summary:
      'Top-level --help describes mcp as read-only (status, doctor, preview), whereas mcp help and cmd/agentic-sdd/mcp_write.go expose mcp apply and entry-level mcp restore (list, preview, apply).',
    topLevelArgv: ['--help'],
    familyArgv: ['mcp', 'help'],
    sources: ['cmd/agentic-sdd/main.go', 'cmd/agentic-sdd/mcp.go', 'cmd/agentic-sdd/mcp_write.go'],
  },
  {
    id: 'top-level-orchestration-flags',
    summary:
      'Top-level Flags omits --with-opencode and --opencode-free-first even though main.go registers them on the root FlagSet; help orchestrate documents both.',
    topLevelArgv: ['--help'],
    familyArgv: ['help', 'orchestrate'],
    sources: ['cmd/agentic-sdd/main.go'],
  },
];

const EXPERIMENTAL_EXCLUSIONS = [
  {
    names: ['opencode', 'opencode-harness'],
    buildTag: 'agentic_sdd_experimental',
    defaultBuildBehavior: 'unavailable in default core builds (exits 2)',
    sources: ['cmd/agentic-sdd/main.go', 'cmd/agentic-sdd/opencode_disabled.go'],
  },
];

function isPlainObject(value) {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) return false;
  const proto = Object.getPrototypeOf(value);
  return proto === Object.prototype || proto === null;
}

function validateRepoRoot(repoRoot) {
  if (typeof repoRoot !== 'string' || repoRoot.trim() === '' || repoRoot.includes('\0')) {
    throw new Error('CLI collector requires a non-empty repoRoot path');
  }
  const resolved = path.resolve(repoRoot);
  let linkStat;
  try {
    linkStat = lstatSync(resolved);
  } catch (error) {
    throw new Error(`Missing repoRoot "${repoRoot}": ${error.code ?? error.message}`);
  }
  if (linkStat.isSymbolicLink()) {
    throw new Error(`Unsafe symlink repoRoot: "${repoRoot}"`);
  }
  const realRoot = realpathSync(resolved);
  const realStat = statSync(realRoot);
  if (!realStat.isDirectory()) {
    throw new Error(`repoRoot is not a directory: "${repoRoot}"`);
  }
  return realRoot;
}

function validateBinaryPath(binaryPath) {
  if (typeof binaryPath !== 'string' || binaryPath.trim() === '' || binaryPath.includes('\0')) {
    throw new Error('CLI collector requires a non-empty explicit binaryPath');
  }
  if (!path.isAbsolute(binaryPath)) {
    throw new Error(`CLI binaryPath must be an explicit absolute path (no PATH resolution): "${binaryPath}"`);
  }
  let stats;
  try {
    stats = lstatSync(binaryPath);
  } catch (error) {
    throw new Error(`Missing CLI binary "${binaryPath}": ${error.code ?? error.message}`);
  }
  if (stats.isSymbolicLink()) {
    throw new Error(`Refusing symlink CLI binary: "${binaryPath}"`);
  }
  if (!stats.isFile()) {
    throw new Error(`CLI binary is not a regular file: "${binaryPath}"`);
  }
  try {
    accessSync(binaryPath, constants.X_OK);
  } catch (error) {
    throw new Error(`CLI binary is not executable "${binaryPath}": ${error.code ?? error.message}`);
  }
  return binaryPath;
}

function verifyContainedSources(realRoot, relativePaths) {
  const verified = new Set();
  const rootPrefix = realRoot.endsWith(path.sep) ? realRoot : `${realRoot}${path.sep}`;
  for (const relative of relativePaths) {
    if (typeof relative !== 'string' || relative === '' || relative.includes('\\') || relative.startsWith('/')) {
      throw new Error(`Invalid repository-relative source path: "${relative}"`);
    }
    const segments = relative.split('/');
    if (segments.some((segment) => segment === '' || segment === '.' || segment === '..')) {
      throw new Error(`Unsafe source path traversal: "${relative}"`);
    }
    let cursor = realRoot;
    for (let index = 0; index < segments.length; index += 1) {
      cursor = path.join(cursor, segments[index]);
      let stats;
      try {
        stats = lstatSync(cursor);
      } catch (error) {
        throw new Error(`Missing CLI source file "${relative}": ${error.code ?? error.message}`);
      }
      if (stats.isSymbolicLink()) {
        throw new Error(`Refusing symlink CLI source path "${relative}"`);
      }
      if (index < segments.length - 1) {
        if (!stats.isDirectory()) {
          throw new Error(`Invalid directory component in CLI source path "${relative}"`);
        }
      } else if (!stats.isFile()) {
        throw new Error(`CLI source path is not a regular file: "${relative}"`);
      }
    }
    const realTarget = realpathSync(cursor);
    if (!realTarget.startsWith(rootPrefix)) {
      throw new Error(`CLI source path escapes repoRoot: "${relative}"`);
    }
    verified.add(relative);
  }
  return [...verified].sort();
}

function extractHelpFlags(stdout, flagMode) {
  const lines = stdout.split('\n');
  let mode = null;
  const flags = new Set();
  for (const line of lines) {
    if (/^Usage:\s*/.test(line)) {
      mode = flagMode === 'usage-and-flags' ? 'usage' : null;
    } else if (/^(?:Flags|Options|OpenCode Orchestration Options):\s*$/.test(line)) {
      mode = 'flags';
    } else if (line.trim() === '') {
      mode = null;
    } else if (mode && !/^\s+/.test(line)) {
      mode = null;
    }
    if (mode) {
      const matches = line.match(FLAG_TOKEN_PATTERN) ?? [];
      for (const token of matches) {
        flags.add(token.replace(/^[\s\[,]+/, ''));
      }
    }
  }
  return [...flags].sort();
}

function markdownToken(value) {
  return String(value).replace(/[\u0000-\u001f`]/g, ' ').trim();
}

function normalizeExpectedCommit(value) {
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error('--commit expects a 7-40 character hexadecimal Git commit');
  }
  const normalized = value.trim().toLowerCase();
  if (!COMMIT_PATTERN.test(normalized)) {
    throw new Error(`--commit expects a 7-40 character hexadecimal Git commit, got "${value}"`);
  }
  return normalized;
}

function commitSatisfies(observed, expected) {
  if (expected.length === 40) return observed === expected;
  return observed.startsWith(expected);
}

function parseVersionIdentity(stdout) {
  const text = stdout.replace(/\r\n/g, '\n');
  const header = (text.split('\n', 1)[0] ?? '').trim();
  const product = /^asdx-core (\S+)$/.exec(header);
  if (!product) {
    throw new Error(`CLI version probe did not identify the canonical asdx-core binary: "${header}"`);
  }
  const field = (label) => {
    const match = new RegExp(`^\\s*${label}:\\s+(\\S.*?)\\s*$`, 'm').exec(text);
    return match ? match[1] : '';
  };
  const commit = field('commit');
  const date = field('date');
  const goVersion = field('go version');
  if (!COMMIT_PATTERN.test(commit)) {
    throw new Error(`CLI binary has no usable build commit ("${commit || 'missing'}"); rebuild it from the inspected revision`);
  }
  if (!date) throw new Error('CLI version probe is missing the build date; rebuild it from the inspected revision');
  if (!goVersion) throw new Error('CLI version probe is missing the Go version; rebuild it from the inspected revision');
  return {
    product: 'asdx-core',
    version: product[1],
    commit: commit.toLowerCase(),
    date,
    goVersion,
  };
}

function runAllowlistedHelp(binaryPath, argv, { cwd, timeoutMs, maxBuffer }) {
  if (!ALLOWLISTED_ARGV_KEYS.has(JSON.stringify(argv))) {
    throw new Error(`Refusing non-allowlisted CLI argv: ${JSON.stringify(argv)}`);
  }
  return runCapturedProcess(binaryPath, argv, { cwd, timeoutMs, maxBuffer, kind: 'help' });
}

function runAllowlistedVersion(binaryPath, { cwd, timeoutMs, maxBuffer }) {
  const argv = ['version'];
  if (!ALLOWLISTED_PROBE_KEYS.has(JSON.stringify(argv))) {
    throw new Error(`Refusing non-allowlisted CLI probe argv: ${JSON.stringify(argv)}`);
  }
  return runCapturedProcess(binaryPath, argv, { cwd, timeoutMs, maxBuffer, kind: 'version probe' });
}

function runCapturedProcess(binaryPath, argv, { cwd, timeoutMs, maxBuffer, kind }) {
  const argvLabel = argv.join(' ');
  return new Promise((resolve, reject) => {
    const child = spawn(binaryPath, [...argv], {
      shell: false,
      cwd,
      env: { ...SANITIZED_ENV },
      stdio: ['ignore', 'pipe', 'pipe'],
      windowsHide: true,
    });

    const stdoutChunks = [];
    const stderrChunks = [];
    let stdoutBytes = 0;
    let stderrBytes = 0;
    let settled = false;
    let killTimer = null;

    const finishError = (error) => {
      if (settled) return;
      settled = true;
      clearTimeout(timeoutTimer);
      if (killTimer) clearTimeout(killTimer);
      reject(error);
    };

    const forceKill = () => {
      try {
        child.kill('SIGTERM');
      } catch {
        // Ignore already-exited child.
      }
      killTimer = setTimeout(() => {
        try {
          child.kill('SIGKILL');
        } catch {
          // Ignore already-exited child.
        }
      }, 100);
      killTimer.unref();
    };

    const timeoutTimer = setTimeout(() => {
      forceKill();
      finishError(new Error(`CLI ${kind} timed out after ${timeoutMs}ms for "${argvLabel}"`));
    }, timeoutMs);
    timeoutTimer.unref();

    child.stdout.on('data', (chunk) => {
      stdoutBytes += chunk.length;
      if (stdoutBytes > maxBuffer) {
        forceKill();
        finishError(new Error(`CLI ${kind} stdout exceeded maxBuffer (${maxBuffer} bytes) for "${argvLabel}"`));
        return;
      }
      stdoutChunks.push(chunk);
    });

    child.stderr.on('data', (chunk) => {
      stderrBytes += chunk.length;
      if (stderrBytes > maxBuffer) {
        forceKill();
        finishError(new Error(`CLI ${kind} stderr exceeded maxBuffer (${maxBuffer} bytes) for "${argvLabel}"`));
        return;
      }
      stderrChunks.push(chunk);
    });

    child.on('error', (error) => {
      finishError(new Error(`Failed to execute CLI binary for "${argvLabel}": ${error.message}`));
    });

    child.on('close', (code, signal) => {
      if (settled) return;
      settled = true;
      clearTimeout(timeoutTimer);
      if (killTimer) clearTimeout(killTimer);

      if (signal) {
        reject(new Error(`CLI ${kind} terminated by signal ${signal} for "${argvLabel}"`));
        return;
      }

      const stdout = Buffer.concat(stdoutChunks).toString('utf8').replace(/\r\n/g, '\n');
      const stderr = Buffer.concat(stderrChunks).toString('utf8').replace(/\r\n/g, '\n');

      if (ANSI_CONTROL_PATTERN.test(stdout) || ANSI_CONTROL_PATTERN.test(stderr)) {
        reject(new Error(`CLI ${kind} emitted unexpected ANSI/control escape sequences for "${argvLabel}"`));
        return;
      }
      if (code !== 0) {
        const detail = stderr.trim() ? `: ${stderr.trim().split('\n')[0]}` : '';
        reject(new Error(`CLI ${kind} exited with code ${code} for "${argvLabel}"${detail}`));
        return;
      }
      if (stdout.trim().length === 0) {
        reject(new Error(`CLI ${kind} returned empty stdout for "${argvLabel}"`));
        return;
      }
      resolve({ exit: code, stdout, stderr });
    });
  });
}

function buildMarkdown({ commands, sources, provenance }) {
  const byId = new Map(commands.map((command) => [command.id, command]));
  const binary = provenance.binary;
  const verification = provenance.commitVerified
    ? ` The expected inspected-revision commit \`${provenance.expectedCommit}\` was verified before capture.`
    : '';
  const lines = [
    `Facts below were captured by running the candidate \`${binary.product}\` binary at commit \`${binary.commit}\` `
      + `(version \`${markdownToken(binary.version)}\`, built \`${markdownToken(binary.date)}\` with `
      + `\`${markdownToken(binary.goVersion)}\`).${verification}`,
    '',
    '## Entrypoints and compatibility wrappers',
    '',
    '- Canonical commands: `asdx-core` (CLI) and `asdx` (TUI companion), defined in `internal/buildinfo/buildinfo.go` and `cmd/agentic-sdd/main.go`.',
    '- Legacy compatibility wrappers (one deprecation window): `agentic-sdd` forwards to same-directory `asdx-core` after a stderr notice (only exact `--help`, `-h`, and `--version` are answered locally); `agentic-sdd-tui` wraps `asdx`.',
    '- Bare `asdx-core` defaults to read-only `preview` and never opens the terminal UI (`cmd/agentic-sdd/main.go`).',
    '- Experimental commands `opencode` and `opencode-harness` require `-tags agentic_sdd_experimental` and exit `2` in default core builds (`cmd/agentic-sdd/opencode_disabled.go`).',
    '',
    '## Core installer, launcher, and utility commands',
    '',
    'Flags are command-specific (standard-library `flag.FlagSet` per handler, not universal global flags):',
    '',
    '| Command | Forms / subcommands | Accepted flags | Write behavior & exit notes | Source |',
    '|---|---|---|---|---|',
    '| `preview` (default) | `asdx-core [preview]` | `--repo PATH`, `--home PATH`, `--with-opencode`, `--opencode-free-first`, `--apply` (default form only), `--version` (root flag), `-h`, `--help` | Read-only. `--version` is a root flag: it is accepted in the default, `preview`, and `apply` forms and short-circuits to version output (`0`); it is not bare-only. `--opencode-free-first` requires `--with-opencode` (`2`); `--apply` combined with an explicit `preview`/`apply` command (`2`); unknown flag/argument (`2`). Exits `0`, `1` (runtime/I/O error), or `2` (usage/flag error). | `cmd/agentic-sdd/main.go` |',
    '| `apply` | `asdx-core apply` | `--repo PATH`, `--home PATH`, `--with-opencode`, `--opencode-free-first`, `--version` (root flag), `-h`, `--help` | Writes backups + skills + log under `<home>/.agentic-sdd/logs`. Exits `0` success, `1` runtime error, `2` usage/flag error. | `cmd/agentic-sdd/main.go`, `cmd/agentic-sdd/logs.go` |',
    '| `version` | `asdx-core version` | None (extra args including `--help` exit `2`; use `asdx-core help version`) | Read-only. Prints version, commit, date, and Go version (`0`). | `cmd/agentic-sdd/main.go` |',
    '| `backups` | `asdx-core backups` | `--repo PATH`, `--home PATH`, `-h`, `--help` | Read-only; identical listing to `restore list`. Exits `0`, `1`, or `2`. | `cmd/agentic-sdd/main.go` |',
    '| `logs` | `asdx-core logs` | `--home PATH`, `-h`, `--help` | Read-only; prints latest apply/restore log. Exits `1` if missing/unreadable, `2` usage. | `cmd/agentic-sdd/logs.go` |',
    '| `restore` | `asdx-core restore [ID]` / `restore list\\|preview\\|apply [ID]` | `--repo PATH`, `--home PATH`, `--apply` (bare form only; rejected on sub-words), `-h`, `--help` | Read-only without `--apply`/`apply`. Interactive mode prompts when no `ID`; non-TTY without `ID` exits `2`. | `cmd/agentic-sdd/main.go` |',
    '| `guide` | `asdx-core guide` | `--repo PATH`, `--home PATH`, `-h`, `--help` | Interactive numbered wizard; refuses non-TTY with exit `2`. Writes only after confirmation. | `cmd/agentic-sdd/guide.go` |',
    '| `tui` | `asdx-core tui` | `--home PATH`, `--repo PATH`, `--fleet-config PATH`, `--with-opencode` (default `true`, `--with-opencode=false` to disable), `--opencode-free-first`, `-h`, `--help` | Requires real stdin+stdout TTY and rejects `AGENTIC_SDD_FORCE_INTERACTIVE=1` (`2`). Missing/mismatched `asdx` exits `1`. | `cmd/agentic-sdd/tui.go` |',
    '| `help` | `asdx-core help [topic]` | Topics: `preview`, `apply`, `backups`, `logs`, `restore`, `guide`, `version`, `tui`, `orchestrate`, `fleet`, `mcp`, `run`, `task`, `review`; experimental-build `opencode` and `opencode-harness` are accepted and print the unavailable note in default builds | Read-only (`0`); unknown topic or extra args exits `2`. `help orchestrate`, `help fleet`, `help mcp` print dedicated blocks without repeating root usage; the default-build `help opencode`/`help opencode-harness` topics describe commands that require `-tags agentic_sdd_experimental`. | `cmd/agentic-sdd/main.go`, `cmd/agentic-sdd/helptext.go` |',
    '',
    '## `fleet` family',
    '',
    'Exit codes (`cmd/agentic-sdd/fleet.go`, `cmd/agentic-sdd/fleet_plan.go`, `cmd/agentic-sdd/fleet_doctor.go`): `0` success/report produced, `1` runtime error / lock held / stale plan / unmanaged conflict, `2` usage / flag / invalid configuration / malformed plan / invalid backup ID.',
    '',
    '| Subcommand | Accepted flags | Write & contract notes | Source |',
    '|---|---|---|---|',
    '| `fleet status` | `--config PATH` (required), `--home PATH`, `--repo PATH` | Read-only, offline inspection of `fleet.json` and installed targets. | `cmd/agentic-sdd/fleet.go` |',
    '| `fleet plan` | `--config PATH` (required), `--output PATH`, `--adopt`, `--home PATH`, `--repo PATH` | Read-only on destinations; `--output PATH` writes the canonical plan JSON file. Exits `1` when plan has conflicts. | `cmd/agentic-sdd/fleet_plan.go` |',
    '| `fleet apply` | `--plan PATH` (required), `--home PATH` | Applies a saved plan under operation lock. | `cmd/agentic-sdd/fleet.go` |',
    '| `fleet restore` | `--backup ID` (required), `--home PATH` | Flag form only; restores managed agents from backup ID. | `cmd/agentic-sdd/fleet.go` |',
    '| `fleet doctor` | `--opencode-bin PATH`, `--config PATH` | Offline model-route diagnostic; exits `0` when report is produced even if routes are unavailable (`2` on invalid config). | `cmd/agentic-sdd/fleet_doctor.go` |',
    '',
    '## `mcp` family (including incomplete top-level help note)',
    '',
    '> **Known top-level help discrepancy (`cmd/agentic-sdd/main.go` vs `cmd/agentic-sdd/mcp.go`, `cmd/agentic-sdd/mcp_write.go`):** Top-level `asdx-core --help` summarizes `mcp` as read-only (`status|doctor|preview`) and omits `apply` and `restore`. Actual `mcp help` and Go dispatch implement `status`, `doctor`, `preview`, `apply`, and entry-level `restore list|preview|apply`.',
    '',
    'Exit codes (`cmd/agentic-sdd/mcp.go`, `cmd/agentic-sdd/mcp_write.go`): `0` success / produced report / real no-op, `1` stale plan / conflict / unsupported runtime / rollback / engine or I/O error, `2` usage / flag / validation / unsafe target / unknown subcommand / deferred `--network` or `--connect`.',
    '',
    '| Subcommand | Accepted flags | Write & contract notes | Source |',
    '|---|---|---|---|',
    '| `mcp` / `mcp status` | `--home PATH`, `--config PATH`, `--project PATH`, `--json` | Read-only, offline; default when no subcommand or a flag is passed after `mcp`. | `cmd/agentic-sdd/mcp.go` |',
    '| `mcp doctor` | `--home PATH`, `--config PATH`, `--project PATH`, `--opencode-bin PATH`, `--codegraph-bin PATH`, `--json` | Offline diagnostics; `--network` and `--connect` are explicitly rejected with exit `2`. | `cmd/agentic-sdd/mcp.go` |',
    '| `mcp preview` | `--servers NAME,...`, `--auth SERVER=MODE`, `--home PATH`, `--config PATH`, `--json` | Read-only structured plan + digest/token (`context7=oauth|api-key`, `exa=keyless|oauth|api-key`). Exits `1` if `Conflicts` is non-empty. | `cmd/agentic-sdd/mcp.go` |',
    '| `mcp apply` | `--expect-config SHA256` (required), `--opencode-bin PATH`, `--servers NAME,...`, `--auth SERVER=MODE`, `--home PATH`, `--config PATH`, `--json` | Writes managed entries into existing global OpenCode config only when digest matches and local runtime probe succeeds; never creates missing config. | `cmd/agentic-sdd/mcp_write.go` |',
    '| `mcp restore list` | `--home PATH`, `--config PATH` (parsed, unused by list), `--json` | Read-only listing of managed MCP backups. | `cmd/agentic-sdd/mcp_write.go` |',
    '| `mcp restore preview ID` | `--home PATH`, `--config PATH`, `--json` | Read-only entry-level restore preview for positional `ID`; exits `1` on conflicts. | `cmd/agentic-sdd/mcp_write.go` |',
    '| `mcp restore apply ID` | `--expect-config SHA256` (required), `--home PATH`, `--config PATH`, `--json` | Entry-level restore for positional `ID`; prompts `[y/N]` when `env.Interactive` is true. | `cmd/agentic-sdd/mcp_write.go` |',
    '',
    '## Execution family (`run`, `task`, `review`)',
    '',
    'Shared execution flags (`cmd/agentic-sdd/execution_cli.go`): `--repo DIR` (default: current working tree), `--run-root DIR` (default: `<repo>/.agentic-sdd/runs`), `--format text|json` (default `text`).',
    '',
    'Execution flag defaults (`internal/execution/types.go`): `run check --timeout 15m` (`0` or negative disables the wall-clock limit); `run check --max-output-bytes 67108864` per stream (`0` or negative selects the default; effective range `1024`–`1073741824`); `review package --max-file-bytes 2097152` and `--max-total-bytes 33554432` (`0` or negative selects the default).',
    '',
    '| Subcommand | Additional flags / positional contract | Distinct exit codes & contract notes | Source |',
    '|---|---|---|---|',
    '| `run init` | `--feature DIR` + `--batch ID`, or `--direct` + optional `--label TEXT` | `--direct` cannot combine with `--feature`/`--batch` (`2`). Exits `0`, `1` runtime, `2` usage/unsafe/conflict/missing scope. | `cmd/agentic-sdd/run.go` |',
    '| `run status` | `--run ID` | Missing run for `status` exits `1` (unlike `init` missing batch `2`). | `cmd/agentic-sdd/run.go`, `cmd/agentic-sdd/execution_cli.go` |',
    '| `run record` | `--run ID` (required), `--task T01` (required for feature runs, defaults to `direct` for `--direct`), `--result FILE\\|-` (required) | Ingests untrusted claim record; `0` recorded, `1` runtime, `2` usage/validation. | `cmd/agentic-sdd/run.go` |',
    '| `run clean` | `--run ID`, `--yes` | Without `--yes`, previews without deleting; with `--yes`, deletes only that run directory. | `cmd/agentic-sdd/run.go` |',
    '| `run check` | `--run ID` (required), `--task T01`, `--timeout DURATION` (default `15m`; `0` disables), `--max-output-bytes N` (default `67108864` per stream, clamped to `1024`–`1073741824`), `-- COMMAND [ARG...]` | `--` separator before command is required (even `--help` without `--` exits `2`). Exits `0` pass, `1` fail/error, `124` timed out, `2` usage. | `cmd/agentic-sdd/run.go` |',
    '| `task brief` | `--run ID` (required), `--task T01` (required), `--force` | Refuses `--direct` runs (`2`) and foreign checkpoints (`2`); `--force` refreshes differing brief bytes without bypassing changed scope. | `cmd/agentic-sdd/task.go` |',
    '| `review package` | `--run ID` (required), `--base COMMIT` (required), `--head COMMIT` (defaults to current `HEAD`), `--max-file-bytes N` (default `2097152`), `--max-total-bytes N` (default `33554432`) | Exits `0` complete package, `3` partial package with explicit omissions, `1` runtime/missing run, `2` usage/unsafe. | `cmd/agentic-sdd/review.go` |',
    '',
    '## Presentation and trace environment variables',
    '',
    '- `NO_COLOR`: non-empty disables ANSI color styling (`cmd/agentic-sdd/env.go`).',
    '- `TERM=dumb`: switches to ASCII glyphs and disables color (`cmd/agentic-sdd/env.go`).',
    '- `COLUMNS`: positive integer sets wrap width (defaults to `80`, `cmd/agentic-sdd/env.go`).',
    '- `AGENTIC_SDD_ANIMATE=1` / `AGENTIC_SDD_REDUCED_MOTION=1`: CLI animation opt-in (`cmd/agentic-sdd/env.go`) and TUI reduced-motion control (`experiments/tui/cmd/internal/tuirun/run.go`).',
    '- `AGENTIC_SDD_FORCE_INTERACTIVE=1`: enables interactive CLI prompts on scripted streams, while `asdx-core tui` refuses it with exit `2` (`cmd/agentic-sdd/env.go`, `cmd/agentic-sdd/tui.go`).',
    '- `AGENTIC_SDD_TRACE`: path to append JSONL trace events (`0600` regular file, `cmd/agentic-sdd/main.go`, `cmd/agentic-sdd/trace.go`).',
    '',
    '## Captured help inventory',
    '',
    '| ID | Argv | Exit | Flags observed in help | Primary source |',
    '|---|---|---|---|---|',
  ];

  for (const cmd of commands) {
    const argvText = `\`asdx-core ${cmd.argv.join(' ')}\``;
    const flagsText = cmd.flags.length ? cmd.flags.map((flag) => `\`${flag}\``).join(', ') : 'none';
    lines.push(`| \`${cmd.id}\` | ${argvText} | \`${cmd.exit}\` | ${flagsText} | \`${cmd.source}\` |`);
  }

  const topOut = byId.get('top')?.stdout.trim() ?? '';
  const mcpOut = byId.get('mcp')?.stdout.trim() ?? '';
  if (topOut) {
    lines.push('', '## Top-level help excerpt (`asdx-core --help`)', '', '```text', topOut, '```');
  }
  if (mcpOut) {
    lines.push('', '## MCP family help excerpt (`asdx-core mcp help`)', '', '```text', mcpOut, '```');
  }

  lines.push('', '## Evidence sources', '');
  for (const source of sources) {
    lines.push(`- \`${source}\``);
  }

  return `${lines.join('\n')}\n`;
}

export async function collectCLI(options = {}) {
  if (!isPlainObject(options)) {
    throw new Error('CLI collector options must be a plain object');
  }
  for (const key of Object.keys(options)) {
    if (!ALLOWED_OPTION_KEYS.has(key)) {
      throw new Error(`Unknown CLI collector option "${key}"`);
    }
  }

  const {
    repoRoot,
    binaryPath,
    timeoutMs = 5000,
    maxBuffer = 1048576,
    expectedCommit,
  } = options;

  if (typeof timeoutMs !== 'number' || !Number.isFinite(timeoutMs) || timeoutMs <= 0) {
    throw new Error('CLI collector timeoutMs must be a positive finite number');
  }
  if (typeof maxBuffer !== 'number' || !Number.isInteger(maxBuffer) || maxBuffer <= 0) {
    throw new Error('CLI collector maxBuffer must be a positive integer');
  }

  const normalizedCommit = expectedCommit === undefined ? null : normalizeExpectedCommit(expectedCommit);
  const realRoot = validateRepoRoot(repoRoot);
  const checkedBinary = validateBinaryPath(binaryPath);
  const sources = verifyContainedSources(realRoot, [
    ...SOURCE_PATHS,
    ...ALLOWLISTED_COMMANDS.map((entry) => entry.source),
  ]);

  const versionResult = await runAllowlistedVersion(checkedBinary, {
    cwd: realRoot,
    timeoutMs,
    maxBuffer,
  });
  const binary = parseVersionIdentity(versionResult.stdout);
  if (normalizedCommit !== null && !commitSatisfies(binary.commit, normalizedCommit)) {
    throw new Error(
      `Reference provenance mismatch: candidate binary commit ${binary.commit} does not match the expected commit ${normalizedCommit}; rebuild the binary from the inspected revision`,
    );
  }
  const provenance = {
    producer: 'website/scripts/reference/cli.mjs',
    binary,
    expectedCommit: normalizedCommit,
    commitVerified: normalizedCommit !== null,
  };

  const commands = [];
  for (const entry of ALLOWLISTED_COMMANDS) {
    const result = await runAllowlistedHelp(checkedBinary, entry.argv, {
      cwd: realRoot,
      timeoutMs,
      maxBuffer,
    });
    commands.push({
      id: entry.id,
      family: entry.family,
      argv: [...entry.argv],
      exit: result.exit,
      stdout: result.stdout,
      stderr: result.stderr,
      flags: extractHelpFlags(result.stdout, entry.flagMode),
      source: entry.source,
    });
  }

  const markdown = buildMarkdown({ commands, sources, provenance });
  return {
    sections: [
      {
        pageId: 'reference/cli',
        title: 'CLI reference',
        markdown,
        sources,
        metadata: structuredClone(provenance),
      },
    ],
    inventory: {
      commands,
      discrepancies: structuredClone(DISCREPANCIES),
      experimentalExclusions: structuredClone(EXPERIMENTAL_EXCLUSIONS),
      sources: [...sources],
      provenance: structuredClone(provenance),
    },
  };
}
