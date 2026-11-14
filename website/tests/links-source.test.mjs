// SPEC020 B4-W1-D — source-mode link, fragment, marker, ownership and stub
// negative fixtures. Each case asserts an actionable file/line/fix diagnostic,
// not an internal implementation detail.
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { spawnSync } from 'node:child_process';
import { linkSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { computeReviewDigest } from '../scripts/lib/manifest.mjs';
import { classifyLinkTarget } from '../scripts/lib/public-paths.mjs';
import { checkSource } from '../scripts/check.mjs';

const testsDir = path.dirname(fileURLToPath(import.meta.url));
const websiteDir = path.resolve(testsDir, '..');
const checkScript = path.join(websiteDir, 'scripts/check.mjs');
const DOCS = 'website/src/content/docs';

function sha256(value) {
  return createHash('sha256').update(value).digest('hex');
}

function entry({ id, source, body, extra = {}, sources }) {
  const text = `---\ntitle: ${id}\n---\n\n${body}\n`;
  const evidence = sources ?? [{ path: source, sha256: sha256(text) }];
  return {
    id,
    source,
    type: 'reference',
    audience: ['maintainer'],
    class: 'L',
    sources: evidence,
    reviewDigest: computeReviewDigest(evidence),
    appliesTo: 'unreleased',
    reviewStatus: 'unreviewed',
    ...extra,
  };
}

function makeRepo(t, { pages, base = '/agentic-sdd', aliases, stubs = {} }) {
  mkdirSync('/tmp/opencode', { recursive: true });
  const repoRoot = mkdtempSync('/tmp/opencode/spec020-b4-links-');
  t.after(() => rmSync(repoRoot, { recursive: true, force: true }));

  const manifest = {
    schemaVersion: 1,
    site: 'https://docs.example.test',
    base,
    repository: 'https://github.com/example/repo',
    docsVersion: 'dev',
    pages: [],
  };
  if (aliases !== undefined) manifest.aliases = aliases;
  const bodies = new Map();
  for (const page of pages) {
    const absolute = path.join(repoRoot, page.source);
    mkdirSync(path.dirname(absolute), { recursive: true });
    const text = `---\ntitle: ${page.id}\n---\n\n${page.body}\n`;
    writeFileSync(absolute, text, 'utf8');
    bodies.set(page.id, text);
    manifest.pages.push(entry({ ...page, body: page.body }));
  }
  for (const [stubPath, stubBody] of Object.entries(stubs)) {
    const absolute = path.join(repoRoot, stubPath);
    mkdirSync(path.dirname(absolute), { recursive: true });
    writeFileSync(absolute, `# Compatibility stub\n\n${stubBody}\n`, 'utf8');
  }
  const manifestPath = path.join(repoRoot, 'website/docs-manifest.json');
  mkdirSync(path.dirname(manifestPath), { recursive: true });
  writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`, 'utf8');
  return { repoRoot, manifestPath, manifest, bodies };
}

function codes(result) {
  return result.errors.map((entry) => entry.code).sort();
}

function find(result, code) {
  return result.errors.find((entry) => entry.code === code);
}

test('reports a broken heading fragment with path, line and fix', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [
      { id: 'a', source: `${DOCS}/a.md`, body: 'Intro\n\n[b](./b.md#missing)' },
      { id: 'b', source: `${DOCS}/b.md`, body: '## Present\n\nText.' },
    ],
  });
  const result = checkSource({ repoRoot });
  assert.equal(result.ok, false);
  const error = find(result, 'links/broken-anchor');
  assert.ok(error, `expected links/broken-anchor, got ${JSON.stringify(codes(result))}`);
  assert.equal(error.path, `${DOCS}/a.md`);
  assert.equal(error.line, 7);
  assert.match(error.message, /#missing/);
  assert.match(error.fix, /heading/);
});

test('reports a missing relative target instead of accepting the link', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [{ id: 'index', source: `${DOCS}/index.md`, body: 'Intro\n\n[missing](./missing.md)' }],
  });
  const result = checkSource({ repoRoot });
  assert.equal(result.ok, false);
  const error = find(result, 'links/unknown-target');
  assert.ok(error, `expected links/unknown-target, got ${JSON.stringify(codes(result))}`);
  assert.equal(error.path, `${DOCS}/index.md`);
  assert.equal(error.line, 7);
  assert.ok(error.fix.length > 20);
});

test('ignores links inside fenced code and inline code', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [
      {
        id: 'index',
        source: `${DOCS}/index.md`,
        body: [
          '````markdown',
          '[fake](./missing.md)',
          '````',
          '',
          'Inline `[also fake](./missing-too.md)` is not a link.',
        ].join('\n'),
      },
    ],
  });
  const result = checkSource({ repoRoot });
  assert.equal(result.ok, true, JSON.stringify(result.errors));
  assert.equal(result.summary.links, 0);
});

test('resolves percent-encoded Markdown paths once', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [
      { id: 'a', source: `${DOCS}/a.md`, body: 'Encoded [space](./some%20page.md#present)' },
      { id: 'space-page', source: `${DOCS}/some page.md`, body: '## Present' },
    ],
  });
  const result = checkSource({ repoRoot });
  assert.equal(result.ok, true, JSON.stringify(result.errors));
  assert.equal(result.summary.links, 1);
});

test('rejects a hard-coded project base in source links', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [
      { id: 'a', source: `${DOCS}/a.md`, body: '[cli](/agentic-sdd/reference/cli/)' },
      { id: 'reference/cli', source: `${DOCS}/reference/cli.md`, body: '## CLI' },
    ],
  });
  const result = checkSource({ repoRoot });
  assert.equal(result.ok, false);
  const error = find(result, 'links/path-escape');
  assert.ok(error, `expected links/path-escape, got ${JSON.stringify(codes(result))}`);
  assert.match(error.fix, /\/agentic-sdd/);
});

test('fails duplicate body ownership while allowing distinct bodies', (t) => {
  const shared = '## Same\n\nThe same body.';
  const { repoRoot } = makeRepo(t, {
    pages: [
      { id: 'a', source: `${DOCS}/a.md`, body: shared },
      { id: 'b', source: `${DOCS}/b.md`, body: shared },
    ],
  });
  const result = checkSource({ repoRoot });
  const error = find(result, 'ownership/duplicate-body');
  assert.ok(error, `expected ownership/duplicate-body, got ${JSON.stringify(codes(result))}`);
  assert.equal(error.path, `${DOCS}/b.md`);
  assert.equal(result.summary.pages, 2);
});

test('fails duplicate canonical source ownership through the manifest contract', (t) => {
  const { repoRoot, manifestPath } = makeRepo(t, {
    pages: [{ id: 'a', source: `${DOCS}/a.md`, body: '## One' }],
  });
  const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
  manifest.pages.push({ ...manifest.pages[0], id: 'b' });
  writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`, 'utf8');
  const result = checkSource({ repoRoot });
  assert.equal(result.ok, false);
  assert.ok(codes(result).includes('manifest/duplicate-page-source'), JSON.stringify(codes(result)));
  assert.equal(result.errors[0].path, 'website/docs-manifest.json');
  assert.ok(result.errors[0].fix.length > 20);
});

