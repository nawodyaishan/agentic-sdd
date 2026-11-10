package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Brand identity. These are the only definitions of the product name, tagline
// and personal attribution in the TUI: header, welcome, confirmation, results
// and About all render these constants instead of repeating strings. The author
// spelling is the GitHub handle, so it must never be re-cased or abbreviated.
const (
	// BrandName is the product name.
	BrandName = "AGENTIC SDD"
	// BrandTagline is README's one-line vision.
	BrandTagline = "Build software with AI. Stay the engineer."
	// BrandAuthor is the personal attribution shown beside the product.
	BrandAuthor = "nawodyaishan"

	// BrandFrames is the number of frames in the compact logo reveal. The root
	// model owns the ticker and advances once per frame; at the 100 ms tick this
	// stays within Feature 013 FR-08's one-second and ten-frames-per-second
	// budget. This component never schedules, sleeps or repeats a frame.
	BrandFrames = 10

	// brandLastFrame is the settled frame index. A state at or past it is done.
	brandLastFrame = BrandFrames - 1

	// brandMark is the small terminal-prompt motif beside the wordmark. It is
	// ASCII, single-width and never carries meaning on its own: the product name
	// next to it is the label.
	brandMark = ">_"
	// brandCredit is the secondary attribution line.
	brandCredit = "by " + BrandAuthor

	brandNameWidth    = len(BrandName)
	brandContentWidth = len(brandMark) + 1 + brandNameWidth
	// brandLogoWidth is the outer width of the boxed wordmark: content plus one
	// padding space and one border cell on each side.
	brandLogoWidth = brandContentWidth + 4
)

// BrandState is an immutable reveal cursor for the compact logo. Construct it
// with NewBrandState and advance it from the root model's tick; every method
// returns a value, so a model can copy or discard a state without side effects.
//
// The state only counts frames. It holds no timer, terminal, writer or global,
// which keeps the reveal deterministic in tests and makes a skipped animation
// (motion off, narrow terminal, hidden view) a matter of simply not advancing.
// The zero value is the first motion frame; prefer NewBrandState for intent.
type BrandState struct {
	frame int
}

// NewBrandState returns the starting state. With motion=false it starts already
// settled, so a static, reduced-motion or accessibility session never needs a
// tick. With motion=true it starts on the first reveal frame and becomes Done
// after at most BrandFrames frames.
func NewBrandState(motion bool) BrandState {
	if !motion {
		return BrandState{frame: brandLastFrame}
	}
	return BrandState{}
}

// Advance returns the next reveal frame. Advancing a finished state returns it
// unchanged, so a stray late tick cannot restart or loop the animation.
func (b BrandState) Advance() BrandState {
	if b.Done() {
		return b
	}
	return BrandState{frame: b.frame + 1}
}

// Done reports whether the reveal has settled. Callers stop their ticker when
// it is true and render the static header from then on.
func (b BrandState) Done() bool {
	return b.frame >= brandLastFrame
}

// Header renders the branded header block at the given terminal width: the
// compact logo (or a plain product name when the box cannot fit), the tagline,
// and the personal attribution. ascii selects 7-bit borders for plain,
// accessibility and ASCII sessions; the Unicode variant differs only in
// box-drawing glyphs, so both carry identical text.
//
// Every line is guaranteed to fit width terminal cells. Narrow widths drop the
// logo box first, then the attribution, so About remains the place that always
// preserves attribution. A non-positive width returns an empty string: there is
// no branded block to show, and the caller collapses the header space.
//
// The returned text is unstyled: no colour, hyperlink or other control
// sequence. Colour, focus and terminal modes belong to the root shell's theme.
func (b BrandState) Header(width int, ascii bool) string {
	if width <= 0 {
		return ""
	}

	name := brandRevealedName(b.frame)
	lines := make([]string, 0, 6)
	switch {
	case width >= brandLogoWidth:
		lines = append(lines, brandLogo(name, ascii)...)
	case width >= brandNameWidth:
		lines = append(lines, name)
	default:
		// Neither the box nor the product name fits; branding collapses rather
		// than truncating the identity into something unrecognisable.
		return ""
	}

	if tagline, ok := brandTaglineLines(width); ok {
		lines = append(lines, tagline...)
	}
	if width >= ansi.StringWidth(brandCredit) {
		lines = append(lines, brandCredit)
	}
	return strings.Join(lines, "\n")
}

// brandRevealedName returns the product name prefix visible on frame. The
// reveal is a simple left-to-right type-in: every letter appears exactly once,
// no frame invents content, and the settled frame is the complete name.
//
// Progress is counted in visible letters rather than runes so no frame is spent
// revealing only the word space, which would look like a stalled animation.
func brandRevealedName(frame int) string {
	if frame <= 0 {
		return ""
	}
	if frame >= brandLastFrame {
		return BrandName
	}

	letters := []rune(BrandName)
	visible := len(letters) - strings.Count(BrandName, " ")
	target := (frame*visible + brandLastFrame - 1) / brandLastFrame
	revealed := 0
	for i, r := range letters {
		if r != ' ' {
			revealed++
		}
		if revealed >= target {
			return string(letters[:i+1])
		}
	}
	return BrandName
}

// brandLogo draws the compact three-line wordmark. The box width is fixed by the
// product name, so every reveal frame has identical outer dimensions and the
// header never reflows as letters appear.
func brandLogo(name string, ascii bool) []string {
	topLeft, topRight := "╭", "╮"
	bottomLeft, bottomRight, horizontal, vertical := "╰", "╯", "─", "│"
	if ascii {
		topLeft, topRight, bottomLeft, bottomRight = "+", "+", "+", "+"
		horizontal, vertical = "-", "|"
	}

	border := strings.Repeat(horizontal, brandContentWidth+2)
	return []string{
		topLeft + border + topRight,
		vertical + " " + padCells(brandMark+" "+name, brandContentWidth) + " " + vertical,
		bottomLeft + border + bottomRight,
	}
}

// brandTaglineLines wraps the tagline to width terminal cells. It reports false
// when a single word cannot fit, in which case the caller omits the tagline
// instead of splitting prose or overflowing the width.
func brandTaglineLines(width int) ([]string, bool) {
	words := strings.Fields(BrandTagline)
	if len(words) == 0 {
		return nil, false
	}
	for _, word := range words {
		if ansi.StringWidth(word) > width {
			return nil, false
		}
	}

	lines := make([]string, 0, len(words))
	line := ""
	for _, word := range words {
		candidate := word
		if line != "" {
			candidate = line + " " + word
		}
		if ansi.StringWidth(candidate) <= width {
			line = candidate
			continue
		}
		lines = append(lines, line)
		line = word
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines, true
}

// padCells right-pads s with spaces to width terminal cells. It measures cells
// rather than bytes or runes so wide or combining glyphs cannot shift the box
// border, and it never truncates: a caller that needs truncation must decide
// what to drop.
func padCells(s string, width int) string {
	if pad := width - ansi.StringWidth(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}
