package main

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

// Brand identity shown once at the start of the terminal guide. The tagline is
// README's existing one-line vision; the docs URL is the complete, copyable
// address. Nothing here launches a browser or contacts the network.
const (
	brandProduct   = "Agentic SDD"
	brandTagline   = "Build software with AI. Stay the engineer."
	brandCredit    = "by nawodyaishan"
	brandDocsURL   = "https://github.com/nawodyaishan/agentic-sdd#readme"
	brandWordmark1 = "+------------------------------------------------+"
	brandWordmark2 = "|                  AGENTIC SDD                   |"
	brandWordmark3 = "+------------------------------------------------+"
	brandName      = "AGENTIC SDD"
	brandMotif     = "UNDERSTAND -> PLAN -> BUILD -> VERIFY"
	brandProseMax  = 72
	brandArtWidth  = 60
)

// The opt-in sweep has two equal passes over the single name line: ten frames
// forward, then ten returning. Twenty 75 ms frames are 1.5 seconds total.
const (
	brandSweepFrames    = 20
	brandSweepDelay     = 75 * time.Millisecond
	brandSweepHighlight = 3
)

// accent returns s in the cyan product accent when color is enabled. Color is
// supplementary: the same text must stand alone without it.
func accent(env *Env, s string) string { return env.style("36", s) }

// heading returns s bold when color is enabled.
func heading(env *Env, s string) string { return env.style("1", s) }

// caution returns s yellow when color is enabled. Yellow marks meaningful
// caution only; status and action always remain words.
func caution(env *Env, s string) string { return env.style("33", s) }

// brandWidth resolves the layout width, treating an unset or nil Env as the
// usual 80-column default. Callers never need a non-zero Env.
func brandWidth(env *Env) int {
	if env == nil || env.Width <= 0 {
		return 80
	}
	return env.Width
}

// printBrandedGuideIntro writes the guide entry once: compact ASCII wordmark or
// narrow text title, workflow motif, tagline, credit, installed version and docs
// URL. Animation happens only when explicitly requested and every capability
// still allows it. The function is terminal-only at its call site and never
// reads input, environment or build globals.
func printBrandedGuideIntro(out io.Writer, env *Env, installed string, animate bool, wait func(time.Duration)) {
	width := brandWidth(env)
	if !shouldAnimateBrand(env, animate) || wait == nil {
		if width < brandArtWidth {
			fmt.Fprintln(out, accent(env, brandProduct))
		} else {
			for _, line := range []string{brandWordmark1, brandWordmark2, brandWordmark3} {
				fmt.Fprintln(out, accent(env, line))
			}
		}
		printBrandedMotif(out, env)
		printBrandedIntroDetails(out, env, installed)
		return
	}

	// Only the fixed-width name line is redrawn. The frame comes from the same
	// static constant, so every frame has identical visible width; borders and
	// all later output stay append-only. Every segment resets SGR before waiting.
	fmt.Fprintln(out, accent(env, brandWordmark1))
	for frame, start := range brandSweepStarts() {
		if frame > 0 {
			fmt.Fprint(out, "\r")
		}
		end := start + brandSweepHighlight
		fmt.Fprint(out, accent(env, brandWordmark2[:start]),
			env.style("1;35", brandWordmark2[start:end]), accent(env, brandWordmark2[end:]))
		wait(brandSweepDelay)
	}
	// Settle on the exact static name before printing the border, motif, details
	// or any prompts.
	fmt.Fprint(out, "\r")
	fmt.Fprintln(out, accent(env, brandWordmark2))
	fmt.Fprintln(out, accent(env, brandWordmark3))
	printBrandedMotif(out, env)
	printBrandedIntroDetails(out, env, installed)
}

