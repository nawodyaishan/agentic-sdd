import { existsSync, lstatSync, mkdirSync, readFileSync, realpathSync, renameSync, unlinkSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const pagePattern = /^(?:index|[a-z0-9]+(?:-[a-z0-9]+)*(?:\/[a-z0-9]+(?:-[a-z0-9]+)*)*)$/;

function pageTarget(repoRoot, pageId) {
  if (typeof pageId !== 'string' || !pagePattern.test(pageId)) throw new Error('Unsafe reference page ID');
  if (lstatSync(repoRoot).isSymbolicLink()) throw new Error('Unsafe symlink repository root');
  const root = realpathSync(repoRoot);
  const target = path.join(root, 'website', 'src', 'content', 'docs', `${pageId}.md`);
  let cursor = root;
  for (const segment of path.relative(root, target).split(path.sep)) {
    cursor = path.join(cursor, segment);
    if (!existsSync(cursor)) {
      // A dangling link still exists as a filesystem entry.
      try {
        if (lstatSync(cursor).isSymbolicLink()) throw new Error(`Unsafe reference symlink: ${cursor}`);
      } catch (error) {
        if (error.code !== 'ENOENT') throw error;
      }
      continue;
    }
    const stat = lstatSync(cursor);
    if (stat.isSymbolicLink()) throw new Error(`Unsafe reference symlink: ${cursor}`);
    if (cursor === target ? !stat.isFile() : !stat.isDirectory()) throw new Error(`Unsafe reference file type: ${cursor}`);
  }
  return { root, target };
}

export function renderGeneratedPage({ pageId, title, markdown, existing }) {
  if (typeof pageId !== 'string' || !pagePattern.test(pageId)) throw new Error('Unsafe reference page ID');
  if (typeof title !== 'string' || !title.trim()) throw new Error('Reference title is required');
  if (typeof markdown !== 'string' || !markdown.trim()) throw new Error('Reference facts must not be empty');
  if (markdown.includes('<!-- generated:')) throw new Error('Generated section payload contains a marker');
  const start = `<!-- generated:${pageId}:start -->`;
  const end = `<!-- generated:${pageId}:end -->`;
  const block = `${start}\n${markdown.replace(/\r\n/g, '\n').replace(/\n+$/, '')}\n${end}`;
  if (existing === undefined) {
    return `---\ntitle: ${JSON.stringify(title)}\ndescription: "Source-derived development reference facts; Spec 020 B1."\n---\n\nGenerated facts for the current source checkout. This is not runtime or release certification.\n\n${block}\n`;
  }
  if (typeof existing !== 'string') throw new Error('Existing reference content must be text');
  const markers = existing.match(/<!-- generated:[\s\S]*?-->/g) ?? [];
  if (!markers.length) return `${existing}${existing.endsWith('\n') ? '\n' : '\n\n'}${block}\n`;
  if (markers.length !== 2 || markers[0] !== start || markers[1] !== end) {
    throw new Error(`Malformed or duplicate generated section markers: ${pageId}`);
  }
  const begin = existing.indexOf(start);
  const finish = existing.indexOf(end, begin + start.length);
  return existing.slice(0, begin) + block + existing.slice(finish + end.length);
}

export function applySections({ repoRoot, sections, mode = 'check' }) {
  if (!['check', 'generate'].includes(mode)) throw new Error('Reference mode must be check or generate');
  if (!Array.isArray(sections) || sections.length === 0) throw new Error('Reference sections must not be empty');
  const seen = new Set();
  // Validate/render the entire set before changing any source file.
  const prepared = sections.map((section) => {
    if (seen.has(section.pageId)) throw new Error(`Duplicate reference section: ${section.pageId}`);
    seen.add(section.pageId);
    const { root, target } = pageTarget(repoRoot, section.pageId);
    const existing = existsSync(target) ? readFileSync(target, 'utf8') : undefined;
    const rendered = renderGeneratedPage({ ...section, existing });
    return { root, target, existing, rendered };
  });
  const changed = prepared.filter((item) => item.existing !== item.rendered);
  if (mode === 'generate') {
    for (const item of changed) {
      mkdirSync(path.dirname(item.target), { recursive: true });
      pageTarget(repoRoot, path.relative(path.join(item.root, 'website/src/content/docs'), item.target).replaceAll(path.sep, '/').slice(0, -3));
      const temporary = `${item.target}.tmp-${process.pid}-${prepared.indexOf(item)}`;
      try {
        writeFileSync(temporary, item.rendered, { flag: 'wx', mode: 0o644 });
        renameSync(temporary, item.target);
      } finally {
        if (existsSync(temporary)) unlinkSync(temporary);
      }
    }
  }
  return { changed: changed.map((item) => path.relative(item.root, item.target).replaceAll(path.sep, '/')) };
}

async function main(args) {
  let repoRoot = path.resolve(fileURLToPath(new URL('../..', import.meta.url)));
  let binaryPath;
  let mode = 'check';
  let modeExplicit = false;
  let json = false;
  for (let index = 0; index < args.length; index += 1) {
    const arg = args[index];
    if (arg === '--repo' || arg === '--binary') {
      const value = args[++index];
      if (!value || value.startsWith('--')) throw new Error(`Missing value for ${arg}`);
      if (arg === '--repo') repoRoot = path.resolve(value);
      else binaryPath = value;
    } else if (arg === '--check' || arg === '--generate') {
      if (modeExplicit) throw new Error('Choose exactly one reference mode');
      mode = arg.slice(2);
      modeExplicit = true;
    } else if (arg === '--json') json = true;
    else throw new Error(`Unknown reference option: ${arg}`);
  }
  if (!binaryPath || !path.isAbsolute(binaryPath)) throw new Error('--binary requires an explicit absolute executable path');
  const [{ collectCLI }, { collectCatalogs }, { collectConfiguration }] = await Promise.all([
    import('./reference/cli.mjs'),
    import('./reference/catalogs.mjs'),
    import('./reference/configuration.mjs'),
  ]);
  const outputs = await Promise.all([
    collectCLI({ repoRoot, binaryPath }),
    collectCatalogs({ repoRoot }),
    collectConfiguration({ repoRoot }),
  ]);
  const sections = outputs.flatMap((output) => output.sections);
  const expected = ['cli', 'skills', 'profiles', 'configuration', 'fleet', 'mcp', 'execution', 'files-and-backups'].map((id) => `reference/${id}`).sort();
  const actual = sections.map((section) => section.pageId).sort();
  if (JSON.stringify(actual) !== JSON.stringify(expected)) throw new Error('Reference collectors do not cover the exact B1 section set');
  const result = applySections({ repoRoot, sections, mode });
  if (json) console.log(JSON.stringify({ mode, ...result }, null, 2));
  else console.log(`${mode}: ${result.changed.length} reference page(s) ${mode === 'check' ? 'drifted' : 'updated'}`);
  if (mode === 'check' && result.changed.length) process.exitCode = 1;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main(process.argv.slice(2)).catch((error) => {
    console.error(`reference: ${error.message}`);
    process.exitCode = 1;
  });
}