test('fails duplicate legacy alias ownership', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [{ id: 'index', source: `${DOCS}/index.md`, body: '## Index' }],
    aliases: ['docs/old.md', 'docs/old.md'],
  });
  const result = checkSource({ repoRoot });
  assert.ok(codes(result).includes('ownership/duplicate-alias'), JSON.stringify(codes(result)));
});

test('fails a legacy alias that collides with a canonical URL', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [{ id: 'index', source: `${DOCS}/index.md`, body: '## Index' }],
    aliases: ['https://docs.example.test/agentic-sdd/'],
  });
  const result = checkSource({ repoRoot });
  const error = find(result, 'ownership/alias-shadows-url');
  assert.ok(error, `expected ownership/alias-shadows-url, got ${JSON.stringify(codes(result))}`);
  assert.match(error.fix, /unique/);
});

test('fails duplicate authored heading slugs', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [{ id: 'index', source: `${DOCS}/index.md`, body: '## Same\n\n## Same' }],
  });
  const result = checkSource({ repoRoot });
  const error = find(result, 'headings/duplicate');
  assert.ok(error, `expected headings/duplicate, got ${JSON.stringify(codes(result))}`);
  assert.match(error.fix, /unique/);
});

test('reports external links as observations and never resolves them', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [
      {
        id: 'index',
        source: `${DOCS}/index.md`,
        body:
          '[site](https://example.test/page), [mail](mailto:docs@example.test) and <https://example.test/auto>',
      },
    ],
  });
  const result = checkSource({ repoRoot });
  assert.equal(result.ok, true, JSON.stringify(result.errors));
  assert.equal(result.observations.length, 1);
  const observation = result.observations[0];
  assert.equal(observation.code, 'links/external');
  assert.equal(observation.details.hrefs.length, 3);
  assert.ok(observation.details.hrefs.includes('https://example.test/auto'));
  assert.equal(result.summary.external, 3);
});

