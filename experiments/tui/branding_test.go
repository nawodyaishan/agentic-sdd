package tui_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"agentic-sdd/experiments/tui"
	"github.com/charmbracelet/x/ansi"
)

// brandWidths are the layout breakpoints Feature 013 FR-02 names. Every branded
// line must fit each of them in terminal cells, not runes or bytes.
var brandWidths = []int{40, 60, 80, 110}

// TestBrandIdentityConstants pins the exact product identity. The expected
// values are the strings Feature 013 FR-08 requires verbatim; the author
// spelling is the GitHub handle, so a rename here would break the About link.
func TestBrandIdentityConstants(t *testing.T) {
	if got, want := tui.BrandName, "AGENTIC SDD"; got != want {
		t.Errorf("BrandName = %q, want %q", got, want)
	}
	if got, want := tui.BrandTagline, "Build software with AI. Stay the engineer."; got != want {
		t.Errorf("BrandTagline = %q, want %q", got, want)
	}
	if got, want := tui.BrandAuthor, "nawodyaishan"; got != want {
		t.Errorf("BrandAuthor = %q, want %q", got, want)
	}
}

// TestBrandStaticStateIsComplete proves the no-motion state starts finished, so
// a static/reduced-motion session never schedules a reveal tick. Advancing a
// finished state must not change its output or reopen the animation.
func TestBrandStaticStateIsComplete(t *testing.T) {
	state := tui.NewBrandState(false)
	if !state.Done() {
		t.Fatal("NewBrandState(false).Done() = false, want true: static mode is complete immediately")
	}

	first := state.Header(80, false)
	if !strings.Contains(first, tui.BrandName) {
		t.Errorf("static header = %q, want it to contain %q", first, tui.BrandName)
	}
	if !strings.Contains(first, "by "+tui.BrandAuthor) {
		t.Errorf("static header = %q, want the exact attribution %q", first, "by "+tui.BrandAuthor)
	}

	advanced := state.Advance()
	if !advanced.Done() {
		t.Fatal("Advance() reopened a finished static state")
	}
	if got := advanced.Header(80, false); got != first {
		t.Errorf("Advance() changed a finished static header:\n got %q\nwant %q", got, first)
	}
}

// TestBrandRevealIsFiniteAndBounded proves the reveal is a short, terminating
// sequence: Feature 013 FR-08 allows at most ten frames, which is at most one
// second at the root's 100 ms tick. A finished reveal stays finished.
func TestBrandRevealIsFiniteAndBounded(t *testing.T) {
	state := tui.NewBrandState(true)
	if state.Done() {
		t.Fatal("NewBrandState(true).Done() = true, want an unfinished reveal")
	}

	frames := 0
	for !state.Done() {
		if frames > 50 {
			t.Fatalf("reveal still running after %d advances; it must terminate", frames)
		}
		state = state.Advance()
		frames++
	}
	if frames < 2 {
		t.Fatalf("reveal finished after %d advance(s), want a multi-frame reveal", frames)
	}
	if frames > 10 {
		t.Fatalf("reveal took %d advances, want at most 10 frames (FR-08)", frames)
	}

	settled := state.Header(80, false)
	for i := 0; i < 3; i++ {
		state = state.Advance()
		if !state.Done() {
			t.Fatalf("Advance() %d after completion reopened the reveal", i+1)
		}
		if got := state.Header(80, false); got != settled {
			t.Errorf("Advance() %d after completion changed the header:\n got %q\nwant %q", i+1, got, settled)
		}
	}

	// A single-frame reveal would satisfy "finite" while showing no animation at
	// all, so require a few genuinely different frames.
	seen := make(map[string]bool)
	for _, header := range brandFrameHeaders(t, 80, false) {
		seen[header] = true
	}
	if len(seen) < 3 {
		t.Errorf("reveal produced %d distinct frame(s), want at least 3", len(seen))
	}
}

