// SPEC020 B4-W1-E — focused tests for the public allowlisted export.
//
// Covered contract:
// - output is exactly llms.txt, index.json and one Markdown twin per L/C page;
// - A/F pages, specs/, docs/, handovers and evidence never reach the output;
// - a page registered outside the public content root, an unregistered content
//   file, a symlink, a special file and an .mdx source all fail closed;
// - two runs are byte-identical and carry no timestamps or absolute paths;
// - project base and docsVersion/applicability parity are asserted;
// - `--check` writes nothing and reports the first difference;
// - a stale page source digest fails closed;
// - twin links become absolute public URLs or pinned source links.

import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import {
  existsSync,
  linkSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { computeReviewDigest } from '../scripts/lib/manifest.mjs';
import { buildExport, checkExport, writeExport } from '../scripts/export.mjs';

const CONTENT = 'website/src/content/docs';
const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const SCRIPT = fileURLToPath(new URL('../scripts/export.mjs', import.meta.url));
const RELEASE_EVIDENCE = {
  tag: 'v0.11.0',
  commit: '2fc31e9f9c4d019e2b9f3b4f004978028808c74b',
  url: 'https://github.com/example/project/releases/tag/v0.11.0',
  observedAt: '2026-11-14',
  verification: 'public metadata only',
};

function sha256(value) {
  return createHash('sha256').update(value).digest('hex');
}

function makeTempDir(t, prefix) {
  mkdirSync('/tmp/opencode', { recursive: true });
  const dir = mkdtempSync(`/tmp/opencode/${prefix}`);
  t.after(() => rmSync(dir, { recursive: true, force: true }));
  return dir;
}

function writeText(root, relative, content) {
  const absolute = path.join(root, relative);
  mkdirSync(path.dirname(absolute), { recursive: true });
  writeFileSync(absolute, content, 'utf8');
  return absolute;
}

function pageBody(title, text = 'Public page body.') {
  return `---\ntitle: ${JSON.stringify(title)}\ndescription: ${JSON.stringify(`${title} description.`)}\n---\n\n${text}\n`;
}

function pageRecord(spec) {
  const source = spec.source ?? (spec.id === 'index' ? `${CONTENT}/index.md` : `${CONTENT}/${spec.id}.md`);
  const sources = [{ path: source, sha256: sha256(spec.body) }];
  for (const evidence of spec.evidence ?? []) {
    sources.push({ path: evidence.path, sha256: sha256(evidence.content) });
  }
  return {
    id: spec.id,
    source,
    type: spec.type ?? 'reference',
    audience: spec.audience ?? ['maintainer'],
    class: spec.class ?? 'L',
    sources,
    reviewDigest: computeReviewDigest(sources),
    appliesTo: spec.appliesTo ?? 'unreleased',
    reviewStatus: 'unreviewed',
  };
}

function makeRepo(t, options = {}) {
  const {
    pages = [
      { id: 'index', class: 'L', body: pageBody('Home', 'Start here.') },
      { id: 'reference/cli', class: 'C', body: pageBody('CLI', 'Command facts.') },
    ],
    docsVersion = 'dev',
    base = '/project',
    site = 'https://example.github.io',
    repository = 'https://github.com/example/project',
    includeReleaseEvidence = true,
    extraFiles = [],
  } = options;
  const root = makeTempDir(t, 'export-repo-');
  for (const spec of pages) {
    const source = spec.source ?? (spec.id === 'index' ? `${CONTENT}/index.md` : `${CONTENT}/${spec.id}.md`);
    writeText(root, source, spec.body);
    for (const evidence of spec.evidence ?? []) writeText(root, evidence.path, evidence.content);
  }
  for (const extra of extraFiles) writeText(root, extra.path, extra.content);
  const manifest = {
    schemaVersion: 1,
    site,
    base,
    repository,
    docsVersion,
    ...(includeReleaseEvidence ? { releaseEvidence: [RELEASE_EVIDENCE] } : {}),
    pages: pages.map(pageRecord),
  };
  writeText(root, 'website/docs-manifest.json', `${JSON.stringify(manifest, null, 2)}\n`);
  return root;
}

function expectCode(fn, code) {
  assert.throws(fn, (error) => {
    assert.equal(error.code, code, `expected ${code}, got ${error.code}: ${error.message}`);
    return true;
  });
}

function snapshot(root) {
  const files = new Map();
  const visit = (dir, prefix) => {
    for (const entry of readdirSync(dir, { withFileTypes: true }).sort((l, r) => (l.name < r.name ? -1 : 1))) {
      const relative = prefix ? `${prefix}/${entry.name}` : entry.name;
      const absolute = path.join(dir, entry.name);
      if (entry.isDirectory()) visit(absolute, relative);
      else files.set(relative, readFileSync(absolute, 'utf8'));
    }
  };
  if (existsSync(root)) visit(root, '');
  return files;
}

function runCli(args) {
  return spawnSync(process.execPath, [SCRIPT, ...args], {
    encoding: 'utf8',
    env: { PATH: process.env.PATH, HOME: '/tmp/opencode' },
  });
}

test('exports exactly the canonical L/C twins, index.json and llms.txt', (t) => {
  const root = makeRepo(t, {
    pages: [
      {
        id: 'index',
        class: 'L',
        body: pageBody('Home', 'Start here.'),
        evidence: [
          { path: 'specs/020-documentation-ecosystem/spec.md', content: 'PRIVATE-EVIDENCE-MARKER spec body\n' },
        ],
      },
      { id: 'reference/cli', class: 'C', body: pageBody('CLI', 'Command facts.') },
      { id: 'architecture/overview', class: 'A', body: pageBody('Architecture', 'Deferred page.') },
      { id: 'releases/migration', class: 'F', body: pageBody('Migration', 'Future page.') },
    ],
  });

  assert.equal(existsSync(path.join(root, 'specs/020-documentation-ecosystem/spec.md')), true);
  const plan = buildExport({ repoRoot: root });
  assert.deepEqual(
    plan.files.map((file) => file.path),
    ['index.json', 'index.md', 'llms.txt', 'reference/cli/index.md'],
  );
  assert.deepEqual(
    plan.excluded.map((entry) => `${entry.id}:${entry.class}`),
    ['architecture/overview:A', 'releases/migration:F'],
  );

  const index = JSON.parse(plan.files.find((file) => file.path === 'index.json').content);
  assert.deepEqual(
    index.pages.map((entry) => entry.id),
    ['index', 'reference/cli'],
  );
  const llms = plan.files.find((file) => file.path === 'llms.txt').content;
  assert.equal(llms.match(/^- \[/gm).length, 2);

  const combined = plan.files.map((file) => file.content).join('\n');
  for (const forbidden of [
    'PRIVATE-EVIDENCE-MARKER',
    'specs/',
    'docs/handovers',
    'architecture/overview',
    'releases/migration',
    'llms-full.txt',
  ]) {
    assert.equal(combined.includes(forbidden), false, `export leaked ${forbidden}`);
  }
});

test('refuses pages outside the public allowlist, unregistered files and unsafe entry types', (t) => {
  const escapedSource = makeRepo(t, {
    pages: [{ id: 'reference/cli', class: 'L', source: 'specs/020-documentation-ecosystem/spec.md', body: pageBody('CLI') }],
  });
  expectCode(() => buildExport({ repoRoot: escapedSource }), 'invalid-source-path');

  const deniedRoute = makeRepo(t, {
    pages: [
      { id: 'index', class: 'L', body: pageBody('Home') },
      { id: 'specs/leak', class: 'L', body: pageBody('Leak') },
    ],
  });
  expectCode(() => buildExport({ repoRoot: deniedRoute }), 'denied-output-path');

  const unregistered = makeRepo(t, {
    extraFiles: [{ path: `${CONTENT}/notes/extra.md`, content: pageBody('Extra', 'Unregistered page.') }],
  });
  expectCode(() => buildExport({ repoRoot: unregistered }), 'unregistered-page');

  const symlinked = makeRepo(t);
  writeText(symlinked, `${CONTENT}/target.md`, pageBody('Target'));
  rmSync(path.join(symlinked, CONTENT, 'reference/cli.md'));
  symlinkSync('target.md', path.join(symlinked, CONTENT, 'reference/cli.md'));
  expectCode(() => buildExport({ repoRoot: symlinked }), 'symlink-path');

  const special = makeRepo(t);
  rmSync(path.join(special, CONTENT, 'reference/cli.md'));
  mkdirSync(path.join(special, CONTENT, 'reference/cli.md'));
  expectCode(() => buildExport({ repoRoot: special }), 'not-a-regular-file');

  const mdx = makeRepo(t, {
    extraFiles: [{ path: `${CONTENT}/page.mdx`, content: pageBody('MDX') }],
  });
  expectCode(() => buildExport({ repoRoot: mdx }), 'unsupported-source');
});

test('is deterministic, writes byte-identical output and carries no host paths', (t) => {
  const root = makeRepo(t);
  const first = buildExport({ repoRoot: root });
  const second = buildExport({ repoRoot: root });
  assert.deepEqual(first.files, second.files);

  const outA = path.join(makeTempDir(t, 'export-out-a-'));
  const outB = path.join(makeTempDir(t, 'export-out-b-'));
  const writeA = writeExport({ repoRoot: root, outDir: outA });
  const writeB = writeExport({ repoRoot: root, outDir: outB });
  assert.deepEqual(writeA.written, ['index.json', 'index.md', 'llms.txt', 'reference/cli/index.md']);
  assert.deepEqual(writeA.unchanged, []);
  assert.deepEqual(writeB, writeA);
  assert.deepEqual(snapshot(outA), snapshot(outB));

  const repeat = writeExport({ repoRoot: root, outDir: outA });
  assert.deepEqual(repeat.written, []);
  assert.equal(repeat.unchanged.length, 4);

  for (const file of first.files) {
    assert.equal(file.content.includes(root), false, `${file.path} contains the absolute repository path`);
    assert.equal(/\/tmp\/opencode\//.test(file.content), false, `${file.path} contains a temporary path`);
    assert.equal(/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/.test(file.content), false, `${file.path} contains a timestamp`);
    assert.equal(file.content.includes('\r'), false, `${file.path} contains CRLF`);
  }
});

test('asserts the project base and docsVersion/applicability parity', (t) => {
  const project = makeRepo(t, {
    pages: [
      { id: 'index', class: 'L', body: pageBody('Home') },
      { id: 'reference/cli', class: 'C', body: pageBody('CLI') },
    ],
  });
  const plan = buildExport({ repoRoot: project });
  const index = JSON.parse(plan.files.find((file) => file.path === 'index.json').content);
  assert.equal(index.base, '/project');
  assert.equal(index.site, 'https://example.github.io');
  for (const page of index.pages) {
    assert.match(page.url, /^https:\/\/example\.github\.io\/project\//);
    assert.match(page.markdownUrl, /^https:\/\/example\.github\.io\/project\//);
  }
  assert.equal(index.pages.find((page) => page.id === 'index').url, 'https://example.github.io/project/');
  assert.equal(
    index.pages.find((page) => page.id === 'reference/cli').markdownUrl,
    'https://example.github.io/project/reference/cli/index.md',
  );
  const llms = plan.files.find((file) => file.path === 'llms.txt').content;
  assert.match(llms, /https:\/\/example\.github\.io\/project\/reference\/cli\//);

  const rootBase = makeRepo(t, { base: '/' });
  const rootPlan = buildExport({ repoRoot: rootBase });
  assert.equal(rootPlan.pages.find((page) => page.id === 'reference/cli').url, 'https://example.github.io/reference/cli/');
  assert.equal(rootPlan.files.some((file) => file.path === 'reference/cli/index.md'), true);

  const badBase = makeRepo(t, { base: 'project' });
  expectCode(() => buildExport({ repoRoot: badBase }), 'invalid-base');

  const trailingBase = makeRepo(t, { base: '/project/' });
  expectCode(() => buildExport({ repoRoot: trailingBase }), 'invalid-base');

  const unreleasedOnRelease = makeRepo(t, {
    docsVersion: 'v0.11.0',
    pages: [{ id: 'index', class: 'L', body: pageBody('Home'), appliesTo: 'unreleased' }],
  });
  expectCode(() => buildExport({ repoRoot: unreleasedOnRelease }), 'version-parity');

  const releaseOnDev = makeRepo(t, {
    docsVersion: 'dev',
    pages: [{ id: 'index', class: 'L', body: pageBody('Home'), appliesTo: 'v0.11.0' }],
  });
  expectCode(() => buildExport({ repoRoot: releaseOnDev }), 'version-parity');

  const matched = makeRepo(t, {
    docsVersion: 'v0.11.0',
    pages: [{ id: 'index', class: 'L', body: pageBody('Home'), appliesTo: 'v0.11.0' }],
  });
  const matchedPlan = buildExport({ repoRoot: matched });
  assert.equal(matchedPlan.ref, 'v0.11.0');
  assert.equal(matchedPlan.pages[0].appliesTo, 'v0.11.0');
  assert.match(
    matchedPlan.files.find((file) => file.path === 'llms.txt').content,
    /Documentation version: v0\.11\.0 · Page applicability: v0\.11\.0/,
  );
});

test('--check writes nothing and reports the first difference', (t) => {
  const root = makeRepo(t);
  const out = path.join(makeTempDir(t, 'export-check-'));
  const first = writeExport({ repoRoot: root, outDir: out });
  assert.equal(first.written.length, 4);

  const clean = runCli(['--repo', root, '--out', out, '--check']);
  assert.equal(clean.status, 0);
  assert.match(clean.stdout, /4 file\(s\) match/);

  const tampered = path.join(out, 'llms.txt');
  writeFileSync(tampered, 'hand-edited parallel prose\n', 'utf8');
  const beforeCheck = snapshot(out);
  const drift = runCli(['--repo', root, '--out', out, '--check']);
  assert.equal(drift.status, 1);
  assert.match(drift.stderr, /llms\.txt/);
  assert.match(drift.stderr, /differs/);
  assert.deepEqual(snapshot(out), beforeCheck, '--check must not rewrite output');

  const missing = path.join(makeTempDir(t, 'export-missing-'), 'absent');
  const absent = runCli(['--repo', root, '--out', missing, '--check']);
  assert.equal(absent.status, 1);
  assert.match(absent.stderr, /missing-output-directory/);
  assert.equal(existsSync(missing), false, '--check must not create the output directory');

  const noOut = runCli(['--repo', root]);
  assert.equal(noOut.status, 1);
  assert.match(noOut.stderr, /--out <dir> is required/);

  const unknown = runCli(['--repo', root, '--out', out, '--publish']);
  assert.equal(unknown.status, 1);
  assert.match(unknown.stderr, /Unknown export option/);

  const api = checkExport({ repoRoot: root, outDir: out });
  assert.equal(api.ok, false);
  assert.equal(api.firstDifference.path, 'llms.txt');
  assert.equal(api.firstDifference.reason, 'differs');

  const staleOut = path.join(makeTempDir(t, 'export-stale-'));
  writeExport({ repoRoot: root, outDir: staleOut });
  writeText(staleOut, 'reference/old/index.md', '# stale twin\n');
  const stale = runCli(['--repo', root, '--out', staleOut, '--check']);
  assert.equal(stale.status, 1);
  assert.match(stale.stderr, /unexpected-output/);
  assert.equal(readFileSync(path.join(staleOut, 'reference/old/index.md'), 'utf8'), '# stale twin\n');

  const deferredOut = path.join(makeTempDir(t, 'export-deferred-'));
  writeExport({ repoRoot: root, outDir: deferredOut });
  writeText(deferredOut, 'llms-full.txt', 'deferred corpus\n');
  const deferred = runCli(['--repo', root, '--out', deferredOut, '--check']);
  assert.equal(deferred.status, 1);
  assert.match(deferred.stderr, /deferred-output/);
  assert.equal(existsSync(path.join(deferredOut, 'llms-full.txt')), true);
});

test('--ref reaches the export builder instead of being ignored', (t) => {
  const root = makeRepo(t, {
    pages: [
      {
        id: 'reference/cli',
        class: 'C',
        body: pageBody('CLI', '[Source](../../../../../cmd/agentic-sdd/main.go)'),
      },
    ],
  });
  const defaultPlan = buildExport({ repoRoot: root });
  const pinnedPlan = buildExport({ repoRoot: root, ref: 'v9.9.9' });
  const defaultIndex = JSON.parse(defaultPlan.files.find((file) => file.path === 'index.json').content);
  const pinnedIndex = JSON.parse(pinnedPlan.files.find((file) => file.path === 'index.json').content);
  assert.equal(defaultIndex.sourceRef, 'main');
  assert.equal(pinnedIndex.sourceRef, 'v9.9.9');

  const defaultTwin = defaultPlan.files.find((file) => file.path === 'reference/cli/index.md').content;
  const pinnedTwin = pinnedPlan.files.find((file) => file.path === 'reference/cli/index.md').content;
  assert.match(defaultTwin, /blob\/main\/cmd\/agentic-sdd\/main\.go/);
  assert.match(pinnedTwin, /blob\/v9\.9\.9\/cmd\/agentic-sdd\/main\.go/);
  assert.notEqual(defaultTwin, pinnedTwin, 'the pinned ref must change the rendered twin');

  const outDefault = path.join(makeTempDir(t, 'export-ref-default-'));
  const outPinned = path.join(makeTempDir(t, 'export-ref-pinned-'));
  assert.equal(runCli(['--repo', root, '--out', outDefault]).status, 0);
  assert.equal(runCli(['--repo', root, '--out', outPinned, '--ref', 'v9.9.9']).status, 0);
  const defaultBytes = readFileSync(path.join(outDefault, 'index.json'), 'utf8');
  const pinnedBytes = readFileSync(path.join(outPinned, 'index.json'), 'utf8');
  assert.notEqual(defaultBytes, pinnedBytes, '--ref must change the written export');
  assert.equal(JSON.parse(pinnedBytes).sourceRef, 'v9.9.9');
  assert.match(
    readFileSync(path.join(outPinned, 'reference/cli/index.md'), 'utf8'),
    /blob\/v9\.9\.9\//,
  );
});

test('--check reports extra files and symlinks outside the exact output set', (t) => {
  const root = makeRepo(t);
  const out = path.join(makeTempDir(t, 'export-extra-'));
  writeExport({ repoRoot: root, outDir: out });

  writeText(out, 'secret.txt', 'not export output\n');
  writeText(out, 'notes/extra.md', '# extra\n');
  const extra = checkExport({ repoRoot: root, outDir: out });
  assert.equal(extra.ok, false);
  assert.deepEqual(
    extra.differences.map((entry) => `${entry.path}:${entry.reason}`),
    ['notes/extra.md:unexpected-output', 'secret.txt:unexpected-output'],
  );
  const cliExtra = runCli(['--repo', root, '--out', out, '--check']);
  assert.equal(cliExtra.status, 1);
  assert.match(cliExtra.stderr, /notes\/extra\.md: unexpected-output/);

  symlinkSync('llms.txt', path.join(out, 'llms-alias.txt'));
  const linked = checkExport({ repoRoot: root, outDir: out });
  assert.equal(linked.ok, false);
  assert.deepEqual(
    linked.differences.map((entry) => `${entry.path}:${entry.reason}`),
    [
      'llms-alias.txt:symlink-output',
      'notes/extra.md:unexpected-output',
      'secret.txt:unexpected-output',
    ],
  );
  const cliLinked = runCli(['--repo', root, '--out', out, '--check']);
  assert.equal(cliLinked.status, 1);
  assert.match(cliLinked.stderr, /llms-alias\.txt: symlink-output/);
});

test('refuses hard-linked sources and output files', (t) => {
  const root = makeRepo(t);
  linkSync(path.join(root, CONTENT, 'reference/cli.md'), path.join(root, 'mirror.md'));
  expectCode(() => buildExport({ repoRoot: root }), 'hardlink-path');

  const cleanRoot = makeRepo(t);
  const out = path.join(makeTempDir(t, 'export-hardlink-'));
  writeExport({ repoRoot: cleanRoot, outDir: out });
  linkSync(path.join(out, 'llms.txt'), path.join(out, 'llms-copy.txt'));

  const checked = checkExport({ repoRoot: cleanRoot, outDir: out });
  assert.equal(checked.ok, false);
  assert.deepEqual(
    checked.differences.map((entry) => `${entry.path}:${entry.reason}`),
    ['llms.txt:hardlink', 'llms-copy.txt:unexpected-output'],
  );
  expectCode(() => writeExport({ repoRoot: cleanRoot, outDir: out }), 'hardlink-output');
});

test('fails closed on a stale or unregistered reviewed source', (t) => {
  const root = makeRepo(t);
  buildExport({ repoRoot: root });
  writeFileSync(
    path.join(root, CONTENT, 'reference/cli.md'),
    `${pageBody('CLI', 'Silently changed after review.')}`,
    'utf8',
  );
  expectCode(() => buildExport({ repoRoot: root }), 'stale-source');

  const missingSelf = makeRepo(t, {
    pages: [
      {
        id: 'index',
        class: 'L',
        body: pageBody('Home'),
        evidence: [{ path: 'specs/020-documentation-ecosystem/spec.md', content: 'evidence\n' }],
      },
    ],
  });
  const manifestPath = path.join(missingSelf, 'website/docs-manifest.json');
  const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
  manifest.pages[0].sources = manifest.pages[0].sources.filter((entry) => !entry.path.endsWith('index.md'));
  manifest.pages[0].reviewDigest = computeReviewDigest(manifest.pages[0].sources);
  writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`, 'utf8');
  expectCode(() => buildExport({ repoRoot: missingSelf }), 'stale-source');
});

test('rewrites twin links to absolute public URLs or pinned source links', (t) => {
  const root = makeRepo(t, {
    pages: [
      {
        id: 'reference/cli',
        class: 'L',
        body: pageBody(
          'CLI',
          [
            '[Peer page](./peer.md#details)',
            '[Source file](../../../../../cmd/agentic-sdd/main.go)',
            '[External](https://example.com/guide)',
            '![Diagram](../../assets/diagram.svg)',
            '`[inline code](./peer.md)`',
            '',
            '[ref]: ./peer.md',
            '',
            '```sh',
            '[fenced](./peer.md)',
            '```',
          ].join('\n'),
        ),
      },
      { id: 'reference/peer', class: 'C', body: pageBody('Peer', 'Peer body.') },
    ],
  });
  const plan = buildExport({ repoRoot: root });
  const twin = plan.files.find((file) => file.path === 'reference/cli/index.md').content;
  assert.match(twin, /\[Peer page\]\(https:\/\/example\.github\.io\/project\/reference\/peer\/#details\)/);
  assert.match(
    twin,
    /\[Source file\]\(https:\/\/github\.com\/example\/project\/blob\/main\/cmd\/agentic-sdd\/main\.go\)/,
  );
  assert.match(twin, /\[External\]\(https:\/\/example\.com\/guide\)/);
  assert.match(twin, /!\[Diagram\]\(\.\.\/\.\.\/assets\/diagram\.svg\)/);
  assert.match(twin, /`\[inline code\]\(\.\/peer\.md\)`/);
  assert.match(twin, /\[ref\]: https:\/\/example\.github\.io\/project\/reference\/peer\//);
  assert.match(twin, /\[fenced\]\(\.\/peer\.md\)/);

  const unknown = makeRepo(t, {
    pages: [
      {
        id: 'reference/cli',
        class: 'L',
        body: pageBody('CLI', '[Ghost](./ghost.md)'),
      },
    ],
  });
  expectCode(() => buildExport({ repoRoot: unknown }), 'unknown-target');
});

test('refuses twin links into private or history trees and keeps public engineering sources', (t) => {
  const cases = [
    ['handover', '../../../../../docs/handovers/2026-11-14_020.md', 'docs/handovers/2026-11-14_020.md'],
    ['feature spec', '../../../../../specs/020-documentation-ecosystem/spec.md', 'specs/020-documentation-ecosystem/spec.md'],
    ['run scratch', '../../../../../.agentic-sdd/runs/020/checkpoint.json', '.agentic-sdd/runs/020/checkpoint.json'],
  ];
  for (const [label, href, target] of cases) {
    const root = makeRepo(t, {
      pages: [
        { id: 'reference/cli', class: 'L', body: pageBody('CLI', `[${label}](${href})`) },
      ],
    });
    assert.throws(
      () => buildExport({ repoRoot: root }),
      (error) => {
        assert.equal(error.code, 'private-link-target', `${label}: expected private-link-target, got ${error.code}`);
        assert.equal(error.path, 'reference/cli');
        assert.ok(error.message.includes(target), `${label}: the refusal must name ${target}`);
        return true;
      },
      `${label} must not become a public export URL`,
    );
  }

  const publicSource = makeRepo(t, {
    pages: [
      {
        id: 'reference/cli',
        class: 'L',
        body: pageBody('CLI', '[Skill](../../../../../skills-files/agentic-sdd-router/SKILL.md)'),
      },
    ],
  });
  const twin = buildExport({ repoRoot: publicSource }).files.find(
    (file) => file.path === 'reference/cli/index.md',
  ).content;
  assert.match(
    twin,
    /blob\/main\/skills-files\/agentic-sdd-router\/SKILL\.md/,
    'a public engineering source keeps its repository source URL',
  );
});

test('the deployable dist carries llms.txt, index.json and one twin per registered page', (t) => {
  // `build:site` is the deployable artifact command: the Astro build followed
  // by the export into the same output directory. `build` stays the pure site
  // build used by existing tests.
  const pkg = JSON.parse(readFileSync(path.join(REPO_ROOT, 'website/package.json'), 'utf8'));
  assert.equal(pkg.scripts.build, 'astro build');
  assert.equal(pkg.scripts['build:site'], 'astro build && node scripts/export.mjs --out dist');

  const manifest = JSON.parse(readFileSync(path.join(REPO_ROOT, 'website/docs-manifest.json'), 'utf8'));
  const plan = buildExport({ repoRoot: REPO_ROOT });
  assert.equal(
    plan.pages.length + plan.excluded.length,
    manifest.pages.length,
    'every registered page is either public (twin emitted) or explicitly deferred',
  );
  const deployDist = path.join(makeTempDir(t, 'export-deploy-dist-'));
  const first = writeExport({ repoRoot: REPO_ROOT, outDir: deployDist });
  const second = writeExport({ repoRoot: REPO_ROOT, outDir: deployDist });
  assert.deepEqual(second.written, [], 'a second export into the same dist must write nothing');
  assert.equal(second.unchanged.length, first.written.length);

  const files = snapshot(deployDist);
  assert.equal(files.has('llms.txt'), true, 'dist/llms.txt is missing');
  assert.equal(files.has('index.json'), true, 'dist/index.json is missing');
  const twins = [...files.keys()].filter((name) => name === 'index.md' || name.endsWith('/index.md'));
  assert.equal(twins.length, plan.pages.length, 'one Markdown twin per public registered page');
  assert.equal(files.size, plan.pages.length + 2, 'the dist export is llms.txt, index.json and twins only');
  for (const page of plan.pages) {
    const twin = page.id === 'index' ? 'index.md' : `${page.id}/index.md`;
    assert.equal(files.has(twin), true, `dist/${twin} is missing`);
  }
  for (const page of plan.excluded) {
    const twin = page.id === 'index' ? 'index.md' : `${page.id}/index.md`;
    assert.equal(files.has(twin), false, `deferred page ${page.id} must not emit a twin`);
  }
  assert.equal(files.has('llms-full.txt'), false);

  // A clean second destination must reproduce the same bytes exactly.
  const cleanDist = path.join(makeTempDir(t, 'export-deploy-clean-'));
  assert.deepEqual(writeExport({ repoRoot: REPO_ROOT, outDir: cleanDist }), first);
  assert.deepEqual(snapshot(cleanDist), files, 'two runs must be byte-identical');
});