test('rejects raw HTML navigation rather than silently ignoring it', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [
      {
        id: 'index',
        source: `${DOCS}/index.md`,
        body: '<a href="./missing.md">raw link</a>',
      },
    ],
  });
  const result = checkSource({ repoRoot });
  const error = find(result, 'html/raw-navigation');
  assert.ok(error, `expected html/raw-navigation, got ${JSON.stringify(codes(result))}`);
  assert.equal(error.path, `${DOCS}/index.md`);
  assert.equal(error.line, 5);
  assert.equal(error.details.href, './missing.md');
  assert.match(error.fix, /Markdown link/);
});

test('still rejects raw HTML navigation whose opening tag spans lines', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [
      {
        id: 'index',
        source: `${DOCS}/index.md`,
        body: '<a\n  class="nav"\n  href="./missing.md">raw link</a>',
      },
    ],
  });
  const result = checkSource({ repoRoot });
  const error = find(result, 'html/raw-navigation');
  assert.ok(error, `expected html/raw-navigation, got ${JSON.stringify(codes(result))}`);
  assert.equal(error.line, 5);
  assert.equal(error.details.href, './missing.md');
});

test('keeps a retained <a id> anchor usable as a fragment target', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [
      {
        id: 'index',
        source: `${DOCS}/index.md`,
        body: 'Jump to [the retained anchor](./legacy.md#legacy-anchor).',
      },
      {
        id: 'legacy',
        source: `${DOCS}/legacy.md`,
        body: [
          '<a id="legacy-anchor"></a>',
          '<a name="legacy-name"></a>',
          '',
          '## Current heading',
          '',
          '[Back to the index](./index.md).',
        ].join('\n'),
      },
    ],
  });
  const result = checkSource({ repoRoot });
  assert.equal(result.ok, true, JSON.stringify(result.errors));
  assert.deepEqual(codes(result), []);
  assert.equal(find(result, 'html/raw-navigation'), undefined);
});

test('still rejects raw HTML navigation that carries an id and an href', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [
      {
        id: 'index',
        source: `${DOCS}/index.md`,
        body: '<a id="spot" href="./missing.md">raw link</a>',
      },
    ],
  });
  const result = checkSource({ repoRoot });
  const error = find(result, 'html/raw-navigation');
  assert.ok(error, `expected html/raw-navigation, got ${JSON.stringify(codes(result))}`);
  assert.equal(error.details.href, './missing.md');
  assert.equal(result.ok, false);
});

test('keeps a retained <a id> anchor usable in a compatibility stub', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [{ id: 'reference/cli', source: `${DOCS}/reference/cli.md`, body: '## CLI' }],
    stubs: {
      'docs/getting-started.md': [
        '<a id="install-linux"></a>',
        '',
        'This guide moved to the [CLI reference](../website/src/content/docs/reference/cli.md).',
      ].join('\n'),
    },
  });
  const result = checkSource({
    repoRoot,
    stubs: [
      { path: 'docs/getting-started.md', canonicalId: 'reference/cli', anchors: ['install-linux'] },
    ],
  });
  assert.equal(result.ok, true, JSON.stringify(result.errors));
  assert.deepEqual(codes(result), []);
  assert.deepEqual(
    result.errors.filter((entry) => entry.code.startsWith('stub/')),
    [],
    'the stub id anchor is not raw navigation and it satisfies the retained-anchor declaration',
  );
});

test('rejects a hard-linked canonical page source', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [{ id: 'a', source: `${DOCS}/a.md`, body: '## A' }],
  });
  linkSync(path.join(repoRoot, `${DOCS}/a.md`), path.join(repoRoot, 'mirror.md'));
  const result = checkSource({ repoRoot });
  const error = find(result, 'source/hardlink');
  assert.ok(error, `expected source/hardlink, got ${JSON.stringify(codes(result))}`);
  assert.equal(error.path, `${DOCS}/a.md`);
  assert.match(error.message, /hard link/i);
  assert.match(error.fix, /single-linked/);
  assert.equal(result.ok, false);
});

