// Spec 020 B1-W1-D — independent F01–F11 counterexamples for the frozen
// resolveLink contract (plan C1 route/plugin interface, B1 contract §C1).
//
// Expectations below are written from the frozen URL contract, not from the
// implementation: root/project base, one base prefix, Markdown twins, GitHub
// source links, fragment preservation without double encoding, and the
// unchanged/failing reference classes. No production resolver is recreated here.

import assert from 'node:assert/strict';
import path from 'node:path';
import test from 'node:test';

import {
  INDEX_SOURCE,
  PUBLIC_DOCS,
  VALID_MANIFEST,
  assertApiError,
  capture,
  clone,
  fixtureTree,
  isErrorCode,
  isNonEmptyString,
  loadFrozenApi,
  readJson,
  reportPending,
  snapshot,
} from './foundation-negative-support/support.mjs';

const api = await loadFrozenApi('website/scripts/lib/routes.mjs', ['resolveLink']);
reportPending('foundation-negative-routes', api);

const skip = api.skip;
const R = api.mod;
const manifest = readJson(VALID_MANIFEST);
const GUIDE_SOURCE = `${PUBLIC_DOCS}/guides/other page.md`;
const CLI_SOURCE = `${PUBLIC_DOCS}/reference/cli.md`;

const resolve = (overrides = {}) =>
  R.resolveLink({
    manifest,
    repoRoot: fixtureTree,
    sourcePath: INDEX_SOURCE,
    href: './index.md',
    ...overrides,
  });

const routeFields = {
  code: isErrorCode,
  source: isNonEmptyString,
  href: isNonEmptyString,
};

function expectRoute(label, overrides, expected) {
  test(label, { skip }, () => {
    assert.equal(resolve(overrides), expected);
  });
}

function expectRouteError(label, overrides) {
  test(label, { skip }, () => {
    const href = overrides.href;
    const input = snapshot(overrides);
    const { threw, error } = capture(() => resolve(overrides));
    assert.ok(threw, `${label}: resolveLink accepted an unresolvable reference`);
    assertApiError(error, { name: 'RouteError', fields: routeFields });
    assert.equal(error.href, href, `${label}: RouteError.href must echo the offending href`);
    assert.equal(snapshot(overrides), input, `${label}: resolveLink must not mutate its arguments`);
  });
}

// ---------------------------------------------------------------------------
// F01 — project base and root index (target html, the default)
// ---------------------------------------------------------------------------

expectRoute('html: relative page link uses the project base', { href: './reference/cli.md' },
  '/agentic-sdd/reference/cli/');
expectRoute('html: root index canonical route is base plus one slash', { href: './index.md' },
  '/agentic-sdd/');
expectRoute('html: cross-directory page link resolves through the manifest source map',
  { sourcePath: GUIDE_SOURCE, href: '../reference/cli.md' }, '/agentic-sdd/reference/cli/');
expectRoute('html: parent link back to the root index', { sourcePath: CLI_SOURCE, href: '../index.md' },
  '/agentic-sdd/');

test('html: result carries exactly one base prefix and never the site origin', { skip }, () => {
  const result = resolve({ href: './reference/cli.md' });
  assert.equal(result.split('/agentic-sdd').length - 1, 1, `expected one base prefix: ${result}`);
  assert.ok(!result.includes('example.github.io'), `site origin leaked into a relative link: ${result}`);
  assert.ok(result.startsWith('/'), `html result must be rooted: ${result}`);
});

test('html: absolute-contained sourcePath matches the repository-relative result', { skip }, () => {
  const absolute = path.join(fixtureTree, INDEX_SOURCE);
  assert.equal(resolve({ sourcePath: absolute, href: './reference/cli.md' }),
    resolve({ sourcePath: INDEX_SOURCE, href: './reference/cli.md' }));
});

test('html: resolution is deterministic', { skip }, () => {
  assert.equal(resolve({ href: './reference/cli.md' }), resolve({ href: './reference/cli.md' }));
});

