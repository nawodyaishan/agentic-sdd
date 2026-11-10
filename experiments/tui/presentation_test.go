package tui

import (
	"reflect"
	"strings"
	"testing"

	"agentic-sdd/internal/skillsync"
	"github.com/charmbracelet/x/ansi"
)

// These tests cover the pure presentation helpers only. They never touch the
// filesystem, the network or a real user home. Test names start with
// TestPresentation or TestHomeOverview so the focused run
// `go test -run 'TestPresentation|TestHomeOverview' .` selects all of them.

func assertNothingWider(t *testing.T, lines []string, limit int) {
	t.Helper()
	for _, line := range lines {
		if got := ansi.StringWidth(line); got > limit {
			t.Fatalf("line width %d exceeds %d: %q", got, limit, line)
		}
	}
}

func hasDoubleBlank(lines []string) bool {
	for i := 1; i < len(lines); i++ {
		if lines[i] == "" && lines[i-1] == "" {
			return true
		}
	}
	return false
}

// flattenWords removes every whitespace character so a check can prove that
// wrapping dropped no text without caring where row breaks or indents landed.
func flattenWords(text string) string {
	return strings.Join(strings.Fields(text), "")
}

// ---------------------------------------------------------------------------
// Paragraph
// ---------------------------------------------------------------------------

func TestPresentationParagraphWrapsOnWordBoundaries(t *testing.T) {
	got := Paragraph("A readable description with useful guidance.", 22, 0)
	want := []string{"A readable description", "with useful guidance."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("prose should wrap between words: got %q want %q", got, want)
	}
}

