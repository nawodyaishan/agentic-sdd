// Fixture project for the C3 example runner tests.
//
// The runner must be exercised without the real candidate binary, so each test
// generates a throwaway Node stub in an owned temp directory. The stub is
// executable (`0o755`) with an absolute `#!<node>` shebang, so it works even
// though the runner sanitizes PATH. It records its argv/environment into a log
// path baked into the script (not passed through env) so the record survives
// the runner's cleanup of its own temp root.
import { chmodSync, writeFileSync } from 'node:fs';
import path from 'node:path';

export const STUB_VERSION_LINE = 'asdx-core 9.9.9-fixture';
export const STUB_COMMIT = 'fixture-commit-0000000';

/**
 * Build the stub source. `logPath` is an absolute test-owned file outside the
 * runner's temp root. `mutateHome` simulates a preview that wrongly writes.
 */
export function stubSource({
  logPath,
  mutateHome = false,
  versionExit = 0,
  previewExit = 0,
  silentPreview = false,
} = {}) {
  if (typeof logPath !== 'string' || !path.isAbsolute(logPath)) {
    throw new Error('stubSource requires an absolute logPath');
  }
  return `#!${process.execPath}
'use strict';
const fs = require('node:fs');
const path = require('node:path');
const args = process.argv.slice(2);
const record = {
  argv: args,
  cwd: process.cwd(),
  home: process.env.HOME === undefined ? null : process.env.HOME,
  path: process.env.PATH === undefined ? null : process.env.PATH,
  envKeys: Object.keys(process.env).sort(),
};
fs.appendFileSync(${JSON.stringify(logPath)}, JSON.stringify(record) + '\\n');
const command = args[0];
if (command === 'version') {
  if (${versionExit} !== 0) {
    process.stderr.write('fixture: version failed\\n');
    process.exit(${versionExit});
  }
  process.stdout.write(${JSON.stringify(`${STUB_VERSION_LINE}\ncommit ${STUB_COMMIT}\nbuilt 2026-01-01T00:00:00Z\n`)});
  process.exit(0);
}
if (command === 'preview') {
  const homeIndex = args.indexOf('--home');
  const home = homeIndex === -1 ? null : args[homeIndex + 1];
  ${silentPreview ? '' : "process.stdout.write('preview: no changes\\nskills target: ' + String(home) + '\\n');"}
  if (home) {
    ${
      mutateHome
        ? "fs.mkdirSync(path.join(home, '.agents', 'skills'), { recursive: true });\n    fs.writeFileSync(path.join(home, '.agents', 'skills', 'leak.txt'), 'preview must not write\\n');"
        : 'fs.accessSync(home);'
    }
  }
  if (${previewExit} !== 0) {
    process.stderr.write('fixture: preview failed\\n');
    process.exit(${previewExit});
  }
  process.exit(0);
}
process.stderr.write('fixture: unexpected argv: ' + args.join(' ') + '\\n');
process.exit(2);
`;
}

/** Write an executable stub into `directory` and return its absolute path. */
export function writeStubCli(directory, options = {}) {
  const filePath = path.join(directory, options.name ?? 'asdx-core-fixture');
  writeFileSync(filePath, stubSource(options), { mode: 0o755 });
  chmodSync(filePath, 0o755);
  return filePath;
}
