package ui

import (
	"strings"

	"github.com/muesli/reflow/ansi"
)

const (
	// minColumnWidth is the narrowest readable column; two-column mode needs
	// room for two of these plus the divider gap.
	minColumnWidth = 40
	// columnGap is the exact width of " ║ ".
	columnGap = 3
	// dividerGlyph separates the columns.
	dividerGlyph = "║"
)

func shouldUseColumns(width int) bool {
	return width >= 2*minColumnWidth+columnGap
}

// displayWidth is the number of cells a string occupies on screen. ANSI
// escape sequences (lipgloss styles on headers/comments) count as zero.
func displayWidth(s string) int {
	return ansi.PrintableRuneWidth(s)
}

// stripANSI removes CSI escape sequences (SGR styles) from s.
func stripANSI(s string) string {
	var sb strings.Builder
	inEscape := false
	for _, r := range s {
		switch {
		case inEscape:
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
		case r == '\x1b':
			inEscape = true
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// expandRows soft-wraps rendered lines to colWidth, returning one string per
// display row. Lines that fit pass through untouched so their styles stay
// intact; only over-length lines are wrapped, stripped to plain text first
// (in practice those are unstyled chord/tab rows).
func expandRows(lines []string, colWidth int) []string {
	if colWidth <= 0 {
		colWidth = 1
	}
	var rows []string
	for _, line := range lines {
		if displayWidth(line) <= colWidth {
			rows = append(rows, line)
			continue
		}
		r := []rune(stripANSI(line))
		for start := 0; start < len(r); start += colWidth {
			end := min(start+colWidth, len(r))
			rows = append(rows, string(r[start:end]))
		}
	}
	return rows
}

// layoutColumns arranges lines into two side-by-side columns of the given
// height separated by a styled divider. Lines flow top-to-bottom in the left
// column, then continue in the right. Returns nil when the content fits in a
// single column or the width cannot fit two columns — callers fall back to
// single-column rendering.
func layoutColumns(lines []string, width, height int, divider string) []string {
	if !shouldUseColumns(width) || height <= 0 {
		return nil
	}
	if divider == "" {
		divider = dividerGlyph
	}
	colWidth := (width - columnGap) / 2

	rows := expandRows(lines, colWidth)
	// Only split when there is more content than one column can hold.
	if len(rows) <= height {
		return nil
	}

	pad := strings.Repeat(" ", colWidth)

	var out []string
	for row := 0; row < height; row++ {
		left := pad
		right := pad
		if row < len(rows) {
			left = padRight(rows[row], colWidth)
		}
		if height+row < len(rows) {
			right = padRight(rows[height+row], colWidth)
		}
		out = append(out, left+" "+divider+" "+right)
	}
	return out
}

// padRight pads s with spaces so its display width reaches width. Styled
// strings keep their escape codes; only printable cells are counted.
func padRight(s string, width int) string {
	d := displayWidth(s)
	if d >= width {
		return s
	}
	return s + strings.Repeat(" ", width-d)
}
