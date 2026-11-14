package tui

// Documentation link registry and fallback rendering (Feature 020). The
// destinations themselves are the Link constants in opener.go; this file only
// orders them and renders the visible fallback line. Nothing here converts an
// arbitrary string into a URL, composes one at run time, starts a process,
// calls a shell or requires a clipboard.

// DocsLinks returns the fixed documentation destinations in display order:
// getting started, the CLI and TUI references, skills and updating,
// troubleshooting, then contributing. Every element is trusted; the registry
// returns a fresh slice per call, so a caller may trim or reorder its copy
// without affecting the canonical order.
func DocsLinks() []Link {
	return []Link{
		LinkDocsGettingStarted,
		LinkDocsCLI,
		LinkDocsTUI,
		LinkDocsSkills,
		LinkDocsUpdates,
		LinkDocsTroubleshooting,
		LinkDocsContributing,
	}
}

// DocsFallbackLine renders the selectable fallback for one documentation link
// as "Label: URL", with the literal URL always visible. It is what remains on
// screen when no browser can be opened — a nil opener, a headless or remote
// session, an unsupported platform, or a terminal without OSC 8 hyperlink
// support — so the user can copy the destination by hand. Rendering it starts
// nothing and touches no clipboard.
//
// An untrusted link, including the zero Link, returns an empty string: there is
// no fixed destination to show, and no arbitrary text is promoted to a URL.
func DocsFallbackLine(link Link) string {
	url := link.URL()
	label := link.Label()
	if url == "" || label == "" {
		return ""
	}
	return label + ": " + url
}