test('rejects a hard-linked non-page link target', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [{ id: 'index', source: `${DOCS}/index.md`, body: '[policy](../../assets/policy.txt)' }],
  });
  const assetPath = path.join(repoRoot, 'website/src/assets/policy.txt');
  mkdirSync(path.dirname(assetPath), { recursive: true });
  writeFileSync(assetPath, 'policy\n', 'utf8');
  linkSync(assetPath, path.join(repoRoot, 'policy-copy.txt'));
  const result = checkSource({ repoRoot });
  const error = find(result, 'links/hardlink-target');
  assert.ok(error, `expected links/hardlink-target, got ${JSON.stringify(codes(result))}`);
  assert.equal(error.details.target, 'website/src/assets/policy.txt');
  assert.equal(result.ok, false);
});

test('accepts the root project base without a double-prefix false positive', (t) => {
  const { repoRoot } = makeRepo(t, {
    base: '/',
    pages: [
      { id: 'a', source: `${DOCS}/a.md`, body: '[b](./b.md#present)' },
      { id: 'b', source: `${DOCS}/b.md`, body: '## Present' },
    ],
  });
  const result = checkSource({ repoRoot });
  assert.equal(result.ok, true, JSON.stringify(result.errors));
});

test('requires registered generated markers on the page', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [
      {
        id: 'reference/cli',
        source: `${DOCS}/reference/cli.md`,
        body: '## CLI',
        extra: {
          generatedSections: [
            {
              id: 'reference/cli',
              producer: 'website/scripts/example.mjs',
              markerStart: '<!-- generated:reference/cli:start -->',
              markerEnd: '<!-- generated:reference/cli:end -->',
              inputs: [`${DOCS}/reference/cli.md`],
            },
          ],
        },
      },
    ],
  });
  const result = checkSource({ repoRoot });
  assert.ok(codes(result).includes('generated/marker-missing'), JSON.stringify(codes(result)));
  const error = find(result, 'generated/marker-missing');
  assert.equal(error.path, `${DOCS}/reference/cli.md`);
  assert.match(error.fix, /generated/);
});

test('accepts a registered generated marker pair', (t) => {
  const body = [
    'Inline `<!-- generated:reference/cli:start -->` is an example, not a marker.',
    '',
    '```md',
    '<!-- generated:reference/cli:end -->',
    '```',
    '',
    '<!-- generated:reference/cli:start -->',
    '## Generated',
    '<!-- generated:reference/cli:end -->',
  ].join('\n');
  const { repoRoot } = makeRepo(t, {
    pages: [
      {
        id: 'reference/cli',
        source: `${DOCS}/reference/cli.md`,
        body,
        extra: {
          generatedSections: [
            {
              id: 'reference/cli',
              producer: 'website/scripts/example.mjs',
              markerStart: '<!-- generated:reference/cli:start -->',
              markerEnd: '<!-- generated:reference/cli:end -->',
              inputs: [`${DOCS}/reference/cli.md`],
            },
          ],
        },
      },
    ],
  });
  const result = checkSource({ repoRoot });
  assert.equal(result.ok, true, JSON.stringify(result.errors));
});

test('rejects a compatibility stub containing a code fence', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [{ id: 'reference/cli', source: `${DOCS}/reference/cli.md`, body: '## CLI' }],
    stubs: {
      'docs/getting-started.md': [
        'This guide moved to the [CLI reference](../website/src/content/docs/reference/cli.md).',
        '',
        '```sh',
        'asdx-core --help',
        '```',
      ].join('\n'),
    },
  });
  const result = checkSource({
    repoRoot,
    stubs: [{ path: 'docs/getting-started.md', canonicalId: 'reference/cli' }],
  });
  assert.equal(result.ok, false);
  const error = find(result, 'stub/code-fence');
  assert.ok(error, `expected stub/code-fence, got ${JSON.stringify(codes(result))}`);
  assert.equal(error.path, 'docs/getting-started.md');
  assert.equal(error.line, 5);
  assert.match(error.fix, /canonical links/);
});

test('rejects a compatibility stub containing a numbered procedure', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [{ id: 'reference/cli', source: `${DOCS}/reference/cli.md`, body: '## CLI' }],
    stubs: {
      'docs/getting-started.md': [
        'Canonical guide: [CLI](../website/src/content/docs/reference/cli.md).',
        '',
        '1. Install the toolkit.',
      ].join('\n'),
    },
  });
  const result = checkSource({
    repoRoot,
    stubs: [{ path: 'docs/getting-started.md', canonicalId: 'reference/cli' }],
  });
  const error = find(result, 'stub/procedure');
  assert.ok(error, `expected stub/procedure, got ${JSON.stringify(codes(result))}`);
  assert.equal(error.line, 5);
});

