// Spec 020 B1-W1-D — offline lifecycle/native-hook and read-only-dependency
// counterexamples for the pinned website workspace (plan C4 dependency
// provisioning, T01 work item "dependency provisioning cannot run candidate
// scripts").
//
// These checks are static, data-only reads of the workspace metadata. Actual
// provisioning, `npm ci --ignore-scripts`, egress-denied builds and native-hook
// audits require the disposable container and are reported as not_run here.

import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

import { readJson, repoRoot } from './foundation-negative-support/support.mjs';

const websiteDir = path.join(repoRoot, 'website');
const packagePath = path.join(websiteDir, 'package.json');
const lockPath = path.join(websiteDir, 'package-lock.json');
const nodeVersionPath = path.join(websiteDir, '.node-version');

const pkgExists = fs.existsSync(packagePath);
const pending = 'integration-pending: website/package.json not present (B1-W1-A workspace)';
if (!pkgExists) console.error(`[foundation-negative] foundation-negative-dependencies: ${pending}`);

const skip = pkgExists ? false : pending;
const pkg = pkgExists ? readJson(packagePath) : null;
const lockExists = fs.existsSync(lockPath);
const lockSkip = lockExists ? skip : `${pending}; website/package-lock.json not present`;

const LIFECYCLE = ['preinstall', 'install', 'postinstall', 'prepare', 'prepublish', 'prepublishOnly'];
const DEP_FIELDS = ['dependencies', 'devDependencies', 'optionalDependencies', 'peerDependencies'];

const allScripts = (manifest) => Object.entries(manifest.scripts ?? {});
const depEntries = (manifest, fields) =>
  fields.flatMap((field) => Object.entries(manifest[field] ?? {}).map(([name, spec]) => ({ field, name, spec })));

test('workspace declares no install lifecycle scripts', { skip }, () => {
  for (const [name, command] of allScripts(pkg)) {
    assert.ok(!LIFECYCLE.includes(name), `lifecycle script "${name}" would run during provisioning: ${command}`);
  }
});

test('build and test scripts never install dependencies', { skip }, () => {
  for (const [name, command] of allScripts(pkg)) {
    assert.doesNotMatch(
      command,
      /(^|[\s;&|()])(npm|pnpm|yarn)\s+(ci|install|add|update)\b/,
      `script "${name}" provisions at build time: ${command}`,
    );
    assert.doesNotMatch(command, /--ignore-scripts=false|--no-ignore-scripts/, `script "${name}" enables lifecycle scripts: ${command}`);
    assert.doesNotMatch(command, /\b(curl|wget|git\s+clone|ssh)\b/, `script "${name}" reaches the network: ${command}`);
  }
});

test('dependencies are registry-only, not file, link or Git sources', { skip }, () => {
  for (const { field, name, spec } of depEntries(pkg, DEP_FIELDS)) {
    assert.doesNotMatch(
      String(spec),
      /^(file:|link:|portal:|workspace:|git\+|git:|github:|https?:|\.{0,2}\/)/,
      `${field}.${name} is not a registry dependency: ${spec}`,
    );
    assert.notEqual(String(spec), '*', `${field}.${name} must be pinned, not floating`);
    assert.notEqual(String(spec), 'latest', `${field}.${name} must not track latest`);
  }
});

test('scripts do not write into a shared node_modules tree', { skip }, () => {
  for (const [name, command] of allScripts(pkg)) {
    assert.doesNotMatch(command, /node_modules/, `script "${name}" writes a shared dependency tree: ${command}`);
  }
});

test('pins a Node runtime in .node-version', { skip }, () => {
  assert.ok(fs.existsSync(nodeVersionPath), 'website/.node-version is required for a pinned runtime');
  const pinned = fs.readFileSync(nodeVersionPath, 'utf8').trim();
  assert.match(pinned, /^v?\d+(\.\d+){0,2}$/, `.node-version must be a plain version: "${pinned}"`);
});

test('engines.node agrees with the pinned runtime major', { skip }, () => {
  const engines = pkg.engines?.node;
  if (!engines) return; // engines is not mandatory in the frozen contract
  const pinned = fs.readFileSync(nodeVersionPath, 'utf8').trim().replace(/^v/, '').split('.')[0];
  assert.match(String(engines), new RegExp(`(^|[^0-9])${pinned}([^0-9]|$)`), `engines.node "${engines}" disagrees with .node-version ${pinned}.x`);
});

test('package-lock is a registry lock without link entries', { skip: lockSkip }, () => {
  const lock = readJson(lockPath);
  assert.ok(Number(lock.lockfileVersion) >= 2, 'a modern lockfile is required for pinned provisioning');
  const packages = lock.packages ?? {};
  for (const [entryPath, entry] of Object.entries(packages)) {
    if (!entryPath) continue;
    assert.notEqual(entry.link, true, `lock entry ${entryPath} is a filesystem link`);
    if (entry.resolved !== undefined) {
      assert.match(String(entry.resolved), /^https:\/\//, `lock entry ${entryPath} resolves outside HTTPS: ${entry.resolved}`);
    }
    if (entryPath.startsWith('node_modules/')) {
      assert.ok(entry.integrity, `lock entry ${entryPath} lacks an integrity hash`);
    }
  }
});

test('lock and package.json dependency specifiers agree', { skip: lockSkip }, () => {
  const lock = readJson(lockPath);
  const rootEntry = lock.packages?.[''] ?? {};
  for (const field of DEP_FIELDS) {
    const declared = pkg[field] ?? {};
    const locked = rootEntry[field] ?? {};
    assert.deepEqual(
      Object.keys(locked).sort(),
      Object.keys(declared).sort(),
      `${field} keys diverge between package.json and the lock root`,
    );
    for (const [name, spec] of Object.entries(declared)) {
      assert.equal(locked[name], spec, `${field}.${name} specifier differs from the lock (${locked[name]} vs ${spec})`);
    }
  }
});

test('website source tree carries no symlinks outside installed dependencies', { skip }, () => {
  const ignored = new Set(['node_modules', 'dist', '.astro', '.cache']);
  const offenders = [];
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      if (ignored.has(entry.name)) continue;
      const absolute = path.join(dir, entry.name);
      if (entry.isSymbolicLink()) {
        offenders.push(path.relative(repoRoot, absolute));
        continue;
      }
      if (entry.isDirectory()) walk(absolute);
    }
  };
  walk(websiteDir);
  assert.deepEqual(offenders, [], 'tracked website sources must not be symlinks (Pages artifact rule)');
});
