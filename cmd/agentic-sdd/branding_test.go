package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

const (
	brandTestTagline = "Build software with AI. Stay the engineer."
	brandTestCredit  = "by nawodyaishan"
	brandTestDocs    = "Docs: https://github.com/nawodyaishan/agentic-sdd#readme"
	brandTestURL     = "https://github.com/nawodyaishan/agentic-sdd#readme"
)

// brandTestEnv is a fully capable terminal that may animate.
func brandTestEnv() *Env {
	return &Env{Interactive: true, StdoutTTY: true, Color: true, Width: 80}
}

// assertBrandOnlySGR rejects every control sequence except SGR styling. That
// covers cursor movement/hiding, OSC links, screen clears and carriage-return
// redraws, none of which the branded entry may emit.
func assertBrandOnlySGR(t *testing.T, out string) {
	t.Helper()
	if strings.ContainsRune(out, '\r') {
		t.Fatalf("branding output contains a carriage return: %q", out)
	}
	for i := 0; i < len(out); i++ {
		if out[i] != 0x1b {
			continue
		}
		if i+1 >= len(out) || out[i+1] != '[' {
			t.Fatalf("branding output contains a non-SGR escape at byte %d: %q", i, out)
		}
		j := i + 2
		for j < len(out) && out[j] != 'm' {
			if out[j] != ';' && (out[j] < '0' || out[j] > '9') {
				t.Fatalf("branding output contains a non-SGR escape at byte %d: %q", i, out)
			}
			j++
		}
		if j >= len(out) {
			t.Fatalf("branding output has an unterminated escape at byte %d: %q", i, out)
		}
		i = j
	}
}

// TestBrandingStaticIntroWide pins the default presentation: the compact art,
// the exact tagline/credit/version/docs lines, no waits and only SGR styling.
func TestBrandingStaticIntroWide(t *testing.T) {
	var out bytes.Buffer
	waits := 0
	printBrandedGuideIntro(&out, brandTestEnv(), "dev", false, func(time.Duration) {
		waits++
	})
	got := out.String()

	if waits != 0 {
		t.Fatalf("static intro performed %d animation waits, want 0", waits)
	}
	assertBrandOnlySGR(t, got)

	art := []string{
		"+------------------------------------------------+",
		"|                  AGENTIC SDD                   |",
		"+------------------------------------------------+",
	}
	for _, line := range art {
		if !strings.Contains(got, "\x1b[36m"+line+"\x1b[0m") {
			t.Fatalf("static intro is missing cyan wordmark line %q:\n%q", line, got)
		}
	}
	if len(brandWordmark2) > 52 {
		t.Fatalf("brand name line is %d columns, want at most 52", len(brandWordmark2))
	}
	if strings.Contains(got, "A G E N T I C") {
		t.Fatalf("brand name still uses spaced-out letters:\n%s", got)
	}
	if n := strings.Count(got, "\x1b[36m"+brandMotif+"\x1b[0m"); n != 1 {
		t.Fatalf("static intro shows the cyan motif %d times, want once:\n%s", n, got)
	}
	for _, want := range []string{brandTestTagline, brandTestCredit, "Version: dev", brandTestDocs} {
		if n := strings.Count(got, want); n != 1 {
			t.Fatalf("intro shows %q %d times, want once:\n%s", want, n, got)
		}
	}
	if strings.Count(got, brandTestURL) != 1 {
		t.Fatalf("intro must show the complete documentation URL once:\n%s", got)
	}
}

// TestBrandingStaticNoColorHasNoANSI proves the plain fallback stays readable
// and byte-clean when color is disabled.
func TestBrandingStaticNoColorHasNoANSI(t *testing.T) {
	env := brandTestEnv()
	env.Color = false

	var out bytes.Buffer
	printBrandedGuideIntro(&out, env, "0.1.0", false, func(time.Duration) {
		t.Fatal("static intro must not wait")
	})
	got := out.String()

	want := "+------------------------------------------------+\n" +
		"|                  AGENTIC SDD                   |\n" +
		"+------------------------------------------------+\n" +
		"UNDERSTAND -> PLAN -> BUILD -> VERIFY\n" +
		brandTestTagline + "\n" +
		brandTestCredit + "\n" +
		"Version: 0.1.0\n" +
		brandTestDocs + "\n\n"
	if got != want {
		t.Fatalf("plain wide intro mismatch:\n got: %q\nwant: %q", got, want)
	}
	if strings.ContainsRune(got, 0x1b) {
		t.Fatalf("color-disabled intro emitted ANSI escapes: %q", got)
	}
}