func TestPresentationParagraphDoesNotSplitWordsThatFit(t *testing.T) {
	got := Paragraph("alpha bravo charlie", 11, 0)
	want := []string{"alpha bravo", "charlie"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPresentationParagraphIndentsEveryLine(t *testing.T) {
	got := Paragraph("alpha beta gamma delta", 12, 2)
	want := []string{"  alpha beta", "  gamma", "  delta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
	for _, line := range got {
		if strings.HasPrefix(line, "   ") {
			t.Fatalf("indent more than requested: %q", line)
		}
	}
}

func TestPresentationParagraphIndentIsBoundedAndNotDuplicated(t *testing.T) {
	// An already indented line keeps exactly the requested indent; it is not
	// indented a second time.
	if got := Paragraph("  hello world", 30, 2); !reflect.DeepEqual(got, []string{"  hello world"}) {
		t.Fatalf("indent was duplicated: %q", got)
	}
	// Only the indent introduces leading spaces: deeper source indentation is
	// normalized to the requested indent instead of accumulating with it.
	if got := Paragraph("      nested item", 30, 2); !reflect.DeepEqual(got, []string{"  nested item"}) {
		t.Fatalf("source indentation must normalize to the requested indent: %q", got)
	}
	// An indent larger than the width is clamped so content still fits.
	got := Paragraph("alpha beta gamma delta", 6, 99)
	assertNothingWider(t, got, 6)
	if flattenWords(strings.Join(got, "")) != flattenWords("alpha beta gamma delta") {
		t.Fatalf("clamped indent lost content: %q", got)
	}
}

func TestPresentationParagraphPreservesBlankBoundaries(t *testing.T) {
	got := Paragraph("aa bb\n\ncc dd", 10, 0)
	want := []string{"aa bb", "", "cc dd"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("a blank source line must stay exactly one blank row: got %q want %q", got, want)
	}
	got = Paragraph("aa bb\ncc dd", 10, 0)
	want = []string{"aa bb", "cc dd"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("a source line break must be preserved: got %q want %q", got, want)
	}
}

func TestPresentationParagraphHardwrapsSingleLongTokenCellAware(t *testing.T) {
	text := "https://example.com/a/very/long/path/token"
	got := Paragraph(text, 12, 0)
	if len(got) < 2 {
		t.Fatalf("a token wider than the line must wrap, got %q", got)
	}
	if strings.Join(got, "") != text {
		t.Fatalf("hardwrap lost or duplicated content: %q", got)
	}
	assertNothingWider(t, got, 12)
}

func TestPresentationParagraphHardwrapKeepsGraphemeClusters(t *testing.T) {
	text := strings.Repeat("ab👩‍💻", 4)
	got := Paragraph(text, 8, 0)
	if strings.Join(got, "") != text {
		t.Fatalf("grapheme-aware wrap changed content: %q", got)
	}
	assertNothingWider(t, got, 8)
	for _, line := range got {
		if strings.HasSuffix(line, "\u200d") || strings.HasPrefix(line, "\u200d") {
			t.Fatalf("a ZWJ grapheme cluster was split: %q", got)
		}
	}
}

func TestPresentationParagraphUnicodeCellWidths(t *testing.T) {
	text := "中文测试字"
	got := Paragraph(text, 5, 0)
	assertNothingWider(t, got, 5)
	if strings.Join(got, "") != text {
		t.Fatalf("wide-character wrap lost content: %q", got)
	}
	// A wide emoji counts as two cells, so a row must respect the width.
	got = Paragraph("ab🚀 cd", 4, 0)
	assertNothingWider(t, got, 4)
	if flattenWords(strings.Join(got, "")) != flattenWords("ab🚀 cd") {
		t.Fatalf("emoji wrap lost content: %q", got)
	}
}

func TestPresentationParagraphANSIWidthsAndEscapes(t *testing.T) {
	input := "\x1b[1;35mImportant\x1b[0m notice \x1b[3mabout\x1b[0m something"
	got := Paragraph(input, 20, 0)
	assertNothingWider(t, got, 20)
	joined := strings.Join(got, " ")
	if strings.Count(joined, "\x1b[0m") != strings.Count(input, "\x1b[0m") {
		t.Fatalf("an escape sequence was broken across rows: %q", got)
	}
	if flattenWords(ansi.Strip(joined)) != flattenWords(ansi.Strip(input)) {
		t.Fatalf("styled text lost words: got %q want %q", ansi.Strip(joined), ansi.Strip(input))
	}
}

func TestPresentationParagraphKeepsOnlyIndentLeadingSpace(t *testing.T) {
	got := Paragraph("  alpha  beta  ", 20, 0)
	if len(got) != 1 || got[0] != "alpha beta" {
		t.Fatalf("outer spacing should be trimmed, got %q", got)
	}
	for _, line := range got {
		if strings.HasSuffix(line, " ") {
			t.Fatalf("trailing space must not be kept: %q", line)
		}
	}

	got = Paragraph("   ", 10, 2)
	if len(got) != 1 || got[0] != "" {
		t.Fatalf("blank input should stay a blank line, got %q", got)
	}
}

func TestPresentationParagraphSmallWidthStaysWithinWidth(t *testing.T) {
	text := "hello world a-much-longer-token"
	for _, width := range []int{1, 2, 5, 8} {
		got := Paragraph(text, width, 2)
		assertNothingWider(t, got, width)
		if len(got) == 0 {
			t.Fatalf("width %d returned no rows", width)
		}
		if flattenWords(strings.Join(got, "")) != flattenWords(text) {
			t.Fatalf("width %d lost content: %q", width, got)
		}
	}
	// Without a usable width the text is returned unwrapped instead of being
	// cut: the parent viewport still scrolls every row.
	got := Paragraph("hello world", 0, 2)
	if !reflect.DeepEqual(got, []string{"hello world"}) {
		t.Fatalf("width 0 must keep the source line intact, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// Section
// ---------------------------------------------------------------------------

func TestPresentationSectionSpacingAndIndent(t *testing.T) {
	got := Section("Title", []string{"one line", "two line"}, 40)
	want := []string{"Title", "", "  one line", "  two line"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPresentationSectionDoesNotDoubleBlanksOrIndents(t *testing.T) {
	got := Section("Title", []string{"", "one", "", "", "  already indented", ""}, 40)
	want := []string{"Title", "", "  one", "", "  already indented"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q want %q", got, want)
	}
	if hasDoubleBlank(got) {
		t.Fatalf("section contains a double blank: %q", got)
	}
}

func TestPresentationSectionWrapsWithinWidth(t *testing.T) {
	got := Section("Install and manage", []string{"A body sentence that wraps"}, 12)
	assertNothingWider(t, got, 12)
	if len(got) < 3 {
		t.Fatalf("narrow section should wrap, got %q", got)
	}
	blanks := 0
	for _, line := range got {
		if line == "" {
			blanks++
			continue
		}
		if blanks == 1 && !strings.HasPrefix(line, "  ") {
			t.Fatalf("body line lacks the 2-space indent: %q", line)
		}
		if strings.HasPrefix(line, "   ") {
			t.Fatalf("body line has doubled indent: %q", line)
		}
	}
	if blanks != 1 {
		t.Fatalf("section must have exactly one blank between title and body, got %q", got)
	}
}

func TestPresentationSectionEmptyBodyHasNoSeparatorBlank(t *testing.T) {
	if got := Section("Only title", nil, 40); !reflect.DeepEqual(got, []string{"Only title"}) {
		t.Fatalf("empty body must not add a separator blank: %q", got)
	}
	if got := Section("Only title", []string{"", "   "}, 40); !reflect.DeepEqual(got, []string{"Only title"}) {
		t.Fatalf("a blank-only body must collapse to the title: %q", got)
	}
	// A blank title must not turn into a leading blank row.
	if got := Section("", []string{"body line"}, 40); !reflect.DeepEqual(got, []string{"  body line"}) {
		t.Fatalf("blank title produced bad spacing: %q", got)
	}
}

// ---------------------------------------------------------------------------
// HomeOverview
// ---------------------------------------------------------------------------

func testOverviewCatalog() Catalog {
	return Catalog{
		Items: []CatalogItem{
			{
				ID: "core-skill", Kind: skillsync.InventoryKindSkill, Origin: "core",
				Clients: []ClientStatus{
					{Client: "claude", Status: skillsync.InventoryStatusMatchesSource},
					{Client: "opencode", Status: skillsync.InventoryStatusMissing},
				},
			},
			{
				ID: "pack-skill", Kind: skillsync.InventoryKindSkill, Origin: skillsync.InventoryOriginPack,
				Clients: []ClientStatus{
					{Client: "claude", Status: skillsync.InventoryStatusDiffersFromSource},
					{Client: "opencode", Status: skillsync.InventoryStatusMatchesSource},
					{Client: "cursor", Status: skillsync.InventoryStatusUnavailable},
				},
			},
			{
				ID: "free-first", Kind: skillsync.InventoryKindProfile, Origin: skillsync.InventoryOriginPreset,
				Clients: []ClientStatus{
					{Client: "claude", Status: skillsync.InventoryStatusMatchesSource},
				},
			},
			{
				ID: "pack-profile", Kind: skillsync.InventoryKindProfile, Origin: skillsync.InventoryOriginPack,
				Clients: []ClientStatus{
					{Client: "opencode", Status: skillsync.InventoryStatusMissing},
				},
			},
		},
		Notices: []string{"Example notice: optional source missing"},
	}
}

func TestHomeOverviewDerivesCountsFromCatalog(t *testing.T) {
	got := strings.Join(HomeOverview(testOverviewCatalog(), 90), "\n")

	for _, want := range []string{
		"Skills: 2",
		"Profiles: 2",
		"Core skills: 1",
		// Per-client coverage is scoped to the supported core skill unit and
		// always shows a present/total fraction with explicit modified text.
		"claude: present 1/1 (1 matching + 0 modified) · 0 not present · 0 refused",
		"opencode: present 0/1 (0 matching + 0 modified) · 1 not present · 0 refused",
		"OpenCode pack: present in this selection (2 items)",
		"Free-first presets: present in this selection (1 item)",
		"Example notice: optional source missing",
		"Set up / Install Agentic-SDD",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("overview missing %q:\n%s", want, got)
		}
	}
	// A pack-only destination is outside the core scope of this summary.
	if strings.Contains(got, "cursor:") {
		t.Fatalf("per-client coverage must stay scoped to core skills:\n%s", got)
	}
	// No fixed mock numbers and no runtime claims.
	for _, forbidden := range []string{
		"4 clients", "18 skills", "installed", "installing", "enabled",
		"authenticated", "approved", "running", "sandboxed", "runtime",
		"provider", "online", "connected",
	} {
		if strings.Contains(strings.ToLower(got), strings.ToLower(forbidden)) {
			t.Fatalf("overview must not claim %q:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(strings.ToLower(got), "read-only") {
		t.Fatalf("overview must state that it is read-only:\n%s", got)
	}
}

func TestHomeOverviewMarksAbsentOptionalContent(t *testing.T) {
	cat := Catalog{
		Items: []CatalogItem{{
			ID: "core-skill", Kind: skillsync.InventoryKindSkill, Origin: "core",
			Clients: []ClientStatus{{Client: "claude", Status: skillsync.InventoryStatusMatchesSource}},
		}},
		Notices: []string{"OpenCode pack source not found; showing core skills only"},
	}
	got := strings.Join(HomeOverview(cat, 60), "\n")
	for _, want := range []string{
		"Skills: 1",
		"Profiles: 0",
		"OpenCode pack: not present in this selection",
		"Free-first presets: not present in this selection",
		"OpenCode pack source not found; showing core skills only",
		"Recovery:",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("overview missing %q:\n%s", want, got)
		}
	}
	// A pack notice must not read as an empty inventory.
	if strings.Contains(got, "No managed skills") {
		t.Fatalf("pack notice was reported as an empty inventory:\n%s", got)
	}
}

func TestHomeOverviewEmptyCatalogIsReadable(t *testing.T) {
	got := HomeOverview(Catalog{}, 50)
	if len(got) == 0 {
		t.Fatal("empty catalog must still render a readable overview")
	}
	text := strings.Join(got, "\n")
	for _, want := range []string{
		"No managed skills",
		"Recovery:",
		"Set up / Install Agentic-SDD",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("empty overview missing %q:\n%s", want, text)
		}
	}
	// No client coverage may be fabricated for an empty inventory.
	if strings.Contains(text, "matching") || strings.Contains(text, "present 0/") {
		t.Fatalf("empty overview fabricated client coverage:\n%s", text)
	}
	if !strings.Contains(strings.ToLower(text), "read-only") {
		t.Fatalf("empty overview must state that it is read-only:\n%s", text)
	}
	if hasDoubleBlank(got) {
		t.Fatalf("empty overview contains a double blank: %q", got)
	}
	if got[0] == "" {
		t.Fatalf("empty overview must not start with a blank line: %q", got)
	}
}

func TestHomeOverviewRespectsWidth(t *testing.T) {
	cat := testOverviewCatalog()
	for _, width := range []int{1, 5, 10, 20, 40, 80} {
		lines := HomeOverview(cat, width)
		assertNothingWider(t, lines, width)
		if hasDoubleBlank(lines) {
			t.Fatalf("width %d produced a double blank: %q", width, lines)
		}
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			t.Fatalf("width %d produced a trailing blank line", width)
		}
	}
	// A degenerate width cannot bound rows, but nothing may be dropped.
	lines := HomeOverview(cat, 0)
	if hasDoubleBlank(lines) {
		t.Fatalf("width 0 produced a double blank: %q", lines)
	}
	if flattenWords(strings.Join(lines, "")) == "" {
		t.Fatal("width 0 dropped the overview content")
	}
}

func TestHomeOverviewLongPathsFullyRecoverable(t *testing.T) {
	home := "/home/user/" + strings.Repeat("nested-directory/", 12) + "HOME-END"
	repo := "/src/checkouts/" + strings.Repeat("very-long-segment/", 10) + "REPO-END"
	backup := "/var/backups/" + strings.Repeat("tree/", 20) + "BACKUP-END"
	notice := "Optional pack unavailable; showing core skills only"
	cat := Catalog{
		Home: home, Repo: repo, BackupRoot: backup,
		Notices: []string{notice},
		Items: []CatalogItem{{
			ID: "core-skill", Kind: skillsync.InventoryKindSkill, Origin: "core",
			Clients: []ClientStatus{{Client: "claude", Status: skillsync.InventoryStatusMissing}},
		}},
	}

	lines := HomeOverview(cat, 30)
	assertNothingWider(t, lines, 30)

	// Rows are reassembled the way a reader does: the per-row indent is
	// layout, everything else must survive wrapping in order.
	var recovered strings.Builder
	for _, line := range lines {
		recovered.WriteString(strings.TrimLeft(line, " "))
	}
	text := recovered.String()
	for _, want := range []string{home, repo, backup} {
		if !strings.Contains(text, want) {
			t.Fatalf("long path %q is not fully recoverable:\n%s", want, text)
		}
	}
	if !strings.Contains(flattenWords(text), flattenWords(notice)) {
		t.Fatalf("notice words are not preserved in order:\n%s", text)
	}
}

func TestHomeOverviewDeterministicAndReadOnly(t *testing.T) {
	build := func() Catalog {
		return Catalog{
			Items: []CatalogItem{{
				ID: "core-skill", Kind: skillsync.InventoryKindSkill, Origin: "core",
				Clients: []ClientStatus{
					{Client: "zed", Status: skillsync.InventoryStatusMissing},
					{Client: "alpha", Status: skillsync.InventoryStatusMatchesSource},
					{Client: "mid", Status: skillsync.InventoryStatusDiffersFromSource},
				},
			}},
		}
	}
	cat := build()
	first := HomeOverview(cat, 90)
	second := HomeOverview(cat, 90)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("overview is not deterministic:\n%q\n%q", first, second)
	}
	if !reflect.DeepEqual(cat, build()) {
		t.Fatalf("overview mutated the catalog it reads: %+v", cat)
	}

	joined := strings.Join(first, "\n")
	alpha := strings.Index(joined, "alpha:")
	mid := strings.Index(joined, "mid:")
	z := strings.Index(joined, "zed:")
	if alpha < 0 || mid < 0 || z < 0 || !(alpha < mid && mid < z) {
		t.Fatalf("clients must render in sorted order:\n%s", joined)
	}
	for _, want := range []string{
		"alpha: present 1/1 (1 matching + 0 modified) · 0 not present · 0 refused",
		"mid: present 1/1 (0 matching + 1 modified) · 0 not present · 0 refused",
		"zed: present 0/1 (0 matching + 0 modified) · 1 not present · 0 refused",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("overview missing %q:\n%s", want, joined)
		}
	}
}

func TestHomeOverviewUnknownMetadataIsNotFabricated(t *testing.T) {
	cat := Catalog{Items: []CatalogItem{
		{ID: "mystery-item"}, // no kind, origin, group or rich metadata
		{ID: "skill-without-origin", Kind: skillsync.InventoryKindSkill},
	}}
	got := strings.Join(HomeOverview(cat, 70), "\n")
	for _, want := range []string{
		"Skills: 1",
		"Profiles: 0",
		"Core skills: 0",
		"Other skills: 1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("overview missing %q:\n%s", want, got)
		}
	}
	for _, forbidden := range []string{
		"mystery-item", "Core skills: 1", "Purpose:", "Example:", "Caveat:",
	} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("overview fabricated %q:\n%s", forbidden, got)
		}
	}
}