test('html: a root base produces a single leading slash', { skip }, () => {
  const rootBase = clone(manifest);
  rootBase.base = '';
  const call = (href) =>
    R.resolveLink({ manifest: rootBase, repoRoot: fixtureTree, sourcePath: INDEX_SOURCE, href });
  assert.equal(call('./reference/cli.md'), '/reference/cli/');
  assert.equal(call('./index.md'), '/');
  for (const href of ['./reference/cli.md', './index.md']) {
    assert.doesNotMatch(call(href), /\/\//, `double slash with a root base: ${call(href)}`);
  }
});

// ---------------------------------------------------------------------------
// F02/F03 — fragments and encoded spaces (never double encoded)
// ---------------------------------------------------------------------------

expectRoute('html: fragment on the root index is preserved', { href: './index.md#foundation-validation' },
  '/agentic-sdd/#foundation-validation');
expectRoute('html: encoded page path resolves by manifest source', { href: './guides/other%20page.md' },
  '/agentic-sdd/guides/other-page/');
expectRoute('html: fragment on a resolved page keeps the separator', { href: './guides/other%20page.md#fixture-heading' },
  '/agentic-sdd/guides/other-page/#fixture-heading');

test('html: an already encoded fragment is not double encoded', { skip }, () => {
  const result = resolve({ href: './guides/other%20page.md#a%20b' });
  assert.ok(result.endsWith('#a%20b'), `expected a single-encoded fragment: ${result}`);
  assert.ok(!result.includes('%2520'), `fragment was double encoded: ${result}`);
});

test('html: a pure fragment reference is unchanged', { skip }, () => {
  assert.equal(resolve({ href: '#real-heading' }), '#real-heading');
});

// ---------------------------------------------------------------------------
// F05/F06/F07 — external, mail and image references stay untouched
// ---------------------------------------------------------------------------

test('external HTTPS reference is unchanged', { skip }, () => {
  const href = 'https://example.com/guide/path?q=1#frag';
  assert.equal(resolve({ href }), href);
});

test('mailto reference is unchanged', { skip }, () => {
  const href = 'mailto:docs@example.com';
  assert.equal(resolve({ href }), href);
});

test('image reference stays in the asset pipeline', { skip }, () => {
  const href = './assets/diagram.png';
  assert.equal(resolve({ href }), href);
});

// ---------------------------------------------------------------------------
// F04 — Markdown twins
// ---------------------------------------------------------------------------

expectRoute('markdown twin for a nested page', { href: './reference/cli.md', target: 'markdown' },
  '/agentic-sdd/reference/cli/index.md');
expectRoute('markdown twin for the root index', { href: './index.md', target: 'markdown' },
  '/agentic-sdd/index.md');
expectRoute('markdown twin keeps the fragment', { href: './reference/cli.md#flag-list', target: 'markdown' },
  '/agentic-sdd/reference/cli/index.md#flag-list');

test('markdown twin carries exactly one base prefix', { skip }, () => {
  const result = resolve({ href: './reference/cli.md', target: 'markdown' });
  assert.equal(result.split('/agentic-sdd').length - 1, 1, result);
  assert.ok(!result.includes('example.github.io'), result);
});

// ---------------------------------------------------------------------------
// F06 — repository source links
// ---------------------------------------------------------------------------

const blob = (ref, repoPath) => `https://github.com/example/repo/blob/${ref}/${repoPath}`;

expectRoute('source link for a nested page source', { href: './reference/cli.md', target: 'source' },
  blob('main', CLI_SOURCE));
expectRoute('source link honours the explicit ref', { href: './reference/cli.md', target: 'source', ref: 'v0.11.0' },
  blob('v0.11.0', CLI_SOURCE));
expectRoute('source link encodes a space in the repository path',
  { href: './guides/other%20page.md', target: 'source' }, blob('main', 'website/src/content/docs/guides/other%20page.md'));
expectRoute('a contained non-public Markdown path becomes a repository source link',
  { href: '../../../../docs/getting-started.md', target: 'source' }, blob('main', 'docs/getting-started.md'));

test('html target never invents a site route for a non-public source', { skip }, () => {
  const href = '../../../../docs/getting-started.md';
  const { threw, value, error } = capture(() => resolve({ href }));
  if (threw) {
    assertApiError(error, { name: 'RouteError', fields: routeFields });
    assert.equal(error.href, href);
  } else {
    assert.match(String(value), /^https:\/\/github\.com\/example\/repo\//,
      `a non-public source must not become a site route: ${String(value)}`);
    assert.ok(!String(value).startsWith('/agentic-sdd/'), `invented site route: ${String(value)}`);
  }
});

// ---------------------------------------------------------------------------
// Unsafe schemes, unknown targets, escapes and missing context
// ---------------------------------------------------------------------------

expectRouteError('unknown public Markdown target fails', { href: './missing.md' });
expectRouteError('path escape out of the repository fails', { href: '../../../../../etc/passwd.md' });
expectRouteError('javascript scheme fails', { href: 'javascript:alert(1)' });
expectRouteError('data scheme fails', { href: 'data:text/html,<script>alert(1)</script>' });
expectRouteError('file scheme fails', { href: 'file:///etc/passwd' });
expectRouteError('source target path escape fails', { href: '../../../../../etc/passwd.md', target: 'source' });
expectRouteError('sourcePath outside the repository root fails', { sourcePath: '/etc/hosts', href: './index.md' });

test('missing source-file context fails', { skip }, () => {
  const { threw, error } = capture(() =>
    R.resolveLink({ manifest, repoRoot: fixtureTree, href: './index.md' }));
  assert.ok(threw, 'resolveLink must reject a missing sourcePath');
  // No sourcePath exists to echo, so only the code and href fields are required here.
  assertApiError(error, { name: 'RouteError', fields: { code: isErrorCode, href: isNonEmptyString } });
});

test('an unsupported target option is rejected', { skip }, () => {
  assert.throws(() => resolve({ href: './index.md', target: 'pdf' }));
});

test('resolveLink does not mutate the manifest', { skip }, () => {
  const before = snapshot(manifest);
  resolve({ href: './reference/cli.md', target: 'source' });
  resolve({ href: '#fragment' });
  assert.equal(snapshot(manifest), before);
});
