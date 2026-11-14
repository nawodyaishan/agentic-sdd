// Spec 020 B4-W1-G — unsafe examples and a wrong project base in built output.
//
// Frozen sources: plan C3 (examples have an effect class; the runner never runs
// real-home `apply`, a user-controlled `curl | sh`, provider calls or network
// installation) and C1/C4 (one authoritative `site`/`base`, one base prefix in
// every emitted URL; T09 verification "unsafe example", "wrong project base").
// The real producers are exercised here: the frozen C3 runner table and
// `checkBuilt` over the built-artifact fixtures. The C3 runner keeps its table
// in code (it has no external-table validator), so a fixture-driven rejection
// of a mislabelled offline action remains `blocked` rather than faked.

import assert from 'node:assert/strict';
import path from 'node:path';
import test from 'node:test';

import {
  BASELINE_MANIFEST,
  blockedOptions,
  errorsWith,
  fixture,
  fixtureJson,
  fixturesRoot,
  loadFrozenApi,
  loadProducer,
  readJson,
  rel,
  repoRoot,
  requireError,
} from './negative-fixtures/lib/harness.mjs';

const api = await loadFrozenApi('website/scripts/lib/manifest.mjs', ['validateManifest']);
const skip = api.skip;
const checkApi = await loadProducer('check', ['checkBuilt']);
const examplesApi = await loadProducer('examples', ['EXAMPLES', 'EXAMPLE_IDS', 'HOME_PLACEHOLDER', 'runExamples']);
const SHIPPED_MANIFEST = path.join(repoRoot, 'website', 'docs-manifest.json');
const APPROVED_BASE = '/agentic-sdd';
const ORIGIN = 'https://example.github.io';

const EXAMPLES = examplesApi.mod?.EXAMPLES ?? [];
const HOME_PLACEHOLDER = examplesApi.mod?.HOME_PLACEHOLDER;
const MANUAL_EFFECTS = new Set(['manual', 'manual/external']);

const blocked = {};
function blockedCase(key, producer, absent, unadapted) {
  blocked[key] = blockedOptions(producer, absent, unadapted);
  return blocked[key];
}

const unsafe = fixtureJson('examples/unsafe.json');
const safe = fixtureJson('examples/safe.json');

/** Internal pathnames referenced by one built HTML fixture. */
function internalPaths(html) {
  return [...html.matchAll(/href="([^"]+)"/g)]
    .map((match) => new URL(match[1], `${ORIGIN}/`).pathname);
}

const built = (name) => fixture(path.join('built', name, 'index.html'));
const builtIndex = (name) => rel(path.join(fixturesRoot, 'built', name, 'index.html'));
const builtResult = (name) =>
  checkApi.mod.checkBuilt({
    repoRoot,
    manifestPath: BASELINE_MANIFEST,
    builtDir: path.join(fixturesRoot, 'built', name),
  });


// ---------------------------------------------------------------------------
// Example-table fixtures: the unsafe set and the accepting control
// ---------------------------------------------------------------------------

test('the unsafe example fixtures carry the required unsafe actions and no offline label', {}, () => {
  const required = ['apply-real-home', 'curl-pipe-sh', 'network-npm-install', 'provider-auth', 'remote-mcp-connect'];
  const ids = unsafe.cases.map((entry) => entry.id);
  for (const id of required) assert.ok(ids.includes(id), `unsafe.json must cover ${id}`);
  for (const entry of unsafe.cases) {
    assert.equal(entry.expectedVerdict, 'reject', `${entry.id}: an unsafe action must be expected to fail`);
    assert.notEqual(entry.effect, 'manual/external', `${entry.id}: fixture must model a mislabelled offline example`);
    assert.ok(entry.violation.length > 0, `${entry.id}: the fixture must state why the action is unsafe`);
  }
});