// TestBrandingNarrowFallback covers the width threshold: below 60 columns the
// art is replaced by a text title, prose wraps and the docs URL stays whole.
func TestBrandingNarrowFallback(t *testing.T) {
	for _, width := range []int{40, 59} {
		var out bytes.Buffer
		env := brandTestEnv()
		env.Color = false
		env.Width = width
		printBrandedGuideIntro(&out, env, "dev", false, func(time.Duration) {
			t.Fatalf("width %d: static intro must not wait", width)
		})
		got := out.String()

		if strings.Contains(got, "+---") || strings.Contains(got, brandWordmark2) {
			t.Fatalf("width %d used art instead of the narrow title:\n%s", width, got)
		}
		if !strings.HasPrefix(got, "Agentic SDD\n") {
			t.Fatalf("width %d must start with the text title:\n%s", width, got)
		}
		if !strings.Contains(got, brandMotif+"\n") {
			t.Fatalf("width %d did not keep the workflow motif:\n%s", width, got)
		}
		if width == 40 {
			if !strings.Contains(got, "Build software with AI. Stay the\nengineer.\n") {
				t.Fatalf("width %d did not wrap the tagline:\n%s", width, got)
			}
		} else if !strings.Contains(got, brandTestTagline+"\n") {
			t.Fatalf("width %d should keep the tagline on one line:\n%s", width, got)
		}
		if !strings.Contains(got, brandTestDocs+"\n") {
			t.Fatalf("width %d did not keep the complete docs line:\n%s", width, got)
		}
		if width == 40 {
			for _, line := range strings.Split(got, "\n") {
				if strings.HasPrefix(line, "Docs: ") {
					continue // indivisible value: terminal soft-wraps it
				}
				if n := utf8.RuneCountInString(line); n > width {
					t.Fatalf("width %d line exceeds width (%d runes): %q", width, n, line)
				}
			}
		}
	}
}

// TestBrandingArtAtSixtyColumns confirms 60 columns is the art threshold.
func TestBrandingArtAtSixtyColumns(t *testing.T) {
	for _, width := range []int{60, 80} {
		var out bytes.Buffer
		env := brandTestEnv()
		env.Width = width
		printBrandedGuideIntro(&out, env, "dev", false, nil)
		if !strings.Contains(out.String(), brandWordmark2) {
			t.Fatalf("width %d must show the wordmark art:\n%s", width, out.String())
		}
	}
}

// settledBrandDisplay models the single logo line being overwritten by CR.
// Frames must have equal visible width (asserted below), so no stale suffix can
// survive. All other terminal controls are rejected by assertBrandOnlySGR.
func settledBrandDisplay(text string) string {
	text = regexp.MustCompile("\x1b\\[[0-9;]*m").ReplaceAllString(text, "")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if pos := strings.LastIndexByte(line, '\r'); pos >= 0 {
			lines[i] = line[pos+1:]
		}
	}
	return strings.Join(lines, "\n")
}