test('requires the stub canonical link and declared retained anchors', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [{ id: 'reference/cli', source: `${DOCS}/reference/cli.md`, body: '## CLI' }],
    stubs: {
      'docs/getting-started.md': [
        '## Quick start',
        '',
        'Historical pointer: [old guide](./gone.md).',
      ].join('\n'),
    },
  });
  const result = checkSource({
    repoRoot,
    stubs: [
      {
        path: 'docs/getting-started.md',
        canonicalId: 'reference/cli',
        anchors: ['quick-start', 'support-and-release-scope'],
      },
    ],
  });
  assert.ok(codes(result).includes('stub/missing-canonical'), JSON.stringify(codes(result)));
  const anchor = find(result, 'stub/missing-anchor');
  assert.ok(anchor, `expected stub/missing-anchor, got ${JSON.stringify(codes(result))}`);
  assert.match(anchor.message, /support-and-release-scope/);
  assert.ok(codes(result).includes('links/missing-file'), JSON.stringify(codes(result)));
});

test('fails a public content page that is absent from the manifest', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [{ id: 'index', source: `${DOCS}/index.md`, body: '## Home' }],
  });
  const extra = path.join(repoRoot, DOCS, 'notes/extra.md');
  mkdirSync(path.dirname(extra), { recursive: true });
  writeFileSync(extra, '---\ntitle: Extra\n---\n\n## Unregistered\n', 'utf8');

  const result = checkSource({ repoRoot });
  const error = find(result, 'source/unregistered-page');
  assert.ok(error, `expected source/unregistered-page, got ${JSON.stringify(codes(result))}`);
  assert.equal(error.path, `${DOCS}/notes/extra.md`);
  assert.equal(error.severity, 'error');
  assert.match(error.fix, /docs-manifest\.json/);
  assert.match(error.fix, /remove/);
  assert.equal(result.ok, false);
});

test('rejects links into private and history trees but keeps public engineering sources linkable', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [
      {
        id: 'index',
        source: `${DOCS}/index.md`,
        body: [
          '[handover](../../../../docs/handovers/2026-11-14_020.md)',
          '',
          '[spec](../../../../specs/020-documentation-ecosystem/spec.md)',
          '',
          '[run](../../../../.agentic-sdd/runs/020/checkpoint.json)',
          '',
          '[skill](../../../../skills-files/agentic-sdd-router/SKILL.md)',
        ].join('\n'),
      },
    ],
  });
  for (const relative of [
    'docs/handovers/2026-11-14_020.md',
    'specs/020-documentation-ecosystem/spec.md',
    '.agentic-sdd/runs/020/checkpoint.json',
    'skills-files/agentic-sdd-router/SKILL.md',
  ]) {
    const absolute = path.join(repoRoot, relative);
    mkdirSync(path.dirname(absolute), { recursive: true });
    writeFileSync(absolute, relative.endsWith('.md') ? '## Present\n' : '{}\n', 'utf8');
  }

  const result = checkSource({ repoRoot });
  const privateErrors = result.errors.filter((entry) => entry.code === 'links/private-target');
  assert.deepEqual(
    privateErrors.map((entry) => `${entry.details.target}:${entry.details.category}`).sort(),
    [
      '.agentic-sdd/runs/020/checkpoint.json:private',
      'docs/handovers/2026-11-14_020.md:history',
      'specs/020-documentation-ecosystem/spec.md:history',
    ],
  );
  for (const error of privateErrors) {
    assert.equal(error.path, `${DOCS}/index.md`);
    assert.ok(error.line > 0);
    assert.match(error.fix, /public page/);
  }
  assert.deepEqual(
    result.errors.filter((entry) => entry.details.target === 'skills-files/agentic-sdd-router/SKILL.md'),
    [],
    'a public engineering source stays linkable through its repository source URL',
  );
  assert.equal(result.ok, false);
});