// brandSweepStarts returns one highlight start offset per frame: ten positions
// sweeping left to right across the name, then the same positions returning.
// The range keeps the highlight over the name's glyphs on every frame; it never
// sweeps the invisible padding. The list length is always brandSweepFrames, and
// only start offsets are generated, so waiting stays separate from frame
// construction.
func brandSweepStarts() []int {
	passes := brandSweepFrames / 2
	nameStart := strings.Index(brandWordmark2, brandName)
	if nameStart < 1 {
		nameStart = 1
	}
	first := nameStart - 1
	last := nameStart + len(brandName) - brandSweepHighlight
	travel := last - first
	starts := make([]int, 0, brandSweepFrames)
	for frame := 0; frame < passes; frame++ {
		starts = append(starts, first+frame*travel/(passes-1))
	}
	for frame := passes - 1; frame >= 0; frame-- {
		starts = append(starts, first+frame*travel/(passes-1))
	}
	return starts
}

// printBrandedMotif writes the static workflow motif once. It is not a progress
// indicator: it wraps only as prose on narrow terminals and never animates.
func printBrandedMotif(out io.Writer, env *Env) {
	width := brandWidth(env)
	if width > brandProseMax {
		width = brandProseMax
	}
	for _, line := range wrapProse(brandMotif, width) {
		fmt.Fprintln(out, accent(env, line))
	}
}

// printBrandedIntroDetails writes the tagline, credit, version and docs. The
// tagline wraps at the guide's prose width; long indivisible values (the docs
// URL) are left whole for the terminal's own soft wrapping.
func printBrandedIntroDetails(out io.Writer, env *Env, installed string) {
	width := brandWidth(env)
	if width > brandProseMax {
		width = brandProseMax
	}
	for _, line := range wrapProse(brandTagline, width) {
		fmt.Fprintln(out, line)
	}
	fmt.Fprintln(out, brandCredit)
	fmt.Fprintf(out, "Version: %s\n", installed)
	if _, stable := parseUpdateVersion(installed); !stable {
		printWrappedProse(out, "", "Development build or custom version; release comparison unavailable.", width)
	}
	fmt.Fprintln(out, "Docs: "+brandDocsURL)
	fmt.Fprintln(out)
}

// shouldAnimateBrand reports whether the opt-in animation may run. Static is
// the default; narrow, redirected, plain, dumb or non-interactive runs always
// win over the request.
func shouldAnimateBrand(env *Env, animate bool) bool {
	return animate &&
		env != nil &&
		env.Interactive &&
		env.StdoutTTY &&
		env.Color &&
		!env.ASCII &&
		brandWidth(env) >= brandArtWidth
}

// wrapProse splits prose into lines of at most width runes. Wrapping counts
// Unicode code points, not bytes; it deliberately makes no display-width claim
// for wide glyphs, which the terminal soft-wraps. Words are never split, so a
// long path or command passed as one token survives intact. Prose only: callers
// must print copyable commands and paths without wrapping.
func wrapProse(text string, width int) []string {
	if width < 1 {
		width = brandProseMax
	}
	words := splitProseWords(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	line := ""
	for _, word := range words {
		switch {
		case line == "":
			line = word
		case utf8.RuneCountInString(line)+1+utf8.RuneCountInString(word) <= width:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	return append(lines, line)
}

// splitProseWords collapses runs of whitespace without splitting interior
// characters, so wrapping never alters a token.
func splitProseWords(text string) []string {
	var words []string
	start := -1
	for i, r := range text {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if start >= 0 {
				words = append(words, text[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		words = append(words, text[start:])
	}
	return words
}

// printStackedCounts writes alignment-free labeled counts for narrow terminals:
// one label line and one line per count. No padding, truncation or styling, so
// any label or number stays readable.
func printStackedCounts(out io.Writer, label string, install, replace, upToDate int) {
	fmt.Fprintf(out, "  %s\n", label)
	fmt.Fprintf(out, "    %d to install\n", install)
	fmt.Fprintf(out, "    %d to replace\n", replace)
	fmt.Fprintf(out, "    %d up to date\n", upToDate)
}
