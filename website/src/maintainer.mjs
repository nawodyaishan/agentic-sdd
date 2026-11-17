// Single shared owner for the public creator/maintainer identity.
//
// The homepage byline is the only visible site credit and repeats these values
// as ordinary Markdown so repository and exported-source readers still see the
// attribution; `website/tests/entrypoints-attribution.test.mjs` checks that
// authored copy against this module instead of generating prose from it. Deep
// pages carry no site-wide footer branding.
export const maintainer = Object.freeze({
	name: 'Nawodya Ishan',
	github: 'https://github.com/nawodyaishan',
	website: 'https://nawodyaishan.github.io/',
});