test('the documented historical-spec exception is narrow and verified against the retained stub', () => {
  const stub = readFileSync(path.join(websiteDir, '..', 'docs', 'mcp-tools.md'), 'utf8');
  assert.ok(
    stub.includes('](../specs/015-fleet-manager-tui/spec.md)'),
    'the retained MCP migration stub must still carry its deliberate historical-spec pointer',
  );

  const allowed = classifyLinkTarget('specs/015-fleet-manager-tui/spec.md', { source: 'docs/mcp-tools.md' });
  assert.deepEqual(allowed, { allowed: true, category: 'documented-exception' });

  const fromPage = classifyLinkTarget('specs/015-fleet-manager-tui/spec.md', {
    source: `${DOCS}/index.md`,
  });
  assert.deepEqual(fromPage, { allowed: false, category: 'history' });

  const otherSpec = classifyLinkTarget('specs/020-documentation-ecosystem/spec.md', {
    source: 'docs/mcp-tools.md',
  });
  assert.deepEqual(otherSpec, { allowed: false, category: 'history' }, 'the exception names one exact target');
});

test('fails a fragment whose existing target cannot be re-parsed', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [
      { id: 'index', source: `${DOCS}/index.md`, body: '[large](../../../tests/large-target.md#present)' },
    ],
  });
  const target = path.join(repoRoot, 'website/tests/large-target.md');
  mkdirSync(path.dirname(target), { recursive: true });
  writeFileSync(target, `## Present\n\n${'x'.repeat(1100 * 1024)}\n`, 'utf8');

  const result = checkSource({ repoRoot });
  const error = find(result, 'links/fragment-unverified');
  assert.ok(error, `expected links/fragment-unverified, got ${JSON.stringify(codes(result))}`);
  assert.equal(error.severity, 'error');
  assert.equal(error.details.target, 'website/tests/large-target.md');
  assert.match(error.fix, /parse cap|fails closed/);
  assert.equal(
    result.observations.some((entry) => entry.code === 'links/fragment-unverified'),
    false,
    'an existing target is never downgraded to an observation',
  );
  assert.equal(result.ok, false);
});

test('markdownlint config is valid JSON with exactly scoped rule exceptions', () => {
  const config = JSON.parse(readFileSync(path.join(websiteDir, '.markdownlint.json'), 'utf8'));
  assert.equal(config.default, true);
  assert.deepEqual(Object.keys(config).sort(), ['MD013', 'default']);
  assert.equal(config.MD013, false);
});

test('CLI emits deterministic JSON and exits nonzero on a failure', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [{ id: 'index', source: `${DOCS}/index.md`, body: '[missing](./missing.md)' }],
  });
  const first = spawnSync(process.execPath, [checkScript, '--repo', repoRoot, '--json'], { encoding: 'utf8' });
  const second = spawnSync(process.execPath, [checkScript, '--repo', repoRoot, '--json'], { encoding: 'utf8' });
  assert.equal(first.status, 1, first.stderr);
  assert.equal(first.stdout, second.stdout);
  const report = JSON.parse(first.stdout);
  assert.equal(report.ok, false);
  assert.equal(report.errors[0].path, `${DOCS}/index.md`);
  assert.ok(report.errors[0].line > 0);
  assert.ok(report.errors[0].fix.length > 0);
});

test('CLI accepts an exact --stub declaration', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [{ id: 'reference/cli', source: `${DOCS}/reference/cli.md`, body: '## CLI' }],
    stubs: {
      'docs/getting-started.md': [
        'Moved to [CLI](../website/src/content/docs/reference/cli.md).',
        '',
        '```sh',
        'asdx-core --help',
        '```',
      ].join('\n'),
    },
  });
  const run = spawnSync(
    process.execPath,
    [
      checkScript,
      '--repo',
      repoRoot,
      '--stub',
      'docs/getting-started.md=reference/cli',
      '--json',
    ],
    { encoding: 'utf8' },
  );
  assert.equal(run.status, 1, run.stderr);
  const report = JSON.parse(run.stdout);
  assert.ok(
    report.errors.some((entry) => entry.code === 'stub/code-fence'),
    JSON.stringify(report.errors.map((entry) => entry.code)),
  );
});

test('CLI passes a clean fixture with no diagnostics', (t) => {
  const { repoRoot } = makeRepo(t, {
    pages: [
      { id: 'a', source: `${DOCS}/a.md`, body: '[b](./b.md#present)' },
      { id: 'b', source: `${DOCS}/b.md`, body: '## Present' },
    ],
  });
  const run = spawnSync(process.execPath, [checkScript, '--repo', repoRoot, '--json'], { encoding: 'utf8' });
  assert.equal(run.status, 0, run.stderr);
  const report = JSON.parse(run.stdout);
  assert.equal(report.ok, true);
  assert.equal(report.summary.pages, 2);
});