// TestBrandingAnimationChunksAndLimits checks the visible 1.5-second two-pass
// sweep: ten forward and ten return highlights, reset before waits, and a
// static end. Positions must advance on the first pass and retreat on the
// second, and the settled display must equal the static one.
func TestBrandingAnimationChunksAndLimits(t *testing.T) {
	env := brandTestEnv()

	var static bytes.Buffer
	printBrandedGuideIntro(&static, env, "dev", false, nil)

	var out bytes.Buffer
	var waits []time.Duration
	var prefixes []string
	printBrandedGuideIntro(&out, env, "dev", true, func(d time.Duration) {
		waits = append(waits, d)
		prefixes = append(prefixes, out.String())
	})

	if len(waits) != 20 {
		t.Fatalf("animated intro used %d waits, want 20 visible frames", len(waits))
	}
	var total time.Duration
	for i, d := range waits {
		if d != 75*time.Millisecond {
			t.Fatalf("wait %d is %s, want 75ms", i, d)
		}
		total += d
	}
	if total != 1500*time.Millisecond {
		t.Fatalf("planned delay is %s, want 1.5s", total)
	}
	if total > 1500*time.Millisecond {
		t.Fatalf("planned delay %s exceeds the 1.5s cap", total)
	}
	if got, want := settledBrandDisplay(out.String()), settledBrandDisplay(static.String()); got != want {
		t.Fatalf("settled animated display differs from static:\n got: %q\nwant: %q", got, want)
	}
	if strings.Count(out.String(), "\r") != 20 {
		t.Fatalf("want nineteen sweep redraws plus one final static redraw: %q", out.String())
	}
	assertBrandOnlySGR(t, strings.ReplaceAll(out.String(), "\r", ""))
	positions := make([]int, 0, len(prefixes))
	for i, prefix := range prefixes {
		if !strings.HasSuffix(prefix, "\x1b[0m") {
			t.Fatalf("frame %d did not reset styling before its wait: %q", i+1, prefix)
		}
		if n := strings.Count(prefix, "\n"); n != 1 {
			t.Fatalf("frame %d redrew outside the single logo line: %q", i+1, prefix)
		}
		frame := prefix[strings.LastIndexAny(prefix, "\r\n")+1:]
		if got := settledBrandDisplay(frame); got != brandWordmark2 {
			t.Fatalf("frame %d changes visible logo text/width: %q", i+1, got)
		}
		highlight := strings.Index(frame, "\x1b[1;35m")
		if highlight < 0 {
			t.Fatalf("frame %d has no visible magenta highlight: %q", i+1, frame)
		}
		positions = append(positions, len(settledBrandDisplay(frame[:highlight])))
	}
	// The highlight must stay on the visible name, never on frame padding.
	nameStart := strings.Index(settledBrandDisplay(brandWordmark2), brandName)
	for i, position := range positions {
		if position < nameStart-1 || position+brandSweepHighlight > nameStart+len(brandName) {
			t.Fatalf("frame %d highlight %d..%d leaves the name bounds %d..%d",
				i+1, position, position+brandSweepHighlight, nameStart, nameStart+len(brandName))
		}
	}
	// First ten frames sweep forward, the next ten return to the left.
	for i := 1; i < 10; i++ {
		if positions[i] <= positions[i-1] {
			t.Fatalf("forward frame %d does not advance highlight: %d after %d", i+1, positions[i], positions[i-1])
		}
	}
	for i := 11; i < 20; i++ {
		if positions[i] >= positions[i-1] {
			t.Fatalf("return frame %d does not move highlight back: %d after %d", i+1, positions[i], positions[i-1])
		}
	}
	if positions[9] <= positions[0] || positions[19] >= positions[10] {
		t.Fatalf("highlight did not make a forward-then-return pass: %v", positions)
	}
}

// TestBrandingAnimationSuppressed proves every disabled condition wins over
// the explicit opt-in and still yields the static bytes.
func TestBrandingAnimationSuppressed(t *testing.T) {
	cases := []struct {
		name    string
		animate bool
		env     *Env
	}{
		{"opt-in absent", false, brandTestEnv()},
		{"stdin not interactive", true, &Env{StdoutTTY: true, Color: true, Width: 80}},
		{"stdout not a terminal", true, &Env{Interactive: true, Color: true, Width: 80}},
		{"color disabled", true, &Env{Interactive: true, StdoutTTY: true, Width: 80}},
		{"dumb terminal", true, &Env{Interactive: true, StdoutTTY: true, Color: true, ASCII: true, Width: 80}},
		{"narrow terminal", true, &Env{Interactive: true, StdoutTTY: true, Color: true, Width: 59}},
		{"nil environment", true, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			waits := 0
			printBrandedGuideIntro(&out, tc.env, "dev", tc.animate, func(time.Duration) {
				waits++
			})
			if waits != 0 {
				t.Fatalf("disabled animation performed %d waits", waits)
			}

			var static bytes.Buffer
			printBrandedGuideIntro(&static, tc.env, "dev", false, nil)
			if got, want := out.String(), static.String(); got != want {
				t.Fatalf("suppressed animation bytes differ from static:\n got: %q\nwant: %q", got, want)
			}
			if tc.env == nil || !tc.env.Color {
				if strings.ContainsRune(out.String(), 0x1b) {
					t.Fatalf("intro emitted ANSI without color: %q", out.String())
				}
			}
		})
	}
}

// TestBrandingAnimationNilWait keeps the opt-in safe without an injected wait.
func TestBrandingAnimationNilWait(t *testing.T) {
	env := brandTestEnv()
	var out bytes.Buffer
	printBrandedGuideIntro(&out, env, "dev", true, nil)

	var static bytes.Buffer
	printBrandedGuideIntro(&static, env, "dev", false, nil)
	if got, want := out.String(), static.String(); got != want {
		t.Fatalf("nil-wait animation bytes differ from static:\n got: %q\nwant: %q", got, want)
	}
	if !strings.Contains(out.String(), "\x1b[36m") {
		t.Fatalf("nil-wait animation lost its color: %q", out.String())
	}
}