test('the accepting control keeps genuinely offline examples and a correct manual classification', {}, () => {
  const ids = safe.cases.map((entry) => entry.id);
  for (const entry of safe.cases) {
    assert.equal(entry.expectedVerdict, 'accept', `${entry.id}: the control must be acceptable`);
    if (entry.effect === 'offline/read-only') {
      assert.ok(!entry.argv.includes('apply'), `${entry.id}: apply is never offline/read-only`);
      assert.ok(!entry.argv.includes('install'), `${entry.id}: installation is never offline/read-only`);
    }
    if (entry.effect === 'offline/fixture-write') {
      const home = entry.argv[entry.argv.indexOf('--home') + 1] ?? '';
      assert.match(home, /^\/tmp\//, `${entry.id}: a fixture write must target a disposable root`);
    }
  }
  assert.ok(ids.includes('real-release-install-manual'), 'a correctly classified manual action must be allowed');
});

// ---------------------------------------------------------------------------
// Built-artifact base fixtures
// ---------------------------------------------------------------------------

test('the built-artifact fixtures distinguish one base prefix from a wrong one', {}, () => {
  const correct = internalPaths(built('correct'));
  assert.ok(correct.length >= 3, 'the correct fixture must carry canonical and navigation links');
  for (const url of correct) {
    assert.ok(url.startsWith(`${APPROVED_BASE}/`), `correct fixture ${url} must use the approved base`);
    assert.ok(!url.startsWith(`${APPROVED_BASE}${APPROVED_BASE}/`), `correct fixture ${url} must apply the base once`);
  }

  for (const [name, expected] of [
    ['wrong-base', (urls) => urls.every((url) => !url.startsWith(`${APPROVED_BASE}/`))],
    ['root-base', (urls) => urls.every((url) => !url.startsWith(`${APPROVED_BASE}/`))],
    ['double-base', (urls) => urls.some((url) => url.startsWith(`${APPROVED_BASE}${APPROVED_BASE}/`))],
  ]) {
    const urls = internalPaths(built(name));
    assert.ok(urls.length > 0, `${name} must contain links`);
    assert.ok(expected(urls), `${name} must actually exhibit its base defect: ${urls.join(', ')}`);
  }
});

test('the shipped manifest uses the one approved project base from the page contract', { skip }, () => {
  const manifest = readJson(SHIPPED_MANIFEST);
  assert.equal(
    manifest.base,
    APPROVED_BASE,
    'sitemap.md publishes /agentic-sdd/... URLs; a different base is a contract change, not a fixture edit',
  );
  assert.ok(manifest.pages.length > 0);
});

// ---------------------------------------------------------------------------
// Producer-dependent enforcement
// ---------------------------------------------------------------------------

test('the example table rejects an unsafe action mislabelled as offline', blockedCase(
  'example-table',
  'examples',
  'the C3 example runner is not present in this worktree',
  'the C3 runner owns a frozen in-code EXAMPLES table and exposes no validator for an external fixture table or its effect class, so it cannot ingest examples/unsafe.json to reject a mislabelled offline action (owner: T03/B2-W1-D examples.mjs)',
), () => {
  // Fixtures: examples/unsafe.json (reject) and examples/safe.json (control).
});

test('the example runner never executes a manual/external action', { skip: examplesApi.skip }, async () => {
  const manual = EXAMPLES.filter((example) => MANUAL_EFFECTS.has(example.effect));
  assert.ok(manual.length > 0, 'the frozen table must classify at least one manual/external example');
  for (const example of manual) {
    assert.deepEqual(example.argv, [], `${example.id}: a manual/external example has no runnable argv`);
    assert.ok(
      typeof example.reason === 'string' && example.reason.length > 0,
      `${example.id}: a manual exclusion must be explained on the entry`,
    );
    const summary = await examplesApi.mod.runExamples({ only: example.id });
    assert.equal(summary.status, 'pass', `${example.id}: selecting only a manual example must not fail`);
    assert.equal(summary.binaryPath, null, `${example.id}: a manual example must never require an executable`);
    assert.equal(summary.examples.length, 1, `${example.id}: the selection must report exactly that example`);
    const [result] = summary.examples;
    assert.equal(result.id, example.id);
    assert.equal(result.status, 'manual', `${example.id}: a manual example is reported, never executed`);
    assert.deepEqual(result.argv, [], `${example.id}: a manual result exposes no argv to run`);
    assert.equal(result.steps, undefined, `${example.id}: a manual example must run no step`);
  }
});

test('applying against a real home is never a runnable example', { skip: examplesApi.skip }, () => {
  assert.ok(EXAMPLES.length > 0, 'the frozen example table must not be empty');
  const templates = EXAMPLES.flatMap((example) => example.argv.map((argv) => ({ id: example.id, argv })));
  assert.ok(templates.length > 0, 'the frozen table must expose its runnable argv templates');
  // The runner refuses an `apply` argv at module load, so reaching this line is
  // itself evidence that no frozen example can write a real home.
  for (const { id, argv } of templates) {
    assert.ok(!argv.includes('apply'), `${id}: the runner must never execute apply`);
    assert.ok(!argv.includes('install'), `${id}: installation is not a runnable example`);
    const homeIndex = argv.indexOf('--home');
    if (homeIndex !== -1) {
      assert.equal(argv[homeIndex + 1], HOME_PLACEHOLDER, `${id}: --home must target the owned fixture root`);
    }
    for (const token of argv) {
      assert.ok(!/^\/(?:home|Users)\//.test(token), `${id}: "${token}" names a host home`);
    }
  }
  assert.ok(
    templates.some(({ argv }) => argv[0] === 'preview' && argv.includes('--home')),
    'the read-only preview example must still exercise --home',
  );
});

test('a provider or network install is never a runnable example', { skip: examplesApi.skip }, () => {
  const allowlisted = new Set(['version', 'preview']);
  const forbidden = new Set([
    'npm', 'npx', 'pnpm', 'yarn', 'curl', 'wget', 'brew', 'sh', 'bash', 'opencode',
    'install', 'login', 'auth', 'connect', '--remote',
  ]);
  for (const example of EXAMPLES) {
    for (const argv of example.argv) {
      assert.ok(allowlisted.has(argv[0]), `${example.id}: "${argv[0]}" is outside the frozen fixed-argv allowlist`);
      for (const token of argv) {
        assert.ok(!forbidden.has(token), `${example.id}: "${token}" is a provider/network action, never a runnable example`);
      }
    }
  }
  assert.ok(
    EXAMPLES.some((example) => example.argv.length > 0),
    'the frozen table must keep at least one runnable read-only example',
  );
});

test('the built-artifact check rejects a wrong project base', { skip: checkApi.skip }, () => {
  const result = builtResult('wrong-base');
  const baseErrors = requireError(result, 'built/wrong-base', builtIndex('wrong-base'));
  assert.equal(baseErrors.length, 2, 'every internal link with the wrong base must be reported');
  for (const entry of baseErrors) {
    assert.match(entry.details.href, /^\/documentation\//, 'the diagnostic must name the offending href');
    assert.match(entry.message, /does not start with the manifest base "\/agentic-sdd"/);
  }
  const [canonical] = requireError(result, 'built/canonical-mismatch', builtIndex('wrong-base'));
  assert.equal(canonical.details.actual, 'https://example.github.io/documentation/reference/cli/');
  assert.equal(canonical.details.expected, 'https://example.github.io/agentic-sdd/');
  assert.equal(result.ok, false);

  const control = builtResult('correct');
  assert.deepEqual(
    errorsWith(control, 'built/wrong-base', builtIndex('correct')),
    [],
    'the approved base must not be reported as a wrong base',
  );
});

test('the built-artifact check rejects a root base or a doubled project base', { skip: checkApi.skip }, () => {
  const rootBase = builtResult('root-base');
  const [rootError] = requireError(rootBase, 'built/wrong-base', builtIndex('root-base'));
  assert.match(rootError.message, /"\/reference\/cli\/" does not start with the manifest base "\/agentic-sdd"/);
  const [rootCanonical] = requireError(rootBase, 'built/canonical-mismatch', builtIndex('root-base'));
  assert.equal(rootCanonical.details.actual, 'https://example.github.io/reference/cli/');
  assert.equal(rootBase.ok, false);

  const doubled = builtResult('double-base');
  const doubledHref = '/agentic-sdd/agentic-sdd/reference/cli/';
  const [doubledCanonical] = requireError(doubled, 'built/canonical-mismatch', builtIndex('double-base'));
  assert.equal(doubledCanonical.details.actual, `https://example.github.io${doubledHref}`);
  assert.deepEqual(
    errorsWith(doubled, 'built/wrong-base'),
    [],
    'a doubled base still starts with the approved base, so it must not be misreported as a wrong base',
  );
  const named = doubled.errors.filter((entry) => `${entry.message} ${JSON.stringify(entry.details)}`.includes(doubledHref));
  assert.ok(named.length > 0, 'the doubled-base link must be named in a failure, never silently accepted');
  assert.equal(doubled.ok, false);
});

