// Single shared owner for the public creator/maintainer identity.
//
// The shared footer imports these values directly. The homepage byline repeats
// them as ordinary Markdown so repository and exported-source readers still see
// the attribution; `website/tests/entrypoints-attribution.test.mjs` checks that
// authored copy against this module instead of generating prose from it.
export const maintainer = Object.freeze({
	name: 'Nawodya Ishan',
	github: 'https://github.com/nawodyaishan',
	website: 'https://nawodyaishan.github.io/',
});
