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

// displayRow is one display row plus its keep-with-next flag.
type displayRow struct {
	text         string
	keepWithNext bool
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

// expandDisplayRows is expandRows with pair tracking: each row carries its
// keepWithNext flag, and only the last wrapped chunk of a line inherits the
// original keep flag (internal wrap points are breakable). keep covers chord/
// lyric pairs and header→first-row units; keepTab covers tab blocks.
func expandDisplayRows(lines []string, keep, keepTab []bool, colWidth int) []displayRow {
	if colWidth <= 0 {
		colWidth = 1
	}
	rows := make([]displayRow, 0, len(lines))
	for i, line := range lines {
		kw := (keep != nil && keep[i]) || (keepTab != nil && keepTab[i])
		if displayWidth(line) <= colWidth {
			rows = append(rows, displayRow{text: line, keepWithNext: kw})
			continue
		}
		r := []rune(stripANSI(line))
		for start := 0; start < len(r); start += colWidth {
			end := min(start+colWidth, len(r))
			rows = append(rows, displayRow{text: string(r[start:end]), keepWithNext: end == len(r) && kw})
		}
	}
	return rows
}

// layoutColumns arranges lines into two side-by-side columns of the given
// height separated by a styled divider. Lines flow top-to-bottom in the left
// column, then continue in the right. Returns nil when the content fits in a
// single column or the width cannot fit two columns — callers fall back to
// single-column rendering.
//
// A chord/lyric pair, tab block, or section header with its first row (an
// unbreakable run) is never split: if a run would straddle the divider it is
// pushed entirely into the right column, and the bottom of the right column
// never ends inside a run that fits. A run taller than one column cannot be
// kept whole, so a break is allowed there. Whole sections may straddle the
// divider beyond their header+first-row.
func layoutColumns(lines []string, keep, keepTab []bool, width, height int, divider string) []string {
	if !shouldUseColumns(width) || height <= 0 {
		return nil
	}
	if divider == "" {
		divider = dividerGlyph
	}
	colWidth := (width - columnGap) / 2

	rows := expandDisplayRows(lines, keep, keepTab, colWidth)
	// Only split when there is more content than one column can hold.
	if len(rows) <= height {
		return nil
	}

	split := height
	if split < len(rows) && split > 0 && rows[split-1].keepWithNext {
		if s, e := displayRunBounds(rows, split-1); e-s+1 <= height {
			split = s
		}
	}
	bottom := min(split+height, len(rows))
	for bottom > split && bottom < len(rows) && bottom > 0 && rows[bottom-1].keepWithNext {
		if s, e := displayRunBounds(rows, bottom-1); e-s+1 > height {
			break
		} else {
			bottom = s
		}
	}

	pad := strings.Repeat(" ", colWidth)

	var out []string
	for row := 0; row < height; row++ {
		left := pad
		right := pad
		if row < split {
			left = padRight(rows[row].text, colWidth)
		}
		if split+row < bottom {
			right = padRight(rows[split+row].text, colWidth)
		}
		out = append(out, left+" "+divider+" "+right)
	}
	return out
}

// displayRunBounds returns the inclusive start/end indices of the maximal
// unbreakable run of display rows containing index i.
func displayRunBounds(rows []displayRow, i int) (int, int) {
	start := i
	for start > 0 && rows[start-1].keepWithNext {
		start--
	}
	end := i
	for end+1 < len(rows) && rows[end].keepWithNext {
		end++
	}
	return start, end
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