// TestBrandingWrapProse covers rune-aware wrapping, indivisible values and
// the explicit no-display-width-engine approximation.
func TestBrandingWrapProse(t *testing.T) {
	cases := []struct {
		name  string
		text  string
		width int
		want  []string
	}{
		{
			name:  "tagline wraps at forty",
			text:  brandTestTagline,
			width: 40,
			want:  []string{"Build software with AI. Stay the", "engineer."},
		},
		{
			name:  "tagline fits at seventy-two",
			text:  brandTestTagline,
			width: 72,
			want:  []string{brandTestTagline},
		},
		{
			name:  "long indivisible value survives",
			text:  "/very/long/path/with/verylongsegment/file.go",
			width: 10,
			want:  []string{"/very/long/path/with/verylongsegment/file.go"},
		},
		{
			name:  "non-ASCII indivisible path survives",
			text:  "/tmp/日本語/ファイル.go",
			width: 8,
			want:  []string{"/tmp/日本語/ファイル.go"},
		},
		{
			name:  "unset width defaults to prose maximum",
			text:  brandTestTagline,
			width: 0,
			want:  []string{brandTestTagline},
		},
		{
			name:  "wide runes counted as runes",
			text:  "日本語 テスト 文章",
			width: 5,
			want:  []string{"日本語", "テスト", "文章"},
		},
		{
			name:  "motif wraps on narrow terminals",
			text:  brandMotif,
			width: 20,
			want:  []string{"UNDERSTAND -> PLAN", "-> BUILD -> VERIFY"},
		},
		{
			name:  "empty prose",
			text:  "   ",
			width: 20,
			want:  nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := wrapProse(tc.text, tc.width)
			if len(got) != len(tc.want) {
				t.Fatalf("wrapProse(%q, %d) = %q, want %q", tc.text, tc.width, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("wrapProse(%q, %d) = %q, want %q", tc.text, tc.width, got, tc.want)
				}
				if tc.name != "long indivisible value survives" && tc.name != "wide runes counted as runes" {
					continue
				}
				// Indivisible values may exceed the width; ordinary prose may not.
				if strings.Contains(got[i], " ") && utf8.RuneCountInString(got[i]) > tc.width {
					t.Fatalf("wrapped line exceeds width: %q", got[i])
				}
			}
		})
	}
}

// TestBrandingStyles pins the small semantic styling helpers used by later
// summaries: cyan accent, bold heading and yellow caution.
func TestBrandingStyles(t *testing.T) {
	env := brandTestEnv()
	if got, want := accent(env, "x"), "\x1b[36mx\x1b[0m"; got != want {
		t.Fatalf("accent = %q, want %q", got, want)
	}
	if got, want := heading(env, "x"), "\x1b[1mx\x1b[0m"; got != want {
		t.Fatalf("heading = %q, want %q", got, want)
	}
	if got, want := caution(env, "x"), "\x1b[33mx\x1b[0m"; got != want {
		t.Fatalf("caution = %q, want %q", got, want)
	}
	plain := brandTestEnv()
	plain.Color = false
	for name, got := range map[string]string{
		"accent":  accent(plain, "x"),
		"heading": heading(plain, "x"),
		"caution": caution(plain, "x"),
	} {
		if got != "x" {
			t.Fatalf("%s with color disabled = %q, want plain %q", name, got, "x")
		}
	}
	if accent(nil, "x") != "x" || heading(nil, "x") != "x" || caution(nil, "x") != "x" {
		t.Fatal("nil Env must return plain text")
	}
}

// TestBrandingStackedCounts covers the narrow-width, alignment-free count
// layout that later summaries can use instead of padded rows.
func TestBrandingStackedCounts(t *testing.T) {
	var out bytes.Buffer
	printStackedCounts(&out, "claude", 1, 0, 2)
	want := "  claude\n    1 to install\n    0 to replace\n    2 up to date\n"
	if got := out.String(); got != want {
		t.Fatalf("stacked counts mismatch:\n got: %q\nwant: %q", got, want)
	}
	if strings.ContainsRune(out.String(), 0x1b) {
		t.Fatalf("stacked counts must stay plain: %q", out.String())
	}
}