// TestBrandLogoBoxStaysAligned proves the compact logo is a real box: every
// line of the mark occupies the same cell width on every frame, so the reveal
// never shifts the border or reflows the header.
func TestBrandLogoBoxStaysAligned(t *testing.T) {
	for _, width := range brandWidths {
		for _, ascii := range []bool{false, true} {
			for frame, header := range brandFrameHeaders(t, width, ascii) {
				lines := strings.Split(header, "\n")
				if len(lines) < 3 {
					t.Fatalf("width %d ascii=%v frame %d: header has %d line(s), want a three-line logo", width, ascii, frame, len(lines))
				}
				want := ansi.StringWidth(lines[0])
				for i, line := range lines[:3] {
					if got := ansi.StringWidth(line); got != want {
						t.Errorf("width %d ascii=%v frame %d: logo line %d = %q is %d cells, want %d", width, ascii, frame, i, line, got, want)
					}
				}
			}
		}
	}
}

// TestBrandRevealIsDeterministic proves the frames depend only on the state, so
// a resize or re-render cannot desynchronise two models and tests can compare
// frames directly.
func TestBrandRevealIsDeterministic(t *testing.T) {
	left := tui.NewBrandState(true)
	right := tui.NewBrandState(true)
	for frame := 0; frame <= 10; frame++ {
		for _, ascii := range []bool{false, true} {
			got, want := left.Header(80, ascii), right.Header(80, ascii)
			if got != want {
				t.Fatalf("frame %d (ascii=%v) differs between identical states:\n got %q\nwant %q", frame, ascii, got, want)
			}
		}
		left = left.Advance()
		right = right.Advance()
	}
}

// TestBrandRevealStartsPartialAndSettlesOnIdentity proves the reveal actually
// reveals: the first frame must not already show the finished header, and the
// settled frame must show the complete product name.
func TestBrandRevealStartsPartialAndSettlesOnIdentity(t *testing.T) {
	settled := tui.NewBrandState(false).Header(80, false)
	start := tui.NewBrandState(true).Header(80, false)

	if start == settled {
		t.Fatalf("first reveal frame already equals the settled header %q", settled)
	}
	if strings.Contains(start, tui.BrandName) {
		t.Errorf("first reveal frame already shows the whole name: %q", start)
	}
	if !strings.Contains(settled, tui.BrandName) {
		t.Errorf("settled header = %q, want it to contain %q", settled, tui.BrandName)
	}
}

// TestBrandHeaderFitsTerminalWidth checks every branded line against the cell
// width it was given, at every reveal frame and in both symbol modes. The
// tagline may wrap, but no word may be dropped and no line may overflow.
func TestBrandHeaderFitsTerminalWidth(t *testing.T) {
	for _, width := range brandWidths {
		for _, ascii := range []bool{false, true} {
			for frame, header := range brandFrameHeaders(t, width, ascii) {
				if header == "" {
					t.Fatalf("width %d ascii=%v frame %d: empty header", width, ascii, frame)
				}
				for i, line := range strings.Split(header, "\n") {
					if got := ansi.StringWidth(line); got > width {
						t.Errorf("width %d ascii=%v frame %d line %d = %q is %d cells", width, ascii, frame, i, line, got)
					}
				}
			}
		}

		settled := tui.NewBrandState(false).Header(width, false)
		if !strings.Contains(settled, tui.BrandName) {
			t.Errorf("width %d: header = %q, want the product name", width, settled)
		}
		if !strings.Contains(settled, "by "+tui.BrandAuthor) {
			t.Errorf("width %d: header = %q, want the exact attribution %q", width, settled, "by "+tui.BrandAuthor)
		}
		// Wrapping may split the tagline across lines but must never lose a word.
		if flat := strings.Join(strings.Fields(settled), " "); !strings.Contains(flat, tui.BrandTagline) {
			t.Errorf("width %d: header = %q, want the complete tagline %q", width, settled, tui.BrandTagline)
		}
	}
}

