package ui

import "strings"

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

// expandRows soft-wraps rendered lines to colWidth, returning one string per
// display row. Nothing is dropped: a line longer than colWidth becomes
// consecutive rows.
func expandRows(lines []string, colWidth int) []string {
	if colWidth <= 0 {
		colWidth = 1
	}
	var rows []string
	for _, line := range lines {
		r := []rune(line)
		if len(r) == 0 {
			rows = append(rows, "")
			continue
		}
		for start := 0; start < len(r); start += colWidth {
			end := min(start+colWidth, len(r))
			rows = append(rows, string(r[start:end]))
		}
	}
	return rows
}

// layoutColumns arranges lines into two side-by-side columns of the given
// height separated by a styled divider. Lines flow top-to-bottom in the left
// column, then continue in the right. The right column is blank-filled when
// content runs out. Returns nil when the width cannot fit two columns.
func layoutColumns(lines []string, width, height int, divider string) []string {
	if !shouldUseColumns(width) || height <= 0 {
		return nil
	}
	if divider == "" {
		divider = dividerGlyph
	}
	colWidth := (width - columnGap) / 2

	rows := expandRows(lines, colWidth)
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

func padRight(s string, width int) string {
	r := []rune(s)
	if len(r) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(r))
}