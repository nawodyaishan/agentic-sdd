// Local object fixtures for the C1 manifest tests.
//
// Fixtures are built by hand (not by the module under test) so a digest or
// normalisation bug cannot validate itself. `ONE_SOURCE_DIGEST` is the
// SHA-256 of the compact JSON for `[{path:"README.md",sha256:"a"*64}]`.

export const SHA_A = 'a'.repeat(64);
export const SHA_B = 'b'.repeat(64);
export const ONE_SOURCE_DIGEST = '2ad05d5ae5ecfceb8080abe738cfcb53d8097f9696a4ff92dede2e96ad96cdba';
export const TWO_SOURCE_DIGEST = '1186cce150c43917df64e4a21ff8f528fa755b94fba6fd267f9258a8cbd33f05';
export const EMPTY_DIGEST = '4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945';

export function page(overrides = {}) {
  return {
    id: 'index',
    source: 'website/src/content/docs/index.md',
    type: 'explanation',
    audience: ['maintainer'],
    class: 'L',
    sources: [{ path: 'README.md', sha256: SHA_A }],
    reviewDigest: ONE_SOURCE_DIGEST,
    reviewedAt: '2026-11-14',
    appliesTo: 'unreleased',
    ...overrides,
  };
}

export function manifest(overrides = {}) {
  return {
    schemaVersion: 1,
    site: 'https://example.github.io',
    base: '/project',
    repository: 'https://github.com/example/project',
    docsVersion: 'dev',
    pages: [page()],
    ...overrides,
  };
}

export function releaseEvidence(overrides = {}) {
  return {
    tag: 'v0.11.0',
    commit: '2fc31e9f9c4d019e2b9f3b4f004978028808c74b',
    url: 'https://github.com/example/project/releases/tag/v0.11.0',
    observedAt: '2026-11-14',
    verification: 'public metadata only',
    asset: 'example_0.11.0_darwin_all.tar.gz',
    sha256: SHA_A,
    ...overrides,
  };
}