// TestBrandHeaderCollapsesWithoutWidth covers zero, negative and very small
// widths: the component must not panic, must never overflow, and must not
// attempt attribution it cannot fit. Root About keeps attribution in that case.
func TestBrandHeaderCollapsesWithoutWidth(t *testing.T) {
	for _, width := range []int{0, -1, -80} {
		if got := tui.NewBrandState(false).Header(width, false); got != "" {
			t.Errorf("Header(%d) = %q, want empty: no width means no branding block", width, got)
		}
	}

	for _, width := range []int{1, 5, 10, 14} {
		for frame, header := range brandFrameHeaders(t, width, false) {
			for i, line := range strings.Split(header, "\n") {
				if got := ansi.StringWidth(line); got > width {
					t.Errorf("width %d frame %d line %d = %q is %d cells", width, frame, i, line, got)
				}
			}
			if strings.Contains(header, "by "+tui.BrandAuthor) {
				t.Errorf("width %d frame %d = %q, want attribution omitted: it cannot fit", width, frame, header)
			}
		}
	}
}

// TestBrandHeaderASCIIIsSevenBit proves the ASCII mode emits only 7-bit bytes
// in every frame and at every width, including the widest layout.
func TestBrandHeaderASCIIIsSevenBit(t *testing.T) {
	for _, width := range append([]int{12, 16, 20}, brandWidths...) {
		for frame, header := range brandFrameHeaders(t, width, true) {
			for i := 0; i < len(header); i++ {
				if header[i] >= utf8.RuneSelf {
					t.Fatalf("width %d frame %d: ASCII header contains non-ASCII byte %#x in %q", width, frame, header[i], header)
				}
			}
		}
	}
}

// TestBrandHeaderUnicodeKeepsTextLabels proves the Unicode mode differs only in
// glyphs: the product name and attribution stay literal text, so no meaning is
// carried by a symbol or by colour.
func TestBrandHeaderUnicodeKeepsTextLabels(t *testing.T) {
	settled := tui.NewBrandState(false).Header(80, false)
	if strings.Contains(settled, "+") {
		t.Errorf("Unicode header = %q, want the box-drawing variant, not ASCII borders", settled)
	}
	if !strings.ContainsAny(settled, "╭╰") {
		t.Errorf("Unicode header = %q, want box-drawing corners", settled)
	}
	if !strings.Contains(settled, tui.BrandName) || !strings.Contains(settled, "by "+tui.BrandAuthor) {
		t.Errorf("Unicode header = %q, want the literal product name and attribution", settled)
	}

	ascii := tui.NewBrandState(false).Header(80, true)
	if !strings.Contains(ascii, "+") || strings.ContainsAny(ascii, "╭╰│") {
		t.Errorf("ASCII header = %q, want ASCII borders only", ascii)
	}
}

// TestBrandHeaderIsUnstyledText proves the component returns plain text: the
// root shell owns colour, and a stray control byte here would corrupt the
// terminal in plain/headless modes.
func TestBrandHeaderIsUnstyledText(t *testing.T) {
	for _, width := range append([]int{1, 14, 16}, brandWidths...) {
		for _, ascii := range []bool{false, true} {
			for frame, header := range brandFrameHeaders(t, width, ascii) {
				if strings.ContainsAny(header, "\x1b\r\x00\t") {
					t.Errorf("width %d ascii=%v frame %d: header contains a control byte: %q", width, ascii, frame, header)
				}
				if strings.ContainsAny(header, "%") {
					t.Errorf("width %d ascii=%v frame %d: header looks like progress output: %q", width, ascii, frame, header)
				}
			}
		}
	}
}

// brandFrameHeaders returns every header the reveal produces, starting with the
// initial frame and ending with the settled one. The loop is bounded so a
// non-terminating state fails the test instead of hanging it.
func brandFrameHeaders(t *testing.T, width int, ascii bool) []string {
	t.Helper()
	state := tui.NewBrandState(true)
	headers := []string{state.Header(width, ascii)}
	for !state.Done() {
		if len(headers) > 50 {
			t.Fatalf("reveal at width %d (ascii=%v) did not terminate", width, ascii)
		}
		state = state.Advance()
		headers = append(headers, state.Header(width, ascii))
	}
	return headers
}
