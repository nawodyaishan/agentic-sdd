package tui

// Glyphs defines the centralized safe Unicode and 7-bit ASCII glyph vocabulary
// for Feature 013 presentation. Decorative emoji and Nerd Fonts are not used;
// all glyphs are single-cell and pair with clear textual labels.
type Glyphs struct {
	Cursor        string // "▸" in Unicode, ">" in ASCII (1 cell)
	Gutter        string // " " (single-cell space for unselected rows)
	DividerH      string // "─" in Unicode, "-" in ASCII
	DividerV      string // "│" in Unicode, "|" in ASCII
	CornerTL      string // "┌" in Unicode, "+" in ASCII
	CornerTR      string // "┐" in Unicode, "+" in ASCII
	CornerBL      string // "└" in Unicode, "+" in ASCII
	CornerBR      string // "┘" in Unicode, "+" in ASCII
	TeeLeft       string // "├" in Unicode, "+" in ASCII
	TeeRight      string // "┤" in Unicode, "+" in ASCII
	BreadcrumbSep string // " > " (standard single-cell separator)
	Bullet        string // "·" in Unicode, "-" in ASCII
	CheckMark     string // "[x]" (standard unambiguous checkbox)
	CheckEmpty    string // "[ ]" (standard unambiguous checkbox)
}

// GlyphSet returns the safe glyph set for the given ASCII setting.
func GlyphSet(ascii bool) Glyphs {
	if ascii {
		return Glyphs{
			Cursor:        ">",
			Gutter:        " ",
			DividerH:      "-",
			DividerV:      "|",
			CornerTL:      "+",
			CornerTR:      "+",
			CornerBL:      "+",
			CornerBR:      "+",
			TeeLeft:       "+",
			TeeRight:      "+",
			BreadcrumbSep: " > ",
			Bullet:        "-",
			CheckMark:     "[x]",
			CheckEmpty:    "[ ]",
		}
	}
	return Glyphs{
		Cursor:        "▸",
		Gutter:        " ",
		DividerH:      "─",
		DividerV:      "│",
		CornerTL:      "┌",
		CornerTR:      "┐",
		CornerBL:      "└",
		CornerBR:      "┘",
		TeeLeft:       "├",
		TeeRight:      "┤",
		BreadcrumbSep: " > ",
		Bullet:        "·",
		CheckMark:     "[x]",
		CheckEmpty:    "[ ]",
	}
}
