// SEO metadata for the built documentation site.
//
// Starlight 0.43.1 route middleware runs after the route data (including its
// `<head>` entries) is resolved and before `Page.astro` renders, so changes to
// `locals.starlightRoute.head` are what the prerendered HTML contains.
// See `node_modules/@astrojs/starlight/dist/routes/common.astro`.
import { defineRouteMiddleware } from '@astrojs/starlight/route-data';
import manifest from '../docs-manifest.json';

type HeadTag = {
	tag: 'base' | 'link' | 'meta' | 'noscript' | 'script' | 'style' | 'template' | 'title';
	attrs?: Record<string, string | boolean | undefined>;
	content?: string;
};

type SidebarNode = {
	type?: 'group' | 'link';
	label?: string;
	href?: string;
	isCurrent?: boolean;
	entries?: SidebarNode[];
};

const HOME_TITLE = 'Agentic-SDD — spec-driven engineering skills and CLI';
const OG_IMAGE_ALT = 'Agentic-SDD — spec-driven engineering skills and CLI';
const basePath = manifest.base.endsWith('/') ? manifest.base : `${manifest.base}/`;

function findMeta(head: HeadTag[], key: 'name' | 'property', value: string): number {
	return head.findIndex((tag) => tag.tag === 'meta' && tag.attrs?.[key] === value);
}

function setMeta(head: HeadTag[], key: 'name' | 'property', value: string, content: string): void {
	const entry: HeadTag = { tag: 'meta', attrs: { [key]: value, content } };
	const index = findMeta(head, key, value);
	if (index === -1) head.push(entry);
	else head[index] = entry;
}

function removeTags(head: HeadTag[], predicate: (tag: HeadTag) => boolean): void {
	for (let index = head.length - 1; index >= 0; index -= 1) {
		if (predicate(head[index])) head.splice(index, 1);
	}
}

function setTitle(head: HeadTag[], title: string): void {
	const entry: HeadTag = { tag: 'title', content: title };
	const index = head.findIndex((tag) => tag.tag === 'title');
	if (index === -1) head.unshift(entry);
	else head[index] = entry;
}

function addJsonLd(head: HeadTag[], data: unknown): void {
	head.push({
		tag: 'script',
		attrs: { type: 'application/ld+json' },
		content: JSON.stringify(data).replace(/</g, '\\u003c'),
	});
}

/** Label of the closest sidebar group that owns the current page. */
function currentGroupLabel(entries: SidebarNode[], pathname: string, parent?: string): string | undefined {
	for (const entry of entries) {
		if (entry.type === 'group') {
			const found = currentGroupLabel(entry.entries ?? [], pathname, entry.label ?? parent);
			if (found !== undefined) return found;
			continue;
		}
		if (entry.isCurrent === true || entry.href === pathname) return parent;
	}
	return undefined;
}

export const onRequest = defineRouteMiddleware((context) => {
	const route = context.locals.starlightRoute;
	const head = route.head as HeadTag[];
	const is404 = route.entry.id === '404';
	// Starlight normalizes the docs collection's root `index` entry id to an empty string.
	const isHome = route.entry.id === '';

	const site = context.site ?? new URL(manifest.site);
	const homeUrl = new URL(basePath, site).href;
	const ogImageUrl = new URL(`${basePath}og-card.png`, site).href;

	const canonicalTag = head.find((tag) => tag.tag === 'link' && tag.attrs?.rel === 'canonical');
	const canonical = typeof canonicalTag?.attrs?.href === 'string' ? canonicalTag.attrs.href : undefined;

	if (is404) {
		removeTags(
			head,
			(tag) =>
				(tag.tag === 'link' && tag.attrs?.rel === 'canonical') ||
				(tag.tag === 'meta' && tag.attrs?.property === 'og:url'),
		);
		setMeta(head, 'name', 'robots', 'noindex');
		return;
	}

	if (isHome) {
		setTitle(head, HOME_TITLE);
		setMeta(head, 'property', 'og:type', 'website');
	}

	setMeta(head, 'property', 'og:image', ogImageUrl);
	setMeta(head, 'property', 'og:image:width', '1200');
	setMeta(head, 'property', 'og:image:height', '630');
	setMeta(head, 'property', 'og:image:alt', OG_IMAGE_ALT);
	setMeta(head, 'name', 'twitter:image', ogImageUrl);

	const url = canonical ?? new URL(context.url.pathname, manifest.site).href;
	const title = String(route.entry.data.title ?? route.id);
	const ogDescription = head.find(
		(tag) => tag.tag === 'meta' && tag.attrs?.property === 'og:description',
	)?.attrs?.content;
	const description = route.entry.data.description ?? ogDescription;

	if (!isHome) {
		const group = currentGroupLabel(route.sidebar as SidebarNode[], context.url.pathname);
		const items: Array<{ name: string; item?: string }> = [
			{ name: 'Start here', item: homeUrl },
		];
		if (group !== undefined) items.push({ name: group });
		items.push({ name: title, item: url });
		addJsonLd(head, {
			'@context': 'https://schema.org',
			'@type': 'BreadcrumbList',
			itemListElement: items.map((item, index) => ({
				'@type': 'ListItem',
				position: index + 1,
				...item,
			})),
		});
	}

	addJsonLd(head, {
		'@context': 'https://schema.org',
		'@type': 'TechArticle',
		headline: title,
		...(description === undefined ? {} : { description: String(description) }),
		url,
		inLanguage: 'en',
		isPartOf: {
			'@type': 'CreativeWork',
			name: 'Agentic-SDD',
			url: manifest.repository,
		},
	});
});
